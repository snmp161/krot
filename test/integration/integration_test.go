//go:build integration

package integration

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

var types = []string{"wg", "awg", "awg2", "awg3"}

// SPEC 8.3: полный цикл для каждого типа: create → add-client → клиент в соседнем netns →
// ping → lock (трафик прекращается) → unlock (восстанавливается) → delete-client → delete.
func TestFullCycle(t *testing.T) {
	for i, typ := range types {
		t.Run(typ, func(t *testing.T) {
			tu := newTunnel(t, typ, i)
			tu.create(t, tu.serverIP()+"/24")
			if it := tu.list(t); it.Status != "up" || it.Type != typ {
				t.Fatalf("after create: %+v", it)
			}

			cfg := tu.addClient(t, "a")
			tu.clientUp(t, cfg)
			waitPing(t, tu.serverIP(), 20*time.Second)

			mustKrot(t, "--lock-client", "--interface", tu.name, "--id", "a")
			expectNoPing(t, tu.serverIP())
			if it := tu.list(t); it.ClientsLocked != 1 {
				t.Fatalf("after lock: %+v", it)
			}

			mustKrot(t, "--unlock-client", "--interface", tu.name, "--id", "a")
			waitPing(t, tu.serverIP(), 45*time.Second)

			mustKrot(t, "--delete-client", "--interface", tu.name, "--id", "a")
			expectNoPing(t, tu.serverIP())
			if peers := tu.livePeers(t); len(peers) != 0 {
				t.Fatalf("live peers after delete-client: %q", peers)
			}

			mustKrot(t, "--down", "--interface", tu.name)
			if it := tu.list(t); it.Status != "down" {
				t.Fatalf("after down: %+v", it)
			}
			mustKrot(t, "--up", "--interface", tu.name)
			if out := mustKrot(t, "--up", "--interface", tu.name); !strings.Contains(out, "already up") {
				t.Fatalf("second --up: %q", out)
			}

			mustKrot(t, "--delete", "--interface", tu.name, "--yes")
			if _, err := os.Stat(tu.conf); !os.IsNotExist(err) {
				t.Fatalf("config still exists: %v", err)
			}
			if _, err := run("ip", "-n", nsSrv, "link", "show", "dev", tu.name); err == nil {
				t.Fatal("interface still exists after --delete")
			}
		})
	}
}

var seqRe = regexp.MustCompile(`icmp_seq=(\d+)`)

// SPEC 6.10, 8.3: тест незаметности. Клиент A пингует сервер каждые 0.1 с, в это время
// 20 циклов add/lock/unlock/delete для B, C, D. Потерь у A нет, ifindex не меняется,
// latest-handshake у A не сбрасывается внепланово, конфиг на диске — как ожидается.
// Этот тест нельзя отключать, пропускать или ослаблять (AGENTS.md).
func TestInvisibility(t *testing.T) {
	for i, typ := range types {
		t.Run(typ, func(t *testing.T) {
			tu := newTunnel(t, typ, 10+i)
			tu.create(t, tu.serverIP()+"/24")
			tu.clientUp(t, tu.addClient(t, "a"))
			waitPing(t, tu.serverIP(), 20*time.Second)

			pubA := tu.clientKey(t, "a")
			ifindex := tu.ifindex(t)
			hs := tu.handshake(t, pubA)
			confBefore := readFile(t, tu.conf)

			cmd := exec.Command("ip", "netns", "exec", nsCli, "ping", "-n", "-i", "0.1", "-W", "1", tu.serverIP())
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			var seqs []int
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				s := bufio.NewScanner(stdout)
				for s.Scan() {
					if m := seqRe.FindStringSubmatch(s.Text()); m != nil {
						n, _ := strconv.Atoi(m[1])
						seqs = append(seqs, n)
					}
				}
				io.Copy(io.Discard, stdout)
			}()

			start := time.Now()
			for cycle := 0; cycle < 20; cycle++ {
				for _, id := range []string{"b", "c", "d"} {
					tu.addClient(t, id)
					mustKrot(t, "--lock-client", "--interface", tu.name, "--id", id)
					mustKrot(t, "--unlock-client", "--interface", tu.name, "--id", id)
					mustKrot(t, "--delete-client", "--interface", tu.name, "--id", id)
				}
			}
			elapsed := time.Since(start)
			time.Sleep(time.Second) // последние ответы
			cmd.Process.Signal(syscall.SIGINT)
			wg.Wait()
			cmd.Wait()

			// Потерь нет: все icmp_seq подряд; последний запрос мог быть ещё в пути при остановке.
			if len(seqs) < int(elapsed/(100*time.Millisecond))/2 {
				t.Fatalf("too few replies (%d) for %s of cycles", len(seqs), elapsed)
			}
			for k, s := range seqs {
				if s != k+1 {
					t.Fatalf("packet loss for client A: expected icmp_seq=%d, got %d (replies %d)", k+1, s, len(seqs))
				}
			}
			if got := tu.ifindex(t); got != ifindex {
				t.Fatalf("ifindex changed: %d -> %d", ifindex, got)
			}
			// Плановый rekey возможен не раньше чем через RekeyAfterTime (>= 100 с у krot).
			if got := tu.handshake(t, pubA); got != hs && time.Since(time.Unix(hs, 0)) < 100*time.Second {
				t.Fatalf("latest-handshake of A changed unexpectedly: %d -> %d", hs, got)
			}
			if got := readFile(t, tu.conf); got != confBefore {
				t.Fatalf("config differs after the cycles:\n--- before ---\n%s\n--- after ---\n%s", confBefore, got)
			}
			if peers := tu.livePeers(t); !slices.Equal(peers, []string{pubA}) {
				t.Fatalf("live peers = %q; want only A", peers)
			}
			t.Logf("%s: %d replies without loss during %s of client operations", typ, len(seqs), elapsed)
		})
	}
}

// SPEC 8.3: dual-stack туннель, ping6 через туннель.
func TestDualStack(t *testing.T) {
	for i, typ := range []string{"wg", "awg2"} {
		t.Run(typ, func(t *testing.T) {
			tu := newTunnel(t, typ, 20+i)
			v6 := fmt.Sprintf("fd66:%d::", 20+i)
			tu.create(t, tu.serverIP()+"/24,"+v6+"1/64")
			cfg := mustKrot(t, "--add-client", "--interface", tu.name, "--id", "dual",
				"--allowed-ips", tu.subnet+".0/24,"+v6+"/64")
			if !strings.Contains(cfg, v6+"2/128") {
				t.Fatalf("client has no IPv6 address:\n%s", cfg)
			}
			tu.clientUp(t, cfg)
			waitPing(t, tu.serverIP(), 20*time.Second)
			deadline := time.Now().Add(15 * time.Second)
			for !ping("-6", "-c", "1", "-W", "1", v6+"1") {
				if time.Now().After(deadline) {
					t.Fatal("ping6 through the tunnel failed")
				}
			}
		})
	}
}

// SPEC 5.3, 8.3: чужой конфиг, созданный вручную: list, list-client, add-client
// сохраняют чужие строки.
func TestForeignConfig(t *testing.T) {
	tu := newTunnel(t, "wg", 30)
	priv := strings.TrimSpace(mustRun(t, "wg", "genkey"))
	peerPub := strings.TrimSpace(mustRun(t, "sh", "-c", "wg genkey | wg pubkey"))
	foreign := "# Hand-written config, do not touch\n" +
		"[Interface]\n" +
		"PrivateKey = " + priv + "\n" +
		"Address = " + tu.serverIP() + "/24   # server\n" +
		"ListenPort = " + strconv.Itoa(tu.port) + "\n" +
		"\n" +
		"# Bob's phone\n" +
		"[Peer]\n" +
		"PublicKey = " + peerPub + "\n" +
		"AllowedIPs = " + tu.subnet + ".2/32\n"
	if err := os.MkdirAll(wgDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tu.conf, []byte(foreign), 0o600); err != nil {
		t.Fatal(err)
	}
	mustKrot(t, "--up", "--interface", tu.name)

	if it := tu.list(t); it.Type != "wg" || it.Status != "up" || it.ClientsTotal != 1 {
		t.Fatalf("--list: %+v", it)
	}
	if cs := tu.clients(t); len(cs) != 1 || cs[0].ID != "-" || cs[0].PublicKey != peerPub {
		t.Fatalf("--list-client: %+v", cs)
	}
	// Без krot:endpoint нужен --endpoint (SPEC 6.5).
	if _, _, code := krot("--add-client", "--interface", tu.name, "--id", "n1", "--allowed-ips", tu.subnet+".0/24"); code != 3 {
		t.Fatalf("add-client without endpoint: exit %d; want 3", code)
	}
	cfg := tu.addClient(t, "n1", "--endpoint", srvAddr)
	if got := readFile(t, tu.conf); !strings.HasPrefix(got, foreign) {
		t.Fatalf("foreign lines changed:\n%s", got)
	}
	tu.clientUp(t, cfg)
	waitPing(t, tu.serverIP(), 20*time.Second)
	mustKrot(t, "--delete", "--interface", tu.name, "--yes")
}

// SPEC 5.5, 8.3: 10 одновременных --add-client — все клиенты в конфиге, адреса уникальны.
func TestParallelAddClient(t *testing.T) {
	tu := newTunnel(t, "wg", 40)
	tu.create(t, tu.serverIP()+"/24")
	var wg sync.WaitGroup
	errs := make(chan string, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := []string{"--add-client", "--interface", tu.name, "--id", fmt.Sprintf("p%d", i), "--allowed-ips", tu.subnet + ".0/24"}
			if out, errOut, code := krot(args...); code != 0 {
				errs <- fmt.Sprintf("p%d: exit %d: %s%s", i, code, out, errOut)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	cs := tu.clients(t)
	if len(cs) != 10 {
		t.Fatalf("got %d clients; want 10", len(cs))
	}
	seen := map[string]bool{}
	for _, c := range cs {
		for _, a := range c.Addresses {
			if seen[a] {
				t.Fatalf("duplicate address %s", a)
			}
			seen[a] = true
		}
	}
	if peers := tu.livePeers(t); len(peers) != 10 {
		t.Fatalf("live peers = %d; want 10", len(peers))
	}
}
