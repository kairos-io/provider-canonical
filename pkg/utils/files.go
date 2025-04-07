package utils

import "github.com/twpayne/go-vfs/v4"

func FileExists(fs vfs.FS, path string) bool {
	info, err := fs.Stat(path)
	return err == nil && !info.IsDir()
}
