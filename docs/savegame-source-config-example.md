# 存档下载源配置说明

## 生存战争中文社区存档下载源

这是内置的存档下载源配置，使用生存战争中文社区的API。

### API 信息

- **基础URL**: `https://m.suancaixianyu.cn`
- **资源类型**: `savegames`

### 接口配置

#### 列表接口
```
GET /api/post/list?type=2&orderType=1&fileTypes=1&page={page}&limit={limit}
```

**参数说明：**
- `type=2` - 帖子类型
- `orderType=1` - 排序类型
- `fileTypes=1` - 文件类型（1表示存档，5表示模组）
- `page={page}` - 页码
- `limit={limit}` - 每页数量

#### 搜索接口
```
GET /api/post/list?type=2&orderType=1&fileTypes=1&page={page}&limit={limit}&title={query}
```

**额外参数：**
- `title={query}` - 搜索关键词

### 内置配置

在 `ModSourceManager.ts` 中已添加内置配置：

```typescript
{
  id: 'suancaixianyu-saves',
  type: 'savegames',
  name: '生存战争中文社区',
  description: '生存战争中文社区存档仓库',
  enabled: true,
  api: {
    baseUrl: 'https://m.suancaixianyu.cn',
    endpoint: {
      method: 'GET',
      url: '/api/post/list?type=2&orderType=1&fileTypes=1&page={page}&limit={limit}'
    },
    search: {
      method: 'GET',
      url: '/api/post/list?type=2&orderType=1&fileTypes=1&page={page}&limit={limit}&title={query}'
    }
  }
}
```

### 实际请求示例

**获取存档列表（第1页，每页10条）：**
```
GET https://m.suancaixianyu.cn/api/post/list?type=2&orderType=1&fileTypes=1&page=1&limit=10
```

**搜索存档（搜索"音乐"，第1页，每页10条）：**
```
GET https://m.suancaixianyu.cn/api/post/list?type=2&orderType=1&fileTypes=1&page=1&limit=10&title=音乐
```

### 使用方法

1. 打开启动器，进入"存档管理"页面
2. 点击"下载存档"按钮
3. 选择要下载到的游戏版本
4. 在搜索框中输入关键词或直接浏览列表
5. 点击存档项目查看详情
6. 选择版本并点击下载

### 技术说明

- **文件格式支持**: `.scworld`, `.scword`, `.zip`
- **下载方式**: 通过HTTP下载后自动导入到选定版本
- **临时文件**: 下载过程中使用临时文件，导入后自动清理
- **超时设置**: 30分钟下载超时
- **错误处理**: 完整的错误处理和用户提示

### 自定义配置

如果你想添加其他存档下载源，可以创建自定义配置文件：

1. 在 `.Survivalcraft/mod-sources/` 目录创建 JSON 文件
2. 设置 `"type": "savegames"`
3. 参考模组下载源的配置格式
4. 重启启动器加载新配置

### 注意事项

1. **ID 唯一性**: 自定义源的ID不要与内置源冲突
2. **文件类型**: 确保 `fileTypes` 参数正确（存档为1，模组为5）
3. **响应映射**: 使用与模组相同的响应映射格式
4. **安全限制**: 自定义源不能覆盖内置源
