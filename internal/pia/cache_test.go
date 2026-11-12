package pia

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestSaveCacheAtomicReplacement(t *testing.T) {
	dir := t.TempDir()
	original := &Tunnel{Interface: "pia", PrivateKey: "original-key"}
	if err := original.SaveCache(dir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "pia.json")
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	// An already-open reader must retain a complete snapshot of the old cache.
	reader, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	updated := &Tunnel{Interface: "pia", PrivateKey: "updated-key"}
	if err := updated.SaveCache(dir); err != nil {
		t.Fatal(err)
	}
	var snapshot Tunnel
	if err := json.NewDecoder(reader).Decode(&snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.PrivateKey != original.PrivateKey {
		t.Errorf("existing reader saw key %q, want %q", snapshot.PrivateKey, original.PrivateKey)
	}
	got, err := ReadCache(dir, "pia")
	if err != nil {
		t.Fatal(err)
	}
	if got.PrivateKey != updated.PrivateKey {
		t.Errorf("new reader saw key %q, want %q", got.PrivateKey, updated.PrivateKey)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("cache permissions = %04o, want 0600", got)
	}
	assertCacheEntries(t, dir, "pia.json")
}

func TestSaveCacheEncodingFailure(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing_cache=%t", existing), func(t *testing.T) {
			dir := t.TempDir()
			original := &Tunnel{Interface: "pia", PrivateKey: "original-key"}
			if existing {
				if err := original.SaveCache(dir); err != nil {
					t.Fatal(err)
				}
			}
			invalid := &Tunnel{
				Interface: "pia",
				Token: Token{
					Expiry: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			}
			if err := invalid.SaveCache(dir); err == nil {
				t.Fatal("expected error encoding invalid timestamp")
			}
			if !existing {
				assertCacheEntries(t, dir)
				return
			}
			got, err := ReadCache(dir, "pia")
			if err != nil {
				t.Fatal(err)
			}
			if got.PrivateKey != original.PrivateKey {
				t.Errorf("failed save replaced original cache: %q", got.PrivateKey)
			}
			assertCacheEntries(t, dir, "pia.json")
		})
	}
}

func TestSaveCacheRenameFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pia.json")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	tun := &Tunnel{Interface: "pia"}
	if err := tun.SaveCache(dir); err == nil {
		t.Fatal("expected error replacing a directory with a cache file")
	}
	assertCacheEntries(t, dir, "pia.json")
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		t.Fatalf("failed rename changed destination: info=%v, error=%v", info, err)
	}
}

func assertCacheEntries(t *testing.T, dir string, names ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(names) {
		t.Fatalf("unexpected cache directory entries: %v", entries)
	}
	for i, entry := range entries {
		if entry.Name() != names[i] {
			t.Errorf("unexpected cache directory entry: %s", entry.Name())
		}
	}
}

func TestCacheLockAcrossProcesses(t *testing.T) {
	dir := t.TempDir()
	lock, err := LockCache(dir, "pia")
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	// Interfaces must not unnecessarily block each other.
	other, err := LockCache(dir, "other")
	if err != nil {
		t.Fatal(err)
	}
	other.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCacheLockHelper$")
	cmd.Env = append(os.Environ(), "PIA_TEST_CACHE_DIR="+dir)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	reader := bufio.NewReader(stdout)
	if line, err := reader.ReadString('\n'); err != nil || line != "waiting\n" {
		t.Fatalf("helper did not start: line=%q, error=%v", line, err)
	}
	// The helper must see this state only after acquiring the lock, rather than
	// read old state before waiting and overwrite this update when it resumes.
	updated := &Tunnel{Interface: "pia", PrivateKey: "updated-key"}
	if err := updated.SaveCache(dir); err != nil {
		t.Fatal(err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("cache lock helper failed: %v", err)
	}
	got, err := ReadCache(dir, "pia")
	if err != nil {
		t.Fatal(err)
	}
	if got.PrivateKey != "updated-key" || got.Status != "refreshed" {
		t.Errorf("helper lost cache updates: key=%q, status=%q", got.PrivateKey, got.Status)
	}
}

func TestCacheLockHelper(t *testing.T) {
	dir := os.Getenv("PIA_TEST_CACHE_DIR")
	if dir == "" {
		return
	}
	probe, err := os.OpenFile(filepath.Join(dir, "pia.lock"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	err = syscall.Flock(int(probe.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	probe.Close()
	if !errors.Is(err, syscall.EWOULDBLOCK) {
		t.Fatalf("parent did not hold an exclusive lock: %v", err)
	}
	fmt.Println("waiting")
	lock, err := LockCache(dir, "pia")
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	tun, err := ReadCache(dir, "pia")
	if err != nil {
		t.Fatal(err)
	}
	if tun.PrivateKey != "updated-key" {
		t.Fatalf("read stale cache after acquiring lock: %q", tun.PrivateKey)
	}
	tun.Status = "refreshed"
	if err := tun.SaveCache(dir); err != nil {
		t.Fatal(err)
	}
}
