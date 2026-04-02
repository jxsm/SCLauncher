# SCLauncher - 生存战争启动器

<div align="center">

![Version](https://img.shields.io/badge/version-0.1.0-blue)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![Vue](https://img.shields.io/badge/Vue-3.3+-4FC08D?logo=vue.js)
![Wails](https://img.shields.io/badge/Wails-v2.0+-5A5DF5)

一个现代化的生存战争（SurvivalCraft）游戏启动器

[功能特性](#功能特性) • [快速开始](#快速开始) • [开发文档](#开发文档) • [项目结构](#项目结构)

</div>

## 📖 项目简介

SCLauncher 是一个使用 **Wails** (Go + Vue 3) 构建的现代化游戏启动器，专为《生存战争》游戏设计。提供版本管理、模组安装、一键启动等功能。

### ✨ 功能特性

#### 🎮 核心功能
- **一键启动** - 快速启动游戏，支持自定义启动参数
- **版本管理** - 多版本隔离，轻松切换不同版本
- **版本下载** - 自动从清单文件下载游戏版本
- **模组管理** - 简单的模组安装/卸载功能
- **进度显示** - 实时显示下载和安装进度

#### 🎨 界面设计
- **简洁现代** - 扁平化设计，清爽的视觉体验
- **响应式布局** - 适配不同分辨率
- **流畅动画** - 细腻的交互体验

#### 🔧 技术特性
- **轻量级** - 基于 Wails，体积小、启动快
- **跨平台** - 支持 Windows、macOS、Linux
- **高性能** - Go 后端 + Vue 3 前端
- **数据持久化** - SQLite 本地数据库

## 🚀 快速开始

### 环境要求

- **Go**: 1.21 或更高版本
- **Node.js**: 18 或更高版本
- **Wails CLI**: 最新版本

### 安装依赖

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 安装前端依赖
cd frontend
npm install
cd ..
```

### 开发模式

```bash
# 启动开发服务器（热重载）
wails dev
```

### 构建应用

```bash
# 构建生产版本
wails build

# Windows 构建产物
./build/bin/SCLauncher.exe
```

## 📚 开发文档

### 🏗️ 项目结构

```
SCLauncher/
├── backend/              # Go 后端
│   ├── game/            # 游戏进程管理
│   ├── version/         # 版本管理
│   ├── mod/             # 模组管理
│   ├── download/        # 下载管理
│   └── storage/         # 数据存储
├── frontend/            # Vue 3 前端
│   ├── src/
│   │   ├── api/        # API 调用
│   │   ├── components/ # 组件
│   │   ├── views/      # 页面
│   │   ├── stores/     # 状态管理
│   │   └── types/      # 类型定义
└── resources/          # 资源文件
```

### 🔌 API 接口定义

#### 游戏 API

```go
// 启动游戏
func (a *App) LaunchGame(versionId string) error

// 停止游戏
func (a *App) StopGame() error

// 获取游戏状态
func (a *App) GetGameStatus() GameStatus
```

#### 版本 API

```go
// 获取所有版本
func (a *App) GetVersions() []Version

// 下载版本
func (a *App) DownloadVersion(versionId string) error

// 删除版本
func (a *App) DeleteVersion(versionId string) error

// 切换版本
func (a *App) SwitchVersion(versionId string) error
```

#### 模组 API

```go
// 获取模组列表
func (a *App) GetMods(versionId string) []Mod

// 安装模组
func (a *App) InstallMod(versionId string, modPath string) error

// 卸载模组
func (a *App) UninstallMod(versionId string, modId string) error
```

#### 下载 API

```go
// 获取下载任务列表
func (a *App) GetDownloads() []DownloadTask

// 暂停下载
func (a *App) PauseDownload(taskId string) error

// 取消下载
func (a *App) CancelDownload(taskId string) error
```

### 📊 数据模型

#### Version（版本）

```typescript
interface Version {
  id: string              // 版本 ID
  name: string            // 版本名称
  version: string         // 版本号（如 1.0.0）
  type: 'release' | 'beta' | 'alpha'  // 版本类型
  releaseDate: string     // 发布日期
  size: number            // 文件大小（字节）
  downloadUrl: string     // 下载地址
  installed: boolean      // 是否已安装
  localPath?: string      // 本地路径
  checksum?: string       // 文件校验和
}
```

#### Mod（模组）

```typescript
interface Mod {
  id: string              // 模组 ID
  name: string            // 模组名称
  version: string         // 模组版本
  author: string          // 作者
  description: string     // 描述
  enabled: boolean        // 是否启用
  installDate: string     // 安装日期
  files: string[]         // 模组文件列表
}
```

#### DownloadTask（下载任务）

```typescript
interface DownloadTask {
  id: string              // 任务 ID
  type: 'version' | 'mod' // 下载类型
  name: string            // 名称
  totalSize: number       // 总大小
  downloadedSize: number  // 已下载大小
  speed: number           // 下载速度（字节/秒）
  status: 'pending' | 'downloading' | 'paused' | 'completed' | 'failed'
  error?: string          // 错误信息
}
```

### 🗄️ 清单文件格式

清单文件（manifest.json）格式示例：

```json
{
  "formatVersion": 1,
  "gameId": "survivalcraft",
  "versions": [
    {
      "id": "sc-1.0.0",
      "name": "生存战争 1.0.0",
      "version": "1.0.0",
      "type": "release",
      "releaseDate": "2024-01-01",
      "size": 524288000,
      "downloadUrl": "https://example.com/sc-1.0.0.zip",
      "checksum": {
        "algorithm": "sha256",
        "value": "abc123..."
      },
      "description": "首个正式版本"
    }
  ]
}
```

### 📁 目录结构设计

启动器运行后会创建以下目录结构：

```
SCLauncher/
├── data/
│   ├── database.db       # SQLite 数据库
│   └── config.json       # 配置文件
├── versions/             # 游戏版本目录
│   ├── sc-1.0.0/        # 版本隔离目录
│   │   ├── game.exe     # 游戏可执行文件
│   │   ├── data/        # 游戏数据
│   │   └── mods/        # 模组目录
│   └── sc-1.1.0/
└── downloads/            # 临时下载目录
    └── .tmp/
```

### 🔄 状态管理

使用 Pinia 进行状态管理，主要 Store：

- `gameStore` - 游戏状态（运行状态、当前版本）
- `versionStore` - 版本列表、下载状态
- `modStore` - 模组列表
- `downloadStore` - 下载任务列表

### 🎯 开发路线图

#### Phase 1: 核心功能 ✅
- [x] 项目初始化
- [ ] 基础 UI 框架搭建
- [ ] 版本管理（下载、安装、删除）
- [ ] 游戏启动/停止

#### Phase 2: 扩展功能
- [ ] 模组管理
- [ ] 下载管理器
- [ ] 设置页面

#### Phase 3: 优化提升
- [ ] UI 美化
- [ ] 性能优化
- [ ] 错误处理
- [ ] 日志系统

## 🛠️ 技术栈

### 后端
- **Wails v2** - 桌面应用框架
- **Go 1.21+** - 后端逻辑
- **SQLite** - 数据存储
- **GORM** - ORM 框架（可选）

### 前端
- **Vue 3** - UI 框架
- **TypeScript** - 类型安全
- **Vite** - 构建工具
- **Pinia** - 状态管理
- **Vue Router** - 路由管理

## 📝 开发规范

### Git 提交规范

```
feat: 新功能
fix: 修复 bug
docs: 文档更新
style: 代码格式调整
refactor: 重构
test: 测试相关
chore: 构建/工具变动
```

### 代码规范

- **Go**: 遵循 `gofmt` 格式化
- **Vue/TS**: 使用 ESLint + Prettier

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License

## 👤 作者

jxsm - 3164058616@qq.com

---

<div align="center">

**[⬆ 回到顶部](#sclauncher---生存战争启动器)**

Made with ❤️ by Wails + Vue 3

</div>
