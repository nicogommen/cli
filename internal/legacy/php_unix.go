//go:build darwin || linux
// +build darwin linux

package legacy

import (
	"github.com/platformsh/cli/internal/file"
	"path"
)

// copyPHP to destination, if it does not exist
func (c *CLIWrapper) copyPHP() error {
	return file.CopyIfChanged(c.PHPPath(), phpCLI, phpCLIHash)
}

// PHPPath returns the path that the PHP CLI will reside
func (c *CLIWrapper) PHPPath() string {
	return path.Join(c.cacheDir(), phpPath)
}
