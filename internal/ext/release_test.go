package ext

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeTarGz 构造含单个名为 name 的可执行文件的 tar.gz。
func makeTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg,
	}))
	_, err := tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

// newReleaseServer 起一个 mock GitHub：latest 返回带平台资产的 release，
// 资产与 checksums.txt 各走独立路径。
func newReleaseServer(t *testing.T, repo string, targz []byte, withChecksum bool) *httptest.Server {
	t.Helper()
	assetName := fmt.Sprintf("%s_1.0.0_%s_%s.tar.gz", repo, runtime.GOOS, runtime.GOARCH)
	sum := sha256.Sum256(targz)
	checksums := fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), assetName)

	mux := http.NewServeMux()
	var base string
	mux.HandleFunc("/asset", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(targz) })
	mux.HandleFunc("/sums", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(checksums)) })
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		assets := fmt.Sprintf(`{"name":%q,"browser_download_url":%q}`, assetName, base+"/asset")
		if withChecksum {
			assets += fmt.Sprintf(`,{"name":"checksums.txt","browser_download_url":%q}`, base+"/sums")
		}
		_, _ = fmt.Fprintf(w, `{"tag_name":"v1.0.0","assets":[%s]}`, assets)
	})
	srv := httptest.NewServer(mux)
	base = srv.URL
	t.Cleanup(srv.Close)
	return srv
}

func TestParseModule(t *testing.T) {
	ref, ok := parseModule("github.com/chinayin/goxctl-claude")
	require.True(t, ok)
	assert.Equal(t, "chinayin", ref.owner)
	assert.Equal(t, "goxctl-claude", ref.repo)

	_, ok = parseModule("not-a-module")
	assert.False(t, ok)
}

func TestInstallFromRelease_Success(t *testing.T) {
	const repo = "goxctl-claude"
	want := []byte("#!/bin/sh\necho hi\n")
	srv := newReleaseServer(t, repo, makeTarGz(t, repo, want), true)

	m := &Manager{dir: t.TempDir(), apiBase: srv.URL}
	_, err := m.installFromRelease(context.Background(), repoRef{owner: "chinayin", repo: repo}, "")
	require.NoError(t, err)

	dest := filepath.Join(m.dir, repo)
	got, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, want, got)

	info, err := os.Stat(dest)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&0o111, "二进制应可执行")
}

func TestInstallFromRelease_NoMatchingAsset(t *testing.T) {
	// 资产名平台不匹配 → errNoBinaryRelease
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v1.0.0","assets":[{"name":"goxctl-claude_1.0.0_plan9_mips.tar.gz","browser_download_url":"http://x/a"}]}`))
	}))
	t.Cleanup(srv.Close)

	m := &Manager{dir: t.TempDir(), apiBase: srv.URL}
	_, err := m.installFromRelease(context.Background(), repoRef{owner: "chinayin", repo: "goxctl-claude"}, "")
	assert.ErrorIs(t, err, errNoBinaryRelease)
}

func TestInstallFromRelease_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	m := &Manager{dir: t.TempDir(), apiBase: srv.URL}
	_, err := m.installFromRelease(context.Background(), repoRef{owner: "chinayin", repo: "goxctl-claude"}, "v9.9.9")
	assert.ErrorIs(t, err, errNoBinaryRelease)
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("payload")
	sum := sha256.Sum256(data)
	good := hex.EncodeToString(sum[:])

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/good" {
			_, _ = fmt.Fprintf(w, "%s  asset.tar.gz\n", good)
			return
		}
		_, _ = fmt.Fprint(w, "deadbeef  asset.tar.gz\n")
	}))
	t.Cleanup(srv.Close)
	c := &http.Client{}

	// 一致
	require.NoError(t, verifyChecksum(context.Background(), c, srv.URL+"/good", "asset.tar.gz", data))
	// sha 不一致
	require.ErrorContains(t, verifyChecksum(context.Background(), c, srv.URL+"/bad", "asset.tar.gz", data), "mismatch")
	// 文件名不在 checksums 中
	require.ErrorContains(t, verifyChecksum(context.Background(), c, srv.URL+"/good", "other.tar.gz", data), "not found")
}

func TestLatestVersion(t *testing.T) {
	srv := newReleaseServer(t, "goxctl", makeTarGz(t, "goxctl", []byte("x")), false)
	apiBaseOverride = srv.URL
	t.Cleanup(func() { apiBaseOverride = "" })

	v, err := LatestVersion(context.Background(), "chinayin/goxctl")
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", v)
}

func TestSelfUpdate(t *testing.T) {
	want := []byte("#!/bin/sh\necho new\n")
	srv := newReleaseServer(t, "goxctl", makeTarGz(t, "goxctl", want), true)
	apiBaseOverride = srv.URL
	t.Cleanup(func() { apiBaseOverride = "" })

	dest := filepath.Join(t.TempDir(), "goxctl")
	require.NoError(t, os.WriteFile(dest, []byte("old binary"), 0o755))

	tag, err := SelfUpdate(context.Background(), "chinayin/goxctl", dest)
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", tag)

	got, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, want, got) // 旧二进制被原子替换
}

func TestFindChecksum(t *testing.T) {
	content := "abc123  goxctl_1.0.0_darwin_arm64.tar.gz\ndef456  checksums.txt\n"
	assert.Equal(t, "abc123", findChecksum(content, "goxctl_1.0.0_darwin_arm64.tar.gz"))
	assert.Empty(t, findChecksum(content, "missing.tar.gz"))
}
