package ext

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// dirName 是 extension 默认安装目录（相对用户主目录）。
const dirName = ".goxctl/extensions"

// binPrefix 是 extension 可执行文件名前缀。
const binPrefix = "goxctl-"

// ErrNotFound 表示未找到指定 extension。
var ErrNotFound = errors.New("ext: extension not found")

// Manager 管理 extension 的发现、转发、安装、列举与删除。
type Manager struct {
	dir string // extension 安装目录
}

// NewManager 用默认目录（~/.goxctl/extensions）创建 Manager。
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("ext: home dir: %w", err)
	}
	return &Manager{dir: filepath.Join(home, dirName)}, nil
}

// Dir 返回 extension 安装目录。
func (m *Manager) Dir() string { return m.dir }

// Find 查找 extension 二进制：优先安装目录，回落 PATH；找不到返回 ErrNotFound。
func (m *Manager) Find(name string) (string, error) {
	local := filepath.Join(m.dir, binPrefix+name)
	if isExecutable(local) {
		return local, nil
	}
	if p, err := exec.LookPath(binPrefix + name); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("%w: %s", ErrNotFound, name)
}

// Dispatch 转发到 extension 并继承当前进程的标准输入输出。
func (m *Manager) Dispatch(ctx context.Context, name string, args []string) error {
	bin, err := m.Find(name)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Install 用 go install 把 extension 装到安装目录（GOBIN 指向该目录）。
//
// modulePath 为扩展仓库的 module 路径（如 github.com/chinayin/goxctl-claude），
// 按团队约定其入口在 <module>/cmd/<repo名>，据此推导 go install 目标；version 缺省 latest。
func (m *Manager) Install(ctx context.Context, modulePath, version string) error {
	modulePath = ensureHost(modulePath)
	if version == "" {
		version = "latest"
	}
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return fmt.Errorf("ext: mkdir %q: %w", m.dir, err)
	}

	installPath := modulePath + "/cmd/" + path.Base(modulePath)
	//nolint:gosec // 安装扩展即转发 go install，module path 由用户显式提供，是本工具的设计意图
	cmd := exec.CommandContext(ctx, "go", "install", installPath+"@"+version)
	cmd.Env = append(os.Environ(), "GOBIN="+m.dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ext: go install %s@%s: %w", installPath, version, err)
	}
	return nil
}

// List 返回已安装 extension 的名称（已排序）。
func (m *Manager) List() ([]string, error) {
	entries, err := os.ReadDir(m.dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ext: read dir %q: %w", m.dir, err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if n, ok := strings.CutPrefix(e.Name(), binPrefix); ok {
			names = append(names, n)
		}
	}
	slices.Sort(names)
	return names, nil
}

// Remove 删除已安装 extension；不存在返回 ErrNotFound。
func (m *Manager) Remove(name string) error {
	bin := filepath.Join(m.dir, binPrefix+name)
	if err := os.Remove(bin); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrNotFound, name)
		}
		return fmt.Errorf("ext: remove %q: %w", name, err)
	}
	return nil
}

func isExecutable(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

// ensureHost 为缺少主机名的 module 路径补默认 github.com，
// 与 claude add 的 source 简写（owner/repo）保持一致。
func ensureHost(modulePath string) string {
	if first, _, _ := strings.Cut(modulePath, "/"); !strings.Contains(first, ".") {
		return "github.com/" + modulePath
	}
	return modulePath
}
