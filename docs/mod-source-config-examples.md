# 模组下载源配置示例

本文档展示如何使用新的 v2 配置格式来配置不同类型的模组下载源。

## 🚀 可运行的配置示例

### 生存战争中文社区配置
- **[suancaixianyu-custom.json](suancaixianyu-custom.json)** - 完整的、可立即使用的配置文件
- 详细说明请查看：[suancaixianyu-config-explanation.md](suancaixianyu-config-explanation.md)

### 其他示例文件
- **[example-mod-source.json](example-mod-source.json)** - 通用下载源配置模板
- **[test-mod-source.json](test-mod-source.json)** - 测试用配置文件

## 配置文件位置

配置文件应该放在启动器目录下的 `.Survivalcraft/mod-sources/` 文件夹中，每个下载源一个 JSON 文件。

例如：
- `.Survivalcraft/mod-sources/suancaixianyu.json`
- `.Survivalcraft/mod-sources/my-custom-source.json`

## 配置格式说明

### 基本结构

```json
{
  "id": "unique-source-id",
  "type": "mods",
  "name": "下载源名称",
  "description": "下载源描述",
  "enabled": true,
  "isDefault": false,
  "api": {
    "baseUrl": "https://api.example.com",
    "endpoint": { ... },
    "responseMapping": { ... }
  }
}
```

### 字段说明

- `id`: 唯一标识符（必需）
- `type`: 资源类型，可选值：`mods`, `savegames`, `furniture`, `textures`, `skins`
- `name`: 显示名称
- `description`: 描述信息
- `enabled`: 是否启用
- `isDefault`: 是否为默认源（只能有一个默认源）
- `api`: API 配置对象
  - `baseUrl`: API 基础 URL
  - `endpoint`: 端点配置（当列表和搜索使用相同接口时）
  - `list`: 列表接口配置（可选）
  - `search`: 搜索接口配置（可选）
  - `responseMapping`: 响应数据映射配置

## 示例 1: GET 请求 - URL 参数替换

适用于使用 GET 请求，参数通过 URL 传递的 API。

```json
{
  "id": "example-get-source",
  "type": "mods",
  "name": "GET 示例源",
  "description": "使用 GET 请求和 URL 参数替换的示例",
  "enabled": true,
  "isDefault": false,
  "api": {
    "baseUrl": "https://api.example.com",
    "endpoint": {
      "method": "GET",
      "url": "/v1/mods?page={page}&limit={limit}",
      "headers": {
        "User-Agent": "MyLauncher/1.0"
      },
      "pagination": {
        "pageParam": "page",
        "limitParam": "limit",
        "searchParam": "q",
        "paramLocation": "url"
      }
    },
    "search": {
      "method": "GET",
      "url": "/v1/mods?page={page}&limit={limit}&q={query}",
      "headers": {},
      "pagination": {
        "pageParam": "page",
        "limitParam": "limit",
        "searchParam": "q",
        "paramLocation": "url"
      }
    },
    "responseMapping": {
      "results": "$.data.mods",
      "id": "$.id",
      "title": "$.name",
      "description": "$.description",
      "author": "$.author.name",
      "authorAvatar": "$.author.avatar",
      "views": "$.stats.views",
      "likes": "$.stats.likes",
      "cover": "$.coverImage",
      "versions": "$.versions",
      "version": "$.version",
      "downloadUrl": "$.downloadUrl",
      "fileName": "$.fileName",
      "fileSize": "$.fileSize",
      "total": "$.data.total",
      "totalPages": "$.data.totalPages"
    }
  }
}
```

## 示例 2: POST 请求 - JSON 请求体

适用于使用 POST 请求，参数通过 JSON 请求体传递的 API。

```json
{
  "id": "example-post-json-source",
  "type": "mods",
  "name": "POST JSON 示例源",
  "description": "使用 POST 请求和 JSON 请求体的示例",
  "enabled": true,
  "isDefault": false,
  "api": {
    "baseUrl": "https://api.example.com",
    "endpoint": {
      "method": "POST",
      "url": "/v1/mods/list",
      "headers": {
        "Content-Type": "application/json",
        "Authorization": "Bearer YOUR_API_TOKEN"
      },
      "body": {
        "type": "json",
        "template": {
          "filters": {
            "category": "mods"
          }
        }
      },
      "pagination": {
        "pageParam": "page",
        "limitParam": "pageSize",
        "searchParam": "keyword",
        "paramLocation": "body"
      }
    },
    "search": {
      "method": "POST",
      "url": "/v1/mods/search",
      "headers": {
        "Content-Type": "application/json"
      },
      "body": {
        "type": "json",
        "template": {
          "filters": {
            "category": "mods"
          }
        }
      },
      "pagination": {
        "pageParam": "page",
        "limitParam": "pageSize",
        "searchParam": "keyword",
        "paramLocation": "body"
      }
    },
    "responseMapping": {
      "results": "$.results",
      "id": "$.id",
      "title": "$.title",
      "description": "$.description",
      "author": "$.author",
      "views": "$.viewCount",
      "likes": "$.likeCount",
      "versions": "$.versions",
      "version": "$.version",
      "downloadUrl": "$.downloadLink",
      "fileName": "$.filename",
      "fileSize": "$.size",
      "total": "$.pagination.total",
      "totalPages": "$.pagination.totalPages"
    }
  }
}
```

## 示例 3: POST 请求 - 表单数据

适用于使用 POST 请求，参数通过表单数据传递的 API。

```json
{
  "id": "example-post-form-source",
  "type": "mods",
  "name": "POST 表单示例源",
  "description": "使用 POST 请求和表单数据的示例",
  "enabled": true,
  "isDefault": false,
  "api": {
    "baseUrl": "https://api.example.com",
    "endpoint": {
      "method": "POST",
      "url": "/api/mods.php",
      "headers": {},
      "body": {
        "type": "form-urlencoded",
        "template": {
          "action": "list",
          "format": "json"
        }
      },
      "pagination": {
        "pageParam": "pagenum",
        "limitParam": "perpage",
        "searchParam": "search",
        "paramLocation": "body"
      }
    },
    "responseMapping": {
      "results": "$.mods",
      "id": "$.mod_id",
      "title": "$.mod_name",
      "description": "$.mod_desc",
      "author": "$.author",
      "versions": "$.files",
      "version": "$.version",
      "downloadUrl": "$.file_url",
      "fileName": "$.file_name",
      "fileSize": "$.file_size"
    }
  }
}
```

## 示例 4: 复杂的 URL 参数配置

展示如何使用各种 URL 参数替换符。

```json
{
  "id": "example-complex-url",
  "type": "mods",
  "name": "复杂 URL 示例",
  "description": "展示各种 URL 参数替换符的使用",
  "enabled": true,
  "isDefault": false,
  "api": {
    "baseUrl": "https://api.example.com/v2",
    "endpoint": {
      "method": "GET",
      "url": "/mods?category=mods&sort=date&order=desc&page={page}&size={limit}",
      "headers": {
        "Accept": "application/json"
      },
      "pagination": {
        "pageParam": "page",
        "limitParam": "size",
        "searchParam": "q",
        "paramLocation": "url"
      }
    },
    "search": {
      "method": "GET",
      "url": "/search?type=mods&q={query}&page={page}&size={limit}&timestamp={timestamp}",
      "headers": {},
      "pagination": {
        "pageParam": "page",
        "limitParam": "size",
        "searchParam": "q",
        "paramLocation": "url"
      }
    },
    "responseMapping": {
      "results": "$.data.items",
      "id": "$.id",
      "title": "$.title",
      "description": "$.shortDescription",
      "author": "$.uploader.username",
      "authorAvatar": "$.uploader.avatarUrl",
      "views": "$.stats.views",
      "likes": "$.stats.thumbsUp",
      "cover": "$.thumbnail.url",
      "versions": "$.versions",
      "version": "$.latestVersion",
      "downloadUrl": "$.versions[0].downloadUrl",
      "fileName": "$.versions[0].filename",
      "fileSize": "$.versions[0].sizeInBytes",
      "total": "$.data.totalResults",
      "totalPages": "$.data.totalPages"
    }
  }
}
```

## URL 参数替换符说明

在 `url` 和 POST 请求的 `template` 中，可以使用以下替换符：

- `{page}` - 当前页码
- `{limit}` - 每页数量
- `{query}` - 搜索关键词（自动 URL 编码）
- `{timestamp}` - 当前时间戳（毫秒）
- `{自定义键名}` - 从 `filters` 中获取对应的值

例如：
- `/api/mods?page={page}&limit={limit}` → `/api/mods?page=1&limit=10`
- `/search?q={query}` → `/search?q=关键词`
- `/api/data?token={token}` → 需要在调用时传入 `filters: { token: 'xxx' }`

## JSONPath 说明

`responseMapping` 中的字段使用 JSONPath 表达式来从响应中提取数据：

- `$.fieldName` - 根级字段
- `$.data.list` - 嵌套字段
- `$.data.items[0]` - 数组第一项
- `$.creator.nickname` - 对象嵌套字段

## 旧版本配置兼容性

系统仍然支持旧版本的配置格式（v1），会自动检测并兼容。

**v1 格式示例：**
```json
{
  "id": "old-source",
  "type": "mods",
  "name": "旧格式源",
  "api": {
    "baseUrl": "https://api.example.com",
    "searchPath": "/api/mods",
    "searchParams": {
      "type": "mod"
    },
    "headers": {},
    "responseMapping": { ... }
  }
}
```

## 调试建议

1. 使用浏览器开发者工具检查网络请求
2. 确认 API 的实际响应格式
3. 使用在线 JSONPath 工具测试表达式
4. 检查 API 文档确认请求方法、参数和响应格式

## 完整示例：蒜菜闲鱼源

这是内置的蒜菜闲鱼源配置示例：

```json
{
  "id": "suancaixianyu",
  "type": "mods",
  "name": "生存战争中文社区",
  "description": "生存战争中文社区模组仓库",
  "enabled": true,
  "isDefault": true,
  "api": {
    "baseUrl": "https://m.suancaixianyu.cn",
    "endpoint": {
      "method": "GET",
      "url": "/api/post/list?type=2&fileTypes=5&page={page}&limit={limit}",
      "headers": {},
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
      "headers": {},
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
    "tags": ["中文", "社区", "模组"]
  }
}
```
