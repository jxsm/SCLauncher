import { Version } from '../types/version'
import * as AppBindings from '../../wailsjs/go/main/App'

// 获取所有版本
export async function GetVersions(): Promise<Version[]> {
  return await AppBindings.GetVersions()
}

// 按类型获取版本
export async function GetVersionsByType(type: string): Promise<Version[]> {
  return await AppBindings.GetVersionsByType(type)
}

// 获取已安装的版本
export async function GetInstalledVersions(): Promise<Version[]> {
  return await AppBindings.GetInstalledVersions()
}

// 从清单文件获取版本列表
export async function FetchVersions(): Promise<Version[]> {
  return await AppBindings.FetchVersions()
}

// 下载版本
export async function DownloadVersion(versionId: string): Promise<void> {
  await AppBindings.DownloadVersion(versionId)
}

// 下载版本（使用自定义名称）
export async function DownloadVersionWithCustomName(versionId: string, customName: string): Promise<void> {
  await AppBindings.DownloadVersionWithCustomName(versionId, customName)
}

// 安装版本
export async function InstallVersion(versionId: string): Promise<void> {
  await AppBindings.InstallVersion(versionId)
}

// 删除版本
export async function DeleteVersion(versionId: string): Promise<void> {
  await AppBindings.DeleteVersion(versionId)
}

// 重命名版本
export async function RenameVersion(versionId: string, newName: string): Promise<void> {
  await AppBindings.RenameVersion(versionId, newName)
}

// 取消下载
export async function CancelDownload(versionId: string): Promise<void> {
  await AppBindings.CancelDownload(versionId)
}

// 设置主要版本
export async function SetPrimaryVersion(versionId: string): Promise<void> {
  await AppBindings.SetPrimaryVersion(versionId)
}

// 检查版本是否存在
export async function VersionExists(versionId: string): Promise<boolean> {
  return await AppBindings.VersionExists(versionId)
}

// 格式化文件大小
export async function FormatSize(bytes: number): Promise<string> {
  return await AppBindings.FormatSize(bytes)
}

// 获取主要版本
export async function GetPrimaryVersion(): Promise<Version | null> {
  try {
    const result = await AppBindings.GetPrimaryVersion()
    if (!result) return null
    return result as Version
  } catch (e) {
    return null
  }
}

// 打开版本文件夹
export async function OpenVersionFolder(versionId: string): Promise<void> {
  await AppBindings.OpenVersionFolder(versionId)
}

// 打开版本的mods文件夹
export async function OpenVersionModsFolder(versionId: string): Promise<void> {
  await AppBindings.OpenVersionModsFolder(versionId)
}
