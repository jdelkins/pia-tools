package pia

import (
	"fmt"
	"testing"
	"time"
)

func TestPortForwardSigValid(t *testing.T) {
	valid := PortForwardSig{
		Port:      12345,
		Expiry:    time.Now().Add(time.Hour),
		Signature: "signature",
		Payload:   "payload",
	}
	for _, port := range []int{1, 12345, 65535} {
		t.Run(fmt.Sprintf("valid_port_%d", port), func(t *testing.T) {
			sig := valid
			sig.Port = port
			if !sig.Valid() {
				t.Error("complete, unexpired signature should be valid")
			}
		})
	}
	invalid := []struct {
		name   string
		change func(*PortForwardSig)
	}{
		{"missing signature", func(sig *PortForwardSig) { sig.Signature = "" }},
		{"missing payload", func(sig *PortForwardSig) { sig.Payload = "" }},
		{"missing port", func(sig *PortForwardSig) { sig.Port = 0 }},
		{"negative port", func(sig *PortForwardSig) { sig.Port = -1 }},
		{"out of range port", func(sig *PortForwardSig) { sig.Port = 65536 }},
		{"missing expiry", func(sig *PortForwardSig) { sig.Expiry = time.Time{} }},
		{"expired", func(sig *PortForwardSig) { sig.Expiry = time.Now().Add(-time.Hour) }},
		{"expires now", func(sig *PortForwardSig) { sig.Expiry = time.Now() }},
		{"missing assignment", func(sig *PortForwardSig) { *sig = PortForwardSig{} }},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			sig := valid
			tt.change(&sig)
			if sig.Valid() {
				t.Error("incomplete or expired signature should not be valid")
			}
		})
	}
}
