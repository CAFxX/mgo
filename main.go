//go:build !mgo_launcher

package main

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
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

	stdout := &lwriter{Writer: os.Stdout}
	stderr := &lwriter{Writer: os.Stderr}

	np := runtime.NumCPU()
	if pbi, _ := strconv.Atoi(os.Getenv("MGO_PARALLEL_BUILD")); pbi > 0 {
		np = pbi
	}
	sema := make(chan struct{}, np)

	eg, ctx := errgroup.WithContext(context.Background())
	for _, v := range []string{"v1", "v2", "v3", "v4"} {
		eg.Go(func() error {
			sema <- struct{}{}
			defer func() { <-sema }()
			cmd := exec.CommandContext(ctx, "go")
			cmd.Args = append([]string{"go", "build", "-o", filepath.Join(tmpdir, "mgo."+v)}, args...)
			cmd.Env = append(os.Environ(), "GOAMD64="+v)
			cmd.Stdout = &writer{prefix: []byte(v + ": "), w: stdout}
			cmd.Stderr = &writer{prefix: []byte(v + ": "), w: stderr}
			err := cmd.Run()
			if err != nil {
				return fmt.Errorf("building variant %q: %w", v, err)
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
	cmd.Args = []string{
		"go", "build", 
		"-C", tmpdir, 
		"-o", o, 
		"-trimpath", 
		"-ldflags", "-s -w", 
		"-tags", "mgo_launcher",
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Printf("building launcher: %v\n", err)
		os.Exit(2)
	}
}

//go:embed go.mod go.sum launcher.go
var launcherSource embed.FS

type writer struct {
	prefix []byte
	w      *lwriter
	buf    bytes.Buffer
}

func (w *writer) Write(buf []byte) (int, error) {
	w.w.Lock()
	defer w.w.Unlock()

	r, _ := w.buf.Write(buf)

	for {
		line, err := w.buf.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return r, nil
			}
			return r, err
		}
		_, err = w.write(w.prefix)
		if err != nil {
			return r, err
		}
		_, err = w.write(line)
		if err != nil {
			return r, err
		}
	}
}

func (w *writer) write(buf []byte) (int, error) {
	var r int
	for len(buf) > 0 {
		n, err := w.w.Write(buf)
		r += n
		if err != nil {
			return r, err
		}
		buf = buf[n:]
	}
	return r, nil
}

type lwriter struct {
	sync.Mutex
	io.Writer
}
