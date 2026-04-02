// 版本类型
export type VersionType = 'api' | 'net' | 'original'

// 版本信息
export interface Version {
  id: string                // 唯一 ID
  versionType: string       // 版本类型（使用 string 以兼容 Wails 生成的类型）
  gameVersion: string       // 游戏版本
  subVersion: string        // 子版本
  name: string             // 显示名称
  size: number             // 文件大小（字节）
  downloadUrl: string      // 下载地址
  checksum: string         // SHA256 校验和
  fileFormat: string       // 文件格式
  illustrate: string       // 说明
  releaseDate: any         // 发布日期（Wails 生成的类型）
  installed?: boolean      // 是否已安装
  isPrimary?: boolean      // 是否为主要版本
  localPath?: string       // 本地路径
}

// 下载进度
export interface DownloadProgress {
  versionId: string
  downloaded: number
  total: number
  speed: number
}

// 安装进度
export interface InstallProgress {
  versionId: string
  current: number
  total: number
}
