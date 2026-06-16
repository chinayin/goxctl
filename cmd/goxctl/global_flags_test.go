package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripGlobalFlags(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantRest  []string
		wantProxy string
	}{
		{"无全局开关", []string{"claude", "add"}, []string{"claude", "add"}, ""},
		{"proxy 分离写法", []string{"--proxy", "http://p:1", "upgrade"}, []string{"upgrade"}, "http://p:1"},
		{"proxy 等号写法", []string{"--proxy=http://p:2", "claude"}, []string{"claude"}, "http://p:2"},
		{"verbose 与 proxy 混合", []string{"-v", "--proxy", "http://p:3", "claude", "x"}, []string{"claude", "x"}, "http://p:3"},
		{"开关后即停止剥离", []string{"claude", "--proxy", "http://p:4"}, []string{"claude", "--proxy", "http://p:4"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rest, proxyURL := stripGlobalFlags(tt.args)
			assert.Equal(t, tt.wantRest, rest)
			assert.Equal(t, tt.wantProxy, proxyURL)
		})
	}
}
