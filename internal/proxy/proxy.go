// Package proxy 解析代理来源并写回标准环境变量，使本进程的 HTTP 默认 transport
// 与转发出去的扩展子进程都经同一代理。
//
// 传播方式与 internal/debug 一致：把生效值写进环境变量，转发链上的扩展子进程
// （exec 继承父进程环境）自动获得，无需各自写代理代码。
package proxy

import "os"

// envKey 是 goxctl 专用的代理环境变量名（便于在 sudoers env_keep 中单独放行）。
const envKey = "GOXCTL_PROXY"

// Apply 按优先级解析代理并落地：--proxy 旗标 > GOXCTL_PROXY > 既有 HTTPS_PROXY/HTTP_PROXY。
//
// 命中前两者时写入标准 *_PROXY 环境变量，从而被默认 transport 与扩展子进程继承；
// 都为空则不动，沿用调用方既有的 *_PROXY（默认 transport 自会读取）。
// 必须在首次发起 HTTP 请求前调用——net/http 的 ProxyFromEnvironment 只在首次请求时读一次环境。
func Apply(flagVal string) {
	p := flagVal
	if p == "" {
		p = os.Getenv(envKey)
	}
	if p == "" {
		return
	}
	_ = os.Setenv("HTTPS_PROXY", p)
	_ = os.Setenv("HTTP_PROXY", p)
}
