package legacy

import (
	_ "embed"
)

//go:embed archives/php_darwin_amd64
var phpCLI []byte

//go:embed archives/php_darwin_amd64.sha256
var phpCLIHash string
