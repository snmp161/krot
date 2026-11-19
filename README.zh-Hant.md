# krot

[English](README.md) | [Русский](README.ru.md) | [简体中文](README.zh.md) | **繁體中文** | [Español](README.es.md) | [Português](README.pt.md) | [한국어](README.ko.md) | [日本語](README.ja.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [Deutsch](README.de.md)

`krot` 是一款用於在 Linux 伺服器上管理 **WireGuard** 與 **AmneziaWG**
（1.x、2.0、3.1）隧道的命令列工具：建立與刪除介面、管理客戶端（peer）、
產生客戶端設定檔、QR Code 與 AmneziaVPN 金鑰（`vpn://…`）。

名稱 **krot** 在俄語及其他部分斯拉夫語中意為「鼴鼠」——一種挖掘隧道的動物。

- 單一靜態二進位檔，無額外依賴（Go，`CGO_ENABLED=0`）。
- 客戶端操作**絕不重新啟動或重建介面**：僅執行精確的
  `wg|awg set <if> peer …` 呼叫——其他客戶端不受影響。
- 介面設定檔是唯一事實來源：不需額外資料庫；krot 元資料儲存在註解中，
  並非由 krot 建立的設定檔也會被讀取並保留。
- 輸出結果可預期：供人閱讀的表格，以及供自動化使用的 `--json` 與結束碼。
- krot 不會設定 NAT、轉送或防火牆規則。

完整技術規格：[`docs/SPEC.md`](docs/SPEC.md)。

## 系統需求

| | |
|---|---|
| 作業系統 | Ubuntu 22.04 / 24.04 / 26.04、Debian 12 / 13、AlmaLinux（Rocky、RHEL）9 / 10 |
| 架構 | `amd64`、`arm64` |
| `--type wg` | `wireguard-tools`（`wg`、`wg-quick`）、WireGuard 核心模組 |
| `--type awg*` | `amneziawg-tools`（`awg`、`awg-quick`）與 AmneziaWG 核心模組**或** `amneziawg-go` |
| 一律需要 | `iproute2`（`ip`）、`systemd` |
| 權限 | root（`--help`、`--version`、`--doctor`、`--udp-checker` 除外；`--udp-listener` 僅在連接埠 `<1024` 或 `/proc/net/udp{,6}` 無法讀取時需要 root） |

krot 不會自行安裝上述任何元件。若缺少必要程式，將以結束碼 `6`
結束，並指出需要安裝的套件名稱。

### AmneziaWG 版本

在執行 `--create` 前，krot 會檢查已安裝的 AmneziaWG 是否支援所要求的類型：

| 類型 | amneziawg-tools | 核心模組 | netlink family | amneziawg-go |
|---|---|---|---|---|
| `awg`（1.0） | 1.0.20240213 | — | ≥ 1 | 0.1.0 |
| `awg` 搭配 `--i1`（1.5） | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.13 |
| `awg2`（2.0） | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.15 |
| `awg3`（3.1） | 3.1.20260812 | 3.1.20260812 | ≥ 3 | 3.1.20260812 |

部分建置會回報預留版本號（例如，1.x 工具的 `awg --version`
幾乎總是顯示 `1.0.20210914`，而使用 `make` 建置的模組會回報 `1.0.0`）。
這類版本會產生警告而非錯誤；只有在可靠識別版本且版本確實過舊時，
才會回傳結束碼 `6`。若版本確實過舊，unit 將無法啟動，krot 會還原變更。

## 安裝

請從發布頁面 <https://github.com/snmp161/krot/releases> 下載套件或
二進位檔進行安裝，並務必驗證校驗和。

```sh
VERSION=0.7.1
BASE=https://github.com/snmp161/krot/releases/download/v$VERSION

# 下載套件與校驗和清單
wget "$BASE/krot_${VERSION}_amd64.deb" "$BASE/SHA256SUMS"
# 驗證：應輸出 "krot_..._amd64.deb: OK"
sha256sum --check --ignore-missing SHA256SUMS
# 安裝
sudo dpkg -i "krot_${VERSION}_amd64.deb"
```

使用 RPM 的系統（AlmaLinux、Rocky、RHEL）：

```sh
wget "$BASE/krot-${VERSION}-1.x86_64.rpm" "$BASE/SHA256SUMS"
sha256sum --check --ignore-missing SHA256SUMS
sudo rpm -i "krot-${VERSION}-1.x86_64.rpm"
```

不使用套件的安裝方式：下載 `krot-linux-amd64`（或 `krot-linux-arm64`）
與 `SHA256SUMS`，以相同方式驗證，然後執行
`sudo install -m 0755 krot-linux-amd64 /usr/bin/krot`。

若使用 `arm64`，請選用 `*_arm64.deb`、`*.aarch64.rpm` 與 `krot-linux-arm64` 檔案。

## 使用方式

```
krot <操作> [旗標]
```

每次執行只能指定一個操作；旗標僅支援長格式 `--flag value` 或
`--flag=value`。使用不適用於該操作的旗標會產生錯誤（結束碼 `2`）。
完整說明：`krot --help`。

### 隧道

```sh
# 建立 AmneziaWG 2.0（雙重協定堆疊）；名稱自動選擇：awg0、awg1、…
sudo krot --create --type awg2 --ip 10.8.0.1/24,fd00:8::1/64 --port 51820 \
          --endpoint vpn.example.com --dns 1.1.1.1,9.9.9.9

# 使用明確名稱與 MTU 建立 WireGuard
sudo krot --create --type wg --interface wg0 --ip 10.9.0.1/24 --port 51821 \
          --endpoint 203.0.113.10 --mtu 1420

# 含 I1 特徵封包的 AmneziaWG 1.5（krot 會驗證 CPS 語法與長度——最多 2048 字元）
sudo krot --create --type awg --ip 10.10.0.1/24 --port 51822 --endpoint vpn.example.com \
          --i1 '<b 0xf6ab3267fa><r 16>'

# AmneziaWG 3.1（一律啟用 Header Protection、RandomTrailers、DisableCookies）
sudo krot --create --type awg3 --ip 10.11.0.1/24 --port 51823 --endpoint vpn.example.com

sudo krot --list                         # 所有隧道
sudo krot --list --type awg2 --json      # 篩選器 --type/--interface/--ip/--port，條件採 AND
sudo krot --down --interface awg0        # systemctl stop awg-quick@awg0
sudo krot --up --interface awg0          # systemctl start；重複 --up 顯示 "already up"，結束碼 0
sudo krot --delete --interface awg0      # 若有客戶端，會要求確認
sudo krot --delete --interface awg0 --yes
```

### 診斷

```sh
krot --doctor   # 不需 root：工具、核心模組、AmneziaWG 版本、目錄
```

輸出為 `CHECK/STATUS/DETAIL` 表格，狀態為 `ok` / `warn` / `fail`。
若沒有 `fail` 項目，結束碼為 `0`，否則為 `6`。此命令不會變更任何內容，也不會取得鎖。

#### UDP 連接埠診斷

`krot` 可檢查兩台主機之間實際可通的 UDP 連接埠。當託管供應商或其上游
過濾部分連接埠（例如因 DDoS 防護規則）時，此功能特別有用。

在伺服器上啟動 listener：

```sh
# 單一連接埠
sudo krot --udp-listener --port 51820

# 連接埠範圍（最多 1024 個連接埠，每個連接埠一個 socket）
sudo krot --udp-listener --port 51820:51830
```

Listener 回應三種資料報：
- `ping SEQ:N` → `pong SEQ:N`；
- `ECHO <text>` → `<text>`；
- `TIME` → 目前 Unix 時間（毫秒）。

在客戶端執行 checker：

```sh
# 檢查單一連接埠
krot --udp-checker --address 203.0.113.1 --port 51820

# 檢查連接埠範圍
krot --udp-checker --address 203.0.113.1 --port 51820:51830

# 增加 echo 酬載大小（1–1400 位元組，預設 64）——有助於找出最佳 MTU
krot --udp-checker --address 203.0.113.1 --port 51820 --echo-size 1400

# 設定 worker 數量並顯示進度
krot --udp-checker --address 203.0.113.1 --port 51820:51830 --workers 64 --verbose
```

預設情況下，checker 會對每個連接埠發送 3 個各類型的封包（`--count`，
1–100），並在 `--timeout`（預設 2s）內等待回應。預設並行檢查 32 個連接埠，
`--workers` 可設定為 1–100。`--verbose` 會將連接埠範圍與各連接埠完成後的結果
輸出至 stderr；在 listener 中則會記錄客戶端位址與請求類型。輸出：

```
PORT     STATUS   PING    ECHO    TIME (min/avg/max)   DETAIL
51820    ok       3/3     3/3     12/24/45 ms          -
51821    filtered 0/3     0/3     -                    timeout
```

狀態：
- `ok` — 所有 ping 與 echo 封包都收到有效回應；
- `filtered` — 至少一個封包逾時（該連接埠很可能遭到過濾）；
- `error` — 收到無效回應。

使用 `--json` 時，輸出為物件陣列，欄位包括 `port`、`status`、
`ping_sent`、`ping_ok`、`echo_sent`、`echo_ok`、`time_sent`、`time_ok`、
`rtt_min_ms`、`rtt_avg_ms`、`rtt_max_ms`、`detail`。

按下 `Ctrl+C` 或收到 `SIGTERM` 時，checker 會以選定格式輸出已完整檢查的連接埠結果，
並以結束碼 `10` 結束；未完成的連接埠不會包含在結果中。

`--udp-checker` 不需 root。`--udp-listener` 僅在範圍包含 `<1024` 連接埠，
或目前使用者無法讀取 `/proc/net/udp` 與 `/proc/net/udp6` 時需要 root。
Listener 會持續執行，直到收到 `Ctrl+C`/`SIGTERM`。

`--list` 範例：

```
INTERFACE  TYPE  VERSION  ADDRESS                   PORT   STATUS  CLIENTS
awg0       awg2  2.0      10.8.0.1/24,fd00:8::1/64  51820  up      12 (1 locked)
wg0        wg    -        10.9.0.1/24               51821  down    3
```

預設 MTU：`wg` — 1420，`awg`/`awg2` — 1376，`awg3` — 1280。
混淆參數（Jc、S1–S4、H1–H4 等）會在安全範圍內自動產生並通過驗證
（規格第 7 節）。

### 客戶端

```sh
# 新增客戶端：位址自動分配（第一個可用的 /32 與 /128）；
# 客戶端設定檔僅顯示一次——客戶端私鑰絕不儲存
sudo krot --add-client --interface awg0 --id phone-sergei --allowed-ips 0.0.0.0/0,::/0

# 含 PresharedKey 與終端機 QR Code
sudo krot --add-client --interface awg0 --id laptop --allowed-ips 0.0.0.0/0 --psk --qr

# 額外輸出 AmneziaVPN 金鑰：設定檔後的 vpn://… 行
sudo krot --add-client --interface awg0 --id tablet --allowed-ips 0.0.0.0/0,::/0 --amnezia-key

# KeeneticOS CLI 命令：將 Keenetic 路由器連接為客戶端（僅限 wg，僅限 IPv4）
sudo krot --add-client --interface wg0 --id keenetic --allowed-ips 0.0.0.0/0 --keenetic

# 指定客戶端位址
sudo krot --add-client --interface awg0 --id router --allowed-ips 10.8.0.0/24 --ip 10.8.0.50/32

# 非 krot 建立的設定檔：明確傳入 endpoint（不會寫入伺服器設定檔）
sudo krot --add-client --interface office --id guest --allowed-ips 0.0.0.0/0 --endpoint 203.0.113.10

# 為單一客戶端自訂 DNS（不會寫入伺服器設定檔）
sudo krot --add-client --interface awg0 --id kid-phone --allowed-ips 0.0.0.0/0 --dns 94.140.14.15

sudo krot --list-client --interface awg0
sudo krot --list-client --interface awg0 --id phone-sergei --json
sudo krot --lock-client   --interface awg0 --id laptop    # 重複執行——顯示 "already locked"，結束碼 0
sudo krot --unlock-client --interface awg0 --id laptop
sudo krot --delete-client --interface awg0 --id laptop
```

`--allowed-ips` 會設定**客戶端**設定檔中的 `AllowedIPs`（客戶端透過隧道
轉送的內容）。客戶端 DNS：若有指定 `--dns` 則使用；否則使用介面的
`# krot:dns` 元資料（`--create` 時的 `--dns`）；再否則使用伺服器設定檔
`[Interface]` 中的 `DNS` 行；若各處皆未設定 DNS，客戶端設定檔就不含 `DNS` 行。

客戶端位址從介面子網路分配，並排除網路位址、廣播位址與伺服器位址。
亦支援點對點子網路 `/31` 與 `/127`（RFC 3021、RFC 6164）：兩個位址
皆可使用，因此可透過 `--create … --ip 10.8.0.0/31,fd00:8::/127`
建立「伺服器 + 單一客戶端」隧道。

請立即儲存產生的設定檔：之後無法再次取得，只能刪除客戶端後重新新增。
QR Code 使用 Unicode 半形方塊字元顯示，並假設終端機使用深色背景。

`--amnezia-key` 會在設定檔後顯示 AmneziaVPN 金鑰——單行 `vpn://…`，
可貼到 AmneziaVPN 應用程式（「插入金鑰」），例如透過通訊軟體傳送。
其內容為同一份客戶端設定檔：`vpn://` + Qt `qCompress` 的 base64url
（不含 `=`）（4 位元組大端序長度 + zlib）。此金鑰適用於所有類型，包括 `wg`。
`--qr` 產生的 QR Code 包含設定檔文字——AmneziaVPN 與 AmneziaWG 可掃描；
對於 `wg`，WireGuard 也可掃描。

`--keenetic` 會在設定檔後顯示 KeeneticOS CLI 命令區塊——將 Keenetic 路由器
連接為隧道客戶端。此區塊遵循 KeeneticOS 的 running-config 語法，可原樣
貼到路由器 CLI（SSH/telnet 或 Web 介面的「命令列介面」）。KeeneticOS
支援 AmneziaWG 混淆（ASC 參數）：AWG 1.0 自 KeeneticOS 4.2 起支援，
AWG 1.5/2.0 自 5.1 起支援；對於 awg 隧道，此區塊包含 `wireguard asc …` 行
（CPS 字串 I1–I5 會加上引號；透過 Web 介面匯入設定檔較為可靠）。
不支援 AWG 3.1。僅限 IPv4；若隧道 MTU 與預設值 1420 不同，會顯示提示。
格式請見規格第 6.13 節。

`--list-client` 範例：

```
ID            PUBLIC KEY  ADDRESS                    STATUS  LAST HANDSHAKE  RX       TX
phone-sergei  aB3d…Xy9=   10.8.0.2/32,fd00:8::2/128  active  1m ago          12.4MiB  1.1GiB
laptop-old    Qw7e…Zz1=   10.8.0.3/32                locked  -               -        -
```

### 偽裝（I1–I5 封包）

自 1.5 版起，AmneziaWG 可在每次握手前發送最多五個特徵封包
`I1`–`I5`：這些封包看起來像其他協定（DNS、STUN、NTP、SSDP、DTLS）
的開頭。krot 透過 `--mimic <profile>` 旗標自行產生此類封包序列：

```sh
# 介面所有客戶端的預設 profile
krot --create --type awg --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --mimic dns
# 為單一客戶端覆寫
krot --add-client --interface awg0 --id phone --allowed-ips 0.0.0.0/0 --mimic stun
```

Profile：`dns`（三個 A/AAAA/HTTPS 查詢）、`stun`（Binding Request）、
`ntp`、`ssdp`（M-SEARCH）、`dtls`（ClientHello）。封包序列會在 `--add-client`
時產生（每個客戶端各自獨立），且僅寫入客戶端設定檔：I 封包由握手發起方
發送，伺服器不需要這些封包。客戶端需要 AmneziaWG 1.5+——請見規格第 6.12 節。

除內建 profile 外，krot 也接受在 `--create` 時透過 `--i1` 傳入現成的
CPS 字串——例如用於 QUIC：內建 QUIC profile 不會產生，因為單一 QUIC
Initial 長達 2400–2500 字元，超過 2048 上限（原因請見
[amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69)）。
現成字串可從第三方產生器取得（兩者皆採 MIT 授權，與 krot 無關聯）：

- [payloadGen](https://github.com/Sketchystan1/payloadGen) — 網頁：
  在瀏覽器中開啟 `index.html`，選擇協定，取第一行（I1）；
- 來自 [awg-multi-script](https://github.com/pumbaX/awg-multi-script)
  的 `cpsgen.py`（`src/py/cpsgen.py`）— payloadGen 的 Python 3
  主控台版本，不需 root。Profile：`dns`、`ntp`、`rtp`、`stun`、`sip`、
  `ssdp`、`webrtc`、`quic`、`curl_quic`；第二個參數為網域名稱。

```sh
# 下載產生器，並在執行前檢查其程式碼
git clone --depth 1 https://github.com/pumbaX/awg-multi-script
less awg-multi-script/src/py/cpsgen.py

# 僅產生 I1 並檢查長度（必須 ≤ 2048 字元）
I1=$(python3 awg-multi-script/src/py/cpsgen.py dns example.com --only-i1)
echo ${#I1}

sudo krot --create --type awg2 --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --i1 "$I1"
```

注意事項：

- `I1` 僅能在 `--create` 時設定，會寫入伺服器設定檔並複製到所有客戶端設定檔。
  krot 不接受 `I2`–`I5`。
- 長度不得超過 2048 字元（見[限制](#限制)），否則結束碼為 `3`。
  `quic` 與 `curl_quic` profile 的封包約為 2400 字元，無法容納。
  較短的 profile 可用：`rtp`（約 30 字元）、`dns` 與 `ntp`（約 100）、
  `stun`（約 250–350）、`ssdp`（約 400）、`sip`（約 1200）。
- `--type awg` 搭配 `--i1` 表示 AmneziaWG 1.5，並有各自的最低版本需求
  （見 [AmneziaWG 版本](#amneziawg-版本)）。`<rc N>`/`<rd N>` 標籤
  （由 `sip` profile 產生）自 0.2.15 起受 amneziawg-go 支援，因此 1.5
  建議使用不含這些標籤的 profile。
- 客戶端必須支援 AmneziaWG 1.5 或更新版本。
- 使用較長的 `I1` 時，設定檔可能無法放入 QR Code：`--qr` 將以結束碼 `3`
  結束（客戶端已新增且設定檔已顯示）。請以檔案傳送設定檔或使用
  `--amnezia-key`。

### JSON

`--list --json` 輸出物件陣列，欄位包括 `interface`、`type`、
`version`、`addresses[]`、`port`、`status`、`clients_total`、
`clients_locked`。`--list-client --json` 輸出物件陣列，欄位包括 `id`、
`public_key`、`addresses[]`、`status`、`latest_handshake`（Unix 時間或
`null`）、`rx_bytes`、`tx_bytes`、`endpoint`（或 `null`）、`created`
（非 krot 管理的 peer 為 `null`）。空結果為 `[]`。

## 結束碼

| 代碼 | 含義 |
|---|---|
| 0 | 成功 |
| 1 | 內部或非預期錯誤（包括無法將變更套用至執行中的介面——設定檔會還原） |
| 2 | CLI 使用錯誤（旗標；有客戶端時未使用 `--yes` 且 stdin 非終端機的刪除操作） |
| 3 | 值驗證錯誤（包括缺少 endpoint、「config too large for QR」） |
| 4 | 找不到物件（介面、客戶端） |
| 5 | 衝突：已存在、位址或連接埠被佔用、無可用位址 |
| 6 | 環境：無 root、缺少工具、AmneziaWG 版本不受支援 |
| 7 | 10 秒內無法取得 `/run/lock/krot.lock` 鎖 |
| 10 | 使用者取消操作 |

一般輸出寫入 stdout；錯誤與警告寫入 stderr，並帶有 `krot:` 前綴。

## 資料儲存

| 類型 | 設定檔 | systemd unit | 工具 |
|---|---|---|---|
| `wg` | `/etc/wireguard/<if>.conf` | `wg-quick@<if>` | `wg` |
| `awg`、`awg2`、`awg3` | `/etc/amnezia/amneziawg/<if>.conf` | `awg-quick@<if>` | `awg` |

元資料以 `wg-quick`/`awg-quick` 會忽略的註解形式儲存：

```ini
# krot:type=awg2
# krot:endpoint=vpn.example.com
# krot:dns=1.1.1.1,9.9.9.9
[Interface]
PrivateKey = ...
Address = 10.8.0.1/24, fd00:8::1/64
ListenPort = 51820
MTU = 1376
Jc = 6
...
# PreUp =
# PostUp =
# PreDown =
# PostDown =

# krot:id=phone-sergei
# krot:created=2026-10-17T01:30:00Z
[Peer]
PublicKey = ...
AllowedIPs = 10.8.0.2/32, fd00:8::2/128

# krot:id=laptop-old
# krot:created=2026-10-01T10:00:00Z
# krot:locked=2026-10-16T12:00:00Z
#~ [Peer]
#~ PublicKey = ...
#~ AllowedIPs = 10.8.0.3/32
```

- 已鎖定的客戶端是以 `#~ ` 前綴註解掉的 `[Peer]` 區塊。
- 已註解的 `PreUp`/`PostUp`/`PreDown`/`PostDown` hook 可取消註解並
  填寫（例如 NAT 規則）——krot 不會修改它們。
- 支援不含 krot 元資料的設定檔（手動建立）：`--list` 會顯示它們，
  `--list-client` 會輸出其 peer，且 id 為 `-`（`--id` 操作不適用於此類
  peer）；任何寫入都會逐位元組保留所有原始行、註解與順序。
- 寫入採原子方式（暫存檔 → `fsync` → `rename` → 目錄 `fsync`），
  權限為 `0600`。
- 所有變更操作都會對 `/run/lock/krot.lock` 取得獨佔 `flock`——
  並行執行（例如 Salt 與人工操作）不會遺失變更。
- krot 不會建立備份。

## 限制

- NAT、轉送、防火牆、sysctl 不在 krot 的範圍內（請使用
  `PostUp`/`PostDown` hook）。
- 不支援安裝 WireGuard/AmneziaWG，以及在不同類型之間遷移隧道。
  偽裝特徵產生使用內建 profile（`--mimic`）；QUIC profile 與自訂特徵
  則透過 `--i1` 與第三方產生器處理，見[偽裝](#偽裝i1i5-封包)。
- `--i1` 不得超過 2048 字元：由於
  [amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69)，
  過長的 `I1`–`I5`（約 3600+ 字元）會導致 `awg show` 停滯，
  並使 `awg set`/`setconf` 失敗。
- 不儲存客戶端私鑰；無法重新簽發客戶端設定檔。

## 開發

需要 `go.mod` 指定的 Go（目前為 1.27.2），整合測試另需 Docker。

```sh
make check        # gofmt、go vet（包括 integration 標籤）、go test -race ./...、gosec、staticcheck
make build        # 靜態 bin/krot（CGO_ENABLED=0），版本來自 git describe
make build-arm64
make integration  # 在具 systemd 的特權 Docker 容器中執行整合測試
```

儲存庫規則：[`AGENTS.md`](AGENTS.md)——規格優先，先測試後撰寫程式碼，
每個任務一個提交。

- **單元測試**不需 root，也不會更動系統：目錄會替換為暫存目錄，
  外部命令（`wg`、`awg`、`systemctl`、`ip` 等）透過記錄呼叫的
  模擬執行器處理。另有測試驗證客戶端操作僅呼叫 `wg|awg show` 與
  `wg|awg set <if> peer …`。
- **Golden 檔案**位於各套件的 `testdata/`。產生的輸出（`internal/app`、
  `internal/qr`）僅在明確需要時更新——使用
  `go test ./internal/app ./internal/qr -update` 並在提交中說明；
  其他 golden 檔案根據規格手動撰寫。
- **整合測試**（`test/integration`，建置標籤 `integration`）需要 root
  與 systemd，且僅在獨立的網路命名空間中執行：測試介面 unit 透過
  `NetworkNamespacePath` drop-in 放入 netns。涵蓋所有類型的完整生命週期、
  客戶端操作透明性（持續 ping 無封包遺失、ifindex 與握手不變）、IPv6、
  手動建立的設定檔及並行執行。本地透過 `scripts/integration-docker.sh` 執行，
  CI 中使用 `sudo`。
- `scripts/collect-versions.sh` 會重新建立真實的 AmneziaWG 版本輸出，
  供 golden 測試使用。
- **CI**（`.github/workflows/ci.yml`）：gofmt、vet、含 `-race` 與覆蓋率
  的單元測試、安全稽核 `govulncheck` + `gosec`、`staticcheck`、
  整合測試、`amd64`/`arm64` 建置。**發布**（`release.yml`，標籤 `v*`）：
  二進位檔、`.deb`/`.rpm` 套件（nfpm、`packaging/nfpm.yaml`）、
  所有作業系統的套件冒煙測試（`scripts/smoke.sh`）、附 `SHA256SUMS`
  的 GitHub Release。

## 授權

[Apache License 2.0](LICENSE)。Copyright 2026 Sergey Ovchinnikov。
