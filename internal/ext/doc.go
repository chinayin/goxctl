// Package ext 实现 goxctl 的 extension 发现、转发与管理。
//
// extension 是命名为 goxctl-<name> 的独立可执行文件，安装在 ~/.goxctl/extensions
// 或 PATH 上；goxctl <name> ... 被转发给对应 extension（gh/git 风格）。
package ext
