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
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/cpuid/v2"
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

	var r io.Reader
	r = strings.NewReader(v)

	var err error
	switch mgoCompress {
	case 0:
		// no compression
	case 1:
		r, err = gzip.NewReader(r)
	case 2:
		r, err = zstd.NewReader(r)
	case 3:
		r, err = zstd.NewReader(r, zstd.WithDecoderDictRaw(0, decodeBase()), zstd.WithDecoderMaxWindow(zstd.MaxWindowSize))
	default:
		panicf("invalid mgoCompress: %d", mgoCompress)
	}
	if mgoCompress != 0 && err != nil {
		panicf("decompression failed: %w", err)
	}

	switch os.Getenv("MGODEBUG") {
	case "extract":
		_, err := io.Copy(os.Stdout, r)
		if err != nil {
			panicf("extraction failed: %w", err)
		}
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
	memfd := os.NewFile(uintptr(fd), exe)
	defer memfd.Close()

	_, err = io.Copy(memfd, r)
	if err != nil {
		panicf("writing to memfd: %w", err)
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

func decodeBase() []byte {
	if mgoCompress != 3 {
		panicf("illegal mgoCompress: %d", mgoCompress)
	}

	var buf bytes.Buffer
	r, err := zstd.NewReader(strings.NewReader(v1))
	if err != nil {
		panicf("error start decompressing base: %w", err)
	}
	_, err = io.Copy(&buf, r)
	if err != nil {
		panicf("error decompressing base: %w", err)
	}

	return buf.Bytes()
}
