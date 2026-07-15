// 皮肤在线下载API
import type { SkinApiResponse, SkinApiItem } from '../types/skin-api'
import { HttpRequest } from './http'

const API_BASE_URL = 'https://m.suancaixianyu.cn/api/post/list'

/**
 * 获取皮肤列表
 */
export async function getSkinList(page: number = 1, limit: number = 10): Promise<SkinApiResponse> {
  const url = `${API_BASE_URL}?type=2&orderType=3&fileTypes=4&page=${page}&limit=${limit}`
  // 通过后端代理发起请求（绕过浏览器 CORS 限制）
  const text = await HttpRequest(url)
  const data = JSON.parse(text)
  return data
}

/**
 * 搜索皮肤
 */
export async function searchSkins(title: string, page: number = 1, limit: number = 10): Promise<SkinApiResponse> {
  const url = `${API_BASE_URL}?type=2&orderType=3&fileTypes=4&title=${encodeURIComponent(title)}&page=${page}&limit=${limit}`
  // 通过后端代理发起请求（绕过浏览器 CORS 限制）
  const text = await HttpRequest(url)
  const data = JSON.parse(text)
  return data
}

/**
 * 转换API数据为前端使用的格式
 */
export function transformSkinApiData(apiData: SkinApiItem): any {
  // 获取第一个版本的所有文件
  const firstVersion = apiData.postVersions[0]
  if (!firstVersion || !firstVersion.files || firstVersion.files.length === 0) {
    return null
  }

  return {
    id: apiData.id.toString(),
    title: apiData.title,
    description: apiData.content,
    author: apiData.creator.nickname,
    authorAvatar: apiData.creator.headImg,
    views: apiData.views,
    likes: parseInt(apiData.likeCount) || 0,
    downloadUrl: firstVersion.files[0].url,
    icon: firstVersion.files[0].icon,
    fileName: firstVersion.files[0].filename,
    fileSize: firstVersion.files[0].size,
    // 保存所有文件信息（一个皮肤包可能包含多个皮肤文件）
    files: firstVersion.files.map(file => ({
      url: file.url,
      icon: file.icon,
      filename: file.filename,
      size: file.size
    }))
  }
}
