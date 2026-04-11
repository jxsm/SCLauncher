# GitHub Actions 自动构建和发布流程

这个 workflow 会在合并到主分支时自动构建 Wails 应用并发布到 GitHub Releases。

## 功能特性

- ✅ 自动版本号管理（支持自定义或自动累加）
- ✅ 自动构建 Windows 可执行文件
- ✅ 自动创建 GitHub Release
- ✅ 自动生成 Release Notes
- ✅ 支持手动触发构建

## 触发方式

### 1. 自动触发（合并到 main 分支）

当你将 PR 合并到 `main` 分支时，workflow 会自动运行。

**默认行为：** 自动累加版本号（如 0.7.1 → 0.7.2）

### 2. 提交信息中指定版本号

在合并的提交信息中包含 `release: x.x.x`，例如：

```
release: 1.0.0

添加了新功能并修复了一些 bug
```

### 3. 手动触发

1. 进入 GitHub 仓库的 **Actions** 标签页
2. 选择 **Build and Release** workflow
3. 点击 **Run workflow**
4. 填写参数：
   - **自定义版本号**：留空则自动累加，输入版本号如 `1.0.0` 则使用自定义版本
   - **Release 发布信息**：留空则自动生成，输入自定义内容则使用该内容

## 版本号规则

### 自动累加

当没有指定版本号时，会自动增加补丁版本号：
- `0.7.1` → `0.7.2`
- `1.2.3` → `1.2.4`

### 自定义版本号

支持以下格式：
- `1.0.0` - 使用 `release: 1.0.0` 前缀或手动触发时输入
- `v1.0.0` - 带有 `v` 前缀也可以

## 构建产物

每次构建会生成以下文件并上传到 Release：

1. **SCLauncher-{version}-windows-amd64.exe** - 完整的可执行文件
2. **SCLauncher-{version}-windows-amd64.zip** - 压缩包版本

## 自动生成的 Release Notes

如果没有自定义 Release Notes，系统会自动生成，包含：

- 当前版本的改动（基于 git commit 信息，每条单独一行并编号）
- 下载链接说明
- 完整的变更日志链接

**示例格式：**
```markdown
## What's New

### Changes

1. 更改下载显示 (175ab96)
2. 版本列表修改 (cd1316d)
3. 更改默认下载源 (2b9784f)
...

---

**Download:** Choose the asset below that matches your system.

**Full Changelog:** https://github.com/jxsm/SCLauncher/commits/0.7.2
```

## 本地测试版本号

你可以使用以下命令测试版本号变更：

```bash
# 查看当前版本
grep Version backend/appinfo/appinfo.go

# 创建测试提交
git commit --allow-empty -m "release: 0.8.0" -m "测试版本号功能"
```

## 注意事项

1. **版本号更新**：workflow 会自动更新 `backend/appinfo/appinfo.go` 中的版本号并提交
2. **Git Tag**：每次发布会自动创建对应的 git tag（如 `v0.7.2`）
3. **权限要求**：需要仓库的 `contents: write` 权限（workflow 已自动配置）
4. **构建时间**：首次构建可能需要 5-10 分钟

## 高级用法

### 创建预发布版本

如需创建 beta/rc 版本，可以修改 workflow 中的 `prerelease` 参数：

```yaml
- name: Create GitHub Release
  uses: softprops/action-gh-release@v1
  with:
    prerelease: ${{ contains(steps.new_version.outputs.new_version, 'beta') || contains(steps.new_version.outputs.new_version, 'rc') }}
```

### 多平台构建

如需构建其他平台，可以添加平台参数：

```yaml
- name: Build Wails application (Linux)
  run: wails build -clean -platform linux/amd64

- name: Build Wails application (macOS)
  run: wails build -clean -platform darwin/amd64
```

## 故障排查

### 构建失败

1. 检查 Go 和 Node.js 版本是否正确
2. 确认 frontend 依赖可以正常安装
3. 查看 Actions 日志获取详细错误信息

### 版本号冲突

如果指定的版本号已存在，workflow 会失败。请确保：
- 自定义版本号未被使用
- 或删除旧的 tag 和 release 后重试

### 权限问题

确保 GitHub Token 有足够的权限：
1. 进入仓库 Settings → Actions → General
2. 在 **Workflow permissions** 中选择 **Read and write permissions**

## 工作流程图

```
合并到 main 分支
       ↓
   检测触发方式
       ↓
   确定新版本号
       ↓
   更新 appinfo.go
       ↓
   安装依赖 (Go + Node.js)
       ↓
   运行 wails build
       ↓
   打包构建产物
       ↓
   创建 GitHub Release
       ↓
   创建 Git Tag
       ↓
   完成 ✅
```
