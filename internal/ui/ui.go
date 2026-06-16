// Package ui 提供统一的命令行输出：✓ 成功提示（TTY 上色）与对齐表格。
package ui

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

// color 表示是否对输出上色：stdout 为 TTY 且未设 NO_COLOR。
// 管道/重定向（非字符设备）自动无色。
var color = func() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}()

const (
	green = "\033[32m"
	gray  = "\033[2m"
	reset = "\033[0m"
)

// Successf 打印 "✓ <msg>"，TTY 下 ✓ 为绿色。
func Successf(w io.Writer, format string, args ...any) {
	mark := "✓"
	if color {
		mark = green + "✓" + reset
	}
	fmt.Fprintf(w, "%s %s\n", mark, fmt.Sprintf(format, args...))
}

// Stepf 打印一行“进行中”的步骤提示（无 ✓ 标记，自带换行），
// 例如 "Installing claude v0.7.0 (darwin-arm64)..."；操作完成后再用 Successf。
func Stepf(w io.Writer, format string, args ...any) {
	fmt.Fprintln(w, fmt.Sprintf(format, args...))
}

// Dim 把次要文本（如表格末列的 module 路径）在 TTY 下显示为暗色，
// 管道/NO_COLOR 下原样返回。仅可用于表格最后一列或非表格行——
// ANSI 转义会被 tabwriter 计入宽度，用在中间列会破坏对齐。
func Dim(s string) string {
	if !color {
		return s
	}
	return gray + s + reset
}

// Table 返回对齐表格 writer（列以 \t 分隔）。
// 用法：t := ui.Table(w); fmt.Fprintln(t, "NAME\tVERSION"); ...; t.Flush()。
func Table(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
}
