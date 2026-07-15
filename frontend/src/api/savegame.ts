import { SaveGame, SaveRequiredMod } from '../types/savegame'
import * as AppBindings from '../../wailsjs/go/main/App'

// 获取存档列表
export async function GetSaveGames(versionId: string): Promise<SaveGame[]> {
  return await AppBindings.GetSaveGames(versionId)
}

// 删除存档
export async function DeleteSaveGame(versionId: string, saveId: string): Promise<void> {
  await AppBindings.DeleteSaveGame(versionId, saveId)
}

// 打开存档文件夹
export async function OpenSaveGameFolder(versionId: string, saveId: string): Promise<void> {
  await AppBindings.OpenSaveGameFolder(versionId, saveId)
}

// 重命名存档
export async function RenameSaveGame(versionId: string, saveId: string, newName: string): Promise<void> {
  await AppBindings.RenameSaveGame(versionId, saveId, newName)
}

// 导出存档
export async function ExportSaveGame(versionId: string, saveId: string): Promise<boolean> {
  return await AppBindings.ExportSaveGame(versionId, saveId)
}

// 导出存档为整合包
export async function ExportSaveGameAsModpack(versionId: string, saveId: string): Promise<boolean> {
  return await AppBindings.ExportSaveGameAsModpack(versionId, saveId)
}

// 导入存档
export async function ImportSaveGame(versionId: string, sourcePath: string): Promise<void> {
  await AppBindings.ImportSaveGame(versionId, sourcePath)
}

// 选择要导入的存档文件
export async function SelectSaveGameFile(): Promise<string> {
  return await AppBindings.SelectSaveGameFile()
}

// 预览存档信息
export async function PreviewSaveGame(sourcePath: string): Promise<SaveGame> {
  return await AppBindings.PreviewSaveGame(sourcePath)
}

// 下载存档，返回落地存档目录名（供进一步解析所需模组）
export async function DownloadSaveGameFromURL(downloadURL: string, versionId: string, fileName: string): Promise<string> {
  return await AppBindings.DownloadSaveGameFromURL(downloadURL, versionId, fileName)
}

// 获取已导入存档所需模组列表（解析 Project.xml/json 的 UsedMods）
export async function GetSaveRequiredMods(versionId: string, saveId: string): Promise<SaveRequiredMod[]> {
  return await AppBindings.GetSaveRequiredMods(versionId, saveId)
}

// 从 .scworld/.scword/.zip 归档预览所需模组（不导入）
export async function PreviewSaveRequiredMods(sourcePath: string): Promise<SaveRequiredMod[]> {
  return await AppBindings.PreviewSaveRequiredMods(sourcePath)
}
