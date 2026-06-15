package ext

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFakeExt 在 dir 下写一个名为 goxctl-<name> 的可执行脚本。
func writeFakeExt(t *testing.T, dir, name, script string) {
	t.Helper()
	path := filepath.Join(dir, binPrefix+name)
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
}

func TestManager_Find(t *testing.T) {
	m := &Manager{dir: t.TempDir()}
	writeFakeExt(t, m.dir, "demo", "#!/bin/sh\n")

	p, err := m.Find("demo")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(m.dir, "goxctl-demo"), p)

	_, err = m.Find("missing")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestManager_Dispatch(t *testing.T) {
	m := &Manager{dir: t.TempDir()}
	marker := filepath.Join(t.TempDir(), "ran")
	// 脚本把首个参数当路径 touch，用于验证转发与参数传递
	writeFakeExt(t, m.dir, "demo", "#!/bin/sh\ntouch \"$1\"\n")

	err := m.Dispatch(context.Background(), "demo", []string{marker})
	require.NoError(t, err)
	assert.FileExists(t, marker)
}

func TestManager_Dispatch_NotFound(t *testing.T) {
	m := &Manager{dir: t.TempDir()}
	require.ErrorIs(t, m.Dispatch(context.Background(), "missing", nil), ErrNotFound)
}

func TestManager_List(t *testing.T) {
	m := &Manager{dir: t.TempDir()}
	writeFakeExt(t, m.dir, "claude", "#!/bin/sh\n")
	writeFakeExt(t, m.dir, "scaffold", "#!/bin/sh\n")
	require.NoError(t, os.WriteFile(filepath.Join(m.dir, "not-an-ext"), []byte("x"), 0o644))

	names, err := m.List()
	require.NoError(t, err)
	assert.Equal(t, []string{"claude", "scaffold"}, names)
}

func TestManager_List_MissingDir(t *testing.T) {
	m := &Manager{dir: filepath.Join(t.TempDir(), "nope")}
	names, err := m.List()
	require.NoError(t, err)
	assert.Empty(t, names)
}

func TestManager_Remove(t *testing.T) {
	m := &Manager{dir: t.TempDir()}
	writeFakeExt(t, m.dir, "demo", "#!/bin/sh\n")

	require.NoError(t, m.Remove("demo"))
	_, err := m.Find("demo")
	require.ErrorIs(t, err, ErrNotFound)

	require.ErrorIs(t, m.Remove("demo"), ErrNotFound)
}

func TestEnsureHost(t *testing.T) {
	// 无 host：补默认 github.com
	assert.Equal(t, "github.com/chinayin/goxctl-claude", ensureHost("chinayin/goxctl-claude"))
	// 已有 host：原样保留
	assert.Equal(t, "github.com/chinayin/goxctl-claude", ensureHost("github.com/chinayin/goxctl-claude"))
	assert.Equal(t, "gitlab.com/x/y", ensureHost("gitlab.com/x/y"))
}

func TestManager_ExtVersion(t *testing.T) {
	m := &Manager{dir: t.TempDir()}
	writeFakeExt(t, m.dir, "demo", "#!/bin/sh\n")

	// 未记录 → 空
	assert.Empty(t, m.ExtVersion("demo"))

	// record 后可读 version 与 module
	require.NoError(t, m.record("demo", "github.com/x/goxctl-demo", "v1.2.3"))
	assert.Equal(t, "v1.2.3", m.ExtVersion("demo"))
	assert.Equal(t, "github.com/x/goxctl-demo", m.ExtModule("demo"))

	// List 只列二进制（清单文件在 extensions 目录之外）
	names, err := m.List()
	require.NoError(t, err)
	assert.Equal(t, []string{"demo"}, names)

	// Remove 顺带从清单移除
	require.NoError(t, m.Remove("demo"))
	assert.Empty(t, m.ExtVersion("demo"))
}
