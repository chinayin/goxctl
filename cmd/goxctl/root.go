package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"

	"github.com/chinayin/goxctl/internal/debug"
	"github.com/chinayin/goxctl/internal/ext"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goxctl",
	Short: "Extensible CLI for the gox ecosystem",
	Long: `goxctl dispatches commands for the gox ecosystem: extension management is
built in, any other subcommand is forwarded to a standalone goxctl-<name>
extension (gh/git style).

For example, "goxctl <name> ..." is forwarded to the goxctl-<name> extension.`,
	// 不设 SilenceUsage：参数/flag 用法错误时显示 usage；业务错误在各 RunE 开头抑制。
	SilenceErrors: true, // 错误由 Execute 统一打印，避免与转发的扩展重复
}

// exitCodeError 携带被转发扩展的退出码，由 Execute 翻译为进程退出码（不打印）。
type exitCodeError struct{ code int }

func (e *exitCodeError) Error() string { return fmt.Sprintf("exit status %d", e.code) }

// Execute 是入口：内置命令走 cobra，未知子命令转发给 extension。
func Execute() {
	err := run()
	var ec *exitCodeError
	if errors.As(err, &ec) {
		os.Exit(ec.code) // 扩展已输出，按其退出码静默退出
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	args := os.Args[1:]
	// 剥离前置全局开关：goxctl --verbose <cmd>（转发不经 cobra，需在此处理）
	for len(args) > 0 && args[0] == "--verbose" {
		debug.Enable()
		args = args[1:]
	}

	if len(args) > 0 && !isBuiltin(args[0]) {
		if handled, err := tryForward(ctx, args); handled {
			return err
		}
	}
	return rootCmd.ExecuteContext(ctx)
}

// tryForward 尝试把 args 转发给扩展。handled=false 表示扩展未安装且非已知扩展，
// 应回落 cobra 报 unknown command。
func tryForward(ctx context.Context, args []string) (bool, error) {
	m, err := ext.NewManager()
	if err != nil {
		return true, err
	}
	debug.Logf("forwarding %q with args %v", args[0], args[1:])

	err = m.Dispatch(ctx, args[0], args[1:])
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// 扩展已向 stderr 输出过错误，按其退出码静默退出，避免重复打印。
		return true, &exitCodeError{code: exitErr.ExitCode()}
	}
	if !errors.Is(err, ext.ErrNotFound) {
		return true, err // 成功（nil）或转发本身出错
	}
	if mod, ok := installHint(args[0]); ok {
		return true, fmt.Errorf("extension %q is not installed; run: goxctl extension install %s", args[0], mod)
	}
	return false, nil
}

// isBuiltin 判断子命令是否由 goxctl 核心直接处理（不转发）。
func isBuiltin(name string) bool {
	switch name {
	case "extension", "ext", "version", "upgrade", "help", "completion",
		"-h", "--help", "-v", "--version":
		return true
	default:
		return false
	}
}

func init() {
	rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose debug output (or set GOXCTL_DEBUG=1)")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		if v, _ := cmd.Flags().GetBool("verbose"); v {
			debug.Enable()
		}
	}
	rootCmd.AddCommand(extensionCmd, versionCmd)
}
