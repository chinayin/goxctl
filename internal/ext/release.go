package ext

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chinayin/goxctl/internal/debug"
)

const (
	// defaultAPIBase 是 GitHub API 基址。
	defaultAPIBase = "https://api.github.com"
	// releaseTimeout 是单次 release 相关 HTTP 调用的超时。
	releaseTimeout = 60 * time.Second
	// maxAssetSize 是下载/解压的体积上限，防 decompression bomb。
	maxAssetSize = 200 << 20 // 200MB
)

// errNoBinaryRelease 表示该扩展没有与当前平台匹配的预编译二进制，
// 调用方据此回退到 go install。
var errNoBinaryRelease = errors.New("ext: no prebuilt binary release")

// repoRef 是 owner/repo 引用。
type repoRef struct {
	owner string
	repo  string
}

// parseModule 从 module 路径（github.com/owner/repo）解析出 owner/repo。
func parseModule(modulePath string) (repoRef, bool) {
	parts := strings.Split(modulePath, "/")
	if len(parts) < 3 {
		return repoRef{}, false
	}
	return repoRef{owner: parts[1], repo: parts[2]}, true
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

// installFromRelease 下载与当前平台匹配的预编译二进制并安装到 m.dir。
// 无匹配资产或无该 release 时返回 errNoBinaryRelease（让上层回退 go install）。
func (m *Manager) installFromRelease(ctx context.Context, ref repoRef, version string) error {
	client := &http.Client{Timeout: releaseTimeout}
	rel, err := m.fetchRelease(ctx, client, ref, version)
	if err != nil {
		return err
	}

	bin, sum := pickAssets(rel.Assets, ref.repo)
	if bin.URL == "" {
		debug.Logf("no prebuilt asset matching %s_%s_%s", ref.repo, runtime.GOOS, runtime.GOARCH)
		return errNoBinaryRelease
	}
	debug.Logf("matched asset: %s", bin.Name)

	data, err := httpGet(ctx, client, bin.URL)
	if err != nil {
		return fmt.Errorf("ext: download %q: %w", bin.Name, err)
	}
	if sum.URL != "" {
		if err := verifyChecksum(ctx, client, sum.URL, bin.Name, data); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return fmt.Errorf("ext: mkdir %q: %w", m.dir, err)
	}
	dest := filepath.Join(m.dir, ref.repo) // 二进制名即 repo（goxctl-<name>），与 Find 一致
	debug.Logf("installing prebuilt binary -> %s", dest)
	return extractBinary(data, ref.repo, dest)
}

// fetchRelease 取指定 tag 或 latest 的 release 元数据；404 视为无 release。
func (m *Manager) fetchRelease(ctx context.Context, client *http.Client, ref repoRef, version string) (*ghRelease, error) {
	base := m.apiBase
	if base == "" {
		base = defaultAPIBase
	}
	var url string
	if version == "" || version == "latest" {
		url = fmt.Sprintf("%s/repos/%s/%s/releases/latest", base, ref.owner, ref.repo)
	} else {
		url = fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s", base, ref.owner, ref.repo, version)
	}

	debug.Logf("querying release: %s", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("ext: build release request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ext: query release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errNoBinaryRelease
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ext: query release: github status %d", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxAssetSize)).Decode(&rel); err != nil {
		return nil, fmt.Errorf("ext: decode release: %w", err)
	}
	return &rel, nil
}

// pickAssets 从资产列表中挑出当前平台的二进制包与 checksums.txt。
func pickAssets(assets []ghAsset, repo string) (bin, sum ghAsset) {
	prefix := repo + "_"
	suffix := fmt.Sprintf("_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	for _, a := range assets {
		switch {
		case strings.HasPrefix(a.Name, prefix) && strings.HasSuffix(a.Name, suffix):
			bin = a
		case a.Name == "checksums.txt":
			sum = a
		}
	}
	return bin, sum
}

// verifyChecksum 下载 checksums.txt 并校验 data 的 sha256 与其中 assetName 一行一致。
func verifyChecksum(ctx context.Context, client *http.Client, sumURL, assetName string, data []byte) error {
	sums, err := httpGet(ctx, client, sumURL)
	if err != nil {
		return fmt.Errorf("ext: download checksums: %w", err)
	}
	want := findChecksum(string(sums), assetName)
	if want == "" {
		return fmt.Errorf("ext: checksum for %q not found", assetName)
	}
	got := sha256.Sum256(data)
	if hex.EncodeToString(got[:]) != want {
		return fmt.Errorf("ext: checksum mismatch for %q", assetName)
	}
	return nil
}

// findChecksum 解析 "<sha256>  <filename>" 行，返回匹配文件名的 sha。
func findChecksum(content, name string) string {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == name {
			return fields[0]
		}
	}
	return ""
}

// extractBinary 从 tar.gz 中取出名为 want 的可执行文件写到 dest（0755）。
func extractBinary(targz []byte, want, dest string) error {
	gz, err := gzip.NewReader(bytes.NewReader(targz))
	if err != nil {
		return fmt.Errorf("ext: gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("ext: tar: %w", err)
		}
		if h.Typeflag != tar.TypeReg || filepath.Base(h.Name) != want {
			continue
		}
		f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755) // 扩展二进制必须可执行
		if err != nil {
			return fmt.Errorf("ext: create %q: %w", dest, err)
		}
		if _, err := io.CopyN(f, tr, maxAssetSize); err != nil && !errors.Is(err, io.EOF) {
			_ = f.Close()
			return fmt.Errorf("ext: write %q: %w", dest, err)
		}
		return f.Close()
	}
	return fmt.Errorf("ext: binary %q not found in archive", want)
}

// httpGet 读取 url 全部内容（受 maxAssetSize 限制）。
func httpGet(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxAssetSize))
}
