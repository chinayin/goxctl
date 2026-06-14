# goxctl 架构设计

> 状态：核心已实现。本文只描述 **goxctl 平台本身**；各 extension 的功能与技术细节（命令、数据格式、同步流程等）见其各自仓库。

## 1. 背景与目标

- `gox` 是基础库（github.com/chinayin/gox，已发布 v1.0.0），保持纯净、零 CLI 依赖。
- `goxctl` 是配套的**可扩展 CLI 平台**：本身只做命令分发与 extension 管理，具体能力由独立仓库的 extension 提供。
- 各子能力（未来 `scaffold` / `lint` 等）各自独立仓库、独立版本，被 `goxctl` 转发。

**目标**：gox 库纯净 + goxctl 核心稳定可扩展（加扩展不改核心、不重编译）+ 各 extension 独立仓库可自由演进 + 对外心智统一为 `goxctl <sub>`。

## 2. 命名与仓库边界

| 名称 | 类型 | 职责 |
|---|---|---|
| `gox` | Go 基础库 (v1.0.0) | cli/config/idgen/log/validator |
| `goxctl` | 核心 CLI 二进制 | 命令分发 + extension 管理（install/list/remove） |
| `goxctl-<name>` | extension | 独立仓库、独立版本，被 `goxctl` 转发；**功能与技术细节见各自仓库** |

核心与各 extension 均用 `gox/cli`(cobra 封装) 构建（dogfood）。

## 3. 核心设计决策

- **3.1 可执行子命令转发（gh/git 风格）**：`goxctl <name>` → 查找并 `exec` `goxctl-<name>` 二进制，原样传递剩余参数。Go 无实用运行时插件机制（`plugin` 包限制多：仅 Linux/Mac、版本须完全一致、不能卸载），独立仓库必然产出独立二进制，故转发是务实选择。
- **3.2 这是 extension 模式，不是 plugin 模式**：goxctl 的扩展是**进程隔离的独立可执行文件**（exec 转发），不是进程内动态加载的库。"extension"（向外延伸的独立单元）比"plugin"（插入进程内）在语义上更准确，与 gh / az 一致；kubectl / docker 把同样的可执行转发机制称作 plugin，仅是命名习惯不同。
- **3.3 对外心智统一**：只暴露 `goxctl <name> <action>`；`goxctl-<name>` 二进制是被转发的实现细节，不宣传、不要求用户直接敲（同 gh：有 `gh-foo` 但心智是 `gh foo`）。
- **3.4 扩展安装器可引导核心**：扩展的安装脚本可检测 `goxctl` 不在则先装核心，再装自己，保证用户环境里 `goxctl` 恒在、入口始终统一。（约定，由各扩展实现）

## 4. goxctl 核心

本体布局遵循团队 cli.md：入口 `cmd/goxctl/main.go`，命令在 `cmd/goxctl/`（root/extension/version/registry），核心逻辑在 `internal/ext/`。

- 内置子命令：`extension install/list/remove`、`version`、`help`。
- **转发**：未知子命令 → 查 `~/.goxctl/extensions/goxctl-<name>`（或 PATH）→ `exec` 并继承 stdio。
- **安装**（`extension install <module>`）：用 `go install` 装到 `~/.goxctl/extensions`（`GOBIN` 指向该目录），而非下载预编译二进制——扩展都是 Go 程序，go install 最简。按团队约定扩展入口在 `<module>/cmd/<repo名>`，据此推导 go install 目标；产出的二进制名即 `goxctl-<name>`，转发可直接发现。`module` 可简写 `owner/repo`（无 host 补 github.com，由 `ensureHost` 处理）。
- `extension list`：列已装扩展；`extension remove <name>`：删除。

### 4.1 未装扩展的提示（已知扩展注册表）

仿 [Azure CLI dynamic install](https://learn.microsoft.com/en-us/cli/azure/azure-cli-extensions-overview)，但用**写死的注册表**而非远程 index（团队扩展数量有限，简单可控）：

- `cmd/goxctl/registry.go` 维护 `knownExtensions` map：子命令名 → module 路径（具体已知扩展见该文件，本文不复制其内容）。
- 转发命中 `ErrNotFound`（扩展未安装）时：
  - **在注册表** → 提示 `扩展 "<name>" 未安装，运行：goxctl extension install <module>`；
  - **不在注册表** → 回落 cobra，报 `unknown command`。
- 新增官方扩展时在 `knownExtensions` 登记一行。将来扩展增多，可平滑升级为远程 extension index（az 模式）。
- 取舍：相比 git/docker/gh 的"未知即报错"，这给了 az 式的安装指引，降低摩擦；写死 map 避免引入远程索引的复杂度。

## 5. 安装与引导

```bash
# 已有核心：module 路径可写全，也可简写 owner/repo（无 host 默认补 github.com）
goxctl extension install <owner>/<repo>
```

- `extension install` 的 module 简写规则（`owner/repo` → 补 `github.com`）由 `ensureHost` 处理；各扩展若自身的命令也接受 source 简写，应保持同一规则以求对称。
- 对外心智一律 `goxctl <name> …`，`goxctl-<name>` 二进制不直接对用户宣传。

## 6. 开放问题（待定）

- extension 发现目录 `~/.goxctl/extensions/` vs 复用 PATH（当前两者皆查，优先安装目录）。
- 注册表后续是否需要从写死 map 升级为远程 extension index（取决于扩展数量增长）。
