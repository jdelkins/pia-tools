package main

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/jdelkins/pia-tools/internal/pia"
)

func gen_keypair(tun *pia.Tunnel) error {
	privkey_b, err := exec.Command("wg", "genkey").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v; %s", err, privkey_b)
	}
	tun.PrivateKey = strings.TrimSpace(string(privkey_b))

	cmd := exec.Command("wg", "pubkey")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("StdinPipe: %v", err)
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, tun.PrivateKey)
	}()
	pubkey_b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v; %s", err, pubkey_b)
	}
	tun.PublicKey = strings.TrimSpace(string(pubkey_b))

	return nil
}
