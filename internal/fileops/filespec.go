package fileops

import (
	"encoding"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/jdelkins/pia-tools/internal/pia"
)

// FileSpec describes how to render a template to an output path.
//
// Owner/Group/Mode are pointers; nil means "use runtime defaults".
type FileSpec struct {
	Output   string
	Template string

	Owner *string
	Group *string
	Mode  *os.FileMode
}

var _ encoding.TextMarshaler = (*FileSpec)(nil)

// Parse converts a Kong-parsed map (eg from mapsep=",", sep="=") into a FileSpec.
//
// Expected keys: output=, template=, owner=, group=, mode=
//
// Signature order is intentionally (error, *FileSpec) to match the request.
func Parse(m map[string]string) (*FileSpec, error) {
	if m == nil {
		return nil, fmt.Errorf("missing spec")
	}

	// Normalize keys.
	norm := make(map[string]string, len(m))
	for k, v := range m {
		nk := strings.ToLower(strings.TrimSpace(k))
		norm[nk] = strings.TrimSpace(v)
	}

	allowed := map[string]bool{
		"output":   true,
		"template": true,
		"owner":    true,
		"group":    true,
		"mode":     true,
	}
	for k := range norm {
		if !allowed[k] {
			return nil, fmt.Errorf("unknown key %q", k)
		}
	}

	out := &FileSpec{
		Output:   norm["output"],
		Template: norm["template"],
	}
	if strings.TrimSpace(out.Output) == "" {
		return nil, fmt.Errorf("missing required key %q", "output")
	}
	if strings.TrimSpace(out.Template) == "" {
		out.Template = out.Output + ".tmpl"
	}

	if v := norm["owner"]; v != "" {
		vv := v
		out.Owner = &vv
	}
	if v := norm["group"]; v != "" {
		vv := v
		out.Group = &vv
	}
	if v := norm["mode"]; v != "" {
		u, err := strconv.ParseUint(v, 8, 32) // octal like 0440
		if err != nil {
			return nil, fmt.Errorf("invalid mode %q (expected octal like 0440): %w", v, err)
		}
		mv := os.FileMode(u)
		out.Mode = &mv
	}

	return out, nil
}

// Generate renders the template and writes it to Output, applying owner/group/mode
// overrides if provided.
func (s *FileSpec) Generate(tun *pia.Tunnel) error {
	if s == nil {
		return fmt.Errorf("nil FileSpec")
	}
	if strings.TrimSpace(s.Output) == "" {
		return fmt.Errorf("missing output")
	}
	if strings.TrimSpace(s.Template) == "" {
		return fmt.Errorf("missing template")
	}

	// Provide a helper used by the stock pia.netdev.tmpl.
	wgserver := func(tuni any) any {
		t := tuni.(*pia.Tunnel)
		return any(t.Region.WgServer())
	}
	extraFuncs := template.FuncMap{"server": wgserver}

	tmpl, err := template.New(filepath.Base(s.Template)).Funcs(sprig.TxtFuncMap()).Funcs(extraFuncs).ParseFiles(s.Template)
	if err != nil {
		return fmt.Errorf("error parsing template from %s: %w", s.Template, err)
	}

	// Choose file mode for initial create.
	perm := os.FileMode(0o666)
	if s.Mode != nil {
		perm = *s.Mode
	}

	// Write to a temp file in the same directory and rename into place.
	dir := filepath.Dir(s.Output)
	base := filepath.Base(s.Output)
	tmpPath := filepath.Join(dir, fmt.Sprintf(".%s.tmp.%d", base, os.Getpid()))

	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(tmpPath)
	}()

	if err := tmpl.Execute(f, tun); err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	// Apply overrides (nil => runtime default).
	if s.Owner != nil || s.Group != nil {
		uid := -1
		gid := -1

		if s.Owner != nil {
			u, err := user.Lookup(*s.Owner)
			if err != nil {
				return fmt.Errorf("could not lookup user %q: %w", *s.Owner, err)
			}
			parsed, err := strconv.Atoi(u.Uid)
			if err != nil {
				return fmt.Errorf("invalid uid for user %q: %w", *s.Owner, err)
			}
			uid = parsed
		}
		if s.Group != nil {
			g, err := user.LookupGroup(*s.Group)
			if err != nil {
				return fmt.Errorf("could not lookup group %q: %w", *s.Group, err)
			}
			parsed, err := strconv.Atoi(g.Gid)
			if err != nil {
				return fmt.Errorf("invalid gid for group %q: %w", *s.Group, err)
			}
			gid = parsed
		}

		if err := os.Chown(tmpPath, uid, gid); err != nil {
			return err
		}
	}
	if s.Mode != nil {
		if err := os.Chmod(tmpPath, *s.Mode); err != nil {
			return err
		}
	}

	if err := os.Rename(tmpPath, s.Output); err != nil {
		return err
	}

	return nil
}

// MarshalText is handy for logging/debugging.
func (s *FileSpec) MarshalText() ([]byte, error) {
	if s == nil {
		return []byte("<nil>"), nil
	}
	parts := []string{fmt.Sprintf("output=%s", s.Output)}
	if s.Template != "" {
		parts = append(parts, fmt.Sprintf("template=%s", s.Template))
	}
	if s.Owner != nil {
		parts = append(parts, fmt.Sprintf("owner=%s", *s.Owner))
	}
	if s.Group != nil {
		parts = append(parts, fmt.Sprintf("group=%s", *s.Group))
	}
	if s.Mode != nil {
		parts = append(parts, fmt.Sprintf("mode=%#o", *s.Mode))
	}
	return []byte(strings.Join(parts, ",")), nil
}
