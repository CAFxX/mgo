package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	_ "github.com/klauspost/cpuid/v2"
)

func main() {
	if !((os.Getenv("GOOS") == "" && runtime.GOOS == "linux") || os.Getenv("GOOS") == "linux") {
		fmt.Printf("GOOS=%q is not supported\n", os.Getenv("GOOS"))
		return
	}
	if !((os.Getenv("GOARCH") == "" && runtime.GOARCH == "amd64") || os.Getenv("GOARCH") == "amd64") {
		fmt.Printf("GOARCH=%q is not supported\n", os.Getenv("GOARCH"))
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("getting cwd: %v\n", err)
		return
	}

	o := flag.String("o", filepath.Base(cwd), "output file")
	flag.Parse()

	var variants = []string{"v1", "v2", "v3", "v4"}

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

	for _, v := range variants {
		cmd := exec.Command("go")
		cmd.Args = append([]string{"go", "build", "-o", filepath.Join(tmpdir, "mgo."+v)}, flag.Args()...)
		cmd.Env = append(os.Environ(), "GOAMD64="+v)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Printf("building variant %q: %v\n", v, err)
			return
		}
	}

	cmd := exec.Command("go")
	cmd.Args = append([]string{"go", "build", "-mod=vendor", "-o", filepath.Join(cwd, *o)}, "-trimpath", filepath.Join(tmpdir, "main.go"))
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

const launcher = `
//go:build linux && amd64
package main

import (
	"embed"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	cpuid "github.com/klauspost/cpuid/v2"
)

//go:embed mgo.v1 mgo.v2 mgo.v3 mgo.v4
var f embed.FS

func main() {
	var err error
	switch cpuid.CPU.X64Level() {
	default:
		err = embeddedExec(f, "mgo.v1")
	case 2:
		err = embeddedExec(f, "mgo.v2")
	case 3:
		err = embeddedExec(f, "mgo.v3")
	case 4:
		err = embeddedExec(f, "mgo.v4")
	}
	if err != nil {
		panic(err)
	}
}

func embeddedExec(f embed.FS, s string) error {
	buf, err := f.ReadFile(s)
	if err != nil {
		return fmt.Errorf("reading embedded file: %w", err)
	}

	fd, err := memfdCreate("/"+os.Args[0])
	if err != nil {
		return fmt.Errorf("creating memfd: %w", err)
	}

	err = copyToMem(fd, buf)
	if err != nil {
		return fmt.Errorf("writing to memfd: %w", err)
	}

	err = execveAt(fd)
	if err != nil {
		return fmt.Errorf("executing: %w", err)
	}

	// unreachable
	return fmt.Errorf("embeddedExec: unreachable")
}

func memfdCreate(path string) (r1 uintptr, err error) {
    s, err := syscall.BytePtrFromString(path)
    if err != nil {
        return 0, err
    }
 
    r1, _, errno := syscall.Syscall(319, uintptr(unsafe.Pointer(s)), 0, 0)
    if int(r1) == -1 {
        return r1, errno
    }
 
    return r1, nil
}

func copyToMem(fd uintptr, buf []byte) (err error) {
    _, err = syscall.Write(int(fd), buf)
    if err != nil {
        return err
    }
    return nil
}

func execveAt(fd uintptr) (err error) {
    s, err := syscall.BytePtrFromString("")
    if err != nil {
        return err
    }
    ret, _, errno := syscall.Syscall6(322, fd, uintptr(unsafe.Pointer(s)), 0, 0, 0x1000, 0)
    if int(ret) == -1 {
        return errno
    }
    // unreachable
    return fmt.Errorf("execveAt: unreachable")
}
`
