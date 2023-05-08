# mgo

Build multiple `GOAMD64` variants and bundle them in a launcher capable to pick the
appropriate variant at runtime.

This is mostly useful if you want to provide `GOAMD64` variants because of the extra
runtime performance this yields, but you have no control over which processor the
executable will be run on.

## Install

```
go install github.com/CAFxX/mgo@latest
```

## Usage

When building your code just replace `go build [...]` with `mgo [...]`.

## Notes

- The resulting executable will be over 4 times as large as a normal build output.
- Startup of the resulting executable is going to be a bit slower.
- Currently only GOOS=linux and GOARCH=amd64 are supported. 
