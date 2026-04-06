# 生存战争中文社区下载源配置说明

本文档说明如何配置生存战争中文社区的模组下载源。

## 📋 配置文件

### 完整配置示例

```json
{
  "id": "suancaixianyu_test",
  "type": "mods",
  "name": "生存战争中文社区模组下载源测试",
  "description": "此为测试配置文件，可以参考这个配置文件配置自己的下载源",
  "enabled": true,
  "isDefault": false,
  "api": {
    "baseUrl": "https://m.suancaixianyu.cn",
    "endpoint": {
      "method": "GET",
      "url": "/api/post/list?type=2&fileTypes=5&page={page}&limit={limit}",
      "headers": {
        "Accept": "application/json",
        "User-Agent": "SCLauncher/1.0"
      },
      "pagination": {
        "pageParam": "page",
        "limitParam": "limit",
        "searchParam": "title",
        "paramLocation": "url"
      }
    },
    "search": {
      "method": "GET",
      "url": "/api/post/list?type=2&fileTypes=5&page={page}&limit={limit}&title={query}",
      "headers": {
        "Accept": "application/json",
        "User-Agent": "SCLauncher/1.0"
      },
      "pagination": {
        "pageParam": "page",
        "limitParam": "limit",
        "searchParam": "title",
        "paramLocation": "url"
      }
    },
    "responseMapping": {
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
      "icon": "$.files[0].icon",
      "total": "$.data.total",
      "totalPages": "$.data.totalPages"
    }
  },
  "metadata": {
    "website": "https://m.suancaixianyu.cn",
    "author": "SuanCaiXianYu",
    "version": "1.0.0",
    "tags": ["中文", "社区", "模组", "官方"]
  }
}
```

## 🔑 配置要点

### 1. 基本信息
- **id**: 唯一标识符，不要与内置源冲突
- **type**: 资源类型，固定为 `"mods"`
- **enabled**: 是否启用该下载源
- **isDefault**: 是否设为默认源

### 2. API 配置

#### 基础URL
```
https://m.suancaixianyu.cn
```

#### 列表接口
- **方法**: GET
- **路径**: `/api/post/list`
- **固定参数**:
  - `type=2` - 模组类型
  - `fileTypes=5` - 模组文件类型
- **分页参数**:
  - `page={page}` - 页码
  - `limit={limit}` - 每页数量

#### 搜索接口
- 与列表接口基本相同
- **额外参数**: `title={query}` - 搜索关键词

### 3. 参数替换符
配置中使用的占位符会被自动替换：
- `{page}` → 当前页码
- `{limit}` → 每页数量
- `{query}` → 搜索关键词

### 4. 响应数据映射
使用 JSONPath 从 API 响应中提取数据：
- `$.data.list` - 模组列表
- `$.id` - 模组ID
- `$.title` - 模组标题
- `$.creator.nickname` - 作者名称
- `$.files[0].url` - 下载链接

## 🚀 使用步骤

1. **创建配置文件**
   - 将上面的配置复制到一个 `.json` 文件中
   - 修改 `id` 为唯一值（避免与内置源冲突）

2. **放置配置文件**
   - 将文件放到 `.Survivalcraft/mod-sources/` 目录
   - 文件名如：`suancaixianyu-custom.json`

3. **重启启动器**
   - 重启后自动加载新配置
   - 在设置页面查看并启用该下载源

4. **测试功能**
   - 切换到模组下载页面
   - 选择该下载源
   - 测试列表浏览和搜索功能

## 📝 可运行的配置文件

查看完整的可运行配置文件：
- **[suancaixianyu-custom.json](suancaixianyu-custom.json)** - 完整的测试配置文件

## 🔍 实际请求示例

### 列表请求
```
GET https://m.suancaixianyu.cn/api/post/list?type=2&fileTypes=5&page=1&limit=10
```

### 搜索请求
```
GET https://m.suancaixianyu.cn/api/post/list?type=2&fileTypes=5&page=1&limit=10&title=建筑
```

## ⚠️ 注意事项

1. **ID 冲突**: 不要使用 `suancaixianyu` 作为 ID，这是内置源的 ID
2. **配置验证**: 可以在浏览器中直接访问 API URL 测试
3. **响应格式**: 确保 `responseMapping` 与实际 API 响应结构匹配
4. **日志调试**: 启动器控制台会显示加载日志，方便调试

## 🛠️ 自定义配置

参考此配置，你可以：
- 修改 `baseUrl` 配置其他 API 服务器
- 调整 `responseMapping` 适配不同的响应格式
- 修改固定参数以支持其他类型的资源
- 添加自定义请求头（如认证 token）

更多配置示例请查看：[mod-source-config-examples.md](mod-source-config-examples.md)
