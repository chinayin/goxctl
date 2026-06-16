package ext

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManifest_LoadMissing_ReturnsEmpty 验证文件不存在时返回空清单（无错误）。
func TestManifest_LoadMissing_ReturnsEmpty(t *testing.T) {
	m := &Manager{dir: t.TempDir()}

	man, err := m.loadManifest()

	require.NoError(t, err)
	assert.NotNil(t, man.Extensions)
	assert.Empty(t, man.Extensions)
}

// TestManifest_LoadCorrupt_ReturnsError 验证损坏的 YAML 文件返回含 "parse manifest" 的错误。
func TestManifest_LoadCorrupt_ReturnsError(t *testing.T) {
	m := &Manager{dir: t.TempDir()}
	require.NoError(t, os.WriteFile(m.manifestPath(), []byte("{{{ not yaml"), 0o600))

	_, err := m.loadManifest()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse manifest")
}

// TestManifest_LoadNullExtensions_Normalized 验证 "extensions:\n" 被规范化为空 map（非 nil）。
func TestManifest_LoadNullExtensions_Normalized(t *testing.T) {
	m := &Manager{dir: t.TempDir()}
	require.NoError(t, os.WriteFile(m.manifestPath(), []byte("extensions:\n"), 0o600))

	man, err := m.loadManifest()

	require.NoError(t, err)
	assert.NotNil(t, man.Extensions, "Extensions 应被规范化为空 map 而非 nil")
}

// TestManifest_SaveLoad_RoundTrip 验证 record 写入磁盘后，新 Manager 可读回相同的条目。
func TestManifest_SaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	m := &Manager{dir: dir}
	require.NoError(t, m.record("demo", "github.com/x/goxctl-demo", "v1.2.3"))

	// 用相同的 dir 但新的 Manager 实例，确保走磁盘读取
	m2 := &Manager{dir: dir}
	man, err := m2.loadManifest()

	require.NoError(t, err)
	entry, ok := man.Extensions["demo"]
	require.True(t, ok, "清单中应存在 demo 条目")
	assert.Equal(t, "github.com/x/goxctl-demo", entry.Module)
	assert.Equal(t, "v1.2.3", entry.Version)
}

// TestManifest_Forget_MissingFile_NoError 验证清单文件不存在时 forget 无操作且不报错，也不创建文件。
func TestManifest_Forget_MissingFile_NoError(t *testing.T) {
	m := &Manager{dir: t.TempDir()}

	err := m.forget("nope")

	require.NoError(t, err)
	_, statErr := os.Stat(m.manifestPath())
	assert.True(t, os.IsNotExist(statErr), "forget 不应在缺失清单时创建文件")
}
