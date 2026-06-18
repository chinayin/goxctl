// Package proxy 解析代理来源并写回标准环境变量，使本进程的 HTTP 默认 transport
// 与转发出去的扩展子进程都经同一代理。
//
// 传播方式与 internal/debug 一致：把生效值写进环境变量，转发链上的扩展子进程
// （exec 继承父进程环境）自动获得，无需各自写代理代码。
package proxy

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// envKey 是 goxctl 专用的代理环境变量名（便于在 sudoers env_keep 中单独放行）。
const envKey = "GOXCTL_PROXY"

// announced 记录上次已确认输出的代理值，避免 Apply 被多次调用（run 与
// PersistentPreRun 各一次）时重复打印同一行。
var announced string

// Apply 按优先级解析代理并落地：--proxy 旗标 > GOXCTL_PROXY > 既有 HTTPS_PROXY/HTTP_PROXY。
//
// 命中前两者时写入标准 *_PROXY 环境变量（从而被默认 transport 与扩展子进程继承），
// 并向 stderr 打印一行确认，便于用户核实代理已生效；都为空则不动，沿用调用方既有的
// *_PROXY（默认 transport 自会读取，不另行提示以免每条命令都刷屏）。
// 必须在首次发起 HTTP 请求前调用——net/http 的 ProxyFromEnvironment 只在首次请求时读一次环境。
func Apply(flagVal string) {
	p, source := flagVal, "--proxy"
	if p == "" {
		p, source = os.Getenv(envKey), envKey
	}
	if p == "" {
		return
	}
	_ = os.Setenv("HTTPS_PROXY", p)
	_ = os.Setenv("HTTP_PROXY", p)
	announce(p, source)
}

// InUse 返回当前生效的代理 URL（脱敏后）；无则返回空串。供错误提示判断
// “是否已在用代理”，避免在用户已带 --proxy 时还反过来建议其添加 --proxy。
func InUse() string {
	for _, k := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy"} {
		if v := os.Getenv(k); v != "" {
			return redact(v)
		}
	}
	return ""
}

// announce 向 stderr 打印一次代理确认（值变化时才打印），脱敏 URL 中的用户名口令。
func announce(p, source string) {
	if p == announced {
		return
	}
	announced = p
	fmt.Fprintf(os.Stderr, "Using proxy %s (from %s)\n", redact(p), source)
}

// redact 隐去代理 URL 里的 userinfo（user:password），避免凭据出现在日志里；
// 无凭据或解析失败则原样返回。
func redact(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	host := u.Host
	u.User = nil // 去掉 userinfo 再手工补 "***@"，避免 url.User 把 * 转义成 %2A
	return strings.Replace(u.String(), "://"+host, "://***@"+host, 1)
}
