/**
 * scbbs 社区游戏版本目录（GET /api/version/list）
 * 返回 {code, data: [{id, name, type, version, status}]}，
 * id 即帖子 postVersions[].gameVersionIds 引用的适配游戏版本 ID（对齐 ScbbsPlus BuildVersionNameMap）。
 * 模块级缓存，整个会话只拉一次；失败时清空 pending 允许重试。
 */
import { HttpRequest } from './http'

const COMMUNITY_API_BASE = 'https://m.suancaixianyu.cn'

let cachedNames: Map<number, string> | null = null
let pending: Promise<Map<number, string>> | null = null

/** 获取社区游戏版本目录（id → 版本名，如 12 → "2.4"） */
export async function getCommunityGameVersionNames(): Promise<Map<number, string>> {
  if (cachedNames) return cachedNames
  if (!pending) {
    pending = HttpRequest(`${COMMUNITY_API_BASE}/api/version/list`)
      .then(text => {
        const data = JSON.parse(text)
        const list: any[] = Array.isArray(data?.data) ? data.data : []
        const map = new Map<number, string>()
        for (const item of list) {
          const id = Number(item?.id)
          const name = String(item?.name || '').trim()
          if (id > 0 && name) {
            map.set(id, name)
          }
        }
        cachedNames = map
        return map
      })
      .catch(error => {
        // 失败允许下次重试
        pending = null
        throw error
      })
  }
  return pending
}

/** 把帖子的 gameVersionIds 解析为游戏版本名列表（解析失败返回空数组，交由用户自选） */
export async function resolveGameVersionNames(gameVersionIds?: number[]): Promise<string[]> {
  if (!gameVersionIds || gameVersionIds.length === 0) return []
  try {
    const names = await getCommunityGameVersionNames()
    const result: string[] = []
    for (const id of gameVersionIds) {
      const name = names.get(Number(id))
      if (name && !result.includes(name)) result.push(name)
    }
    return result
  } catch (error) {
    console.warn('[communityVersions] 解析社区游戏版本目录失败:', error)
    return []
  }
}
