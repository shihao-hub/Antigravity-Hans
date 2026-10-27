package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// readRegularFile checks the directory entry before opening and rechecks the
// opened file. Nonblocking/no-follow flags also protect the Unix open itself.
func readRegularFile(path string, limit int64) ([]byte, os.FileInfo, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, nil, err
	}
	defer root.Close()
	return readRootRegularFile(root, filepath.Base(path), limit)
}

func readRootRegularFile(root *os.Root, name string, limit int64) ([]byte, os.FileInfo, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, nil, fmt.Errorf("拒绝非普通文件或超出体积上限的文件: %s", name)
	}
	f, err := root.OpenFile(name, os.O_RDONLY|safeReadFlags(), 0)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, nil, fmt.Errorf("读取期间文件身份发生变化: %s", name)
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, nil, err
	}
	if int64(len(data)) > limit {
		return nil, nil, fmt.Errorf("文件超出体积上限: %s", name)
	}
	return data, opened, nil
}

// Paths used for generated launchers must not traverse links, including links
// that point elsewhere inside the same root.
func ensureRootDirectory(root *os.Root, name string) error {
	if !filepath.IsLocal(name) {
		return fmt.Errorf("无效的相对目录: %s", name)
	}
	current := ""
	for _, part := range strings.Split(filepath.Clean(name), string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := root.Lstat(current)
		if os.IsNotExist(err) {
			if err = root.Mkdir(current, 0755); err != nil && !os.IsExist(err) {
				return err
			}
			info, err = root.Lstat(current)
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("目录不是安全的普通目录: %s", current)
		}
	}
	return nil
}

func writeRootFileAtomic(root *os.Root, name string, data []byte, mode os.FileMode) error {
	if !filepath.IsLocal(name) {
		return fmt.Errorf("无效的相对路径: %s", name)
	}
	if err := ensureRootDirectory(root, filepath.Dir(name)); err != nil {
		return err
	}
	if info, err := root.Lstat(name); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("拒绝覆盖非普通文件: %s", name)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	tmp := filepath.Join(filepath.Dir(name), fmt.Sprintf(".agy-hans-%x.tmp", nonce))
	f, err := root.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer root.Remove(tmp)
	if _, err = f.Write(data); err == nil {
		err = f.Chmod(mode)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	// Rename replaces the directory entry rather than following a final symlink.
	return root.Rename(tmp, name)
}
