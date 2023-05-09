// This is the code of the launcher process that detects the microarchitecture
// the process is being run on, and launches the appropriate GOAMD64 variant
// that is embedded in the binary.
//
// This code is not compiled in this directory: instead it is copied (together
// with the vendor directory) into the temp directory where the compiled
// GOAMD64 variants are outputted, and it is then compiled from there.

//go:build linux && amd64

package main

import (
	"embed"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"

	cpuid "github.com/klauspost/cpuid/v2"
	"golang.org/x/sys/unix"
)

//go:embed mgo.v1 mgo.v2 mgo.v3 mgo.v4
var f embed.FS

func main() {
	level := cpuid.CPU.X64Level()

	switch os.Getenv("GOAMD64") {
	default:
	case "v1":
		level = 1
	case "v2":
		level = 2
	case "v3":
		level = 3
	case "v4":
		level = 4
	}

	var err error
	switch level {
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

	fd, err := unix.MemfdCreate("", unix.MFD_CLOEXEC)
	if err != nil {
		return fmt.Errorf("creating memfd: %w", err)
	}

	_, err = syscall.Write(fd, buf)
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

func execveAt(fd int) (err error) {
	s, err := syscall.BytePtrFromString("")
	if err != nil {
		return err
	}
	argv, err := syscall.SlicePtrFromStrings(os.Args)
	if err != nil {
		return err
	}
	envp, err := syscall.SlicePtrFromStrings(os.Environ())
	if err != nil {
		return err
	}

	ret, _, errno := syscall.Syscall6(
		unix.SYS_EXECVEAT,
		uintptr(fd),
		uintptr(unsafe.Pointer(s)),
		uintptr(unsafe.Pointer(&argv[0])),
		uintptr(unsafe.Pointer(&envp[0])),
		unix.AT_EMPTY_PATH,
		0, /* unused */
	)
	runtime.KeepAlive(s)
	runtime.KeepAlive(argv)
	runtime.KeepAlive(envp)
	if int(ret) == -1 {
		return fmt.Errorf("execveat: %w", errno)
	}

	// unreachable
	return fmt.Errorf("execveAt: unreachable")
}
