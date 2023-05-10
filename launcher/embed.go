package launcher

import "embed"

//go:embed go.mod go.sum cmd vendor
var Source embed.FS
