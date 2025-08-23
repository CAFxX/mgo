# mgo

Build your code for multiple [`GOAMD64` variants][1] and bundle all of them in a 
launcher capable of picking at runtime the most appropriate variant for the
processor in use.

This is mostly useful if you want to provide `GOAMD64` variants because of the extra
runtime performance this yields, but you have no control over which processor 
microarchitecture the executable will be run on.

## Install

```
go install github.com/CAFxX/mgo@latest
```

## Usage

### Build time

When building your code just replace `go build [...]` with `mgo [...]`: the resulting
executable will contain 4 variants, each optimized for one of `GOAMD64=v1`, `GOAMD64=v2`,
`GOAMD64=v3` and `GOAMD64=v4`, and a launcher that will pick the appropriate one at
runtime.

Using `MGO_COMPRESS`, will enable the transparent use of compression of the variants
embedded in the executable. This reduces the size the size of the executable, at the
cost of additional overhead when the executable is launched. The following values are
supported:

- `g$n` with `$n` a number from 1 to 9 will use Gzip with compression level `$n`
- `z$n` with `$n` a number from 1 to 11 will use Zstandard with compression level `$n`
- `Z$n` with `$n` a number from 1 to 11 will use Zstandard with compression level `$n`
  and dictionary compression using the base variant as the dictionary

### Run time

At runtime it is possible to override which variant is used by specifying in the
`GOAMD64` environment variable one of the values `v1`, `v2`, `v3`, or `v4`.

To check which version is being executed you can add the `MGODEBUG=log` environment
variable when starting the compiled binary. In this case the launcher will print on
stderr a line similar to the following at startup (in this example, to signal that
the `v3` variant is being used):

```
[mgo] launcher: starting variant GOAMD64=v3
```

Otherwise you can find out which version is being used by resolving the `/proc/<PID>/exe`
symlink, where `<PID>` is the process ID of the launched process:

```
$ PID=...
$ readlink /proc/$PID/exe
/memfd:/usr/bin/foobar [GOAMD64=v3] (deleted)
```

You can extract a specific embedded executable using `MGODEBUG=extract` when starting the
compiled binary, e.g. `MGODEBUG=extract GOAMD64=v3` will dump to stdout the executable for
`GOAMD64=v3` instead of starting it.

## Notes

- `mgo` requires Go >= 1.20
- The resulting executable will be over 4 times as large as a normal build output
- Startup of the resulting executable is going to be a bit slower (tens of milliseconds)
- Currently only `GOOS=linux` and `GOARCH=amd64` are supported, and only in
  `buildmode`s that produce executables (not archives, plugins, or libraries)

## TODO

- Further minimize launcher overhead
  - Start process with `GOMAXPROCS=1` and `GOGC=off` instead of setting them in `init`
  - Avoid the `write` to the memfd (find ways to populate the new process directly from the binary in `RODATA`)
- Embed build metadata of the built `v1` variant into the launcher, so that introspection
  tools can use it
- Support `GOARCH=arm` and [`GOARCH=arm64`](https://github.com/golang/go/issues/60905)

## Quick sanity check

```bash
echo stage0 && go build && \
for mgo_compress in "" g1 g9 z1 z11 Z1 Z11; do
  (
    echo -e "\033[A\r\033[KMGO_COMPRESS=$mgo_compress"
    export MGO_COMPRESS=$mgo_compress
    rm -f mgo? && \
    echo stage1 && ./mgo -o mgo1 && h1=$(sha1sum mgo1 | cut -c-40) && \
    echo -e "\033[A\r\033[Kstage2" && ./mgo1 -o mgo2 && h2=$(sha1sum mgo2 | cut -c-40) && \
    echo -e "\033[A\r\033[Kstage3" && ./mgo2 -o mgo3 && h3=$(sha1sum mgo3 | cut -c-40) && \
    if [ "$h1" != "$h2" ] || [ "$h3" != "$h3" ]; then
      echo -e "\033[A\r\033[KHashes are not identical: $h1, $h2, $h3\n"
    fi || \
    echo "Build failed\n" && exit 1
  )
done
echo -e "\033[A\r\033[K"
```

If no failures are reported, everything should be working correctly.



[1]: https://go.dev/wiki/MinimumRequirements#amd64
