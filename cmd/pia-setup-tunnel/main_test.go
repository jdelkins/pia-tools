package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jdelkins/pia-tools/internal/pia"
)

func TestSetupTunnelDoesNotSaveCacheOnFailure(t *testing.T) {
	for _, existingCache := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing_cache=%t", existingCache), func(t *testing.T) {
			cacheDir := t.TempDir()
			cachePath := filepath.Join(cacheDir, "pia.json")
			previous := []byte("{\"interface\":\"pia\",\"peer_privkey\":\"existing-private-key\"}\n")
			if existingCache {
				if err := os.WriteFile(cachePath, previous, 0o600); err != nil {
					t.Fatal(err)
				}
			}

			cli := &CLI{
				IfName:   "pia",
				CacheDir: cacheDir,
				WGBinary: filepath.Join(cacheDir, "missing-wg"),
			}
			func() {
				defer func() {
					failure := recover()
					if failure == nil {
						t.Error("expected setup to fail")
					} else if !strings.Contains(fmt.Sprint(failure), "Could not generate keypair") {
						t.Errorf("unexpected setup failure: %v", failure)
					}
				}()
				setupTunnel(cli, &pia.Region{})
			}()

			got, err := os.ReadFile(cachePath)
			if !existingCache {
				if !os.IsNotExist(err) {
					t.Fatalf("failed setup should not create cache; read error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(previous) {
				t.Errorf("failed setup changed cache: got %q, want %q", got, previous)
			}
		})
	}
}
