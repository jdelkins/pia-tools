package fileops

import (
	"fmt"
	"io"
	"maps"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jdelkins/pia-tools/internal/pia"
)

func TestParse(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input map[string]string
		want  FileSpec
	}{
		{"defaults", map[string]string{"output": "/network/pia.netdev"}, FileSpec{Output: "/network/pia.netdev", Template: "/network/pia.netdev.tmpl"}},
		{"explicit template", map[string]string{"output": "out", "template": "source"}, FileSpec{Output: "out", Template: "source"}},
		{"normalize keys and values", map[string]string{" OUTPUT ": " out ", " Template ": " source "}, FileSpec{Output: "out", Template: "source"}},
		{"empty template", map[string]string{"output": "out", "template": " "}, FileSpec{Output: "out", Template: "out.tmpl"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			original := maps.Clone(tt.input)
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(*got, tt.want) {
				t.Errorf("Parse = %#v, want %#v", *got, tt.want)
			}
			if !maps.Equal(tt.input, original) {
				t.Error("Parse modified its input map")
			}
		})
	}
}

func TestParseOverrides(t *testing.T) {
	spec, err := Parse(map[string]string{"output": "out", "owner": "alice", "group": "network", "mode": "0440"})
	if err != nil {
		t.Fatal(err)
	}
	if spec.Owner == nil || *spec.Owner != "alice" || spec.Group == nil || *spec.Group != "network" || spec.Mode == nil || *spec.Mode != 0o440 {
		t.Errorf("unexpected overrides: %#v", spec)
	}
	spec, err = Parse(map[string]string{"output": "out", "mode": "0000"})
	if err != nil || spec.Mode == nil || *spec.Mode != 0 {
		t.Fatalf("explicit zero mode was not preserved: spec=%#v, error=%v", spec, err)
	}
}

func TestParseErrors(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input map[string]string
		want  string
	}{
		{"nil", nil, "missing spec"},
		{"missing output", map[string]string{}, "missing required key"},
		{"blank output", map[string]string{"output": " "}, "missing required key"},
		{"unknown key", map[string]string{"output": "out", "typo": "value"}, "unknown key"},
		{"non octal mode", map[string]string{"output": "out", "mode": "0888"}, "invalid mode"},
		{"negative mode", map[string]string{"output": "out", "mode": "-1"}, "invalid mode"},
		{"overflow mode", map[string]string{"output": "out", "mode": "77777777777777777777777"}, "invalid mode"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Parse(tt.input); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Parse error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestMarshalText(t *testing.T) {
	var nilSpec *FileSpec
	if got, err := nilSpec.MarshalText(); err != nil || string(got) != "<nil>" {
		t.Fatalf("nil MarshalText = %q, %v", got, err)
	}
	for _, input := range []map[string]string{
		{"output": "out"},
		{"output": "out", "template": "source", "owner": "alice", "group": "network", "mode": "0440"},
	} {
		spec, err := Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		got, err := spec.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		want := "output=out,template=out.tmpl"
		if input["owner"] != "" {
			want = "output=out,template=source,owner=alice,group=network,mode=0440"
		}
		if string(got) != want {
			t.Errorf("MarshalText = %q, want %q", got, want)
		}
	}
}

func TestGenerate(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "pia.netdev")
	source := output + ".tmpl"
	if err := os.WriteFile(source, []byte(`{{.Interface}}|{{upper .Region.Name}}|{{(server .).Cn}}|{{spec}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("old configuration"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	group, err := user.LookupGroupId(current.Gid)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := Parse(map[string]string{"output": output, "owner": current.Username, "group": group.Name, "mode": "0640"})
	if err != nil {
		t.Fatal(err)
	}
	tun := &pia.Tunnel{Interface: "wg_pia", Region: pia.Region{
		Name: "Test region", Servers: map[string][]pia.Server{"wg": {{Cn: "server.example"}}},
	}}
	if err := spec.Generate(tun); err != nil {
		t.Fatal(err)
	}
	text, err := spec.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if want := "wg_pia|TEST REGION|server.example|" + string(text); string(got) != want {
		t.Errorf("generated content = %q, want %q", got, want)
	}
	old, err := io.ReadAll(snapshot)
	if err != nil || string(old) != "old configuration" {
		t.Errorf("existing reader lost its original snapshot: %q, %v", old, err)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Errorf("generated mode = %04o, want 0640", info.Mode().Perm())
	}
	assertNoTempFile(t, spec)
}

func TestGenerateFailuresPreserveOutput(t *testing.T) {
	for _, tt := range []struct {
		name     string
		template string
		change   func(*FileSpec)
		want     string
	}{
		{"parse error", "{{", nil, "error parsing template"},
		{"execution error", `partial{{fail "deliberate failure"}}`, nil, "error executing template"},
		{"missing template", "text", func(s *FileSpec) { s.Template += ".missing" }, "error parsing template"},
		{"unknown owner", "text", func(s *FileSpec) { owner := "pia-tools-nonexistent-user"; s.Owner = &owner }, "could not lookup user"},
		{"unknown group", "text", func(s *FileSpec) { group := "pia-tools-nonexistent-group"; s.Group = &group }, "could not lookup group"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			spec := &FileSpec{Output: filepath.Join(dir, "out"), Template: filepath.Join(dir, "source")}
			if err := os.WriteFile(spec.Template, []byte(tt.template), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(spec.Output, []byte("original"), 0o600); err != nil {
				t.Fatal(err)
			}
			if tt.change != nil {
				tt.change(spec)
			}
			if err := spec.Generate(&pia.Tunnel{}); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Generate error = %v, want %q", err, tt.want)
			}
			got, err := os.ReadFile(spec.Output)
			if err != nil || string(got) != "original" {
				t.Errorf("failed generation changed output: %q, %v", got, err)
			}
			assertNoTempFile(t, spec)
		})
	}
}

func TestGenerateInvalidSpec(t *testing.T) {
	for _, spec := range []*FileSpec{nil, {}, {Template: "source"}, {Output: "out"}} {
		if err := spec.Generate(&pia.Tunnel{}); err == nil {
			t.Errorf("expected invalid spec error for %#v", spec)
		}
	}
}

func TestGenerateStockTemplates(t *testing.T) {
	tun := &pia.Tunnel{
		Interface: "wg_test", PrivateKey: "test-private-key", ServerPubkey: "test-public-key",
		ServerIp: "192.0.2.1", ServerPort: 51820, ServerVip: "10.0.0.1", PeerIp: "10.0.0.2",
		DnsServers: []string{"10.0.0.242", "10.0.0.243"},
		Region:     pia.Region{Id: "test", Name: "Test region", Servers: map[string][]pia.Server{"wg": {{Cn: "server.example"}}}},
	}
	for _, tt := range []struct {
		kind string
		want []string
	}{
		{"netdev", []string{"Name=wg_test", "PrivateKey=test-private-key", "PublicKey=test-public-key", "Endpoint=192.0.2.1:51820", "server.example"}},
		{"network", []string{"Name=wg_test", "Address=10.0.0.2/32", "DNS=10.0.0.242", "DNS=10.0.0.243", "Destination=10.0.0.1/32", "Gateway=10.0.0.1"}},
	} {
		t.Run(tt.kind, func(t *testing.T) {
			spec := &FileSpec{
				Output:   filepath.Join(t.TempDir(), "pia."+tt.kind),
				Template: filepath.Join("..", "..", "systemd", "network", "pia."+tt.kind+".tmpl"),
			}
			if err := spec.Generate(tun); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(spec.Output)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tt.want {
				if !strings.Contains(string(got), want) {
					t.Errorf("generated template missing %q", want)
				}
			}
			assertNoTempFile(t, spec)
		})
	}
}

func TestGenerateDoesNotOverwriteExistingTempFile(t *testing.T) {
	dir := t.TempDir()
	spec := &FileSpec{Output: filepath.Join(dir, "out"), Template: filepath.Join(dir, "source")}
	if err := os.WriteFile(spec.Template, []byte("new configuration"), 0o600); err != nil {
		t.Fatal(err)
	}
	tmp := tempPath(spec)
	if err := os.WriteFile(tmp, []byte("existing temporary file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := spec.Generate(&pia.Tunnel{}); !os.IsExist(err) {
		t.Errorf("Generate error = %v, want existing-file error", err)
	}
	if got, err := os.ReadFile(tmp); err != nil || string(got) != "existing temporary file" {
		t.Errorf("existing temporary file changed: %q, %v", got, err)
	}
}

func tempPath(spec *FileSpec) string {
	return filepath.Join(filepath.Dir(spec.Output), fmt.Sprintf(".%s.tmp.%d", filepath.Base(spec.Output), os.Getpid()))
}

func assertNoTempFile(t *testing.T, spec *FileSpec) {
	t.Helper()
	if _, err := os.Stat(tempPath(spec)); !os.IsNotExist(err) {
		t.Errorf("temporary file was not cleaned up: %v", err)
	}
}
