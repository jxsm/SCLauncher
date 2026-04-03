import { Skin } from '../types/skin'
import * as AppBindings from '../../wailsjs/go/main/App'

// 选择皮肤文件
export async function SelectSkinFile(): Promise<string | null> {
  return await AppBindings.SelectSkinFile()
}

// 获取皮肤列表
export async function GetSkins(): Promise<Skin[]> {
  return await AppBindings.GetSkins()
}

// 导入皮肤
export async function ImportSkin(sourcePath: string): Promise<void> {
  await AppBindings.ImportSkin(sourcePath)
}

// 删除皮肤
export async function DeleteSkin(fileName: string): Promise<void> {
  await AppBindings.DeleteSkin(fileName)
}

// 同步皮肤到游戏目录
export async function SyncSkinsToGame(versionId: string): Promise<void> {
  await AppBindings.SyncSkinsToGame(versionId)
}

// 获取皮肤图片的base64编码
export async function GetSkinImage(fileName: string): Promise<string> {
  return await AppBindings.GetSkinImage(fileName)
}
