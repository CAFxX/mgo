package main

import (
	"crypto/sha256"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"

	"golang.org/x/mod/semver"
)

func main() {
	goos := os.Getenv("GOOS")
	if goos == "" {
		goos = runtime.GOOS
	}
	if goos != "linux" {
		fmt.Printf("GOOS=%q is not supported\n", goos)
		return
	}

	goarch := os.Getenv("GOARCH")
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	if goarch != "amd64" {
		fmt.Printf("GOARCH=%q is not supported\n", goarch)
		return
	}

	if goamd64 := os.Getenv("GOAMD64"); goamd64 != "" {
		fmt.Printf("GOAMD64 must not be set (currently %q)\n", goamd64)
		return
	}

	cmd := exec.Command("go")
	cmd.Args = []string{"go", "version"}
	buf, err := cmd.Output()
	if err != nil {
		fmt.Printf("fetching go version: %v\n", err)
		return
	}
	m := regexp.MustCompile(`^go version go([0-9]+\.[0-9]+(\.[0-9]+)?)`).FindSubmatch(buf)
	if m == nil {
		fmt.Printf("parsing go version: malformed: %q\n", string(buf))
		return
	}
	if semver.Compare("v"+string(m[1]), "v1.18") < 0 {
		fmt.Printf("installed go version too old: %q\n", string(m[1]))
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("getting cwd: %v\n", err)
		return
	}

	o := flag.String("o", filepath.Base(cwd), "output file")
	flag.Parse()

	tmpdir, err := os.MkdirTemp("", "mgo")
	if err != nil {
		fmt.Printf("creating temp dir: %v\n", err)
		return
	}
	defer os.RemoveAll(tmpdir)

	err = os.WriteFile(filepath.Join(tmpdir, "main.go"), []byte(launcher), 0600)
	if err != nil {
		fmt.Printf("writing launcher: %v\n", err)
		return
	}

	err = fs.WalkDir(vendor, ".", func(path string, d fs.DirEntry, err error) error {
		dpath := filepath.Join(tmpdir, path)
		switch {
		case err != nil:
			return err
		case d.Type().IsRegular():
			buf, err := fs.ReadFile(vendor, path)
			if err != nil {
				return fmt.Errorf("read file %q: %w", path, err)
			}
			err = os.WriteFile(dpath, buf, 0600)
			if err != nil {
				return fmt.Errorf("write file %q: %w", dpath, err)
			}
		case d.Type().IsDir():
			err := os.MkdirAll(dpath, 0700)
			if err != nil {
				return fmt.Errorf("create dir %q: %w", dpath, err)
			}
		default:
			return fmt.Errorf("unknown dir entry %q, type %q", path, d.Type())
		}
		return nil
	})
	if err != nil {
		fmt.Printf("copying vendored dependencies: %v\n", err)
		return
	}

	hashes := map[[32]byte]string{}
	vmap := map[string]string{}
	for _, v := range []string{"v1", "v2", "v3", "v4"} {
		exe := filepath.Join(tmpdir, "mgo."+v)

		cmd := exec.Command("go")
		cmd.Args = append([]string{"go", "build", "-o", exe}, flag.Args()...)
		cmd.Env = append(os.Environ(), "GOAMD64="+v)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Printf("building variant %q: %v\n", v, err)
			return
		}

		buf, err := os.ReadFile(exe)
		if err != nil {
			fmt.Printf("reading variant executable %q: %v", exe, err)
			return
		}
		h := sha256.Sum256(buf)
		if pv, ok := hashes[h]; ok {
			vmap[v] = pv
		} else {
			vmap[v] = v
			hashes[h] = v
		}
	}

	cmd = exec.Command("go")
	cmd.Args = []string{"go", "build", "-mod=vendor", "-o", filepath.Join(cwd, *o), "-trimpath", filepath.Join(tmpdir, "main.go")}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Printf("building launcher: %v\n", err)
		return
	}
}

//go:embed vendor
var vendor embed.FS

//go:embed launcher/main.go
var launcher []byte
