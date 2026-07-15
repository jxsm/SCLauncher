// 模组依赖（来自 modinfo.json 的 Dependencies）
export interface Dependency {
  packageName: string            // 依赖模组的包名
  versionRange: string           // 版本范围约束（空字符串表示任意）
  displayName?: string           // 显示名（可选）
}

// 玩法影响等级
export type GameplayImpactLevel = '' | 'Cosmetic' | 'Assist' | 'Turbo' | 'Break' | 'Godmode'

// 模组信息（解析自 modinfo.json）
export interface ModInfo {
  name: string                   // 模组名称
  version: string                // 模组版本
  apiVersion: string             // 适配的 API 版本（< 1.8 游戏会警告）
  packageName: string            // 包名，唯一标识
  description: string            // 描述
  scVersion: string              // 适配的游戏版本（无实际作用）
  loadOrder: number              // 加载顺序，越小越先加载
  nonPersistentMod: boolean      // 非持久化（不写入存档）
  gameplayImpactLevel: string    // 玩法影响等级：Cosmetic/Assist/Turbo/Break/Godmode
  link: string                   // 模组链接
  author: string                 // 作者
  dependencies: Dependency[]     // 依赖的其他模组
}

// 模组信息
export interface Mod {
  id: string          // 模组 ID（使用文件名）
  versionId: string   // 所属版本 ID
  name: string        // 模组名称（文件名）
  fileName: string    // 文件名
  enabled: boolean    // 是否启用
  size: number        // 文件大小
  installDate: string // 安装日期
  modInfo?: ModInfo | null // 解析自 modinfo.json 的模组信息（解析失败时为 null）
}
