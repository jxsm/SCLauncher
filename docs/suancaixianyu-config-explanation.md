# 生存战争中文社区配置说明

这是根据生存战争中文社区的实际API逻辑创建的v2格式配置文件。

## 配置解析

### 基本信息
```json
{
  "id": "suancaixianyu",           // 唯一标识符
  "type": "mods",                   // 资源类型：模组
  "name": "生存战争中文社区",       // 显示名称
  "description": "生存战争中文社区模组仓库",  // 描述
  "enabled": true,                 // 是否启用
  "isDefault": true                // 是否为默认源
}
```

### API配置

#### 基础URL
```json
"baseUrl": "https://m.suancaixianyu.cn"
```

#### 列表接口（endpoint）
用于获取模组列表，不使用搜索关键词：

```json
"endpoint": {
  "method": "GET",                          // HTTP方法
  "url": "/api/post/list?type=2&fileTypes=5&page={page}&limit={limit}",
  "headers": {
    "Accept": "application/json",
    "User-Agent": "SCLauncher/1.0"
  }
}
```

**URL解析：**
- `/api/post/list` - API路径
- `type=2` - 固定参数（2表示模组类型）
- `fileTypes=5` - 固定参数（5表示模组文件类型）
- `page={page}` - 页码占位符，会被替换为实际页码
- `limit={limit}` - 每页数量占位符，会被替换为实际数量

**实际请求示例：**
```
GET https://m.suancaixianyu.cn/api/post/list?type=2&fileTypes=5&page=1&limit=10
```

#### 搜索接口（search）
用于搜索模组，包含搜索关键词：

```json
"search": {
  "method": "GET",
  "url": "/api/post/list?type=2&fileTypes=5&page={page}&limit={limit}&title={query}",
  "headers": {
    "Accept": "application/json",
    "User-Agent": "SCLauncher/1.0"
  }
}
```

**URL解析：**
- 与列表接口基本相同
- 增加了 `title={query}` - 搜索关键词占位符

**实际请求示例：**
```
GET https://m.suancaixianyu.cn/api/post/list?type=2&fileTypes=5&page=1&limit=10&title=建筑
```

#### 分页配置
```json
"pagination": {
  "pageParam": "page",           // 页码参数名
  "limitParam": "limit",         // 每页数量参数名
  "searchParam": "title",        // 搜索关键词参数名
  "paramLocation": "url"         // 参数通过URL传递
}
```

### 响应数据映射

#### 实际API响应结构（示例）
```json
{
  "data": {
    "list": [
      {
        "id": "12345",
        "title": "超多模组整合包",
        "content": "<p>这是一个包含多个模组的整合包...</p>",
        "cover": "https://example.com/image.jpg",
        "views": 1523,
        "likeCount": 89,
        "creator": {
          "nickname": "作者名",
          "headImg": "https://example.com/avatar.jpg"
        },
        "postVersions": [
          {
            "version": "1.0.0",
            "files": [
              {
                "url": "https://example.com/mod.zip",
                "filename": "mod.zip",
                "size": "1048576",
                "icon": "https://example.com/icon.png"
              }
            ]
          }
        ]
      }
    ],
    "total": 100,
    "totalPages": 10
  }
}
```

#### JSONPath映射
```json
"responseMapping": {
  "results": "$.data.list",           // 结果列表路径
  "id": "$.id",                       // 模组ID
  "title": "$.title",                 // 模组标题
  "description": "$.content",         // 模组描述（HTML格式）
  "author": "$.creator.nickname",     // 作者昵称
  "authorAvatar": "$.creator.headImg", // 作者头像
  "views": "$.views",                 // 浏览次数
  "likes": "$.likeCount",             // 点赞数
  "cover": "$.cover",                 // 封面图片
  "versions": "$.postVersions",       // 版本列表
  "version": "$.version",             // 版本号
  "downloadUrl": "$.files[0].url",    // 下载链接（第一个文件）
  "fileName": "$.files[0].filename",  // 文件名
  "fileSize": "$.files[0].size",      // 文件大小
  "icon": "$.files[0].icon",          // 图标
  "total": "$.data.total",            // 总数
  "totalPages": "$.data.totalPages"   // 总页数
}
```

### 元数据
```json
"metadata": {
  "website": "https://m.suancaixianyu.cn",
  "author": "SuanCaiXianYu",
  "version": "1.0.0",
  "tags": ["中文", "社区", "模组", "官方"]
}
```

## 使用方法

1. 将配置文件复制到启动器的 `.Survivalcraft/mod-sources/` 目录
2. 重启启动器
3. 在设置页面启用该下载源

## 参数替换示例

### 列表请求（第1页，每页10条）
```
URL: https://m.suancaixianyu.cn/api/post/list?type=2&fileTypes=5&page=1&limit=10
```

### 搜索请求（搜索"建筑"，第2页，每页20条）
```
URL: https://m.suancaixianyu.cn/api/post/list?type=2&fileTypes=5&page=2&limit=20&title=建筑
```

## 与旧版本的区别

### 旧版本（v1格式）
```json
{
  "api": {
    "baseUrl": "https://m.suancaixianyu.cn",
    "searchPath": "/api/post/list",
    "searchParams": {
      "type": 2,
      "fileTypes": 5
    },
    "responseMapping": { ... }
  }
}
```

### 新版本（v2格式）
```json
{
  "api": {
    "baseUrl": "https://m.suancaixianyu.cn",
    "endpoint": {
      "method": "GET",
      "url": "/api/post/list?type=2&fileTypes=5&page={page}&limit={limit}"
    },
    "search": {
      "method": "GET",
      "url": "/api/post/list?type=2&fileTypes=5&page={page}&limit={limit}&title={query}"
    },
    "responseMapping": { ... }
  }
}
```

**新版本的优势：**
1. 更清晰地表达了HTTP方法和URL结构
2. 分离了列表和搜索接口
3. 支持URL参数替换符，更灵活
4. 可以自定义请求头
5. 可以配置分页参数的位置和名称

## 调试建议

如果配置不工作，可以：

1. 在浏览器中直接访问API URL测试：
   ```
   https://m.suancaixianyu.cn/api/post/list?type=2&fileTypes=5&page=1&limit=10
   ```

2. 检查响应格式是否与配置中的JSONPath匹配

3. 查看浏览器开发者工具的网络面板，确认实际请求

4. 检查启动器日志，查看是否有错误信息
