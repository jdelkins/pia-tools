package fileops

import (
	"io/fs"
	"os"
	"os/user"
	"strconv"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/jdelkins/pia-tools/internal/pia"
)

func createFile(path string, gid int, perm fs.FileMode) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return nil, err
	}

	err = file.Chown(0, gid)
	if err != nil {
		file.Close()
		return nil, err
	}

	return file, nil
}

func CreateNetdevFile(tun *pia.Tunnel, output_path, template_path string) error {
	// Generate .netdev
	wgserver := func(tuni interface{}) interface{} {
		tun := tuni.(pia.Tunnel)
		return interface{}(*tun.Region.WgServer())
	}

	extraFuncs := template.FuncMap{
		"server": wgserver,
	}
	nd_tmpl, err := template.New(tun.Interface + ".netdev.tmpl").Funcs(sprig.TxtFuncMap()).Funcs(extraFuncs).ParseFiles(template_path)
	if err != nil {
		return err
	}

	grp, err := user.LookupGroup("systemd-network")
	if err != nil {
		return err
	}

	gid, err := strconv.Atoi(grp.Gid)
	if err != nil {
		return err
	}

	file, err := createFile(output_path, gid, 0o640)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := nd_tmpl.Execute(file, tun); err != nil {
		return err
	}

	return nil
}

func CreateNetworkFile(tun *pia.Tunnel, output_path, template_path string) error {
	tmpl, err := template.New(tun.Interface + ".network.tmpl").Funcs(sprig.TxtFuncMap()).ParseFiles(template_path)
	if err != nil {
		return err
	}

	file, err := createFile(output_path, 0, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := tmpl.Execute(file, tun); err != nil {
		return err
	}
	return nil
}
