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

	"github.com/chinayin/goxctl/internal/debug"
)

// dirName 是 extension 默认安装目录（相对用户主目录）。
const dirName = ".gox/extensions"

// binPrefix 是 extension 可执行文件名前缀。
const binPrefix = "goxctl-"

// defaultHost 是 module 简写（owner/repo）缺省补全的代码托管主机名。
const defaultHost = "github.com"

// ErrNotFound 表示未找到指定 extension。
var ErrNotFound = errors.New("ext: extension not found")

// Manager 管理 extension 的发现、转发、安装、列举与删除。
type Manager struct {
	dir     string // extension 安装目录
	apiBase string // GitHub API 基址（空=默认，主要供测试注入）
}

// NewManager 用默认目录（~/.gox/extensions）创建 Manager。
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
	debug.Logf("exec %s %v", bin, args)
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Install 安装 extension 到安装目录：优先下载与当前平台匹配的预编译二进制
// （无需 Go 环境），无匹配时回退 go install（需本机有 go 工具链）。
//
// modulePath 为扩展仓库的 module 路径（如 github.com/<owner>/goxctl-<name>）；version 缺省 latest。
func (m *Manager) Install(ctx context.Context, modulePath, version string) error {
	modulePath = ensureHost(modulePath)
	debug.Logf("install module=%q version=%q", modulePath, version)

	// 优先安装预编译二进制（无需 Go 环境）
	if ref, ok := parseModule(modulePath); ok {
		debug.Logf("resolved owner=%s repo=%s", ref.owner, ref.repo)
		err := m.installFromRelease(ctx, ref, version)
		if err == nil {
			return nil
		}
		if !errors.Is(err, errNoBinaryRelease) {
			return err // release 流程真实错误（网络/校验），不静默回退
		}
		// 无匹配预编译二进制 → 回退 go install
	}
	return m.installViaGo(ctx, modulePath, version)
}

// installViaGo 用 go install 从源码安装（开发者回退路径，需本机有 go 工具链）。
// 按团队约定扩展入口在 <module>/cmd/<repo名>，据此推导 go install 目标。
func (m *Manager) installViaGo(ctx context.Context, modulePath, version string) error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("ext: no prebuilt binary for this platform and go is not installed: %s", modulePath)
	}
	if version == "" {
		version = "latest"
	}
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return fmt.Errorf("ext: mkdir %q: %w", m.dir, err)
	}

	installPath := modulePath + "/cmd/" + path.Base(modulePath)
	debug.Logf("falling back to: go install %s@%s", installPath, version)
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

// ensureHost 为缺少主机名的 module 路径补默认 host（defaultHost），
// 让 extension install 支持 owner/repo 简写。
func ensureHost(modulePath string) string {
	if first, _, _ := strings.Cut(modulePath, "/"); !strings.Contains(first, ".") {
		return defaultHost + "/" + modulePath
	}
	return modulePath
}
