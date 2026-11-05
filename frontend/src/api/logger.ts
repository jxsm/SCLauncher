import * as AppBindings from '../../wailsjs/go/main/App'

// 打开启动器日志目录（.Survivalcraft/Logs）
export async function OpenLogsFolder(): Promise<void> {
  await AppBindings.OpenLogsFolder()
}

// 前端日志上报（由 utils/feLogger.ts 统一调用，一般无需直接使用）
export async function WriteFrontendLog(level: string, message: string): Promise<void> {
  await AppBindings.WriteFrontendLog(level, message)
}
