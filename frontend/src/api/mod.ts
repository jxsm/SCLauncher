import { Mod } from '../types/mod'
import * as AppBindings from '../../wailsjs/go/main/App'

// 选择模组文件
export async function SelectModFile(): Promise<string | null> {
  return await AppBindings.SelectModFile()
}

// 获取模组列表
export async function GetMods(versionId: string): Promise<Mod[]> {
  return await AppBindings.GetMods(versionId)
}

// 导入模组
export async function ImportMod(versionId: string, sourcePath: string): Promise<void> {
  await AppBindings.ImportMod(versionId, sourcePath)
}

// 切换模组状态
export async function ToggleMod(versionId: string, modId: string, enabled: boolean): Promise<void> {
  await AppBindings.ToggleMod(versionId, modId, enabled)
}

// 删除模组
export async function DeleteMod(versionId: string, modId: string): Promise<void> {
  await AppBindings.DeleteMod(versionId, modId)
}
