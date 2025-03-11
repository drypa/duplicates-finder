package files

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spaolacci/murmur3"
	"io"
	"os"
	"path/filepath"
)

type File struct {
	Size     int64
	FullPath string
	Hash     string
}

func NewFile(fullPath string) (*File, error) {
	stat, err := os.Stat(fullPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to stat file '%s'", fullPath)
	}
	hash, err := hashFile(fullPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to hash file '%s'", fullPath)
	}

	f := &File{FullPath: fullPath, Size: stat.Size(), Hash: hash}
	return f, nil
}

func hashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := murmur3.New128()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	h1, h2 := hasher.Sum128()
	return fmt.Sprintf("%x%x", h1, h2), nil
}

func (f *File) FileName() string {
	return filepath.Base(f.FullPath)
}

func (f *File) Equals(other *File) bool {
	return f.Size == other.Size && f.FileName() == other.FileName() && f.Hash == other.Hash
}
