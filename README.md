# mgo

Build multiple `GOAMD64` variants and bundle them in a launcher capable to pick the
appropriate variant at runtime.

This is mostly useful if you want to provide `GOAMD64` variants because of the extra
runtime performance this yields, but you have no control over which processor 
microarchitecture the executable will be run on.

## Install

```
go install github.com/CAFxX/mgo@latest
```

## Usage

When building your code just replace `go build [...]` with `mgo [...]`: the resulting
executable will contain 4 variants, each optimized for one of `GOAMD64=v1`, `GOAMD64=v2`,
`GOAMD64=v3` and `GOAMD64=v4`, and a launcher that will pick the appropriate one at
runtime.

At runtime it is possible to override which variant is used by specifying in the
`GOAMD64` environment variable one of the values `v1`, `v2`, `v3`, or `v4`.

## Notes

- `mgo` requires Go >= 1.18 (older versions do not support `GOAMD64`)
- The resulting executable will be over 4 times as large as a normal build output
- Startup of the resulting executable is going to be a bit slower (tens of milliseconds)
- Currently only `GOOS=linux` and `GOARCH=amd64` are supported, and only in
  `buildmode`s that produce executables (not archives, plugins, or libraries).

## Quick sanity check

```
echo stage0 && go build &&\
echo stage1 && ./mgo -o vmgo && sha1sum vmgo &&\
echo stage2 && ./vmgo -o vmgo2 && sha1sum vmgo2 &&\
echo stage3 && ./vmgo2 -o vmgo3 && sha1sum vmgo3
```

This command should succeed and produce three identical hashes.
