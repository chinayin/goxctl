package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/chinayin/goxctl/internal/ext"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goxctl",
	Short: "gox 生态的可扩展命令行工具",
	Long: `goxctl 是 gox 生态的命令分发器：内置 extension 管理，
其余子命令转发给独立的 goxctl-<name> 扩展（gh/git 风格）。

例如 goxctl claude update 会转发给 goxctl-claude 扩展。`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute 是入口：内置命令走 cobra，未知子命令转发给 extension。
func Execute() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	args := os.Args[1:]
	if len(args) > 0 && !isBuiltin(args[0]) {
		m, err := ext.NewManager()
		if err != nil {
			return err
		}
		// 未知子命令转发给扩展
		err = m.Dispatch(ctx, args[0], args[1:])
		if !errors.Is(err, ext.ErrNotFound) {
			return err // 成功（nil）或扩展执行出错
		}
		// 扩展未安装：已知官方扩展给安装指引，否则回落 cobra 报 unknown command
		if mod, ok := installHint(args[0]); ok {
			return fmt.Errorf("扩展 %q 未安装，运行：goxctl extension install %s", args[0], mod)
		}
	}

	return rootCmd.ExecuteContext(ctx)
}

// isBuiltin 判断子命令是否由 goxctl 核心直接处理（不转发）。
func isBuiltin(name string) bool {
	switch name {
	case "extension", "ext", "version", "help", "completion",
		"-h", "--help", "-v", "--version":
		return true
	default:
		return false
	}
}

func init() {
	rootCmd.AddCommand(extensionCmd, versionCmd)
}
