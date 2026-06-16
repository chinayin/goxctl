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

- 内置子命令：`extension install/list/remove/upgrade`、`upgrade`、`version`、`help`。
- **转发**：未知子命令 → 查 `~/.gox/extensions/goxctl-<name>`（或 PATH）→ `exec` 并继承 stdio。
- **安装**（`extension install <module>`）：**优先下载与当前平台匹配的预编译二进制**（GitHub Release 资产，无需 Go 环境），无匹配时回退 `go install`（需本机有 go 工具链）。装到 `~/.gox/extensions`，二进制名即 `goxctl-<name>`，转发可直接发现。`module` 可简写 `owner/repo`（无 host 补 github.com，由 `ensureHost` 处理）。机制详见 §5。
- `extension list`：列已装扩展；`extension remove <name>`：删除。
- **调试输出**：`GOXCTL_DEBUG=1`、`goxctl --verbose <cmd>` 或 `-v` 打印解析的 owner/repo、release URL、命中资产、转发 exec 等。环境变量贯穿转发链（扩展子进程继承）；核心 `--verbose`/`-v` 写回该变量，且**扩展自身也全局支持 `--verbose`/`-v`**，故 `goxctl --verbose claude update`（前置）与 `goxctl claude update --verbose`（后置）都生效。注：`-v` 专用于 verbose，版本用 `goxctl version` 或 `goxctl --version`。
- **输出风格**：列表用对齐表格（大写表头），操作成功用 `✓`（TTY 上色，管道自动无色），错误用 `error:`；统一无静默成功（参照 gh/docker/brew）。
- **用法错误体验**：参数/flag 用法错误显示该命令 usage，业务错误不显示（各 RunE 开头设 `SilenceUsage`）；被转发的扩展非 0 退出时，核心按其退出码静默退出，不重复打印 error。
- **自更新**（区别于扩展的数据同步命令）：
  - `goxctl upgrade`：查最新 release → 下载当前平台二进制 → sha256 校验 → **原子替换运行中的二进制**（写临时 + rename，deno/bun 风格）。`--check` 只查不装；目标目录（如 `/usr/local/bin`）不可写时提示用 sudo。复用 `internal/ext` 的下载/校验逻辑（`SelfUpdate` / `LatestVersion`）。
  - `goxctl extension upgrade <name>|--all`：把已装扩展重装到最新 release（gh 风格）。显示「当前版本 → 最新版本」或 already up to date（已最新则跳过不重装）。扩展元数据（module + version）统一记在 `~/.gox/extensions.yaml`（单一清单，类似 package.json，安装/升级写入、remove 删除）；upgrade 从清单读 module 与当前版本，因此**不再依赖写死的注册表**，任意已装扩展都能升级。注册表（registry.go）只用于「未装扩展时的安装提示」。
  - 三层"更新"别混：`goxctl upgrade`=核心二进制、`extension upgrade`=扩展二进制、`goxctl <name> update`（如 `goxctl claude update`）=扩展同步的项目数据。

### 4.1 未装扩展的提示（已知扩展注册表）

仿 [Azure CLI dynamic install](https://learn.microsoft.com/en-us/cli/azure/azure-cli-extensions-overview)，但用**写死的注册表**而非远程 index（团队扩展数量有限，简单可控）：

- `cmd/goxctl/registry.go` 维护 `knownExtensions` map：子命令名 → module 路径（具体已知扩展见该文件，本文不复制其内容）。
- 转发命中 `ErrNotFound`（扩展未安装）时：
  - **在注册表** → 提示 `extension "<name>" is not installed; run: goxctl extension install <module>`；
  - **不在注册表** → 回落 cobra，报 `unknown command`。
- 新增官方扩展时在 `knownExtensions` 登记一行。将来扩展增多，可平滑升级为远程 extension index（az 模式）。
- 取舍：相比 git/docker/gh 的"未知即报错"，这给了 az 式的安装指引，降低摩擦；写死 map 避免引入远程索引的复杂度。

## 5. 分发与安装（零 Go 依赖）

核心目标：**没有 Go 环境的人也能装、能用**（前端 / 运维 / CI / Kiro 用户）。对齐 gh / krew —— 预编译二进制为主，源码编译仅作开发者回退。

### 5.1 二进制分发（goreleaser）

- 核心与各扩展各自用 `.goreleaser.yaml` + `release.yml`（push tag `v*` 触发）构建多平台二进制，发布到自己的 GitHub Releases。
- 平台矩阵：macOS / Linux × amd64 / arm64（4 个二进制）。
- 产物命名**不含版本号**：`<name>_<os>_<arch>.tar.gz` + `checksums.txt`（sha256）。不含版本是为了让安装脚本能用稳定的 `releases/latest/download/<asset>` 直链，下载端也只需按 `_<os>_<arch>.tar.gz` 后缀匹配。
- `os`/`arch` 取值与 Go 的 `runtime.GOOS`/`GOARCH` 一致（`darwin`/`linux`、`amd64`/`arm64`），下载端直接拼接。

### 5.2 安装核心（install.sh，不碰 go）

```bash
curl -sSfL https://raw.githubusercontent.com/chinayin/goxctl-claude/main/install.sh | sh
```

`install.sh` 探测 `uname` → 下载对应平台 tar.gz → 校验 sha256 → 解压到 `~/.gox/bin`（可用 `GOXCTL_BIN_DIR` 覆盖，脚本提示加入 PATH）。全程无 Go。装好核心后由核心安装扩展。

### 5.3 extension install：二进制优先，go install 回退

```
install <owner/repo> [version]：
  1. 查 GitHub Release（tag 或 latest）
  2. 找匹配 <repo>_<os>_<arch>.tar.gz 的资产
     ├─ 命中 → 下载 + 校验 sha256 + 解压 → ~/.gox/extensions/goxctl-<name>   （无需 Go）
     └─ 未命中 ↓
  3. 本机有 go → 回退 go install（开发者便捷路径）
  4. 都不行 → 报错：无预编译二进制且本机无 go
```

- release 流程的真实错误（网络、404 以外的状态、校验失败）直接上报，不静默回退，避免掩盖问题；仅"无该 release / 无匹配资产"才回退 go install。
- 公开仓库匿名下载，无需 token。
- `module` 简写规则（`owner/repo` → 补 `github.com`）由 `ensureHost` 处理；各扩展若自身命令也接受 source 简写，应保持同一规则以求对称。
- 对外心智一律 `goxctl <name> …`，`goxctl-<name>` 二进制不直接对用户宣传。

## 6. 开放问题（待定）

- extension 发现目录 `~/.gox/extensions/` vs 复用 PATH（当前两者皆查，优先安装目录）。
- 注册表后续是否需要从写死 map 升级为远程 extension index（取决于扩展数量增长）。
