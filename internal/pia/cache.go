package pia

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// LockCache serializes operations on an interface's cached tunnel. Callers must
// hold the returned file open throughout the entire read/update/save operation;
// closing it releases the lock, including when the process exits unexpectedly.
// The lock file must not be removed: its inode must remain stable across cache
// replacements and between processes waiting for the lock.
func LockCache(pathCache, ifname string) (*os.File, error) {
	file, err := os.OpenFile(filepath.Join(pathCache, ifname+".lock"), os.O_CREATE|os.O_RDWR, 0o660)
	if err != nil {
		return nil, err
	}
	for {
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
		if !errors.Is(err, syscall.EINTR) {
			break
		}
	}
	if err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

// SaveCache atomically replaces the cached tunnel. Writers must also hold
// LockCache to prevent overwriting another process's updates with stale state.
func (tun *Tunnel) SaveCache(pathCache string) error {
	path := filepath.Join(pathCache, tun.Interface+".json")
	// Use the destination directory so rename is atomic, retaining the original
	// create permissions (subject to umask) and exclusively creating a new file.
	tmpPath := filepath.Join(pathCache, "."+tun.Interface+".json.tmp."+rand.Text())
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o660)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)
	defer file.Close()

	// Replacement must not loosen permissions on an existing cache file.
	if info, err := os.Stat(path); err == nil {
		if err := file.Chmod(info.Mode().Perm()); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := json.NewEncoder(file).Encode(tun); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func ReadCache(pathCache string, ifname string) (*Tunnel, error) {
	file, err := os.Open(filepath.Join(pathCache, ifname+".json"))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var tun Tunnel
	if err := json.NewDecoder(file).Decode(&tun); err != nil {
		return nil, err
	}
	return &tun, nil
}
