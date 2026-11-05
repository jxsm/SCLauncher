/**
 * scbbs 社区游戏版本目录（GET /api/version/list）
 * 返回 {code, data: [{id, name, type, version, status}]}：
 * - id：帖子 postVersions[].gameVersionIds 引用的适配游戏版本 ID，也可作为 post/list 的
 *   gameVersionIds 服务端过滤参数
 * - type：'original'（原版游戏版本 2.4.x/2.3.x…）、'plugin'（插件 API 版本 API1.9.x…）、
 *   'online'（联机版本 x26.10…）
 * - version：用于比较的版本数字串（如 plugin 的 "1.9"、online 的 "x26.10"）
 * 对齐 scbbs-plus-mod 的 FetchVersionListAsync / BuildVersionNameMap。
 * 模块级缓存，整个会话只拉一次；失败时清空 pending 允许重试。
 */
import { HttpRequest } from './http'

const COMMUNITY_API_BASE = 'https://m.suancaixianyu.cn'

export interface CommunityGameVersion {
  id: number
  name: string
  type: string
  version: string
  status: number
}

let cachedVersions: CommunityGameVersion[] | null = null
let pending: Promise<CommunityGameVersion[]> | null = null

/** 获取社区游戏版本目录（按 id 降序，新版本在前） */
export async function getCommunityGameVersions(): Promise<CommunityGameVersion[]> {
  if (cachedVersions) return cachedVersions
  if (!pending) {
    pending = HttpRequest(`${COMMUNITY_API_BASE}/api/version/list`)
      .then(text => {
        const data = JSON.parse(text)
        const list: any[] = Array.isArray(data?.data) ? data.data : []
        const versions: CommunityGameVersion[] = []
        for (const item of list) {
          const id = Number(item?.id)
          const name = String(item?.name || '').trim()
          if (id > 0 && name) {
            versions.push({
              id,
              name,
              type: String(item?.type || ''),
              version: String(item?.version || name),
              status: Number(item?.status ?? 1)
            })
          }
        }
        // id 与版本新旧大致正相关，降序让新版本排在筛选下拉前面
        versions.sort((a, b) => b.id - a.id)
        cachedVersions = versions
        return versions
      })
      .catch(error => {
        // 失败允许下次重试
        pending = null
        throw error
      })
  }
  return pending
}

/** 获取社区游戏版本目录（id → 版本名，如 25 → "2.4.x"） */
export async function getCommunityGameVersionNames(): Promise<Map<number, string>> {
  const versions = await getCommunityGameVersions()
  return new Map(versions.map(v => [v.id, v.name]))
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
