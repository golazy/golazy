package component

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"golazy.dev/lazyview/script"
)

type Npm struct {
	Name    string
	Version string
	Imports ImportMap
	Files   []string
	Scripts []script.Script
}

func (n *Npm) String() string {
	name := "npm:" + n.Name
	if n.Version != "" {
		name += "@" + n.Version
	}
	return name
}

func (n *Npm) Install(opts InstallOptions) error {
	fmt.Println("installing npm package: " + n.Name + " (" + n.Version + ")")

	dest := opts.Cache
	if dest == "" {
		dest = opts.Path
	}
	err := os.MkdirAll(dest, 0755)
	if err != nil {
		return err
	}

	name := n.Name
	if n.Version != "" {
		name += "@" + n.Version
	}

	cmd := exec.Command("npm", "install", n.Name)
	cmd.Dir = dest
	err = cmd.Run()
	if err != nil {
		return err
	}

	// Copy all Imports
	for _, impPath := range n.Imports {
		err = copyFile(n.cachePath(opts, impPath), n.installPath(opts, impPath))
		if err != nil {
			return err
		}
	}

	// Copy all Files
	for _, file := range n.Files {
		err = copyFile(n.cachePath(opts, file), n.installPath(opts, file))
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *Npm) cachePath(opts InstallOptions, path ...string) string {
	d := opts.Cache
	if d == "" {
		d = opts.Path
	}
	p := filepath.Join(d, "node_modules", n.Name, filepath.Join(path...))
	return filepath.Clean(p)
}

func (n *Npm) installPath(opts InstallOptions, path ...string) string {
	p := filepath.Join(opts.Path, n.Name, filepath.Join(path...))
	return filepath.Clean(p)
}

func copyFile(from, to string) error {
	err := os.MkdirAll(filepath.Dir(to), 0755)
	if err != nil {
		return err
	}
	os.Remove(to + "-temp")

	err = os.Link(from, to+"-temp")
	if err != nil {
		return err
	}
	defer os.Remove(to + "-temp")

	return os.Rename(to+"-temp", to)

}

func (n *Npm) Uninstall(opts InstallOptions) error {
	errs := []error{}

	dest := opts.Cache
	if dest == "" {
		dest = opts.Path
	}

	errs = append(errs, os.RemoveAll(n.installPath(opts)))

	cmd := exec.Command("npm", "uninstall", n.Name)
	cmd.Dir = dest
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	errs = append(errs, cmd.Run())

	return errors.Join(errs...)
}

func (n *Npm) ImportMap() ImportMap {
	i := make(map[string]string)
	for k, v := range n.Imports {
		i[k] = "/" + path.Join(n.Name, v)
	}
	return i
}

func (n *Npm) PageScripts() []script.Script {
	return n.Scripts
}
