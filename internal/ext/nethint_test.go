package ext

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetHint(t *testing.T) {
	// 提示文案随是否在用代理而变，逐项清空以保证断言不受运行环境的 *_PROXY 影响。
	clearProxyEnv := func(t *testing.T) {
		for _, k := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy"} {
			t.Setenv(k, "")
		}
	}

	t.Run("超时给友好提示", func(t *testing.T) {
		clearProxyEnv(t)
		err := netHint("download foo.tar.gz", context.DeadlineExceeded)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
		assert.Contains(t, err.Error(), "--proxy")
	})

	t.Run("连接类网络错误给提示", func(t *testing.T) {
		clearProxyEnv(t)
		err := netHint("download foo", &net.OpError{Op: "dial", Err: errors.New("connection refused")})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GOXCTL_PROXY")
	})

	t.Run("已在用代理时改提示代理可能不可达", func(t *testing.T) {
		clearProxyEnv(t)
		t.Setenv("HTTPS_PROXY", "http://127.0.0.1:7890")
		err := netHint("download foo.tar.gz", context.DeadlineExceeded)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "proxy http://127.0.0.1:7890 is in use")
		assert.NotContains(t, err.Error(), "add --proxy")
	})

	t.Run("非网络错误返回 nil", func(t *testing.T) {
		assert.NoError(t, netHint("download foo", errors.New("github status 404")))
	})
}
