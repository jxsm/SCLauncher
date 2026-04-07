// 皮肤API相关类型定义

export interface SkinApiFile {
  id: number
  filename: string
  mimeType: string
  hash: string
  url: string
  icon: string
  type: number
  size: string
  status: number
  remark: string | null
  userid: number
  downloadCount: number
  createdAt: string
  updatedAt: string
}

export interface SkinApiVersion {
  id: number
  title: string
  gameVersionIds: number[]
  postId: number
  version: string
  fileType: number
  createdAt: string
  updatedAt: string
  files: SkinApiFile[]
}

export interface SkinApiCreator {
  id: number
  nickname: string
  headImg: string
  exp: number
  rank: string | null
  roles: any[]
}

export interface SkinApiItem {
  id: number
  title: string
  content: string
  views: number
  cover: string
  creatorId: number
  plateId: number
  tags: any[]
  type: number
  fileType: number
  downloadCount: number
  createdAt: string
  updatedAt: string
  top: number
  visible: number
  put: string
  isLiked: boolean
  likeCount: string
  isBaded: boolean
  badCount: string
  postVersionCount: string
  commentCount: string
  creator: SkinApiCreator
  postVersions: SkinApiVersion[]
}

export interface SkinApiResponse {
  code: number
  data: {
    list: SkinApiItem[]
    total: number
    page: string
    limit: string
  }
  msg: string
}
