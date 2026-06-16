package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/chinayin/goxctl/internal/debug"
	"github.com/chinayin/goxctl/internal/ext"
	"github.com/chinayin/goxctl/internal/proxy"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goxctl",
	Short: "Extensible CLI for the gox ecosystem",
	Long: `goxctl dispatches commands for the gox ecosystem: extension management is
built in, any other subcommand is forwarded to a standalone goxctl-<name>
extension (gh/git style).

For example, "goxctl <name> ..." is forwarded to the goxctl-<name> extension.`,
	Example: `  # Install an extension (prebuilt binary, or go install fallback)
  goxctl extension install chinayin/goxctl-claude

  # List installed extensions
  goxctl extension list

  # Run an extension — forwarded to goxctl-<name>
  goxctl claude add

  # Update goxctl itself to the latest release
  goxctl upgrade`,
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

	// 剥离前置全局开关：goxctl --verbose/-v --proxy <url> <cmd>
	// （转发不经 cobra，需在此手动处理；剥离后还原成扩展真正的参数）
	args, proxyFlag := stripGlobalFlags(os.Args[1:])
	proxy.Apply(proxyFlag)

	if len(args) > 0 && !isBuiltin(args[0]) {
		if handled, err := tryForward(ctx, args); handled {
			return err
		}
	}
	return rootCmd.ExecuteContext(ctx)
}

// stripGlobalFlags 剥离转发前的前置全局开关，返回扩展真正的参数与 --proxy 值。
// -v/--verbose 直接生效（debug.Enable）；--proxy/--proxy=<url> 取值后交由 proxy.Apply 落地。
func stripGlobalFlags(args []string) (rest []string, proxyURL string) {
	for len(args) > 0 {
		switch {
		case args[0] == "--verbose" || args[0] == "-v":
			debug.Enable()
			args = args[1:]
		case args[0] == "--proxy" && len(args) >= 2:
			proxyURL, args = args[1], args[2:]
		case strings.HasPrefix(args[0], "--proxy="):
			proxyURL = strings.TrimPrefix(args[0], "--proxy=")
			args = args[1:]
		default:
			return args, proxyURL
		}
	}
	return args, proxyURL
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
		"-h", "--help", "-V", "--version":
		return true
	default:
		return false
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose debug output (or set GOXCTL_DEBUG=1)")
	rootCmd.PersistentFlags().String("proxy", "", "HTTP/HTTPS proxy URL for downloads (or set GOXCTL_PROXY)")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		if v, _ := cmd.Flags().GetBool("verbose"); v {
			debug.Enable()
		}
		p, _ := cmd.Flags().GetString("proxy")
		proxy.Apply(p)
	}

	// version 走 -V（大写）短旗：-v 已给 verbose，凑齐 -v/-V/-h 三件套。
	// 预注册同名 bool flag，cobra 检测到已存在便不再套用默认（-v 或无短旗）。
	rootCmd.Version = version
	rootCmd.Flags().BoolP("version", "V", false, "print version and exit")

	// 去噪：移除自动生成的 completion 子命令、隐藏 help 子命令。
	// cobra 默认模板用 (eq .Name "help") 硬编码强制列出 help，.Hidden 对它无效；
	// 故自定义一个真名非 "help" 的隐藏命令、用别名 "help" 接管——既不在列表露出，
	// goxctl help / goxctl help <cmd> / -h / --help 仍照常工作。
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetHelpCommand(newHiddenHelpCmd())
	rootCmd.AddCommand(extensionCmd, versionCmd)
}

// newHiddenHelpCmd 复刻 cobra 默认 help 命令的行为，但真名非 "help" 且 Hidden，
// 从而不出现在命令列表里（见 init 注释）。
func newHiddenHelpCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "help-topic [command]",
		Aliases: []string{"help"},
		Short:   "Help about any command",
		Hidden:  true,
		Run: func(c *cobra.Command, args []string) {
			target, _, err := c.Root().Find(args)
			if target == nil || err != nil {
				_ = c.Root().Usage()
				return
			}
			target.InitDefaultHelpFlag()
			target.InitDefaultVersionFlag()
			_ = target.Help()
		},
	}
}
