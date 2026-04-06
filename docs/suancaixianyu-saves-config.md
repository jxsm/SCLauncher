# 生存战争中文社区存档下载源配置

## 配置文件信息

- **文件名**: `suancaixianyu-saves.json`
- **位置**: `.Survivalcraft/mod-sources/`
- **类型**: 存档下载源 (savegames)
- **状态**: 启用

## API 配置

### 基本信息
- **基础URL**: `https://m.suancaixianyu.cn`
- **资源类型**: 存档 (fileTypes=1)
- **排序方式**: 按时间倒序 (orderType=1)

### 列表接口
```
GET /api/post/list?type=2&orderType=1&fileTypes=1&page={page}&limit={limit}
```

**参数说明：**
- `type=2` - 帖子类型
- `orderType=1` - 排序类型（1表示最新）
- `fileTypes=1` - 文件类型（1表示存档）
- `page={page}` - 页码占位符
- `limit={limit}` - 每页数量占位符

### 搜索接口
```
GET /api/post/list?type=2&orderType=1&fileTypes=1&page={page}&limit={limit}&title={query}
```

**额外参数：**
- `title={query}` - 搜索关键词占位符

## 响应数据映射

使用 JSONPath 从 API 响应中提取数据：

```json
{
  "results": "$.data.list",
  "id": "$.id",
  "title": "$.title",
  "description": "$.content",
  "author": "$.creator.nickname",
  "authorAvatar": "$.creator.headImg",
  "views": "$.views",
  "likes": "$.likeCount",
  "cover": "$.cover",
  "versions": "$.postVersions",
  "version": "$.version",
  "downloadUrl": "$.files[0].url",
  "fileName": "$.files[0].filename",
  "fileSize": "$.files[0].size",
  "total": "$.data.total",
  "totalPages": "$.data.totalPages"
}
```

## 实际请求示例

### 获取存档列表（第1页，每页10条）
```
GET https://m.suancaixianyu.cn/api/post/list?type=2&orderType=1&fileTypes=1&page=1&limit=10
```

### 搜索存档（搜索"音乐"，第1页，每页10条）
```
GET https://m.suancaixianyu.cn/api/post/list?type=2&orderType=1&fileTypes=1&page=1&limit=10&title=音乐
```

## 使用方法

1. 将配置文件放到启动器的 `.Survivalcraft/mod-sources/` 目录
2. 重启启动器
3. 在"存档管理"页面点击"下载存档"按钮
4. 在下载源选择器中选择"生存战争中文社区存档仓库"

## 与模组下载源的区别

### 模组下载源配置
- `fileTypes=5` - 模组类型
- URL: `/api/post/list?type=2&fileTypes=5`

### 存档下载源配置  
- `fileTypes=1` - 存档类型
- URL: `/api/post/list?type=2&orderType=1&fileTypes=1`

## 版本数量显示

在存档下载列表中，每个存档卡片会显示版本数量：
- 📦 0 个版本 - 没有上传文件
- 📦 1 个版本 - 有文件可下载
- 📦 3 个版本 - 多个版本可选

## 下载流程

1. 用户选择存档和版本
2. 点击下载按钮
3. 系统从 URL 下载文件（带正确扩展名）
4. 自动导入到选定的游戏版本
5. 刷新存档列表显示新导入的存档

## 注意事项

1. **文件格式**: 支持下载 `.scworld`、`.scword`、`.zip` 格式
2. **下载超时**: 30分钟超时设置
3. **临时文件**: 下载时使用临时文件，导入后自动清理
4. **ID 唯一性**: 使用 `suancaixianyu-saves-custom` 作为ID，避免与内置源冲突
5. **默认源**: 设置为非默认源，让用户自己选择

## 测试建议

1. 搜索常见关键词：建筑、音乐、地图、冒险等
2. 检查有文件的存档是否能正常下载
3. 检查无文件的存档（0版本）是否正确显示
4. 测试分页功能是否正常

## 故障排除

如果下载失败：
1. 检查网络连接
2. 确认选择的游戏版本存在
3. 查看浏览器控制台错误信息
4. 检查 API 响应格式是否变化

## 更新日志

- **v1.0.0** (2026-04-06)
  - 初始版本
  - 支持列表浏览和关键词搜索
  - 支持分页功能
  - 完整的响应数据映射
