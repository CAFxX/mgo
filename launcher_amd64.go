//go:build mgo_launcher && linux && amd64

package main

import (
	_ "embed"
	"os"

	"github.com/klauspost/cpuid/v2"
)

func getVariant() (string, string, string) {
	envVar := "GOAMD64"

	var level int
	switch os.Getenv(envVar) {
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

	switch level {
	default:
		return envVar, "v1", v1
	case 2:
		return envVar, "v2", v2
	case 3:
		return envVar, "v3", v3
	case 4:
		return envVar, "v4", v4
	}
}

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
