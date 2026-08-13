package runtime

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/platform"
	"github.com/cyberkit-x/cyberpilot/internal/rpc"
	"github.com/cyberkit-x/cyberpilot/internal/service"
	store "github.com/cyberkit-x/cyberpilot/internal/storage/sqlite"
)

const tokenFileName = "rpc.token"

type Daemon struct {
	lock     *platform.Lock
	store    *store.Store
	listener net.Listener
	server   *rpc.Server
}

type SessionWorker interface {
	Start(context.Context, domain.Session)
}
type WorkerFactory func(*service.SessionService, *store.Store) SessionWorker

func NewDaemon(paths platform.Paths) (*Daemon, error) {
	return NewDaemonWithWorker(paths, nil)
}

func NewDaemonWithWorker(paths platform.Paths, factory WorkerFactory) (*Daemon, error) {
	if err := platform.Ensure(paths); err != nil {
		return nil, err
	}
	lock, err := platform.AcquireLock(paths.LockFile)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*Daemon, error) { _ = lock.Close(); return nil, err }
	db, err := store.Open(paths.DatabaseFile)
	if err != nil {
		return fail(err)
	}
	token, err := loadOrCreateToken(filepath.Join(paths.RuntimeDir, tokenFileName))
	if err != nil {
		_ = db.Close()
		return fail(err)
	}
	listener, err := platform.LocalTransport().Listen(paths.Endpoint)
	if err != nil {
		_ = db.Close()
		return fail(err)
	}
	server := rpc.NewServer(token)
	sessions := service.NewSessionService(db)
	if err := sessions.Recover(context.Background()); err != nil {
		_ = listener.Close()
		_ = db.Close()
		return fail(err)
	}
	if factory != nil {
		if worker := factory(sessions, db); worker != nil {
			sessions.OnCreate(worker.Start)
		}
	}
	sessions.Register(server)
	return &Daemon{lock: lock, store: db, listener: listener, server: server}, nil
}

func (d *Daemon) Serve(ctx context.Context) error { return d.server.Serve(ctx, d.listener) }

func (d *Daemon) Close() error {
	if d.listener != nil {
		_ = d.listener.Close()
	}
	if d.store != nil {
		_ = d.store.Close()
	}
	if d.lock != nil {
		return d.lock.Close()
	}
	return nil
}

func loadOrCreateToken(path string) (string, error) {
	if data, err := os.ReadFile(path); err == nil {
		value := string(data)
		if value == "" {
			return "", fmt.Errorf("empty RPC token")
		}
		return value, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	value := base64.RawURLEncoding.EncodeToString(data)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	if _, err := file.WriteString(value); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return value, nil
}

func ReadToken(paths platform.Paths) (string, error) {
	return loadOrCreateToken(filepath.Join(paths.RuntimeDir, tokenFileName))
}
