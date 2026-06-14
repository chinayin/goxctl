# goxctl 架构设计

> 状态：已实现（goxctl 核心 + goxctl-claude 扩展），随 goxctl repo 维护。

## 1. 背景与目标

- `gox` 是基础库（github.com/chinayin/gox，已发布 v1.0.0），保持纯净、零 CLI 依赖。
- `goxctl` 是配套的**可扩展 CLI 平台**：本身只做命令分发与 extension 管理，具体能力由独立仓库的 extension 提供。
- 子能力（`claude` AI 协作配置分发，未来 `scaffold`/`lint` 等）各自独立仓库、独立版本，被 `goxctl` 转发。

**目标**：gox 库纯净 + goxctl 核心稳定可扩展（加子模块不改核心、不重编译）+ extension 独立仓库可自由演进 + 对外心智统一为 `goxctl <sub>`。

## 2. 命名与仓库边界

| 名称 | 类型 | 职责 |
|---|---|---|
| `gox` | Go 基础库 (v1.0.0) | cli/config/idgen/log/validator |
| `goxctl` | 核心 CLI 二进制 | 命令分发 + extension 管理（install/list/update/remove） |
| `goxctl-claude` | extension（工具+数据合并一仓库） | 分发团队 AI 协作配置（steering / CLAUDE.md 模板 / 未来 hooks） |
| `goxctl-scaffold` / `goxctl-lint` | 未来 extension | 脚手架生成 / lint 版本管理 |

核心与各 extension 均用 `gox/cli`(cobra 封装) 构建（dogfood）。

## 3. 核心设计决策

- **3.1 可执行子命令转发（gh/git 风格）**：`goxctl <name>` → 查找并 `exec` `goxctl-<name>` 二进制，原样传递剩余参数。Go 无实用运行时插件机制，独立仓库必然产出独立二进制，故转发是务实选择。
- **3.2 对外心智统一**：只暴露 `goxctl claude <action>`；`goxctl-claude` 二进制是被转发的实现细节，不宣传、不要求用户直接敲（同 gh：有 `gh-foo` 但心智是 `gh foo`）。
- **3.3 装扩展自动带核心**：`goxctl-claude` 的 `install.sh` 检测 `goxctl` 不在则先装核心，再装自己；保证用户环境里 `goxctl` 恒在，入口始终统一。
- **3.4 工具 + 数据合并一仓库**：`goxctl-claude` 内 `cmd/`(工具) + `steering/`(规范数据)，tag 统一。**工具二进制版本 ≠ 拉取的规范版本**——工具去拉「项目 pin 的那个 tag 的 `steering/`」，旧工具拉新规范依然成立。数据仍是普通 markdown，Kiro/其它工具可 raw 拉取，合并不损可复用性。
- **3.5 lock 只锚 commit**：lock 记 `resolved` commit sha（唯一完整性锚点，git 内容寻址）+ 自动生成的受管文件列表 + 整体 digest。**不逐文件 hash**（冗余且随文件增减难维护）。
- **3.6 部分托管**：只管受管文件，项目自有的 steering 文件共存、不被触碰；`check` 只校验受管范围。
- **3.7 always-on 注入链不变**：编码规范的 always-on 注入恒为 `CLAUDE.md @import .kiro/steering/*`，那份副本由 `goxctl claude` 版本化同步而来。CLI extension / Claude plugin 都不改变这条链，只让"保持最新"自动化。

## 4. goxctl 核心

本体布局遵循团队 cli.md：入口 `cmd/goxctl/main.go`，命令在 `cmd/goxctl/`（root/extension/version/registry），核心逻辑在 `internal/ext/`。

- 内置子命令：`extension install/list/remove`、`version`、`help`。
- **转发**：未知子命令 → 查 `~/.goxctl/extensions/goxctl-<name>`（或 PATH）→ `exec` 并继承 stdio。
- **安装**（`extension install <module>`）：用 `go install` 装到 `~/.goxctl/extensions`（`GOBIN` 指向该目录），而非下载预编译二进制——扩展都是 Go 程序，go install 最简。按团队约定扩展入口在 `<module>/cmd/<repo名>`，据此推导 go install 目标；产出的二进制名即 `goxctl-<name>`，转发可直接发现。`module` 可简写 `owner/repo`（无 host 补 github.com，与 `add` 的 source 一致）。
- `extension list`：列已装扩展；`extension remove <name>`：删除。

### 4.1 未装扩展的提示（已知扩展注册表）

仿 [Azure CLI dynamic install](https://learn.microsoft.com/en-us/cli/azure/azure-cli-extensions-overview)，但用**写死的注册表**而非远程 index（团队扩展数量有限，简单可控）：

- `cmd/goxctl/registry.go` 维护 `knownExtensions` map：子命令名 → module 路径，例如 `claude → github.com/chinayin/goxctl-claude`。
- 转发命中 `ErrNotFound`（扩展未安装）时：
  - **在注册表** → 提示 `扩展 "claude" 未安装，运行：goxctl extension install <module>`；
  - **不在注册表** → 回落 cobra，报 `unknown command`。
- 新增官方扩展时在 `knownExtensions` 登记一行。将来扩展增多，可平滑升级为远程 extension index（az 模式）。
- 取舍：相比 git/docker/gh 的"未知即报错"，这给了 az 式的安装指引，降低摩擦；写死 map 避免引入远程索引的复杂度。

## 5. goxctl-claude extension

### 5.1 仓库结构
```
goxctl-claude/
├── cmd/goxctl-claude/         # 入口 + 命令（main/root/add/update/remove/list/check），遵循 cli.md
├── internal/claude/           # 业务逻辑（manifest/lock/fetch/sync）
├── steering/                  # 规范数据（英文、通用、无项目专属词）
│   └── rules.md / karpathy-guidelines.md / cli.md / config.md / db-migrations.md / scaffold.md
├── templates/
│   └── CLAUDE.template.md     # 通用 CLAUDE.md 模板
└── install.sh                 # 引导：缺核心则先装 goxctl
```

### 5.2 命令（对外 `goxctl claude …`，对齐 npx skills，无 sync）
| 命令 | 作用 |
|---|---|
| `goxctl claude add <source>` | 首次添加规范源并拉取（写 manifest + lock） |
| `goxctl claude update [version]` | 无参=拉到 lock 锁定版本（新 clone 恢复 / CI 校正，幂等）；带版本=升级到该 tag 并改 lock |
| `goxctl claude remove` | 移除受管文件 + 清 manifest/lock |
| `goxctl claude list` | 显示源 / 版本 / 受管文件 |
| `goxctl claude check` | 校验受管 digest == lock（CI 防漂移） |

### 5.3 manifest `.goxctl-claude.yaml`（进 git）
```yaml
source: github.com/chinayin/goxctl-claude
version: v1.0.0            # 精确 tag（不支持语义范围）
paths: [ steering/ ]       # 按目录/glob，新增规范文件自动带上
target: .kiro/steering     # 落地目录（Kiro + Claude Code 共用）
```

### 5.4 lock `.goxctl-claude.lock`（进 git）
```yaml
source: github.com/chinayin/goxctl-claude
version: v1.1.0
resolved: 9f3a2c1d...      # commit sha —— 唯一完整性锚点
managed:                   # 受管文件列表，update 时自动生成（非手动、非逐文件 hash）
  - rules.md
  - cli.md
digest: sha256:ab12...     # 受管文件整体摘要，仅供离线 check
```

### 5.5 拉取与完整性
- 拉取：GitHub release tarball（按 tag）解出 `paths` 子集 → `target`；纯 HTTP。
- `update` 只覆盖/清理 `managed` 列表内文件，项目自有 steering 原样保留。
- `check` 重算受管 digest 与 lock 比对。

## 6. 安装与引导

```bash
# 一键：扩展安装器确保核心在场
curl -sSfL https://raw.githubusercontent.com/chinayin/goxctl-claude/main/install.sh | sh
# 或已有核心（module 路径写全；go install 按约定从 <module>/cmd/<repo名> 安装）
goxctl extension install github.com/chinayin/goxctl-claude
```
对外一律 `goxctl claude …`。`add` 的 source 与 `extension install` 的 module 均可简写为 `owner/repo`（无 host 时默认补 github.com，由 `ensureHost`/`parseSource` 对称处理）。

## 7. CLAUDE.md 通用模板（templates/CLAUDE.template.md）

英文、零项目词；项目专属隔离到 `## Project context` 或本地 PROJECT.md：
```markdown
# Team Engineering Standards (Go)

Authoritative source: `.kiro/steering/` (synced via `goxctl claude`; reference only).

@.kiro/steering/rules.md
@.kiro/steering/karpathy-guidelines.md

## On-demand standards — read before touching the relevant area
- CLI tools → `.kiro/steering/cli.md`
- Config / struct tags → `.kiro/steering/config.md`
- DB migrations → `.kiro/steering/db-migrations.md`
- Project scaffold → `.kiro/steering/scaffold.md`

## Project context
<!-- project-specific only -->
```

## 8. 版本化与更新流程
```
goxctl-claude 改规范 → 打 tag v1.1.0
  ↓
项目: goxctl claude update v1.1.0     # 改 lock 的 resolved/version/digest，重写受管文件
  ↓
git diff 审阅 .kiro/steering 变化 → 提交（版本 pin 在 .goxctl-claude.lock）
  ↓
CI: goxctl claude check               # 防手改/漏更
```

## 9. 与 Claude Code plugin 的关系（两套 plugin，别混）

| | goxctl extension | Claude Code plugin |
|---|---|---|
| 层 | 命令行（可执行文件转发） | AI 会话（skills/hooks/commands） |
| 例子 | `goxctl-claude` 二进制 | `/standards-sync` 命令、gofumpt hook |
| 协作 | — | Claude plugin 的 `/command` 底层调 `goxctl claude update` |

## 10. 落地阶段

1. **goxctl-claude repo**：把 proxyhive/.kiro 规范英文化、去项目词搬入 `steering/` + 打 v1.0.0；先有数据。
2. **goxctl-claude 工具**：`add/update/remove/list/check` + manifest/lock，用 gox/cli 写。
3. **goxctl 核心壳**：转发 + extension 管理 + install.sh 引导。
4. **接入**：gox / gox-browser 用 `goxctl claude` 同步 `.kiro/steering`，CLAUDE.md 换通用模板。
5. **可选**：Claude Code plugin 封装 `goxctl claude` 命令 + gofumpt hook。

## 11. 开放问题（待定）

- **repo 可见性**：goxctl-claude 公开 or 私有（私有需 token：GH_TOKEN / gh auth）—— **未定**。
- manifest/lock 文件名是否定为 `.goxctl-claude.{yaml,lock}`。
- extension 发现目录 `~/.goxctl/extensions/` vs 复用 PATH。
- 先做 goxctl 核心壳还是先把 goxctl-claude 做成可独立运行、之后再补核心转发。
