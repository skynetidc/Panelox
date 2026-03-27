package filemanager

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	ErrPathTraversal = errors.New("path traversal not allowed")
	ErrNotFound      = errors.New("file or directory not found")
	ErrPermission    = errors.New("permission denied")
)

// allowedRoots are the directories that users can access.
var allowedRoots = []string{"/www", "/home", "/tmp", "/var/log", "/opt"}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// validatePath checks that the resolved path is within an allowed root directory.
func validatePath(path string) (string, error) {
	cleaned := filepath.Clean(path)
	if !filepath.IsAbs(cleaned) {
		return "", ErrPathTraversal
	}

	resolved, err := filepath.EvalSymlinks(filepath.Dir(cleaned))
	if err != nil {
		// Parent doesn't exist yet, use cleaned path
		resolved = filepath.Dir(cleaned)
	}
	fullPath := filepath.Join(resolved, filepath.Base(cleaned))

	for _, root := range allowedRoots {
		if strings.HasPrefix(fullPath, root) {
			return cleaned, nil
		}
	}

	return "", ErrPathTraversal
}

func (s *Service) List(path string) ([]FileEntry, error) {
	safePath, err := validatePath(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(safePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		if os.IsPermission(err) {
			return nil, ErrPermission
		}
		return nil, err
	}

	result := make([]FileEntry, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		fe := FileEntry{
			Name:    entry.Name(),
			Path:    filepath.Join(safePath, entry.Name()),
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			Mode:    info.Mode().String(),
			ModTime: info.ModTime(),
		}

		if entry.Type()&fs.ModeSymlink != 0 {
			fe.IsSymlink = true
			target, err := os.Readlink(filepath.Join(safePath, entry.Name()))
			if err == nil {
				fe.LinkTarget = target
			}
		}

		result = append(result, fe)
	}

	return result, nil
}

func (s *Service) ReadFile(path string) (*ReadFileResponse, error) {
	safePath, err := validatePath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(safePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if info.IsDir() {
		return nil, fmt.Errorf("cannot read directory as file")
	}

	// Limit file size to 10MB for reading
	if info.Size() > 10*1024*1024 {
		return nil, fmt.Errorf("file too large (max 10MB)")
	}

	content, err := os.ReadFile(safePath)
	if err != nil {
		return nil, err
	}

	return &ReadFileResponse{
		Content: string(content),
		Path:    safePath,
		Size:    info.Size(),
		Mode:    info.Mode().String(),
	}, nil
}

func (s *Service) WriteFile(path, content string) error {
	safePath, err := validatePath(path)
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(safePath), 0755); err != nil {
		return err
	}

	return os.WriteFile(safePath, []byte(content), 0644)
}

func (s *Service) Upload(path string, reader io.Reader, filename string) error {
	destDir, err := validatePath(path)
	if err != nil {
		return err
	}

	destPath := filepath.Join(destDir, filepath.Base(filename))
	// Re-validate the full destination path
	if _, err := validatePath(destPath); err != nil {
		return err
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, reader)
	return err
}

func (s *Service) Delete(path string) error {
	safePath, err := validatePath(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(safePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}

	if info.IsDir() {
		return os.RemoveAll(safePath)
	}
	return os.Remove(safePath)
}

func (s *Service) Mkdir(path string) error {
	safePath, err := validatePath(path)
	if err != nil {
		return err
	}

	return os.MkdirAll(safePath, 0755)
}

func (s *Service) Rename(oldPath, newPath string) error {
	safeOld, err := validatePath(oldPath)
	if err != nil {
		return err
	}
	safeNew, err := validatePath(newPath)
	if err != nil {
		return err
	}

	return os.Rename(safeOld, safeNew)
}

func (s *Service) Chmod(path, mode string) error {
	safePath, err := validatePath(path)
	if err != nil {
		return err
	}

	m, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return fmt.Errorf("invalid mode: %s", mode)
	}

	return os.Chmod(safePath, os.FileMode(m))
}
