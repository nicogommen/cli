package file

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/spf13/afero"
)

const hashExt = ".sha256"

// Copy copies a file from the given bytes to destination.
func Copy(destination string, fin []byte) error {
	if _, err := os.Stat(destination); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not stat file: %w", err)
	}

	fout, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	defer fout.Close()

	r := bytes.NewReader(fin)

	if _, err := io.Copy(fout, r); err != nil {
		return fmt.Errorf("could copy file: %w", err)
	}

	return nil
}

// CheckSize checks if a file exists and has an exact size.
func CheckSize(filename string, size int) (bool, error) {
	stat, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("could not stat file: %w", err)
	}
	return stat.Size() == int64(size), nil
}

var testableFS = afero.NewOsFs()

// CheckHash checks if a file has the given SHA256 hash.
// It supports reading the file's current hash from a static file saved next to it with the hashExt extension.
func CheckHash(filename string, hash string) (bool, error) {
	if fh, err := afero.ReadFile(testableFS, filename+hashExt); err == nil {
		return string(fh) == hash, nil
	}
	fh, err := sha256Sum(testableFS, filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return true, err
	}
	return fh == hash, nil
}

func SaveHash(filename string, hash string) error {
	return Copy(filename+hashExt, []byte(hash))
}

// sha256Sum calculates the SHA256 hash of a file.
func sha256Sum(fs afero.Fs, filename string) (string, error) {
	f, err := fs.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// CopyIfChanged copies source data to a destination filename if it has changed.
func CopyIfChanged(destFilename string, source []byte, sourceHash string) error {
	sizeOK, err := CheckSize(destFilename, len(source))
	if err != nil {
		return err
	}
	if sizeOK {
		hashOK, err := CheckHash(destFilename, sourceHash)
		if hashOK || err != nil {
			return err
		}
	}
	if err := Copy(destFilename, source); err != nil {
		return err
	}
	return SaveHash(destFilename, sourceHash)
}
