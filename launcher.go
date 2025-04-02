// This is the code of the launcher process that detects the microarchitecture
// the process is being run on, and launches the appropriate GOAMD64 variant
// that is embedded in the binary.
//
// This code is not compiled in this directory: instead it is copied (together
// with go.mod/go.sum) into the temp directory where the compiled GOAMD64
// variants are output, and it is then compiled from there.

//go:build mgo_launcher && linux && (amd64 || arm64)

package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func init() {
	runtime.GOMAXPROCS(1)
	debug.SetGCPercent(-1)
}

func main() {
	envVar, variant, binary := getVariant()

	switch os.Getenv("MGODEBUG") {
	case "extract":
		fmt.Fprintf(os.Stderr, "[mgo] launcher: dumping variant %s=%s\n", envVar, variant)
		os.Stdout.WriteString(binary)
		os.Exit(0)
	case "log":
		fmt.Fprintf(os.Stderr, "[mgo] launcher: starting variant %s=%s\n", envVar, variant)
	}

	exe := os.Args[0]
	if _exe, _ := os.Executable(); _exe != "" {
		exe = _exe
	}
	exe = fmt.Sprintf("%s [%s=%s]", exe, envVar, variant)

	// TODO: create fd pointing directly to the data embedded in the launcher?
	// TODO: handle via O_TMPFILE the case in which the executable does not fit into a memfd
	fd, err := unix.MemfdCreate(exe, unix.MFD_CLOEXEC)
	if err != nil {
		panicf("creating memfd: %w", err)
	}
	defer unix.Close(fd)

	written, err := syscall.Write(fd, unsafe.Slice(unsafe.StringData(binary), len(binary)))
	if err != nil {
		panicf("writing to memfd: %w", err)
	} else if written != len(v) {
		panic("short write to memfd")
	}

	// TODO: seal the memfd?

	s, err := syscall.BytePtrFromString("")
	if err != nil {
		panicf("converting path: %w", err)
	}
	defer runtime.KeepAlive(s)
	argv, err := syscall.SlicePtrFromStrings(os.Args)
	if err != nil {
		panicf("converting args: %w", err)
	}
	defer runtime.KeepAlive(argv)
	envp, err := syscall.SlicePtrFromStrings(os.Environ())
	if err != nil {
		panicf("converting environ: %w", err)
	}
	defer runtime.KeepAlive(envp)

	ret, _, errno := syscall.Syscall6(
		unix.SYS_EXECVEAT,
		uintptr(fd),
		uintptr(unsafe.Pointer(s)),
		uintptr(unsafe.Pointer(&argv[0])),
		uintptr(unsafe.Pointer(&envp[0])),
		unix.AT_EMPTY_PATH,
		0, /* unused */
	)

	// execveat returns only in case of failure
	panicf("execveat: %d %w", ret, errno)
}

func panicf(format string, args ...any) {
	panic(fmt.Errorf(format, args...))
}
