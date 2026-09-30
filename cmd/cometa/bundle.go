package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// A "complete" distribution ships next to cometa(.exe):
//
//	go/         a Go toolchain
//	modcache/   the Ebitengine module cache (GOMODCACHE)
//	gocache/    a prewarmed build cache (read-only seed)
//
// When go/ is present, games build offline with that toolchain. Without it
// (the normal binary) Cometa uses the Go installed on the machine.
type bundle struct {
	root string
	goEx string
}

func findBundle() *bundle {
	root := os.Getenv("COMETA_HOME")
	if root == "" {
		exe, err := os.Executable()
		if err != nil {
			return nil
		}
		root = filepath.Dir(exe)
	}
	goEx := filepath.Join(root, "go", "bin", "go")
	if runtime.GOOS == "windows" {
		goEx += ".exe"
	}
	if _, err := os.Stat(goEx); err != nil {
		return nil
	}
	return &bundle{root: root, goEx: goEx}
}

// seeding is true while scripts/bundle-windows.ps1 fills the caches: it needs
// the network and writes directly into the bundle directories.
func seeding() bool { return os.Getenv("COMETA_SEMBRAR") != "" }

// env returns the environment overrides that make Go use the bundle.
func (b *bundle) env() ([]string, error) {
	goroot := filepath.Join(b.root, "go")
	modcache := filepath.Join(b.root, "modcache")
	env := []string{
		"GOROOT=" + goroot,
		"GOTOOLCHAIN=local",
		"GOMODCACHE=" + modcache,
		"GOFLAGS=-modcacherw",
		"CGO_ENABLED=0", // Ebitengine on Windows needs no C toolchain
	}
	if seeding() {
		return append(env, "GOCACHE="+filepath.Join(b.root, "gocache")), nil
	}
	cache, err := b.userCache()
	if err != nil {
		return nil, err
	}
	return append(env, "GOCACHE="+cache, "GOPROXY=off", "GOSUMDB=off"), nil
}

// userCache returns a writable build cache, copied from the shipped seed the
// first time (the install directory may be read-only).
func (b *bundle) userCache() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	seed := filepath.Join(b.root, "gocache")
	dest := filepath.Join(base, "cometa", "gocache")
	marker := filepath.Join(dest, ".sembrado")
	if _, err := os.Stat(marker); err == nil {
		return dest, nil
	}
	if _, err := os.Stat(seed); err != nil {
		return dest, os.MkdirAll(dest, 0o755) // no seed: plain empty cache
	}
	fmt.Fprintln(os.Stderr, "preparando la caché de compilación (solo la primera vez)…")
	if err := copyTree(seed, dest); err != nil {
		return "", err
	}
	return dest, os.WriteFile(marker, nil, 0o600)
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if _, err := os.Stat(target); err == nil {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		if _, err = io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	})
}

// useBundledGo makes the toolchain bundled with cometa the one Go's type
// importer (used to validate generated code) finds, even when no Go is on PATH.
func useBundledGo() {
	b := findBundle()
	if b == nil || os.Getenv("COMETA_GO") != "" {
		return
	}
	os.Setenv("GOROOT", filepath.Join(b.root, "go"))
	os.Setenv("GOTOOLCHAIN", "local")
	os.Setenv("PATH", filepath.Dir(b.goEx)+string(os.PathListSeparator)+os.Getenv("PATH"))
}
