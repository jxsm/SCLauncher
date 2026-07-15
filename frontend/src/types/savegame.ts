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

// 存档所需模组（解析自 Project.xml/json 的 UsedMods）
export interface SaveRequiredMod {
  packageName: string  // 包名，唯一标识（用于依赖解析）
  version: string      // 存档记录的该模组版本（当版本约束用，裸版本视为 ">="）
  name: string         // 模组名称（显示用）
  author: string       // 作者
  link: string         // 链接
}
