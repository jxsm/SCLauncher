// 模组信息
export interface Mod {
  id: string          // 模组 ID（使用文件名）
  versionId: string   // 所属版本 ID
  name: string        // 模组名称（文件名）
  fileName: string    // 文件名
  enabled: boolean    // 是否启用
  size: number        // 文件大小
  installDate: string // 安装日期
}
