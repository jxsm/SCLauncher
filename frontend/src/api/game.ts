import { GameStatus, GameProcessInfo } from '../types/game'
import * as AppBindings from '../../wailsjs/go/main/App'

// 启动游戏
export async function LaunchGame(versionId: string): Promise<void> {
  await AppBindings.LaunchGame(versionId)
}

// 停止游戏
export async function StopGame(): Promise<void> {
  await AppBindings.StopGame()
}

// 获取游戏状态
export async function GetGameStatus(): Promise<GameStatus> {
  const status = await AppBindings.GetGameStatus()
  return status as GameStatus
}

// 获取游戏进程信息
export async function GetGameProcessInfo(): Promise<GameProcessInfo | null> {
  const info = await AppBindings.GetGameProcessInfo()
  return info as GameProcessInfo | null
}
