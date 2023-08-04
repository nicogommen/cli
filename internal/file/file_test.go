package file

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckHash(t *testing.T) {
	// Temporarily swap to a memory filesystem.
	testableFS = afero.NewMemMapFs()
	defer func() {
		testableFS = afero.NewOsFs()
	}()
	fs := testableFS

	mockContent := "hello world\n"
	mockContentHash := "a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447"
	diffContent := "hello world?\n"
	diffContentHash := "d441ffff4b6663c3f150bda9c519a58c0685e34cf13d26e881d7e004f704eeba"

	cases := []struct {
		name      string
		content   string
		writeHash string

		checkHash  string
		shouldFail bool
	}{
		{
			name:      "static_hash_good",
			writeHash: mockContentHash,
			checkHash: mockContentHash,
		},
		{
			name:       "static_hash_bad",
			writeHash:  diffContentHash,
			checkHash:  mockContentHash,
			shouldFail: true,
		},
		{
			name:      "dynamic_hash_good",
			content:   diffContent,
			checkHash: diffContentHash,
		},
		{
			name:       "dynamic_hash_bad",
			content:    mockContent,
			checkHash:  diffContentHash,
			shouldFail: true,
		},
	}

	filename := "test"
	for _, c := range cases {
		require.NoError(t, removeIfExists(fs, filename))
		require.NoError(t, removeIfExists(fs, filename+hashExt))
		t.Run(c.name, func(t *testing.T) {
			require.NoError(t, afero.WriteFile(fs, filename, []byte(c.content), 0644))
			if c.writeHash != "" {
				require.NoError(t, afero.WriteFile(fs, filename+hashExt, []byte(c.writeHash), 0644))
			}
			hashOK, err := CheckHash(filename, c.checkHash)
			assert.NoError(t, err)
			assert.Equal(t, !c.shouldFail, hashOK)
		})
	}
}

func removeIfExists(fs afero.Fs, filename string) error {
	if err := fs.Remove(filename); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
