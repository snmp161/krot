package app

import (
	"github.com/sovchinnikov/krot/internal/cli"
	"github.com/sovchinnikov/krot/internal/conf"
	"github.com/sovchinnikov/krot/internal/errs"
)

// clientOp — общая часть --delete-client/--lock-client/--unlock-client: блокировка, поиск
// интерфейса и клиента (только по krot:id), изменение конфига, точечная живая операция
// и откат конфига при её неудаче (SPEC 6.6–6.8, 6.10).
func (a *App) clientOp(cmd *cli.Command, op func(t *tunnel, p conf.Peer) (live func() error, msg string, err error)) error {
	return a.withLock(func() error {
		t, err := a.findTunnel(cmd.Interface)
		if err != nil {
			return err
		}
		if err := a.need(t.tool()); err != nil {
			return err
		}
		p, ok := t.conf.PeerByID(cmd.ID)
		if !ok {
			return errs.NotFoundf("client %s not found on interface %s", cmd.ID, t.name)
		}
		orig := t.conf.Bytes()
		live, msg, err := op(t, p)
		if err != nil {
			return err
		}
		if live != nil {
			if err := a.save(t); err != nil {
				return err
			}
			if err := live(); err != nil {
				return a.rollback(t, orig, err)
			}
		}
		a.printf("%s\n", msg)
		return nil
	})
}

// SPEC 6.6: --delete-client. Заблокированного клиента нет в интерфейсе — живой вызов не нужен.
func (a *App) deleteClient(cmd *cli.Command) error {
	return a.clientOp(cmd, func(t *tunnel, p conf.Peer) (func() error, string, error) {
		if err := t.conf.RemovePeer(p.ID); err != nil {
			return nil, "", err
		}
		live := func() error {
			if p.Locked {
				return nil
			}
			return a.removePeerLive(t, p.PublicKey())
		}
		return live, "Client " + p.ID + " deleted from " + t.name, nil
	})
}

// SPEC 6.7: --lock-client.
func (a *App) lockClient(cmd *cli.Command) error {
	return a.clientOp(cmd, func(t *tunnel, p conf.Peer) (func() error, string, error) {
		if p.Locked {
			return nil, "Client " + p.ID + " is already locked", nil
		}
		if err := t.conf.LockPeer(p.ID, a.Now().UTC().Format("2006-01-02T15:04:05Z")); err != nil {
			return nil, "", err
		}
		return func() error { return a.removePeerLive(t, p.PublicKey()) }, "Client " + p.ID + " locked", nil
	})
}

// SPEC 6.8: --unlock-client — вживую возвращается с его AllowedIPs и PresharedKey.
func (a *App) unlockClient(cmd *cli.Command) error {
	return a.clientOp(cmd, func(t *tunnel, p conf.Peer) (func() error, string, error) {
		if !p.Locked {
			return nil, "Client " + p.ID + " is already unlocked", nil
		}
		if err := t.conf.UnlockPeer(p.ID); err != nil {
			return nil, "", err
		}
		live := func() error { return a.applyPeer(t, p.PublicKey(), p.AllowedIPs(), p.PresharedKey()) }
		return live, "Client " + p.ID + " unlocked", nil
	})
}

// removePeerLive удаляет peer из поднятого интерфейса: `wg|awg set <if> peer <pub> remove`.
func (a *App) removePeerLive(t *tunnel, pub string) error {
	up, err := a.isUp(t)
	if err != nil || !up {
		return err
	}
	_, err = a.Exec.Run(t.tool(), "set", t.name, "peer", pub, "remove")
	return err
}
