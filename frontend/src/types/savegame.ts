// 存档信息
export interface SaveGame {
  id: string           // 存档 ID（目录名）
  name: string         // 世界名称
  gameVersion: string  // 游戏版本
  gameMode: string     // 游戏模式
  lastModified: string // 最后修改时间
  isAutoSave: boolean  // 是否自动保存
  projectPath: string  // Project文件路径
  worldPath: string    // 存档目录路径
  isImported: boolean  // 是否来自导入的版本
}
