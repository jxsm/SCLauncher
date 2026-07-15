// 版本比较与依赖范围判定工具
// 参考 scbbs-plus-mod 的 Update.cs（NormalizeVersionText/CompareVersionText）、
// LocalResources.cs（TryParseDependencyVersionRange/DoesInstalledVersionSatisfyDependency）、
// LocalMods.Pending.cs（IsVersionTextIncompatible）。

/**
 * 规范化版本文本：去掉前导 v/V，取首个数字串（如 "v1.9.2.1-beta" → "1.9.2.1"）。
 */
export function normalizeVersionText(input: string | null | undefined): string {
  if (!input) return ''
  const m = input.replace(/^\s*[vV]/, '').match(/\d+(\.\d+)*/)
  return m ? m[0] : ''
}

/**
 * 提取所有数字段为数字数组（如 "1.9.2.1-beta" → [1,9,2,1]）。
 */
export function extractVersionParts(input: string | null | undefined): number[] {
  if (!input) return []
  const matches = input.match(/\d+/g)
  return matches ? matches.map(Number) : []
}

/**
 * 分量比较版本，缺位补 0。返回 -1/0/1。
 */
export function compareVersionText(a: string | null | undefined, b: string | null | undefined): number {
  const pa = extractVersionParts(a)
  const pb = extractVersionParts(b)
  const len = Math.max(pa.length, pb.length)
  for (let i = 0; i < len; i++) {
    const x = pa[i] ?? 0
    const y = pb[i] ?? 0
    if (x < y) return -1
    if (x > y) return 1
  }
  return 0
}

export interface VersionRange {
  min: string
  includeMin: boolean
  max: string
  includeMax: boolean
}

/**
 * 解析 Nuget 风格的括号范围："[1.0,2.0]"、"(,1.0]"、"[1.0]"、"[1.0,)" 等。
 * 非 Nuget 括号形式返回 null。
 */
export function tryParseDependencyVersionRange(text: string | null | undefined): VersionRange | null {
  if (!text) return null
  const t = text.trim()
  if (t.length < 2) return null
  const first = t[0]
  const last = t[t.length - 1]
  if (!((first === '[' || first === '(') && (last === ']' || last === ')'))) return null

  const includeMin = first === '['
  const includeMax = last === ']'
  const inner = t.slice(1, -1).trim()
  const commaIdx = inner.indexOf(',')
  if (commaIdx < 0) {
    // 精确版本 [1.0] → min==max 且都包含
    if (!inner) return null
    return { min: inner, includeMin: true, max: inner, includeMax: true }
  }
  const min = inner.slice(0, commaIdx).trim()
  const max = inner.slice(commaIdx + 1).trim()
  return { min, includeMin, max, includeMax }
}

/**
 * 判断一个文本是否为"纯版本号"（如 "1.0"、"v1.9.2.1"），即不含范围/比较运算符。
 * 用于决定是否把 versionRange 原样传给服务端接口（服务端可能不支持复杂范围语法）。
 */
export function isPlainVersion(text: string | null | undefined): boolean {
  if (!text) return false
  return /^\s*[vV]?\d+(\.\d+)*\s*$/.test(text)
}

/**
 * 判断 installedVersion 是否满足 rangeText 声明的范围。
 * 实现完整文档集：Nuget 括号范围 + SemVer 前缀（= ^ ~ > >= < <=），裸版本视为">= 该版本"（文档 Nuget 约定）。
 * 空 rangeText 表示任意，返回 true。无法解析时回退到规范化精确匹配。
 */
export function satisfiesVersionRange(installedVersion: string | null | undefined, rangeText: string | null | undefined): boolean {
  const range = (rangeText ?? '').trim()
  if (!range) return true // 任意

  const installed = normalizeVersionText(installedVersion) || (installedVersion ?? '')

  // 1) Nuget 括号范围
  const bracket = tryParseDependencyVersionRange(range)
  if (bracket) {
    if (bracket.min) {
      const c = compareVersionText(installed, bracket.min)
      if (bracket.includeMin ? c < 0 : c <= 0) return false
    }
    if (bracket.max) {
      const c = compareVersionText(installed, bracket.max)
      if (bracket.includeMax ? c > 0 : c >= 0) return false
    }
    return true
  }

  // 2) SemVer 前缀
  const lower = range.toLowerCase()
  if (lower.startsWith('>=')) return compareVersionText(installed, range.slice(2)) >= 0
  if (lower.startsWith('<=')) return compareVersionText(installed, range.slice(2)) <= 0
  if (lower.startsWith('>')) return compareVersionText(installed, range.slice(1)) > 0
  if (lower.startsWith('<')) return compareVersionText(installed, range.slice(1)) < 0
  if (lower.startsWith('=')) return compareVersionText(installed, range.slice(1)) === 0
  if (lower.startsWith('^')) {
    // 同主版本且 >= 指定版本（SC 模组以 1.x 为主，足够覆盖常见情况）
    const rest = range.slice(1)
    const rp = extractVersionParts(rest)
    const ip = extractVersionParts(installed)
    if (rp.length === 0 || ip.length === 0) return compareVersionText(installed, rest) === 0
    return ip[0] === rp[0] && compareVersionText(installed, rest) >= 0
  }
  if (lower.startsWith('~')) {
    // 同主次版本且 >= 指定版本
    const rest = range.slice(1)
    const rp = extractVersionParts(rest)
    const ip = extractVersionParts(installed)
    if (rp.length < 2 || ip.length < 2) return compareVersionText(installed, rest) === 0
    return ip[0] === rp[0] && ip[1] === rp[1] && compareVersionText(installed, rest) >= 0
  }

  // 3) 裸版本 → 视为 ">="（文档 Nuget 约定："1.0" → x≥1.0）
  if (isPlainVersion(range)) {
    return compareVersionText(installed, range) >= 0
  }

  // 4) 回退：规范化精确匹配
  return compareVersionText(installed, range) === 0
}

/**
 * ApiVersion 是否低于 1.8（引擎会显示"可能无法使用"警告，但仍尝试加载）。
 * 无数字时视为不触发（兼容）。
 */
export function isApiVersionPotentiallyUnusable(apiVersion: string | null | undefined): boolean {
  const parts = extractVersionParts(apiVersion)
  if (parts.length === 0) return false
  const major = parts[0]
  const minor = parts[1] ?? 0
  if (major < 1) return true
  if (major === 1 && minor < 8) return true
  return false
}

/**
 * 参考 LocalMods.Pending.cs:484 的 ApiVersion 不兼容判定：
 * 比较前两段数字（主.次），相同即兼容；额外豁免 1.8↔1.9 跨版（主机主=1 且次>=9，模组主=1 且次∈[8,主机次]）。
 * 任一方无数字视为兼容。
 */
export function isVersionTextIncompatible(hostApi: string | null | undefined, modApi: string | null | undefined): boolean {
  const host = extractVersionParts(hostApi)
  const mod = extractVersionParts(modApi)
  if (host.length === 0 || mod.length === 0) return false
  const hostMajor = host[0]
  const hostMinor = host[1] ?? 0
  const modMajor = mod[0]
  const modMinor = mod[1] ?? 0
  if (hostMajor === modMajor && hostMinor === modMinor) return false
  if (hostMajor === 1 && modMajor === 1 && hostMinor >= 9 && modMinor >= 8 && modMinor <= hostMinor) return false
  return true
}

/**
 * 从版本 id 尝试解析出主机 API 版本。
 * 版本 id 形如 api-<scVersion>-<apiVersion 或 api-<scVersion>-<subVersion>（subVersion 可能带
 * 字母前缀如 "API1.60"）。SC API 版本恒为 1.x，因此只接受主版本号为 1 的段；解析不到返回 ''
 * （调用方应跳过跨版本提示，避免误报）。
 */
export function inferApiVersionFromVersionId(versionId: string | null | undefined): string {
  if (!versionId) return ''
  if (!versionId.toLowerCase().startsWith('api-')) return ''
  const rest = versionId.slice(4)
  let best = ''
  let bestParts = 0
  for (const seg of rest.split('-')) {
    // 去掉可能的字母类型前缀（API/NET/Original/Modified/联机/插件 等）
    const cleaned = seg.replace(/^[A-Za-z一-龥]+/i, '')
    if (!/^\d+(\.\d+)+$/.test(cleaned)) continue // 至少两段的纯数字版本
    const parts = cleaned.split('.').map(Number)
    if (parts[0] !== 1) continue // SC API 版本恒为 1.x，拒绝游戏版本段（如 2.4、2.31）
    if (parts.length >= bestParts) { // 同段数时取靠后一段（apiVersion 一般在 scVersion 之后）
      bestParts = parts.length
      best = cleaned
    }
  }
  return best
}
