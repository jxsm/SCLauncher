import { AppConfig } from '../types/config'
import * as AppBindings from '../../wailsjs/go/main/App'

// 获取配置
export async function GetConfig(): Promise<AppConfig> {
  const config = await AppBindings.GetConfig()
  return config as AppConfig
}
