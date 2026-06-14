package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstallHint(t *testing.T) {
	mod, ok := installHint("claude")
	assert.True(t, ok)
	assert.Equal(t, "github.com/chinayin/goxctl-claude", mod)

	_, ok = installHint("nonexistent")
	assert.False(t, ok)
}
