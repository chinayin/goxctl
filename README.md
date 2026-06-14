# goxctl

gox 生态的可扩展命令行工具：本身只做**命令分发**与 **extension 管理**，具体能力由独立仓库的扩展提供（gh / git 风格）。

## 工作方式

```bash
goxctl version                                          # 内置：版本
goxctl extension install github.com/chinayin/goxctl-claude   # 安装扩展（go install 到 ~/.goxctl/extensions）
goxctl extension list                                   # 列出已装扩展
goxctl claude update                                    # 转发给 goxctl-claude 扩展
goxctl extension remove claude                          # 删除扩展
```

- 未知子命令 `goxctl <name> ...` 会被转发给名为 `goxctl-<name>` 的可执行文件（先查 `~/.goxctl/extensions`，再查 PATH）。
- 扩展是独立仓库、独立版本的 Go 程序，用 `go install` 安装。
- 对用户心智统一为 `goxctl <name>`，无需直接调用 `goxctl-<name>`。

## 扩展

| 扩展 | 功能 |
|---|---|
| [`goxctl-claude`](https://github.com/chinayin/goxctl-claude) | 团队 AI 协作配置（steering / CLAUDE.md）版本化同步 |

详见 [架构设计](docs/GOXCTL_ARCHITECTURE.md)。
