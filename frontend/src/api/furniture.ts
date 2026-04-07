import { Furniture } from '../types/furniture'
import * as AppBindings from '../../wailsjs/go/main/App'

// 获取家具列表，如果文件夹不存在返回 null
export async function GetFurnitures(versionId: string): Promise<Furniture[] | null> {
  return await AppBindings.GetFurnitures(versionId)
}

// 删除家具
export async function DeleteFurniture(versionId: string, furnitureId: string): Promise<void> {
  await AppBindings.DeleteFurniture(versionId, furnitureId)
}

// 打开家具文件夹
export async function OpenFurnitureFolder(versionId: string): Promise<void> {
  await AppBindings.OpenFurnitureFolder(versionId)
}

// 重命名家具
export async function RenameFurniture(versionId: string, furnitureId: string, newName: string): Promise<void> {
  await AppBindings.RenameFurniture(versionId, furnitureId, newName)
}

// 选择要导入的家具文件
export async function SelectFurnitureFile(): Promise<string> {
  return await AppBindings.SelectFurnitureFile()
}

// 导入家具
export async function ImportFurniture(versionId: string, sourcePath: string): Promise<void> {
  await AppBindings.ImportFurniture(versionId, sourcePath)
}

// 下载家具
export async function DownloadFurnitureFromURL(downloadURL: string, versionId: string, fileName: string): Promise<void> {
  await AppBindings.DownloadFurnitureFromURL(downloadURL, versionId, fileName)
}
