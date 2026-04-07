# 家具管理 API 实现说明

## 概述
家具管理功能需要实现以下 Go 函数，用于管理游戏版本中的家具包文件。

## 家具包存储位置
由于游戏版本差异，家具包可能存储在以下两个位置之一：
- `[游戏版本]\doc\FurniturePacks\`
- `[游戏版本]/FurniturePacks/`

只有用户启动一次游戏后才会生成这个文件夹。

## 需要实现的 API 函数

### 1. GetFurnitures
```go
func GetFurnitures(versionId string) ([]Furniture, error)
```

**功能：** 获取指定版本的所有家具包

**逻辑：**
1. 根据 versionId 获取游戏版本路径
2. 依次检查两个可能的路径：
   - `{versionPath}/doc/FurniturePacks/`
   - `{versionPath}/FurniturePacks/`
3. 如果两个路径都不存在，返回 nil 和 nil 错误（前端通过 nil 判断文件夹不存在）
4. 如果找到路径，扫描该目录下所有 `.scfpack` 文件
5. 返回家具信息列表

**返回数据结构：**
```go
type Furniture struct {
    ID       string `json:"id"`       // 文件名（不含扩展名）
    Name     string `json:"name"`     // 显示名称（文件名）
    FileName string `json:"fileName"` // 完整文件名（含扩展名）
}
```

### 2. DeleteFurniture
```go
func DeleteFurniture(versionId string, furnitureId string) error
```

**功能：** 删除指定的家具包文件

**逻辑：**
1. 根据 versionId 获取游戏版本路径
2. 找到正确的 FurniturePacks 目录
3. 删除对应的 `.scfpack` 文件（furnitureId 是不含扩展名的文件名）

### 3. RenameFurniture
```go
func RenameFurniture(versionId string, furnitureId string, newName string) error
```

**功能：** 重命名家具包文件

**逻辑：**
1. 根据 versionId 获取游戏版本路径
2. 找到正确的 FurniturePacks 目录
3. 将 `.scfpack` 文件重命名为新的文件名（需要添加 .scfpack 扩展名）
4. newName 只包含新的文件名，不含扩展名

### 4. OpenFurnitureFolder
```go
func OpenFurnitureFolder(versionId string) error
```

**功能：** 打开家具包文件夹

**逻辑：**
1. 根据 versionId 获取游戏版本路径
2. 找到正确的 FurniturePacks 目录
3. 使用系统默认文件管理器打开该目录

## 参考实现

可以参考现有的存档管理函数实现，比如：
- `GetSaveGames` - 获取存档列表
- `DeleteSaveGame` - 删除存档
- `RenameSaveGame` - 重命名存档
- `OpenSaveGameFolder` - 打开存档文件夹

## 注意事项

1. **路径检查顺序：** 先检查 `doc/FurniturePacks/`，再检查 `FurniturePacks/`
2. **文件夹不存在：** GetFurnitures 应该返回 nil 列表和 nil 错误，而不是返回错误
3. **文件名处理：** furnitureId 是不含扩展名的文件名，操作时需要添加 `.scfpack` 扩展名
4. **错误处理：** 其他操作失败时返回相应的错误信息
5. **大小写：** .scfpack 扩展名应该是小写

## 前端调用示例

```typescript
// 获取家具列表
const furnitures = await GetFurnitures(versionId)
if (furnitures === null) {
  // 文件夹不存在，显示"请启动一次游戏"提示
} else {
  // 显示家具列表
}

// 删除家具
await DeleteFurniture(versionId, furnitureId)

// 重命名家具
await RenameFurniture(versionId, furnitureId, "新名称")

// 打开文件夹
await OpenFurnitureFolder(versionId)
```
