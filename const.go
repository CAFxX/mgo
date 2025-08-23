package main

type compressionType int

const (
	compressionTypeNone compressionType = iota
	compressionTypeGzip
	compressionTypeZstd
	compressionTypeZstdWithDict
)

const (
	maxDeflateDict = 1 << 15
	maxZstdDict    = 1 << 31 // https://github.com/klauspost/compress/blob/8e79dc4b98d4c5a09c62a2546b79c14edf7c3e38/zstd/dict.go#L27
)
