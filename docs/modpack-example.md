# 生存战争整合包配置说明

## 概述

生存战争整合包是一个包含了游戏、模组、配置文件的打包方案，通过一个配置文件（`manifest.jsonc`）来描述整合包的所有信息。

## 配置文件结构

### 基本信息

```jsonc
{
  // 固定，表示是生存战争的整合包（必须）
  "manifestType": "SurvivalcraftModpack",

  // 清单版本号，告诉启动器使用哪个版本的解析器解析（当前只有 0.1，请填写 0.1）（必填）
  "manifestVersion": 0.1,

  // 整合包名称（必填）
  "name": "整合包名称",

  // 整合包版本（必填）
  "version": "1.0.0",

  // 作者（必填）
  "author": "作者名称",

  // 简短描述（可选）
  "description": "描述",

  // 图标（可选）
  "icon": "icon.png",

  // 创建时间（可选）
  "created": "2025-01-01",

  // 更新日志（可选）
  "changelog": "首次发布"
}
```

### 游戏核心配置

```jsonc
"survivalcraft": {
  "version": {
    //（可选）[默认 false] 是否手动选择版本
    // true 表示用户需要手动指定一个已安装版本
    // false 表示自动安装指定的版本（安卓端不支持自动安装，只能手动选择）
    "manual": false,

    "android": {
      //（可选）如果不填用户在安装时会显示"该整合包不支持android"
      //（必填）填写整合包的版本号
      "version": "2.4:api-1.8.2.3",

      // 如果整合包支持安卓，则此项必填
      // 用于检测用户是否已经安装了该版本
      // 如果不存在则会要求用户先安装该版本
      "apkPackageName": "com.candy.survivalcraftAPI1_8",

      //（可选）下载 URL
      // 如果填写了此项则下载时会使用这个链接进行下载
      //（!!如果填写了该项目在安装时会提示用户注意风险）
      "path": "URL"
    },

    "windows": {
      //（可选）如果不填用户在安装时会显示"该整合包不支持windows"
      "version": "2.4:api-1.8.2.3",

      //（可选）下载 URL
      // 如果填写了此项则下载时会使用这个链接进行下载
      //（!!如果填写了该项目在安装时会提示用户注意风险）
      "path": "URL"
    }
  },

  "versionList": {
    //（可选）默认使用内置的
    // 安卓生存战争下载列表清单（可选）默认使用启动器所选的
    "android": "URL",

    // Windows 生存战争下载列表（可选）默认使用启动器所选的
    "windows": "URL"
    // 版本列表中必须包含 path，用于下载游戏
  }
}
```

#### 版本号命名规则

格式：`大版本号:版本类型-版本号`

- **大版本号**：如 `2.4` 表示 2.4 的大版本
- **版本类型**：
  - `api`：插件版
  - `net`：联机版
- **版本号**：如 `1.8.2.3`

示例：
- `2.4:api-1.8.2.3` - 2.4 大版本，插件版，1.8.2.3 版本
- `2.4:net-26.02.05` - 2.4 大版本，联机版，26.02.05 版本

#### 特殊情况：完整包

完整包包含所有文件（游戏、Mod、配置）。

此时对应的平台应填写为：`大版本号:carry/文件夹名(或apk名)`

并且在整合包的根目录下创建一个名为 `carry` 的文件夹，里面需要存放好相应的文件夹或 apk 文件。

示例：
```jsonc
"windows": "2.4:carry/1.8.2.3.zip",
"android": "2.4:carry/1.8.2.3.apk"
```

### 模组列表

```jsonc
"mods": [
  // 模组列表（必填）（可以为空列表）
  {
    // 模组 ID（必填）
    // 该 ID 是在 SCBBS 上发布的文章的 ID
    // 示例：https://www.scbbs.top/#/postDetails/1630
    "projectID": 1630,

    // 模组版本（必填）
    // 资源下载中显示的版本号
    // 可以是 "lastest" 则下载最新的哪个版本
    "version": "1.1.0.2",

    // 模组名（可选，方便调试）
    // 资源下载中显示的模组名
    "name": "模组名",

    // 是否必须（必填）
    // 如果为 false 则如果在下载失败的情况下不会中断整合包的下载
    "required": true,

    // 模组下载地址（可选）
    // 如果填写了该地址则下载该地址的模组，否则下载 SCBBS 上的模组
    //（!!如果填写了该项目在安装时会提示用户注意风险）
    "path": "URL"
  }
]
```

### 模组存放路径

```jsonc
// 模组存放路径（可选）默认为 /Mods
// 请注意，如果是联机版请填写 "/NetMods"
"modPath": "/Mods"
```

### 自定义覆盖文件

```jsonc
// 自定义覆盖文件（可选）
// 如果有自定义覆盖文件则必须填写
// 该文件必须放在整合包根目录下
"overrides": "overrides"

// 这个的意思就是，你在整合包根目录下创建一个名为 overrides 的文件夹
// 里面存放好你要覆盖的文件，启动器会自动覆盖到游戏目录下
```

### 校验方式

```jsonc
// 校验方式（可选）默认 sha256
// 用于校验整合包的完整性
"checksum": "sha256"
```

## 完整示例

```jsonc
{
  "manifestType": "SurvivalcraftModpack",
  "manifestVersion": 0.1,
  "name": "示例整合包",
  "version": "1.0.0",
  "author": "作者名称",
  "description": "这是一个示例整合包",
  "icon": "icon.png",
  "created": "2025-01-01",
  "changelog": "首次发布",
  "survivalcraft": {
    "version": {
      "manual": false,
      "android": {
        "version": "2.4:api-1.8.2.3",
        "apkPackageName": "com.candy.survivalcraftAPI1_8",
        "path": "https://example.com/sc.apk"
      },
      "windows": {
        "version": "2.4:api-1.8.2.3",
        "path": "https://example.com/sc.zip"
      }
    },
    "versionList": {
      "android": "https://example.com/android-version-list.json",
      "windows": "https://example.com/windows-version-list.json"
    }
  },
  "mods": [
    {
      "projectID": 1630,
      "version": "1.1.0.2",
      "name": "示例模组",
      "required": true,
      "path": "https://example.com/mod.zip"
    }
  ],
  "modPath": "/Mods",
  "overrides": "overrides",
  "checksum": "sha256"
}
```

## 注意事项

1. **必填字段**：所有标记为"必填"的字段都必须填写，否则整合包可能无法正常工作
2. **自定义下载链接**：如果在 `path` 字段中填写了自定义下载链接，启动器在安装时会提示用户注意风险
3. **安卓平台限制**：安卓端不支持自动安装游戏，只能手动选择已安装的版本
4. **模组路径**：如果是联机版，请将 `modPath` 设置为 `/NetMods`
5. **覆盖文件**：如果使用 `overrides` 功能，需要在整合包根目录下创建相应的文件夹

示例文件路径列表
```
./manifest.jsonc
./icon.png
./overrides/
./overrides/doc/Worlds
./overrides/Mods
...
```
