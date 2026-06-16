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
	t.Run("超时给友好提示", func(t *testing.T) {
		err := netHint("download foo.tar.gz", context.DeadlineExceeded)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
		assert.Contains(t, err.Error(), "--proxy")
	})

	t.Run("连接类网络错误给提示", func(t *testing.T) {
		err := netHint("download foo", &net.OpError{Op: "dial", Err: errors.New("connection refused")})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GOXCTL_PROXY")
	})

	t.Run("非网络错误返回 nil", func(t *testing.T) {
		assert.NoError(t, netHint("download foo", errors.New("github status 404")))
	})
}
