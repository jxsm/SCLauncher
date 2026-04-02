import { AppConfig } from '../types/config'
import * as AppBindings from '../../wailsjs/go/main/App'

// 获取配置
export async function GetConfig(): Promise<AppConfig> {
  const config = await AppBindings.GetConfig()
  return config as AppConfig
}

// 设置清单文件 URL
export async function SetManifestURL(url: string): Promise<void> {
  await AppBindings.SetManifestURL(url)
}

// 设置最大并发下载数
export async function SetMaxConcurrent(max: number): Promise<void> {
  await AppBindings.SetMaxConcurrent(max)
}
