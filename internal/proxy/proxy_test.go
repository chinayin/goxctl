package proxy

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApply_FlagWins(t *testing.T) {
	// t.Setenv 注册自动还原；Apply 内部 os.Setenv 的写入也会被还原到此刻的值
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("HTTP_PROXY", "")
	t.Setenv(envKey, "http://from-env:1")

	Apply("http://from-flag:2")

	assert.Equal(t, "http://from-flag:2", os.Getenv("HTTPS_PROXY"))
	assert.Equal(t, "http://from-flag:2", os.Getenv("HTTP_PROXY"))
}

func TestApply_FallbackToEnvKey(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("HTTP_PROXY", "")
	t.Setenv(envKey, "http://from-env:1")

	Apply("")

	assert.Equal(t, "http://from-env:1", os.Getenv("HTTPS_PROXY"))
	assert.Equal(t, "http://from-env:1", os.Getenv("HTTP_PROXY"))
}

func TestApply_NoneKeepsExisting(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://existing:9")
	t.Setenv("HTTP_PROXY", "http://existing:9")
	t.Setenv(envKey, "")

	Apply("")

	// 都为空时不改动，沿用既有 *_PROXY
	assert.Equal(t, "http://existing:9", os.Getenv("HTTPS_PROXY"))
	assert.Equal(t, "http://existing:9", os.Getenv("HTTP_PROXY"))
}

func TestRedact(t *testing.T) {
	// 无凭据：原样返回
	assert.Equal(t, "http://127.0.0.1:7890", redact("http://127.0.0.1:7890"))
	// 带 user:password：隐去 userinfo
	assert.Equal(t, "http://***@host:8080", redact("http://user:secret@host:8080"))
	// 仅 token 作 user：同样隐去
	assert.Equal(t, "http://***@host:8080", redact("http://tok@host:8080"))
}

func TestInUse(t *testing.T) {
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("http_proxy", "")
	t.Setenv("https_proxy", "")
	t.Setenv("HTTPS_PROXY", "http://user:secret@host:1")

	// 返回脱敏后的生效代理
	assert.Equal(t, "http://***@host:1", InUse())
}
