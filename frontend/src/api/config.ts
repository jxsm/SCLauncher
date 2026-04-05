import { AppConfig } from '../types/config'
import * as AppBindings from '../../wailsjs/go/main/App'

// 获取应用信息
export async function GetAppInfo(): Promise<any> {
  return await AppBindings.GetAppInfo()
}

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

// 设置语言
export async function SetLanguage(lang: string): Promise<void> {
  await AppBindings.SetLanguage(lang)
}

// 自动检测系统语言（首次启动时调用）
export async function AutoDetectLanguage(): Promise<void> {
  await AppBindings.AutoDetectLanguage()
}

// 检查更新
export async function CheckUpdate(): Promise<any> {
  return await AppBindings.CheckUpdate()
}

// 强制检查更新（忽略不再提醒设置）
export async function CheckUpdateForce(): Promise<any> {
  return await AppBindings.CheckUpdateWithForce(true)
}

// 设置不再提醒更新
export async function SetUpdateRemindDisabled(disabled: boolean): Promise<void> {
  await AppBindings.SetUpdateRemindDisabled(disabled)
}

// 选择背景图片文件
export async function SelectBackgroundFile(): Promise<string> {
  return await AppBindings.SelectBackgroundFile()
}

// 设置背景图片
export async function SetBackground(sourcePath: string): Promise<string> {
  return await AppBindings.SetBackground(sourcePath)
}

// 清除背景图片
export async function ClearBackground(): Promise<void> {
  await AppBindings.ClearBackground()
}

// 获取背景图片路径
export async function GetBackgroundImage(): Promise<string> {
  return await AppBindings.GetBackgroundImage()
}

// 检查是否设置了背景图片
export async function HasBackground(): Promise<boolean> {
  return await AppBindings.HasBackground()
}

// 获取背景图片的base64编码
export async function GetBackgroundImageBase64(): Promise<string> {
  return await AppBindings.GetBackgroundImageBase64()
}
