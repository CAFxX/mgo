//go:build mgo_launcher && linux && arm64

package main

import (
	_ "embed"
	"os"
)

func getVariant() (string, string, string) {
	envVar := "GOARM64"

	var level string
	switch v := os.Getenv(envVar); v {
	case "v8.0", "v8.0,lse", "v8.0,crypto", "v8.0,crypto,lse",
		"v8.1", "v8.1,crypto", "v8.2", "v8.2,crypto", "v8.3", "v8.3,crypto", "v8.4", "v8.4,crypto", "v8.5", "v8.5,crypto",
		"v8.6", "v8.6,crypto", "v8.7", "v8.7,crypto", "v8.8", "v8.8,crypto", "v8.9", "v8.9,crypto", "v9.0", "v9.0,crypto",
		"v9.1", "v9.1,crypto", "v9.2", "v9.2,crypto", "v9.3", "v9.3,crypto", "v9.4", "v9.4,crypto", "v9.5", "v9.5,crypto":
		level = v
	case "v8.0,lse,crypto":
		level = "v8.0,crypto,lse"
	default:
		level = "v8.0" // FIXME: autodetect
	}

	switch level {
	default:
		return envVar, "v8.0", v8_0
	case "v8.0,lse":
		return envVar, level, v8_0_lse
	case "v8.0,crypto":
		return envVar, level, v8_0_crypto
	case "v8.0,crypto,lse":
		return envVar, level, v8_0_crypto_lse
	case "v8.1":
		return envVar, level, v8_1
	case "v8.2":
		return envVar, level, v8_2
	case "v8.3":
		return envVar, level, v8_3
	case "v8.4":
		return envVar, level, v8_4
	case "v8.5":
		return envVar, level, v8_5
	case "v8.6":
		return envVar, level, v8_6
	case "v8.7":
		return envVar, level, v8_7
	case "v8.8":
		return envVar, level, v8_8
	case "v8.9":
		return envVar, level, v8_9
	case "v9.0":
		return envVar, level, v9_0
	case "v9.1":
		return envVar, level, v9_1
	case "v9.2":
		return envVar, level, v9_2
	case "v9.3":
		return envVar, level, v9_3
	case "v9.4":
		return envVar, level, v9_4
	case "v9.5":
		return envVar, level, v9_5
	case "v8.1,crypto":
		return envVar, level, v8_1_crypto
	case "v8.2,crypto":
		return envVar, level, v8_2_crypto
	case "v8.3,crypto":
		return envVar, level, v8_3_crypto
	case "v8.4,crypto":
		return envVar, level, v8_4_crypto
	case "v8.5,crypto":
		return envVar, level, v8_5_crypto
	case "v8.6,crypto":
		return envVar, level, v8_6_crypto
	case "v8.7,crypto":
		return envVar, level, v8_7_crypto
	case "v8.8,crypto":
		return envVar, level, v8_8_crypto
	case "v8.9,crypto":
		return envVar, level, v8_9_crypto
	case "v9.0,crypto":
		return envVar, level, v9_0_crypto
	case "v9.1,crypto":
		return envVar, level, v9_1_crypto
	case "v9.2,crypto":
		return envVar, level, v9_2_crypto
	case "v9.3,crypto":
		return envVar, level, v9_3_crypto
	case "v9.4,crypto":
		return envVar, level, v9_4_crypto
	case "v9.5,crypto":
		return envVar, level, v9_5_crypto
	}
}

var (
	//go:embed mgo.v8.0
	v8_0 string
	//go:embed mgo.v8.0,lse
	v8_0_lse string
	//go:embed mgo.v8.0,crypto
	v8_0_crypto string
	//go:embed mgo.v8.0,lse,crypto
	v8_0_crypto_lse string
	//go:embed mgo.v8.1
	v8_1 string
	//go:embed mgo.v8.2
	v8_2 string
	//go:embed mgo.v8.3
	v8_3 string
	//go:embed mgo.v8.4
	v8_4 string
	//go:embed mgo.v8.5
	v8_5 string
	//go:embed mgo.v8.6
	v8_6 string
	//go:embed mgo.v8.7
	v8_7 string
	//go:embed mgo.v8.8
	v8_8 string
	//go:embed mgo.v8.9
	v8_9 string
	//go:embed mgo.v9.0
	v9_0 string
	//go:embed mgo.v9.1
	v9_1 string
	//go:embed mgo.v9.2
	v9_2 string
	//go:embed mgo.v9.3
	v9_3 string
	//go:embed mgo.v9.4
	v9_4 string
	//go:embed mgo.v9.5
	v9_5 string
	//go:embed mgo.v8.1,crypto
	v8_1_crypto string
	//go:embed mgo.v8.2,crypto
	v8_2_crypto string
	//go:embed mgo.v8.3,crypto
	v8_3_crypto string
	//go:embed mgo.v8.4,crypto
	v8_4_crypto string
	//go:embed mgo.v8.5,crypto
	v8_5_crypto string
	//go:embed mgo.v8.6,crypto
	v8_6_crypto string
	//go:embed mgo.v8.7,crypto
	v8_7_crypto string
	//go:embed mgo.v8.8,crypto
	v8_8_crypto string
	//go:embed mgo.v8.9,crypto
	v8_9_crypto string
	//go:embed mgo.v9.0,crypto
	v9_0_crypto string
	//go:embed mgo.v9.1,crypto
	v9_1_crypto string
	//go:embed mgo.v9.2,crypto
	v9_2_crypto string
	//go:embed mgo.v9.3,crypto
	v9_3_crypto string
	//go:embed mgo.v9.4,crypto
	v9_4_crypto string
	//go:embed mgo.v9.5,crypto
	v9_5_crypto string
)
