package filemanager

import "time"

type FileEntry struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	IsDir     bool      `json:"is_dir"`
	Size      int64     `json:"size"`
	Mode      string    `json:"mode"`
	ModTime   time.Time `json:"mod_time"`
	User      string    `json:"user,omitempty"`
	Group     string    `json:"group,omitempty"`
	IsSymlink bool      `json:"is_symlink"`
	LinkTarget string   `json:"link_target,omitempty"`
}

type ReadFileResponse struct {
	Content string `json:"content"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
}

type WriteFileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type MkdirRequest struct {
	Path string `json:"path"`
}

type RenameRequest struct {
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
}

type ChmodRequest struct {
	Path string `json:"path"`
	Mode string `json:"mode"` // e.g. "0755"
}

type DeleteRequest struct {
	Path string `json:"path"`
}

type CompressRequest struct {
	Paths       []string `json:"paths"`
	Destination string   `json:"destination"`
}
