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
	"runtime/debug"
	"syscall"
	"unsafe"

	cpuid "github.com/klauspost/cpuid/v2"
	"golang.org/x/sys/unix"
)

//go:embed mgo.v1 mgo.v2 mgo.v3 mgo.v4
var f embed.FS

func init() {
	runtime.GOMAXPROCS(1)
	debug.SetGCPercent(-1)
}

func main() {
	var level int
	switch os.Getenv("GOAMD64") {
	case "v1":
		level = 1
	case "v2":
		level = 2
	case "v3":
		level = 3
	case "v4":
		level = 4
	default:
		level = cpuid.CPU.X64Level()
	}

	v := "mgo.v1"
	switch level {
	case 2:
		v = "mgo.v2"
	case 3:
		v = "mgo.v3"
	case 4:
		v = "mgo.v4"
	}

	// FIXME: avoid alloc+copy when reading from embed.FS
	buf, err := f.ReadFile(v)
	if err != nil {
		panicf("reading embedded file %q: %w", v, err)
	}

	// TODO: create fd pointing directly to the data embedded?
	fd, err := unix.MemfdCreate("", unix.MFD_CLOEXEC)
	if err != nil {
		panicf("creating memfd: %w", err)
	}

	_, err = syscall.Write(fd, buf)
	if err != nil {
		panicf("writing to memfd: %w", err)
	}

	s, err := syscall.BytePtrFromString("")
	if err != nil {
		panicf("converting path: %w", err)
	}
	argv, err := syscall.SlicePtrFromStrings(os.Args)
	if err != nil {
		panicf("converting args: %w", err)
	}
	envp, err := syscall.SlicePtrFromStrings(os.Environ())
	if err != nil {
		panicf("converting environ: %w", err)
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

	// execveat returns only in case of failure
	panicf("execveat: %d %w", ret, errno)
}

func panicf(format string, args ...any) {
	panic(fmt.Errorf(format, args...))
}
