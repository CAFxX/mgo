// This is the code of the launcher process that detects the microarchitecture
// the process is being run on, and launches the appropriate GOAMD64 variant
// that is embedded in the binary.
//
// This code is not compiled in this directory: instead it is copied (together
// with go.mod/go.sum) into the temp directory where the compiled GOAMD64
// variants are output, and it is then compiled from there.

//go:build mgo_launcher && linux && amd64

package main

import (
	_ "embed"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"syscall"
	"unsafe"

	"github.com/klauspost/cpuid/v2"
	"golang.org/x/sys/unix"
)

var (
	//go:embed mgo.v1
	v1 string
	//go:embed mgo.v2
	v2 string
	//go:embed mgo.v3
	v3 string
	//go:embed mgo.v4
	v4 string
)

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

	v := v1
	switch level {
	case 2:
		v = v2
	case 3:
		v = v3
	case 4:
		v = v4
	}

	switch os.Getenv("MGODEBUG") {
	case "extract":
		os.Stdout.WriteString(v)
		os.Exit(0)
	case "log":
		fmt.Fprintf(os.Stderr, "[mgo] launcher: starting variant GOAMD64=v%d\n", level)
	}

	exe := os.Args[0]
	if _exe, _ := os.Executable(); _exe != "" {
		exe = _exe
	}
	exe = fmt.Sprintf("%s [GOAMD64=v%d]", exe, level)

	// TODO: create fd pointing directly to the data embedded in the launcher?
	// TODO: handle via O_TMPFILE the case in which the executable does not fit into a memfd
	fd, err := unix.MemfdCreate(exe, unix.MFD_CLOEXEC)
	if err != nil {
		panicf("creating memfd: %w", err)
	}
	defer unix.Close(fd)

	written, err := syscall.Write(fd, unsafe.Slice(unsafe.StringData(v), len(v)))
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
