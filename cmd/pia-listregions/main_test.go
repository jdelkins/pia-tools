package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jdelkins/pia-tools/internal/pia"
)

func TestWriteRegions(t *testing.T) {
	regions := []pia.Region{
		{Id: "both", Name: "Both features", PingTime: 12*time.Millisecond + 900*time.Microsecond,
			PortForward: true, Servers: map[string][]pia.Server{"wg": {{Ip: "192.0.2.1"}}}},
		{Id: "wg", Name: "WireGuard only", Servers: map[string][]pia.Server{"wg": {{Ip: "192.0.2.2"}}}},
		{Id: "pf", Name: "Forwarding only", PingTime: time.Millisecond, PortForward: true},
		{Id: "neither", Name: "Neither feature"},
	}
	var output bytes.Buffer
	if err := writeRegions(&output, regions); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n")
	if len(lines) != len(regions)+2 {
		t.Fatalf("unexpected region table: %q", output.String())
	}
	header := lines[0]
	nameColumn := strings.Index(header, "NAME")
	pingColumn := strings.Index(header, "PING")
	wgColumn := strings.Index(header, "WG?")
	pfColumn := strings.Index(header, "PF?")
	if !strings.HasPrefix(header, "ID") || nameColumn <= 0 || pingColumn <= nameColumn || wgColumn <= pingColumn || pfColumn <= wgColumn {
		t.Fatalf("unexpected header: %q", header)
	}
	for i, want := range []struct {
		ping string
		wg   string
		pf   string
	}{{"12 ms", "✓", "✓"}, {"N/A", "✓", ""}, {"1 ms", "", "✓"}, {"N/A", "", ""}} {
		line := lines[i+2]
		if len(line) < pfColumn {
			t.Fatalf("short table row: %q", line)
		}
		got := []string{
			strings.TrimSpace(line[:nameColumn]),
			strings.TrimSpace(line[nameColumn:pingColumn]),
			strings.TrimSpace(line[pingColumn:wgColumn]),
			strings.TrimSpace(line[wgColumn:pfColumn]),
			strings.TrimSpace(line[pfColumn:]),
		}
		expected := []string{regions[i].Id, regions[i].Name, want.ping, want.wg, want.pf}
		for column := range expected {
			if got[column] != expected[column] {
				t.Errorf("row %d column %d = %q, want %q", i, column, got[column], expected[column])
			}
		}
	}
}

func TestWriteRegionsEmpty(t *testing.T) {
	var output bytes.Buffer
	if err := writeRegions(&output, nil); err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(output.String(), "\n"); lines != 2 {
		t.Errorf("empty table has %d lines, want header and separator", lines)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWriteRegionsWriterError(t *testing.T) {
	want := errors.New("output unavailable")
	if err := writeRegions(failingWriter{want}, nil); !errors.Is(err, want) {
		t.Errorf("writeRegions error = %v, want %v", err, want)
	}
}
