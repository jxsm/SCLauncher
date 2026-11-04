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

export type CommunityVersionType = 'original' | 'plugin' | 'online'

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

/**
 * 把本地清单的游戏版本号（如 "2.4"）解析为社区版本目录 ID（匹配 "2.4.x"），
 * 供筛选下拉默认选中当前主版本；找不到返回 null。
 */
export async function resolveGameVersionId(gameVersion: string): Promise<number | null> {
  const target = gameVersion.trim()
  if (!target) return null
  const versions = await getCommunityGameVersions()
  // 先精确匹配，再去掉社区名的 ".x" 后缀匹配（"2.4.x" → "2.4"）
  const exact = versions.find(v => v.name === target)
  if (exact) return exact.id
  const normalized = versions.find(v => v.name.replace(/\.x$/i, '') === target)
  return normalized ? normalized.id : null
}

/** 提取所有数字段为数字数组（如 "api-1.9.5.1" → [1,9,5,1]，"x26.10" → [26,10]） */
function extractVersionNumbers(text: string): number[] {
  const matches = text.match(/\d+/g)
  return matches ? matches.map(Number) : []
}

interface VersionMatchScore {
  // 候选是当前版本数字串的（部分）前缀（候选更短或相等），如当前 [1,9,5,1] vs 候选 [1,9]
  candidateIsPrefix: boolean
  // 公共数字前缀长度，越大越接近
  commonPrefix: number
  // 前缀后首个不同数字段之差，越小越接近
  nextDiff: number
  // 数字段数差，越小越接近
  lenDiff: number
  // 候选数字段数，同分时越长越具体
  candidateLen: number
}

// 数字段比较约定：Infinity 表示无下一个可比较段（视为最接近）
const INF = Number.POSITIVE_INFINITY

function scoreCandidate(target: number[], candidate: number[]): VersionMatchScore | null {
  if (target.length === 0 || candidate.length === 0) return null
  let prefix = 0
  while (prefix < target.length && prefix < candidate.length && target[prefix] === candidate[prefix]) {
    prefix++
  }
  return {
    candidateIsPrefix: prefix === candidate.length,
    commonPrefix: prefix,
    nextDiff: prefix < Math.min(target.length, candidate.length)
      ? Math.abs(target[prefix] - candidate[prefix])
      : INF,
    lenDiff: Math.abs(target.length - candidate.length),
    candidateLen: candidate.length
  }
}

/** 分数比较：返回正数表示 a 更优（参考 scbbs-plus-mod Posts.cs VersionMatchScore.CompareTo） */
function compareMatchScore(a: VersionMatchScore, b: VersionMatchScore): number {
  if (a.candidateIsPrefix !== b.candidateIsPrefix) return a.candidateIsPrefix ? 1 : -1
  if (a.commonPrefix !== b.commonPrefix) return b.commonPrefix - a.commonPrefix
  if (a.nextDiff !== b.nextDiff) return a.nextDiff < b.nextDiff ? 1 : -1
  if (a.lenDiff !== b.lenDiff) return a.lenDiff < b.lenDiff ? 1 : -1
  return b.candidateLen - a.candidateLen
}

/**
 * 在版本目录指定类型中为本地版本号找最佳匹配（参考 scbbs-plus-mod Posts.cs
 * FindBestPluginVersion：仅参与 status==1 且 type 匹配的目录条目，按数字段打分取最优）。
 * versionText 传版本的 subVersion：单机 API 版如 "api-1.9.5.1"（匹配 plugin 类），
 * 联机版如 "26.07.01.01"（匹配 online 类）；找不到返回 null。
 */
export async function findBestCommunityVersion(
  type: CommunityVersionType,
  versionText: string
): Promise<CommunityGameVersion | null> {
  const text = (versionText || '').trim()
  if (!text) return null
  const target = extractVersionNumbers(text)
  if (target.length === 0) return null

  const versions = await getCommunityGameVersions()
  let best: { version: CommunityGameVersion; score: VersionMatchScore } | null = null
  for (const v of versions) {
    if (v.status !== 1 || v.type !== type) continue
    const score = scoreCandidate(target, extractVersionNumbers(v.version))
    if (!score) continue
    if (!best || compareMatchScore(score, best.score) > 0) {
      best = { version: v, score }
    }
  }
  return best?.version ?? null
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
