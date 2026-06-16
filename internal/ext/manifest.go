package ext

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// manifestName 是扩展清单文件名（位于 ~/.gox 下，与 extensions 目录平级）。
const manifestName = "extensions.yaml"

// extManifest 记录所有已装扩展的元数据（类似 package.json），替代每扩展一个 .ver 文件。
type extManifest struct {
	Extensions map[string]extEntry `yaml:"extensions"`
}

type extEntry struct {
	Module  string `yaml:"module"`
	Version string `yaml:"version"`
}

// manifestPath 返回清单路径（~/.gox/extensions.yaml）。
func (m *Manager) manifestPath() string {
	return filepath.Join(filepath.Dir(m.dir), manifestName)
}

// loadManifest 读取清单；不存在返回空清单。
func (m *Manager) loadManifest() (*extManifest, error) {
	man := &extManifest{Extensions: map[string]extEntry{}}
	b, err := os.ReadFile(m.manifestPath())
	if errors.Is(err, os.ErrNotExist) {
		return man, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ext: read manifest: %w", err)
	}
	if err := yaml.Unmarshal(b, man); err != nil {
		return nil, fmt.Errorf("ext: parse manifest: %w", err)
	}
	if man.Extensions == nil {
		man.Extensions = map[string]extEntry{}
	}
	return man, nil
}

// saveManifest 写回清单（2 空格缩进，对齐主流 YAML 风格）。
func (m *Manager) saveManifest(man *extManifest) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(man); err != nil {
		return fmt.Errorf("ext: marshal manifest: %w", err)
	}
	_ = enc.Close()
	b := buf.Bytes()
	if err := os.MkdirAll(filepath.Dir(m.manifestPath()), 0o755); err != nil {
		return fmt.Errorf("ext: mkdir for manifest: %w", err)
	}
	if err := os.WriteFile(m.manifestPath(), b, 0o600); err != nil {
		return fmt.Errorf("ext: write manifest: %w", err)
	}
	return nil
}

// record 记录/更新某扩展的 module 与 version。
func (m *Manager) record(name, module, version string) error {
	man, err := m.loadManifest()
	if err != nil {
		return err
	}
	man.Extensions[name] = extEntry{Module: module, Version: version}
	return m.saveManifest(man)
}

// forget 从清单移除某扩展（不存在则无操作）。
func (m *Manager) forget(name string) error {
	man, err := m.loadManifest()
	if err != nil {
		return err
	}
	if _, ok := man.Extensions[name]; !ok {
		return nil
	}
	delete(man.Extensions, name)
	return m.saveManifest(man)
}

// ExtVersion 返回清单记录的扩展版本；未知返回空串。
func (m *Manager) ExtVersion(name string) string {
	man, err := m.loadManifest()
	if err != nil {
		return ""
	}
	return man.Extensions[name].Version
}

// ExtModule 返回清单记录的扩展 module；未知返回空串。
func (m *Manager) ExtModule(name string) string {
	man, err := m.loadManifest()
	if err != nil {
		return ""
	}
	return man.Extensions[name].Module
}
