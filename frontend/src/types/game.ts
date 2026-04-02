// 游戏状态
export type GameStatus = 'stopped' | 'running' | 'crashed'

// 游戏进程信息
export interface GameProcessInfo {
  pid: number
  versionId: string
  startTime: string
  executable: string
}
