package platform

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestEnsureAndSingleLock(t *testing.T) {
	root := t.TempDir()
	paths := Paths{ConfigDir: filepath.Join(root, "config"), DataDir: filepath.Join(root, "data"), ArtifactsDir: filepath.Join(root, "data", "artifacts"), RuntimeDir: filepath.Join(root, "data", "run"), LockFile: filepath.Join(root, "data", "run", "daemon.lock")}
	if err := Ensure(paths); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(paths.RuntimeDir)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Fatalf("runtime permissions = %o, want 700", info.Mode().Perm())
		}
	}
	first, err := AcquireLock(paths.LockFile)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if _, err := AcquireLock(paths.LockFile); err == nil {
		t.Fatal("expected second lock acquisition to fail")
	}
}

func TestLocalTransport(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("temporary named-pipe name is covered on Windows CI")
	}
	endpoint := filepath.Join(t.TempDir(), "rpc.sock")
	listener, err := LocalTransport().Listen(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		_, err = conn.Write([]byte("ready"))
		done <- err
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := LocalTransport().Dial(ctx, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	data, err := io.ReadAll(conn)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ready" {
		t.Fatalf("got %q", data)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
