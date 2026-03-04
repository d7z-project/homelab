package common

import (
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/afero"
)

// InitVFS 根据 URL scheme 初始化虚拟文件系统并使用 BasePathFs 进行沙箱隔离
// 支持 memory:// 和 local:///path 格式
func InitVFS(vfsURL string) (afero.Fs, error) {
	u, err := url.Parse(vfsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse VFS URL: %w", err)
	}

	var baseFs afero.Fs
	var rootPath string

	switch u.Scheme {
	case "memory":
		baseFs = afero.NewMemMapFs()
		rootPath = "/" // Virtual root for memory FS
	case "local":
		rootPath = u.Path
		if rootPath == "" {
			rootPath = u.Host
		}
		if rootPath == "" {
			return nil, fmt.Errorf("local VFS requires a path (e.g., local:///tmp/homelab)")
		}
		// Ensure physical directory exists
		if err := os.MkdirAll(rootPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create VFS root directory '%s': %w", rootPath, err)
		}
		baseFs = afero.NewOsFs()
	default:
		return nil, fmt.Errorf("unsupported VFS scheme: %s (supported: memory, local)", u.Scheme)
	}

	// Always wrap in BasePathFs to prevent path escape from application logic
	return afero.NewBasePathFs(baseFs, rootPath), nil
}
