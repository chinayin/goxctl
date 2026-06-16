# goxctl

gox 生态的可扩展命令行工具：本身只做**命令分发**与 **extension 管理**，具体能力由独立仓库的扩展提供（gh / git 风格）。

## 安装

预编译二进制，**无需 Go 环境**（macOS / Linux，amd64 / arm64）。安装到 `/usr/local/bin`（默认在 PATH；不可写则回退 sudo）：

```bash
curl -sSfL https://github.com/chinayin/goxctl/releases/latest/download/install.sh | sh
```

开发者也可 `go install github.com/chinayin/goxctl/cmd/goxctl@latest`。

## 工作方式

```bash
goxctl version                            # 内置：版本
goxctl upgrade                            # 自更新核心到最新 release（--check 只查不装）
goxctl extension install <owner>/<repo>   # 安装扩展（优先预编译二进制，回退 go install）
goxctl extension list                     # 列出已装扩展
goxctl extension upgrade <name>|--all     # 更新扩展到最新 release
goxctl extension remove <name>            # 删除扩展
goxctl <name> ...                         # 转发给 goxctl-<name> 扩展
goxctl -v / --verbose <name> ...          # 调试输出（等价 GOXCTL_DEBUG=1）
```

- `extension install` **优先下载与当前平台匹配的预编译二进制**（无需 Go），无匹配时回退 `go install`（需本机有 go）。module 可简写 `owner/repo`（无 host 默认补 `github.com`）。
- 未知子命令 `goxctl <name> ...` 会被转发给名为 `goxctl-<name>` 的可执行文件（先查 `~/.gox/extensions`，再查 PATH）。
- 扩展是独立仓库、独立版本的 Go 程序；各扩展的功能与用法见其各自仓库。
- 对用户心智统一为 `goxctl <name>`，无需直接调用 `goxctl-<name>`。
- 调试：`GOXCTL_DEBUG=1` 或 `goxctl --verbose`/`-v`，环境变量贯穿转发链（扩展也支持，前置/后置 `--verbose` 都生效）。`-v` 专给 verbose；版本用 `goxctl version` 或 `--version`。

详见 [架构设计](docs/GOXCTL_ARCHITECTURE.md)。
