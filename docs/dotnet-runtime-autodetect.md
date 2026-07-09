# 自动检测并安装 .NET 运行时环境 — 实现方案

> 状态：待评审 / 未实现
> 关联：用户点击「启动游戏」时，在真正拉起游戏进程之前，自动补齐缺失的 .NET 运行时。

---

## 1. 背景与目标

新版 Survivalcraft（联机版 / .NET 版）基于 **.NET Desktop Runtime** 运行。
当用户机器没有安装对应大版本的运行时时，游戏会直接闪退（启动后立即退出，退出码非 0），
用户体验差且很难自查原因。

本功能目标：**在 `LaunchGame` 流程里插入一个前置检查**，
自动判断游戏所需运行时 → 比对本地是否已装 → 若缺失则自动补装 → 再继续启动游戏。

### 1.1 已确认的产品决策（来自与用户的确认）

| 决策点 | 选择 |
|---|---|
| 缺失运行时时的处理方式 | **弹窗确认后安装**：弹出对话框告知「需要安装 .NET X 运行时」，用户点「安装」后才执行 |
| winget 失败后的回退方式 | **下载微软官方安装程序并自动运行**（被动模式 `/passive`） |
| 安装程序运行模式 | **被动模式 `/passive`**（显示进度条、无需交互、自动关闭） |
| 安装完成后是否自动启动游戏 | **自动启动**：安装成功后立即继续启动游戏，无需再次点击 |
| 是否提供开关 | **提供设置开关，默认开启**（新增配置项 `AutoCheckRuntime`） |

---

## 2. 整体流程

```
用户点击「启动游戏」(LaunchGame)
        │
        ▼
GameManager.Launch(versionID)
        │
        ├─ 原有前置检查（是否已运行、版本是否存在、查找 exe、工作目录）……
        │
        ▼
【新增】 ensureDotNetRuntime(versionID)   ← 运行时环境检查 + 补装
        │
        │  ① 探测游戏所需 .NET 大版本（major）
        │  ② 查询本地是否已安装该大版本的 Desktop Runtime
        │  ③ 若已装 → 直接返回 nil，继续启动
        │  ④ 若未装 → 弹窗确认 → winget 安装 → 失败回退下载安装程序 → 运行
        │  ⑤ 安装成功 → 返回 nil，继续启动
        ▼
原有后续流程（同步皮肤 → 启动进程 → 监控）……
```

注入点：[backend/game/process.go](../backend/game/process.go#L100) 的 `Launch` 方法内，
在 `getGameWorkDirectory(...)` 之后、`SyncSkinsToGame` 之前插入对 `ensureDotNetRuntime` 的调用。

---

## 3. 模块划分

新建独立包 `backend/dotnet/`，与 `backend/version`、`backend/game` 等并列，
保证逻辑可单元测试、与 Wails 上下文解耦。

```
backend/dotnet/
├── detect.go        # ① 从游戏目录探测所需 .NET 版本
├── detect_test.go
├── installed.go     # ② 查询本机已安装的 .NET Desktop Runtime 版本列表
├── installed_test.go
├── releases.go      # ③ 解析微软 releases.json，定位官方安装包 URL
├── releases_test.go
├── installer.go     # ④ winget 安装 + 下载并运行安装程序（/passive）
├── installer_test.go
├── manager.go       # ⑤ 编排：detect → installed → install 的高层入口
└── manager_test.go
```

每个文件配套 `_test.go`，目标覆盖率 **≥ 90%**。

### 3.1 关键设计原则（为了可测试 / 高覆盖）

- **所有外部副作用抽象成接口**：HTTP 抓取、命令执行、文件下载、UAC 进程拉起。
  生产代码注入默认实现，单测注入 fake/mock。
- **纯函数下沉**：解析 `runtimeconfig.json`、解析 `deps.json`、解析 `dotnet --list-runtimes` 输出、
  从 `releases.json` 挑选安装包 URL 等，全部做成纯函数（输入字节 / 输入结构 → 输出结果），
  用表驱动测试覆盖，零外部依赖即可跑到高覆盖。
- **不直接依赖 `wails runtime`**：`dotnet` 包不 import wails；
  需要向前端发事件 / 弹窗时，由 `GameManager` 通过回调传入。

---

## 4. 详细设计

### 4.1 探测游戏所需版本 — `detect.go`

优先级（与用户描述一致）：

1. **优先**：找 `*.runtimeconfig.json`，读 `runtimeOptions.tfm`
   - 例：`"tfm": "net10.0"` → 目标大版本 = `10`
   - 校验：若 `runtimeOptions.configProperties` / `frameworks` 为空且目录里自带大量 `Microsoft.*.dll`，
     视为 **自包含(self-contained)**，无需安装 → 直接返回「无需检查」。
2. **回退 A**：找 `*.deps.json`，读 `runtimeTarget.name`
   - 例：`".NETCoreApp,Version=v10.0"` → 用正则 `v(\d+)\.` 提取大版本 = `10`
3. **都没有**：视为旧版原生游戏（2.3- 及更早），跳过检查。

**接口：**

```go
// RequiredRuntime 描述一次探测结果
type RequiredRuntime struct {
    Needed      bool   // 是否需要安装系统运行时（false = 自包含或旧版，跳过）
    MajorVersion int   // 例如 10、9
    Source      string // "runtimeconfig" | "deps" | "none"，便于日志
    TFM         string // 原始 tfm，例如 "net10.0"
}

// DetectRequired 从游戏目录探测所需运行时
func DetectRequired(gameDir string) (RequiredRuntime, error)
```

探测策略：遍历 `gameDir`（含子目录）查找目标文件，
- `*.runtimeconfig.json`：用 `encoding/json` 解析，结构体只声明需要的字段（`runtimeOptions.tfm`）。
- `*.deps.json`：解析 `runtimeTarget.name`。

### 4.2 查询本机已安装版本 — `installed.go`

**主路径**：执行 `dotnet --list-runtimes`，解析输出。

输出示例：
```
Microsoft.WindowsDesktop.App 9.0.16 [...\DesktopRuntime]
Microsoft.WindowsDesktop.App 10.0.5  [...\DesktopRuntime]
Microsoft.NETCore.App 9.0.16 [...]
```

只关心 `Microsoft.WindowsDesktop.App` 这一行（游戏是 WPF/WinForms 桌面应用），
取版本号的「主版本」集合。

**回退路径**：当 `dotnet` 不在 PATH（极端情况），
读取注册表 `HKLM\SOFTWARE\WOW6432Node\dotnet\Setup\InstalledVersions\...\sharedhost\...`
或扫 `C:\Program Files\dotnet\shared\Microsoft.WindowsDesktop.App\*` 目录名。
> 实现时优先把 `dotnet --list-runtimes` 跑通；注册表回退作为可选增强，先打 TODO。

**接口：**

```go
// InstalledVersions 列出本机已安装的 WindowsDesktop 运行时主版本集合
func ListInstalledDesktopMajors(execFn CommandRunner) (map[int]bool, error)
```

`CommandRunner` 是一个接口（`Run(name string, args ...string) (stdout, stderr string, err error)`），
便于单测注入模拟输出。

### 4.3 解析微软 releases.json — `releases.go`

接口固定：`https://builds.dotnet.microsoft.com/dotnet/release-metadata/{MAJOR}.0/releases.json`
（例：.NET 10 → `.../10.0/releases.json`，.NET 9 → `.../9.0/releases.json`）。

已核实 JSON 结构（关键字段）：

```jsonc
{
  "channel-version": "9.0",
  "latest-release": "9.0.16",
  "releases": [
    {
      "release-version": "9.0.16",
      "windowsdesktop": {
        "version": "9.0.16",
        "files": [
          { "name": "windowsdesktop-runtime-win-x64.exe", "rid": "win-x64",
            "url": "https://builds.dotnet.microsoft.com/dotnet/WindowsDesktop/9.0.16/windowsdesktop-runtime-9.0.16-win-x64.exe",
            "hash": "<sha512>" },
          { "name": "windowsdesktop-runtime-win-x86.exe", "rid": "win-x86", "url": "...", "hash": "..." },
          { "name": "windowsdesktop-runtime-win-arm64.exe", "rid": "win-arm64", "url": "...", "hash": "..." }
        ]
      }
    }
  ]
}
```

**挑选规则**：取 `releases[0]`（即最新）的 `windowsdesktop.files`，
按 OS 体系结构匹配 RID：
- x64 → `win-x64`
- x86 → `win-x86`
- arm64 → `win-arm64`

OS 体系结构由 Go 的 `runtime.GOARCH` 决定（`amd64`→x64，`arm64`→arm64，`386`→x86）。

**接口：**

```go
type InstallerAsset struct {
    Version string // 例如 "9.0.16"
    URL     string
    SHA512  string
    RID     string
}

// PickLatestDesktopInstaller 从 releases.json 字节里挑出指定架构的最新安装包
func PickLatestDesktopInstaller(jsonData []byte, goarch string) (InstallerAsset, error)

// ReleasesURL 由主版本号拼出 releases.json 的 URL
func ReleasesURL(major int) string
```

> hash 字段是 sha512，可选校验：下载完成后 `sha512(文件) == hash`，不一致则报错中止（防下载损坏）。

### 4.4 安装 — `installer.go`

**默认只走「下载 + `/passive` 运行官方安装包」，不使用 winget。**

> 选型理由（实现阶段调整）：winget 没有可对接的下载进度（启动器无法显示进度条），
> 且会弹出 CMD 黑框，用户易产生安全顾虑。下载路径能上报真实进度（`runtime:install:progress`），
> `/passive` 由安装程序自身的图形进度窗口承接，**无控制台窗口**。
> `InstallViaWinget` 作为公开方法保留，供未来「高级：用 winget」开关复用，但 `Manager.Install` 默认不调用。

#### 4.4.1 winget 安装（备选，默认不调用）

命令（非交互、自动接受协议）：
```
winget install --id Microsoft.DotNet.DesktopRuntime.<MAJOR> -e --silent --accept-package-agreements --accept-source-agreements
```
例：`.10`、`.9`。判定成功：`exit code == 0`。

#### 4.4.2 下载 + 运行官方安装程序（默认路径）

- 下载：`http.Get` + 流式写入临时文件，带进度回调；文件名取自 URL basename
  （`windowsdesktop-runtime-<ver>-win-x64.exe`）。
- 校验：sha512 与 `releases.json` 里的 `hash` 比对；hash 为空则跳过校验。
- 运行：`exec.Command(installerPath, "/passive", "/norestart")` 并 `Wait()`。
  `/passive`：安装程序自身图形进度条、无需点击、自动结束；安装包是 GUI 子系统，**无 CMD 黑框**。
  退出码 0 = 成功；3010 = 成功但需重启（视为成功）。

**接口（命令执行同样走 CommandRunner 接口）：**

```go
type Installer struct {
    HTTPGet    func(url string) (*http.Response, error)   // 默认 http.DefaultClient.Get
    Runner     CommandRunner
    Logger     func(format string, args ...any)
    TmpDir     string
}

// InstallViaWinget 备选路径；Manager.Install 默认不调用
func (i *Installer) InstallViaWinget(major int) error

// DownloadAndRun 下载安装包（带进度 + sha512 校验）并以 /passive 运行
func (i *Installer) DownloadAndRun(asset InstallerAsset, progress func(downloaded, total int64)) error
```

### 4.5 编排 — `manager.go`

```go
type Manager struct {
    Runner        CommandRunner
    HTTPClient    *http.Client
    FetchReleases func(major int) ([]byte, error) // 默认从 builds.dotnet.microsoft.com 抓
    Logger        func(format string, args ...any)
    SharedDir     string
}

// Status 检查游戏目录所需运行时 + 本机是否已装
func (m *Manager) Status(gameDir string) (Status, error)

// Install 下载官方安装包并以 /passive 运行（progress 上报下载进度）
func (m *Manager) Install(major int, progress func(downloaded, total int64)) error
```

实际编排（`Status` + `Install`，App 层在两者之间负责弹窗确认）：

1. `Status(gameDir)` → `DetectRequired`；若 `!Needed`，前端直接放行启动。
2. `IsInstalled(major)`（`dotnet --list-runtimes` 为主，安装目录扫描为回退）；若已装，放行。
3. 缺失 → **前端 Vue 弹窗确认**（决策 #1）；取消 → 中止启动。
4. 确认 → `Install(major, progress)`：抓 `releases.json` → `PickLatestDesktopInstaller` → `DownloadAndRun`。
   下载进度由 App 层 `EventsEmit("runtime:install:progress", ...)` 推到前端进度弹窗。
5. 失败 → 返回错误，启动中止，前端给出友好提示。

> 注意：`Ensure` 本身 **不弹窗**（保持 `dotnet` 包纯净、可单测）。
> 弹窗由 `GameManager` 在调用前后通过 Wails runtime 完成（见 §5）。

---

## 5. 与现有代码的集成

### 5.1 `backend/game/process.go` 的 `Launch`

在 [process.go:100](../backend/game/process.go#L100)（拿到 `workDir` 之后、同步皮肤之前）插入：

```go
// 运行时环境检查（可在设置中关闭）
if g.config.AutoCheckRuntime {
    if err := g.ensureDotNetRuntime(workDir); err != nil {
        return fmt.Errorf("运行时环境准备失败: %w", err)
    }
}
```

新增方法 `GameManager.ensureDotNetRuntime(gameDir string) error`，
负责：调用 `dotnet` 包的 `DetectRequired` + `ListInstalledDesktopMajors`，
若需安装则：
1. `runtime.EventsEmit(g.ctx, "runtime:required", {major, version, tfm})` 通知前端；
2. 通过 Wails 的对话框（或前端弹窗 + 事件回调）等待用户确认；
3. 调用 `dotnet.Manager.Ensure` 完成安装。

> 「等待用户确认」的实现：前端监听 `runtime:required` 事件后弹窗，
> 用户点「安装」→ 调用新增的 `App.ConfirmRuntimeInstall()` Wails 方法推进；
> 点「取消」→ 调用 `App.CancelRuntimeInstall()`。
> 为简化首版，可用 `runtime.MessageDialog`（同步确认框）直接拿用户选择，避免异步状态机。
> **推荐先用 `MessageDialog` 同步确认**，简单可靠。

### 5.2 配置项 — `backend/config/config.go`

在 `Config` 结构体（参考 [config.go:19](../backend/config/config.go#L19)）新增：

```go
AutoCheckRuntime bool `json:"autoCheckRuntime"` // 默认 true
```

- `DefaultConfig()` 里设为 `true`。
- 新增 `SetAutoCheckRuntime(bool) error` setter（参考 `SetTheme` 等模式）。
- `GetConfig`（[app.go:173](../app.go#L173)）的返回 map 里加上该字段供前端显示。

### 5.3 Wails 绑定 — `app.go`

新增（自动暴露给前端，参考 `SetTheme` 写法）：

```go
func (a *App) SetAutoCheckRuntime(enabled bool) error
```

（首版若用 `MessageDialog` 同步确认，则无需额外的 `ConfirmRuntimeInstall`/`CancelRuntimeInstall` 方法。）

### 5.4 前端 / 国际化

经核实前端栈：Vue3 + Pinia + **Naive UI** + vue-i18n，入口位于 [frontend/src/stores/game.ts:50](../frontend/src/stores/game.ts#L50) 的 `launchGame`，对话框用 `dialog.create()`（见 [Settings.vue:426](../frontend/src/views/Settings.vue#L426)），进度弹窗可参考 [components/ModpackInstallDialog.vue](../frontend/src/components/ModpackInstallDialog.vue)，事件订阅参考 [stores/download.ts:40](../frontend/src/stores/download.ts#L40)。

- **注入点**：在 `gameStore.launchGame(versionId)` 里，`await gameApi.LaunchGame` 之前插入
  `CheckDotNetRuntime(versionId)` → 缺失则 `dialog.create()` 确认 → `InstallDotNetRuntime`（监听 `runtime:install:progress` 显示进度弹窗）→ 成功后继续 `LaunchGame`。
- **设置开关**：设置页新增一个 `RuntimeSettings.vue`（复用 [SourceListItem.vue:14](../frontend/src/components/settings/SourceListItem.vue#L14) 的 `n-switch` 模式），绑定 `GetConfig.autoCheckRuntime` / `SetAutoCheckRuntime`。
- **国际化**：项目支持 **8 种**语言，全部要覆盖：
  `zh-CN / en-US / ru-RU / pt-BR / hi-IN / id-ID / ar-SA / es-ES`（见 [frontend/src/locales/index.ts](../frontend/src/locales/index.ts)）。
  新增一组 key，例如：
  ```ts
  runtime: {
    notInstalled: '未检测到所需 .NET 运行时',
    installPrompt: '游戏需要安装 .NET {version} 运行时，是否立即安装？',
    installing:   '正在安装 .NET 运行时…',
    success:      '安装成功',
    failed:       '安装失败',
    userCancelled:'已取消，游戏未启动',
  }
  ```
- **进度**：安装过程通过 `runtime.EventsEmit("runtime:install:progress", ...)` 反馈，
  前端显示一个 `n-modal` 进度弹窗（复用 ModpackInstallDialog 风格）。

---

## 6. 错误处理与边界

| 场景 | 处理 |
|---|---|
| 游戏目录无 runtimeconfig / deps | `Needed=false`，跳过，正常启动 |
| 游戏是 self-contained | `Needed=false`，跳过 |
| `dotnet` 未安装 / 不在 PATH | `--list-runtimes` 失败 → 视为「未安装」→ 进入安装流程 |
| winget 不存在 | `exec.LookPath` 失败 → 直接走下载回退 |
| winget 失败、下载也失败 | 返回错误，启动中止，前端提示「请手动安装 .NET X」并附官方下载页 |
| 安装程序退出码 3010 | 视为成功（需重启），继续启动游戏（可能仍失败，但属合理行为） |
| 用户在确认框点「取消」 | 返回 `ErrUserCancelled`，启动中止，不发错误弹窗 |
| 设置里关闭了检查 | 完全跳过，保持原有行为 |

### 6.1 幂等性

- 已安装相同大版本 → 不重复安装（`ListInstalledDesktopMajors` 命中即返回）。
- 安装完成后再次启动同一版本 → 立即放行。

---

## 7. 测试策略（目标覆盖率 ≥ 90%）

### 7.1 纯函数（表驱动，易达 100%）

- `detect.go`
  - `parseRuntimeConfigTFM([]byte) (string, bool)`：net10.0 / net9.0 / 空 / 自包含 / 非法 JSON。
  - `parseDepsRuntimeTarget([]byte) (int, bool)`：`v10.0` / `v9.0` / 无该字段 / 非法。
  - `DetectRequired(dir)`：用 `t.TempDir()` 构造真实文件树（含 / 不含目标文件、自包含标记）。
- `releases.go`
  - `PickLatestDesktopInstaller`：用内置的真实 JSON 片段作 fixture，覆盖 x64/x86/arm64、
    `windowsdesktop` 缺失、`files` 为空、未知 RID、JSON 损坏等分支。
  - `ReleasesURL(major)`：参数化。
- `installed.go`
  - `ParseListRuntimes(stdout) map[int]bool`：多版本 / 无 Desktop 行 / 空输出 / 形如 `Preview`。

### 7.2 接口注入（mock CommandRunner / HTTP）

- `installer.go`
  - `InstallViaWinget`：mock winget 成功 / 失败 / 不存在三种。
  - `DownloadAndRun`：用 `httptest.Server` 提供假安装包（一个小 exe 或 dummy 字节），
    mock `Run` 校验收到 `/passive /norestart` 参数；覆盖下载失败、sha512 校验失败、运行失败。
- `manager.go`
  - `Ensure`：组合 mock，覆盖「不需」「已装」「winget 成功」「winget 失败→下载成功」「全失败」全部路径。

### 7.3 覆盖率门槛

- 在 CI 或本地用 `go test ./backend/dotnet/... -cover`，目标 **≥ 90%**。
- 对无法在单测里覆盖的系统调用层（真实 winget、真实 UAC 提权），
  保证它们只出现在被接口包裹的薄实现里，不拉低整体覆盖率。

---

## 8. 待实现文件清单

新增：
- [backend/dotnet/detect.go](../backend/dotnet/detect.go) + `_test.go`
- [backend/dotnet/installed.go](../backend/dotnet/installed.go) + `_test.go`
- [backend/dotnet/releases.go](../backend/dotnet/releases.go) + `_test.go`
- [backend/dotnet/installer.go](../backend/dotnet/installer.go) + `_test.go`
- [backend/dotnet/manager.go](../backend/dotnet/manager.go) + `_test.go`

修改：
- [backend/game/process.go](../backend/game/process.go)：`Launch` 内加调用 + 新增 `ensureDotNetRuntime` 方法。
- [backend/config/config.go](../backend/config/config.go)：新增 `AutoCheckRuntime` 字段 + setter + 默认值。
- [app.go](../app.go)：`GetConfig` 暴露字段 + 新增 `SetAutoCheckRuntime` 绑定。
- 前端：设置页开关 + i18n 文案 + 进度事件监听。

---

## 9. 待确认 / 风险

1. **确认框方式**：首版用 Wails 同步 `MessageDialog`（简单）；若 UI 团队想要自定义弹窗，再改成事件驱动。
   → 暂按同步框实现，实施时若发现阻塞再调整。
2. **winget 包 ID 大版本映射**：`Microsoft.DotNet.DesktopRuntime.10` / `.9` 是约定俗成，
   实施时先用 `winget search` 复核目标大版本的确切 ID，必要时维护一个 `map[int]string`。
3. **架构识别**：默认按 `runtime.GOARCH`（启动器自身架构）。
   罕见情况下游戏是 32 位、启动器是 64 位——首版不处理，按启动器架构走，后续按需细化。
4. **releases.json 缓存**：首版不缓存，每次需要时实时抓（只在确实要下载安装包时才抓）。
   后续可加内存缓存（24h TTL）减少请求。

---

## 10. 实施顺序（建议）

1. `detect.go` + 测试（独立、零依赖，最先完成）
2. `releases.go` + 测试（已有 JSON fixture）
3. `installed.go` + 测试
4. `installer.go` + 测试（httptest + mock exec）
5. `manager.go` + 测试（编排）
6. `config` 字段 + `app.go` 绑定
7. `game/process.go` 集成 + 同步确认框
8. 前端开关 + i18n + 进度
9. `go test ./backend/dotnet/... -cover` 确认 ≥ 90%，补齐用例
10. 真机联调（找一台没装目标运行时的机器跑一遍 winget 路径与下载路径）
