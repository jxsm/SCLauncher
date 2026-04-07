import { Texture } from '../types/texture'
import * as AppBindings from '../../wailsjs/go/main/App'

// 获取材质列表，如果文件夹不存在返回 null
export async function GetTextures(versionId: string): Promise<Texture[] | null> {
  return await AppBindings.GetTextures(versionId)
}

// 删除材质
export async function DeleteTexture(versionId: string, textureId: string): Promise<void> {
  await AppBindings.DeleteTexture(versionId, textureId)
}

// 打开材质文件夹
export async function OpenTextureFolder(versionId: string): Promise<void> {
  await AppBindings.OpenTextureFolder(versionId)
}

// 重命名材质
export async function RenameTexture(versionId: string, textureId: string, newName: string): Promise<void> {
  await AppBindings.RenameTexture(versionId, textureId, newName)
}

// 选择要导入的材质文件
export async function SelectTextureFile(): Promise<string> {
  return await AppBindings.SelectTextureFile()
}

// 导入材质
export async function ImportTexture(versionId: string, sourcePath: string): Promise<void> {
  await AppBindings.ImportTexture(versionId, sourcePath)
}

// 下载材质
export async function DownloadTextureFromURL(downloadURL: string, versionId: string, fileName: string): Promise<void> {
  await AppBindings.DownloadTextureFromURL(downloadURL, versionId, fileName)
}
