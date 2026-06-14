# goxctl

gox 生态的可扩展命令行工具：本身只做**命令分发**与 **extension 管理**，具体能力由独立仓库的扩展提供（gh / git 风格）。

## 安装

预编译二进制，**无需 Go 环境**（macOS / Linux，amd64 / arm64）：

```bash
# 从 Releases 下载对应平台二进制，解压到 ~/.gox/bin（提示加入 PATH）
curl -sSfL https://raw.githubusercontent.com/chinayin/goxctl/main/install.sh | sh
```

开发者也可 `go install github.com/chinayin/goxctl/cmd/goxctl@latest`。

## 工作方式

```bash
goxctl version                            # 内置：版本
goxctl extension install <owner>/<repo>   # 安装扩展（优先预编译二进制，回退 go install）
goxctl extension list                     # 列出已装扩展
goxctl <name> ...                         # 转发给 goxctl-<name> 扩展
goxctl extension remove <name>            # 删除扩展
```

- `extension install` **优先下载与当前平台匹配的预编译二进制**（无需 Go），无匹配时回退 `go install`（需本机有 go）。module 可简写 `owner/repo`（无 host 默认补 `github.com`）。
- 未知子命令 `goxctl <name> ...` 会被转发给名为 `goxctl-<name>` 的可执行文件（先查 `~/.gox/extensions`，再查 PATH）。
- 扩展是独立仓库、独立版本的 Go 程序；各扩展的功能与用法见其各自仓库。
- 对用户心智统一为 `goxctl <name>`，无需直接调用 `goxctl-<name>`。

详见 [架构设计](docs/GOXCTL_ARCHITECTURE.md)。
