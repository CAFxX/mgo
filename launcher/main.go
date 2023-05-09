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

	fd, err := memfdCreate("/" + os.Args[0])
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
	argv, err := syscall.SlicePtrFromStrings(os.Args)
	if err != nil {
		return err
	}
	envp, err := syscall.SlicePtrFromStrings(os.Environ())
	if err != nil {
		return err
	}
	ret, _, errno := syscall.Syscall6(322, fd, uintptr(unsafe.Pointer(s)), uintptr(unsafe.Pointer(&argv[0])), uintptr(unsafe.Pointer(&envp[0])), 0, 0)
	runtime.KeepAlive(s)
	runtime.KeepAlive(argv)
	runtime.KeepAlive(envp)
	if int(ret) == -1 {
		return errno
	}
	// unreachable
	return fmt.Errorf("execveAt: unreachable")
}
