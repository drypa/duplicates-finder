package files

import (
	"context"
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

var errCancelled = errors.New("operation cancelled")

func NewFile(ctx context.Context, fullPath string) (*File, error) {
	stat, err := os.Stat(fullPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to stat file '%s'", fullPath)
	}
	hash, err := hashFile(ctx, fullPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to hash file '%s'", fullPath)
	}

	f := &File{FullPath: fullPath, Size: stat.Size(), Hash: hash}
	return f, nil
}

func hashFile(ctx context.Context, filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := murmur3.New128()
	buf := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return "", errCancelled
		}
		n, readErr := file.Read(buf)
		if n > 0 {
			_, writeErr := hasher.Write(buf[:n])
			if writeErr != nil {
				return "", writeErr
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}

	h1, h2 := hasher.Sum128()
	return fmt.Sprintf("%x%x", h1, h2), nil
}

func IsCancelled(err error) bool {
	return errors.Is(err, errCancelled) || errors.Is(err, context.Canceled)
}

func (f *File) FileName() string {
	return filepath.Base(f.FullPath)
}

func (f *File) Equals(other *File) bool {
	return f.Size == other.Size && f.FileName() == other.FileName() && f.Hash == other.Hash
}
