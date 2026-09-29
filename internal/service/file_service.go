package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// DirEntry is a simplified directory entry returned by ReadDirectory.
type DirEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"isDir"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"` // unix seconds
}

// FileService handles all local filesystem operations for the frontend.
// Frontend code must never touch the filesystem directly; it always goes
// through this service.
type FileService struct{}

// NewFileService returns a new FileService.
func NewFileService() *FileService {
	return &FileService{}
}

// ReadFile reads the entire file at path as UTF-8 text and returns its content.
func (s *FileService) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		// Fall back to latin-1 so we never corrupt data; Muya expects UTF-8.
		out := make([]rune, 0, len(data))
		for _, b := range data {
			out = append(out, rune(b))
		}
		return string(out), nil
	}
	return string(data), nil
}

// WriteFile writes content to path, creating the file if it does not exist
// and truncating it if it does.
func (s *FileService) WriteFile(path, content string) error {
	dir := filepath.Dir(path)
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// PathExists reports whether path exists on the filesystem.
func (s *FileService) PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// IsDirectory reports whether path is a directory.
func (s *FileService) IsDirectory(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// Stat returns metadata for a file without transferring its contents,
// letting the frontend decide whether a path-based operation is possible.
func (s *FileService) Stat(path string) (DirEntry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return DirEntry{}, err
	}
	abs, absErr := filepath.Abs(path)
	if absErr != nil {
		abs = path
	}
	return DirEntry{
		Name:    info.Name(),
		Path:    abs,
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime().Unix(),
	}, nil
}

// ReadDirectory lists the immediate children of dir. Each entry contains
// name, absolute path, isDir flag, size and modTime.
func (s *FileService) ReadDirectory(dir string) ([]DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]DirEntry, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, DirEntry{
			Name:    e.Name(),
			Path:    filepath.Join(dir, e.Name()),
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
		})
	}
	return out, nil
}

// PickOpenFile shows the native open-file dialog and returns the chosen path,
// or an empty string if the user cancels.
func (s *FileService) PickOpenFile() (string, error) {
	dialog := application.Get().Dialog.OpenFile().
		SetTitle("Open Markdown").
		AddFilter("Markdown", "*.md;*.markdown;*.mdown;*.mkd").
		AddFilter("Plain Text", "*.txt").
		AddFilter("All Files", "*.*")
	return dialog.PromptForSingleSelection()
}

// PickPandocFile shows the native open-file dialog defaulting to executable
// files, used for picking a pandoc binary in the settings dialog.
func (s *FileService) PickPandocFile() (string, error) {
	dialog := application.Get().Dialog.OpenFile().
		SetTitle("选择 pandoc 程序").
		AddFilter("可执行程序", "*.exe").
		AddFilter("All Files", "*.*")
	return dialog.PromptForSingleSelection()
}

// PickSaveFile shows the native save-file dialog seeded with defaultName and
// returns the chosen path, or an empty string if the user cancels.
func (s *FileService) PickSaveFile(defaultName string) (string, error) {
	dialog := application.Get().Dialog.SaveFile().
		SetMessage("Save As").
		SetFilename(defaultName).
		AddFilter("Markdown", "*.md;*.markdown;*.mdown;*.mkd").
		AddFilter("Plain Text", "*.txt")
	return dialog.PromptForSingleSelection()
}

// PickOpenDirectory shows the native directory-picker dialog and returns the
// chosen directory path, or an empty string if the user cancels.
func (s *FileService) PickOpenDirectory() (string, error) {
	dialog := application.Get().Dialog.OpenFile().
		SetTitle("Open Folder").
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true)
	return dialog.PromptForSingleSelection()
}

// CreateDirectory creates a new directory at path, including any necessary
// parents. It is a no-op (returns nil) if the directory already exists.
func (s *FileService) CreateDirectory(path string) error {
	return os.MkdirAll(path, 0o755)
}

// DeleteFile removes a file or directory at path. Directories are removed
// recursively.
func (s *FileService) DeleteFile(path string) error {
	return os.RemoveAll(path)
}

// RenamePath renames the file or directory at oldPath to newName within the
// same directory and returns the new absolute path. It rejects empty names,
// names containing path separators or characters that are illegal on Windows,
// and target names that already exist (duplicate-name check).
func (s *FileService) RenamePath(oldPath, newName string) (string, error) {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return "", errors.New("名称不能为空")
	}
	if strings.ContainsAny(newName, `\/:*?"<>|`) {
		return "", errors.New(`名称不能包含 \ / : * ? " < > | 等字符`)
	}
	newPath := filepath.Join(filepath.Dir(oldPath), newName)
	if _, err := os.Stat(newPath); err == nil {
		return "", fmt.Errorf("「%s」已存在，请换一个名称", newName)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return "", err
	}
	return newPath, nil
}

// MovePath moves the file or directory at srcPath into the directory
// destDirPath, preserving its base name, and returns the new absolute path.
// It rejects: moving into itself, moving a directory into one of its own
// descendants, a missing destination directory and name collisions in the
// target folder (resolved by appending " (n)", same as CopyFile).
func (s *FileService) MovePath(srcPath, destDirPath string) (string, error) {
	srcPath = filepath.Clean(srcPath)
	destDirPath = filepath.Clean(destDirPath)

	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return "", fmt.Errorf("源文件不存在：%s", srcPath)
	}
	destInfo, err := os.Stat(destDirPath)
	if err != nil {
		return "", errors.New("目标文件夹不存在")
	}
	if !destInfo.IsDir() {
		return "", errors.New("只能移动到文件夹中")
	}

	// Normalise with a trailing separator for prefix comparisons.
	srcNorm := filepath.ToSlash(srcPath)
	destNorm := filepath.ToSlash(destDirPath)
	if destNorm == srcNorm {
		return "", errors.New("不能移动到自身")
	}
	if srcInfo.IsDir() && strings.HasPrefix(destNorm+"/", srcNorm+"/") {
		return "", errors.New("不能移动到自身或其子文件夹中")
	}

	newPath := availableCopyPath(destDirPath, filepath.Base(srcPath))

	if err := os.Rename(srcPath, newPath); err != nil {
		return "", err
	}
	return newPath, nil
}

// CopyFile copies a file into destDirPath. Name collisions are resolved by
// appending " (n)" before the extension (e.g. "a (1).md"). Returns the final
// destination path. Directories are not supported.
func (s *FileService) CopyFile(srcPath, destDirPath string) (string, error) {
	srcPath = filepath.Clean(srcPath)
	destDirPath = filepath.Clean(destDirPath)

	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return "", fmt.Errorf("源文件不存在：%s", srcPath)
	}
	if srcInfo.IsDir() {
		return "", errors.New("仅支持复制文件")
	}
	destInfo, err := os.Stat(destDirPath)
	if err != nil || !destInfo.IsDir() {
		return "", errors.New("目标文件夹不存在")
	}

	newPath := availableCopyPath(destDirPath, filepath.Base(srcPath))

	in, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer in.Close()

	out, err := os.OpenFile(newPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, srcInfo.Mode())
	if err != nil {
		return "", err
	}
	if _, err := out.ReadFrom(in); err != nil {
		out.Close()
		os.Remove(newPath)
		return "", err
	}
	if err := out.Close(); err != nil {
		os.Remove(newPath)
		return "", err
	}
	return newPath, nil
}

// availableCopyPath returns destDir/name, or "name (n).ext" variants when the
// plain name already exists in destDir.
func availableCopyPath(destDir, name string) string {
	candidate := filepath.Join(destDir, name)
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		candidate = filepath.Join(destDir, fmt.Sprintf("%s (%d)%s", stem, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

// JoinPath joins path elements using the platform separator and returns the
// result. This lets the frontend build child paths without knowing the OS.
func (s *FileService) JoinPath(elem ...string) (string, error) {
	return filepath.Join(elem...), nil
}

// FileInfo carries detailed metadata for a single file, used by the frontend's
// properties dialog.
type FileInfo struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Size       int64  `json:"size"` // bytes
	IsDir      bool   `json:"isDir"`
	ModTime    int64  `json:"modTime"`    // unix seconds (last modified)
	CreateTime int64  `json:"createTime"` // unix seconds (creation time, 0 if unavailable)
	Mode       uint32 `json:"mode"`       // file mode bits
	ReadOnly   bool   `json:"readOnly"`
}

// GetFileInfo returns detailed file metadata for the properties dialog. The
// ReadOnly flag is derived from the owner write bit (perm & 0200 == 0).
func (s *FileService) GetFileInfo(path string) (FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}
	abs, absErr := filepath.Abs(path)
	if absErr != nil {
		abs = path
	}
	return FileInfo{
		Name:       info.Name(),
		Path:       abs,
		Size:       info.Size(),
		IsDir:      info.IsDir(),
		ModTime:    info.ModTime().Unix(),
		CreateTime: fileCreateTime(abs),
		Mode:       uint32(info.Mode()),
		ReadOnly:   info.Mode().Perm()&0o200 == 0,
	}, nil
}

// RevealInExplorer opens Windows Explorer with the given file selected. On
// non-Windows platforms it returns an error.
func (s *FileService) RevealInExplorer(path string) error {
	if runtime.GOOS != "windows" {
		return errors.New("RevealInExplorer only supported on Windows")
	}
	cmd := exec.Command("explorer", "/select,", path)
	return cmd.Start()
}
