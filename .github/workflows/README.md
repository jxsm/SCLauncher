# GitHub Actions 自动构建和发布流程

这个 workflow 会在**推送新 tag 时**自动构建 Wails 应用并发布到 GitHub Releases。

## 功能特性

- ✅ 推送 `v*` tag 自动触发构建与发布
- ✅ 版本号直接取自 tag 名
- ✅ 自动构建 Windows 可执行文件
- ✅ 自动创建 GitHub Release
- ✅ 自动生成 Release Notes（基于两个 tag 之间的提交）
- ✅ 保留手动触发构建（此时版本号取自 `wails.json`）

## 触发方式

### 1. 推送 tag 自动触发（推荐）

创建并推送一个 `v` 开头的 tag 即可自动构建并创建对应版本的 Release：

```bash
# 1. 发布前先把 wails.json 的 info.productVersion 改成相同版本号并提交
# 2. 打 tag 并推送
git tag v1.1.0
git push origin v1.1.0
```

推送 tag 后 workflow 自动运行，Release 的版本号、名称均取自 tag（`v` 前缀仅用于匹配，产物文件名不含 `v`）。

**发布说明优先级：** 手动触发输入 > **tag 注释内容**（annotated tag 的消息）> 自动生成的提交列表。

推荐打 annotated tag 并把更新说明写进 tag 消息，发布时 Release 会直接使用：

```bash
git tag -a v1.1.0 -m "## 更新内容

- 新功能 A
- 修复 B"
git push origin v1.1.0
```

> ⚠️ 同名 tag 重复推送不会重新触发 workflow。如需重新构建，先删除远程 tag 和对应 Release，再重新推送：
> ```bash
> git push origin :refs/tags/v1.1.0   # 删除远程 tag
> git tag -d v1.1.0                   # 删除本地 tag
> ```

### 2. 手动触发

1. 进入 GitHub 仓库的 **Actions** 标签页
2. 选择 **Build and Release** workflow
3. 点击 **Run workflow**
4. 填写参数：
   - **Release 发布信息**：留空则自动生成，输入自定义内容则使用该内容

手动触发时版本号取自 `wails.json` 的 `info.productVersion`。

## 版本号规则

- **Tag 触发**：版本号 = tag 名去掉 `v` 前缀（如 `v1.1.0` → `1.1.0`）
- **手动触发**：版本号 = `wails.json` 的 `info.productVersion`

### 一致性检查

Tag 触发时，workflow 会检查 `wails.json` 中的版本号是否与 tag 一致。
不一致时构建仍会继续，但会在日志和 Summary 中给出警告——
此时构建出的 exe 内部版本号（关于页/更新检查显示的版本）与 Release 版本号不符。
**发布前请记得同步 `wails.json`。**

## 构建产物

每次构建会生成以下文件并上传到 Release：

1. **SCLauncher-{version}-windows-amd64.exe** - 完整的可执行文件
2. **SCLauncher-{version}-windows-amd64.zip** - 压缩包版本

## 自动生成的 Release Notes

如果没有自定义 Release Notes，系统会自动生成，包含：

- 当前版本的改动（基于上一个 tag 到本次 tag 之间的 git commit 信息，每条单独一行并编号）
- 下载链接说明
- 完整的变更日志链接（两个 tag 的 compare 对比页）

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

**Full Changelog:** https://github.com/jxsm/SCLauncher/compare/v1.0.9...v1.1.0
```

## 注意事项

1. **版本号同步**：发布前确保 `wails.json` 的 `info.productVersion` 与 tag 一致
2. **Tag 命名**：必须以 `v` 开头（如 `v1.1.0`），其他格式的 tag 不会触发
3. **权限要求**：需要仓库的 `contents: write` 权限（workflow 已自动配置）
4. **构建时间**：首次构建可能需要 5-10 分钟

## 高级用法

### 创建预发布版本

如需创建 beta/rc 版本，可以修改 workflow 中的 `prerelease` 参数：

```yaml
- name: Create GitHub Release
  uses: softprops/action-gh-release@v1
  with:
    prerelease: ${{ contains(steps.version.outputs.version, 'beta') || contains(steps.version.outputs.version, 'rc') }}
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

如果该 tag 已存在对应的 Release，workflow 会失败。请确保：
- 删除旧的 Release 后重试
- 或删除远程 tag 后重新推送

### 权限问题

确保 GitHub Token 有足够的权限：
1. 进入仓库 Settings → Actions → General
2. 在 **Workflow permissions** 中选择 **Read and write permissions**

## 工作流程图

```
推送 v* tag
       ↓
   检出代码 (完整历史)
       ↓
   确定版本号 (取自 tag，并校验 wails.json)
       ↓
   安装依赖 (Go + Node.js + Wails)
       ↓
   运行 wails build
       ↓
   打包构建产物 (.exe + .zip)
       ↓
   生成 Release Notes (上一个 tag 到本次提交)
       ↓
   创建 GitHub Release
       ↓
   完成 ✅
```
