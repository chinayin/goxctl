// Package debug 提供受 GOXCTL_DEBUG 环境变量控制的调试输出。
//
// 调试开关有两个来源：环境变量 GOXCTL_DEBUG（贯穿转发链，子进程自动继承）、
// 以及 --verbose flag（经 Enable 设置 GOXCTL_DEBUG，使转发出去的扩展也开启）。
package debug

import (
	"fmt"
	"os"
)

// envKey 是控制调试输出的环境变量名。
const envKey = "GOXCTL_DEBUG"

// enabled 表示是否开启调试输出。
var enabled = isTruthy(os.Getenv(envKey))

// isTruthy 判断环境变量值是否表示开启（非空且非 "0"/"false"）。
func isTruthy(v string) bool {
	return v != "" && v != "0" && v != "false"
}

// Enable 显式开启调试，并写回环境变量，确保转发给扩展子进程时同样生效。
func Enable() {
	enabled = true
	_ = os.Setenv(envKey, "1")
}

// Enabled 返回当前是否开启调试。
func Enabled() bool { return enabled }

// Logf 在调试开启时向 stderr 打印一行（统一 debug: 前缀）。
func Logf(format string, args ...any) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "debug: "+format+"\n", args...)
}
