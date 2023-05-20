package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"unicode"

	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
)

func main() {
	goos := os.Getenv("GOOS")
	if goos == "" {
		goos = runtime.GOOS
	}
	if goos != "linux" {
		fmt.Printf("GOOS=%q is not supported\n", goos)
		os.Exit(1)
	}

	goarch := os.Getenv("GOARCH")
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	if goarch != "amd64" {
		fmt.Printf("GOARCH=%q is not supported\n", goarch)
		os.Exit(1)
	}

	if goamd64 := os.Getenv("GOAMD64"); goamd64 != "" {
		fmt.Printf("GOAMD64 must not be set (currently %q)\n", goamd64)
		os.Exit(1)
	}

	cmd := exec.Command("go")
	cmd.Args = []string{"go", "version"}
	buf, err := cmd.Output()
	if err != nil {
		fmt.Printf("fetching go version: %v\n", err)
		os.Exit(2)
	}
	m := regexp.MustCompile(`^go version go([0-9]+\.[0-9]+(\.[0-9]+)?)`).FindSubmatch(buf)
	if m == nil {
		fmt.Printf("parsing go version: malformed: %q\n", string(buf))
		os.Exit(2)
	}
	if semver.Compare("v"+string(m[1]), "v1.20") < 0 {
		fmt.Printf("installed go version too old: %q\n", string(m[1]))
		os.Exit(2)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("getting cwd: %v\n", err)
		os.Exit(2)
	}

	o := filepath.Base(cwd)
	var args []string
	for i := 1; i < len(os.Args); i++ {
		if strings.TrimSpace(os.Args[i]) == "-o" && len(os.Args) > i+1 {
			o = os.Args[i+1]
			i++
		} else if prefix, arg := "-o=", strings.TrimLeftFunc(os.Args[i], unicode.IsSpace); strings.HasPrefix(arg, prefix) {
			o = arg[len(prefix):]
		} else {
			args = append(args, os.Args[i])
		}
	}
	if !filepath.IsAbs(o) {
		o = filepath.Join(cwd, o)
	}

	tmpdir, err := os.MkdirTemp("", "mgo")
	if err != nil {
		fmt.Printf("creating temp dir: %v\n", err)
		os.Exit(2)
	}
	defer os.RemoveAll(tmpdir)

	err = fs.WalkDir(launcherSource, ".", func(path string, d fs.DirEntry, err error) error {
		dpath := filepath.Join(tmpdir, path)
		switch {
		case err != nil:
			return err
		case d.Type().IsRegular():
			buf, err := fs.ReadFile(launcherSource, path)
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
		os.Exit(2)
	}

	eg, ctx := errgroup.WithContext(context.Background())
	sema := make(chan struct{}, runtime.NumCPU())
	for _, v := range []string{"v1", "v2", "v3", "v4"} {
		v := v
		eg.Go(func() error {
			sema <- struct{}{}
			defer func() { <-sema }()
			cmd := exec.CommandContext(ctx, "go")
			cmd.Args = append([]string{"go", "build", "-o", filepath.Join(tmpdir, "launcher", "mgo."+v)}, args...)
			cmd.Env = append(os.Environ(), "GOAMD64="+v)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("building variant %q: %w\noutput:\n%s", v, err, output)
			}
			return nil
		})
	}
	err = eg.Wait()
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	cmd = exec.Command("go")
	cmd.Args = []string{"go", "build", "-C", tmpdir, "-o", o, "-trimpath", "./launcher"}
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("building launcher: %v\noutput:\n%s\n", err, output)
		os.Exit(2)
	}
}

//go:embed go.mod go.sum launcher
var launcherSource embed.FS
