package version

// dotnetmeta.go — 通过静态解析 .NET 元数据（ECMA-335）从 Survivalcraft.dll 提取版本信息。
//
// 提取规则（已在 SCAPI 1.8 / 1.9.5.1 上实测验证）：
//   游戏大版本 ← Assembly(0x20) 表的程序集版本（如 2.4.0.0 → "2.4"）
//   API 版本   ← ModsManager 类静态构造函数 .cctor() 中的 ldstr 字符串：
//               剔除与游戏版本相同的("2.4.0.0")及其短形式("2.4")后，
//               剩余候选中取段数最多、数值最大者（"1.9.5.1" > "1.9"）
//               ※ 只扫 .cctor —— DeserializeJson 里有 1.80/1.8/1.81/1.8.1 等兼容性噪音字符串
//   deps.json  ← 同目录 Survivalcraft.deps.json 的 "Survivalcraft/x.y.z.w" 作为 API 版本旁证/兜底
//
// 非 .NET 的 dll 或程序集名不是 Survivalcraft（如 Engine.dll）一律不采信，
// 回退到 PE 版本资源（VS_FIXEDFILEINFO）或 Survivalcraft.exe。

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf16"
)

// LocalVersionInfo 本地游戏目录的版本检测结果
type LocalVersionInfo struct {
	GameVersion     string // 游戏大版本（前两段，如 "2.4"，与服务端清单 gameVersion 格式一致）
	GameVersionFull string // 完整版本（如 "2.4.0.0"，来自 PE 资源时可能为其他段数）
	SubVersion      string // 插件版 API 版本（如 "1.9.5.1"，原样保留 DLL 内写法；非插件版为空）
	IsAPI           bool   // 是否为 SCAPI 插件版
	Source          string // 版本来源描述，用于日志
}

// ---------------------------------------------------------------- PE 解析

type peSection struct {
	virtualAddr uint32
	virtualSize uint32
	rawAddr     uint32
	rawSize     uint32
}

type peImage struct {
	data     []byte
	sections []peSection
	ddBase   int // 数据目录起始偏移
}

func openPE(path string) (*peImage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	if len(data) < 0x40 {
		return nil, fmt.Errorf("文件过小")
	}
	peOff := int(binary.LittleEndian.Uint32(data[0x3C:]))
	if peOff+24 > len(data) || string(data[peOff:peOff+4]) != "PE\x00\x00" {
		return nil, fmt.Errorf("不是 PE 文件")
	}
	coff := peOff + 4
	numSections := int(binary.LittleEndian.Uint16(data[coff+2:]))
	optSize := int(binary.LittleEndian.Uint16(data[coff+16:]))
	opt := coff + 20
	if opt+optSize > len(data) {
		return nil, fmt.Errorf("PE 头不完整")
	}
	pe32plus := binary.LittleEndian.Uint16(data[opt:]) == 0x20B
	img := &peImage{data: data}
	img.ddBase = opt + 96
	if pe32plus {
		img.ddBase = opt + 112
	}
	secOff := opt + optSize
	for i := 0; i < numSections; i++ {
		o := secOff + i*40
		if o+40 > len(data) {
			return nil, fmt.Errorf("节表不完整")
		}
		img.sections = append(img.sections, peSection{
			virtualSize: binary.LittleEndian.Uint32(data[o+8:]),
			virtualAddr: binary.LittleEndian.Uint32(data[o+12:]),
			rawSize:     binary.LittleEndian.Uint32(data[o+16:]),
			rawAddr:     binary.LittleEndian.Uint32(data[o+20:]),
		})
	}
	return img, nil
}

func (p *peImage) rvaToOffset(rva uint32) (int, bool) {
	for _, s := range p.sections {
		size := s.virtualSize
		if s.rawSize > size {
			size = s.rawSize
		}
		if rva >= s.virtualAddr && rva < s.virtualAddr+size {
			return int(s.rawAddr) + int(rva-s.virtualAddr), true
		}
	}
	return 0, false
}

// ---------------------------------------------------------------- 元数据表结构（ECMA-335）

type colType uint8

const (
	colFixed  colType = iota // 定宽字节列
	colString                // #Strings 堆索引
	colBlob                  // #Blob 堆索引
	colGuid                  // #GUID 堆索引
	colSimple                // 单表行索引
	colCoded                 // 组合索引
)

type colSpec struct {
	typ     colType
	fixed   int   // colFixed: 字节数
	tagBits int   // colCoded: 标签位数
	tables  []int // colSimple/colCoded: 涉及的表号
}

func f(n int) colSpec    { return colSpec{typ: colFixed, fixed: n} }
func sIdx() colSpec      { return colSpec{typ: colString} }
func bIdx() colSpec      { return colSpec{typ: colBlob} }
func gIdx() colSpec      { return colSpec{typ: colGuid} }
func smpl(t int) colSpec { return colSpec{typ: colSimple, tables: []int{t}} }
func cded(bits int, ts ...int) colSpec {
	return colSpec{typ: colCoded, tagBits: bits, tables: ts}
}

// metaTables ECMA-335 标准表号 → 列结构（计算行宽用）
var metaTables = map[int][]colSpec{
	0:  {f(2), sIdx(), gIdx(), gIdx(), gIdx()},                      // Module
	1:  {cded(2, 0, 26, 35, 39), sIdx(), sIdx()},                    // TypeRef
	2:  {f(4), sIdx(), sIdx(), cded(2, 2, 1, 27), smpl(4), smpl(6)}, // TypeDef
	3:  {smpl(4)},                                                   // FieldPtr
	4:  {f(2), sIdx(), bIdx()},                                      // Field
	5:  {smpl(6)},                                                   // MethodPtr
	6:  {f(4), f(2), f(2), sIdx(), bIdx(), smpl(8)},                 // MethodDef
	7:  {smpl(8)},                                                   // ParamPtr
	8:  {f(2), f(2), sIdx()},                                        // Param
	9:  {smpl(2), cded(2, 2, 1, 27)},                                // InterfaceImpl
	10: {cded(3, 2, 1, 26, 6, 27), sIdx(), bIdx()},                  // MemberRef
	11: {f(1), f(1), cded(2, 4, 8, 23), bIdx()},                     // Constant
	12: {cded(5, 6, 4, 1, 2, 8, 9, 10, 14, 23, 20, 17, 26, 27, 0, 32, 35, 38, 39, 40, 42, 43, 44),
		cded(3, 6, 10), bIdx()}, // CustomAttribute
	13: {cded(1, 4, 8), bIdx()},                                        // FieldMarshal
	14: {f(2), bIdx(), cded(2, 2, 6, 32)},                              // DeclSecurity
	15: {f(2), f(4), smpl(2)},                                          // ClassLayout
	16: {f(4), smpl(4)},                                                // FieldLayout
	17: {bIdx()},                                                       // StandAloneSig
	18: {smpl(2), smpl(20)},                                            // EventMap
	19: {smpl(20)},                                                     // EventPtr
	20: {f(2), sIdx(), cded(2, 2, 1, 27)},                              // Event
	21: {smpl(2), smpl(23)},                                            // PropertyMap
	22: {smpl(23)},                                                     // PropertyPtr
	23: {f(2), sIdx(), bIdx()},                                         // Property
	24: {f(2), cded(1, 6, 10), cded(1, 20, 23)},                        // MethodSemantics
	25: {smpl(2), cded(1, 6, 10), cded(1, 6, 10)},                      // MethodImpl
	26: {sIdx()},                                                       // ModuleRef
	27: {bIdx()},                                                       // TypeSpec
	28: {f(2), cded(1, 4, 6), sIdx(), smpl(26)},                        // ImplMap
	29: {f(4), smpl(4)},                                                // FieldRVA
	30: {f(4), f(4)},                                                   // ENCLog
	31: {f(4)},                                                         // ENCMap
	32: {f(4), f(2), f(2), f(2), f(2), f(4), bIdx(), sIdx(), sIdx()},   // Assembly
	33: {f(4)},                                                         // AssemblyProcessor
	34: {f(4), f(4), f(4), f(4)},                                       // AssemblyOS
	35: {f(2), f(2), f(2), f(2), f(4), bIdx(), sIdx(), sIdx(), bIdx()}, // AssemblyRef
	36: {f(4), smpl(35)},                                               // AssemblyRefProcessor
	37: {f(4), f(4), f(4), f(4), smpl(35)},                             // AssemblyRefOS
	38: {f(4), sIdx(), bIdx()},                                         // File
	39: {f(4), f(4), sIdx(), sIdx(), cded(2, 38, 35, 39)},              // ExportedType
	40: {f(4), f(4), sIdx(), cded(2, 38, 35, 39)},                      // ManifestResource
	41: {smpl(2), smpl(2)},                                             // NestedClass
	42: {f(2), f(2), cded(2, 2, 6), sIdx()},                            // GenericParam
	43: {cded(2, 6, 10, 43), bIdx()},                                   // MethodSpec
	44: {smpl(42), cded(2, 2, 1, 27)},                                  // GenericParamConstraint
}

// ---------------------------------------------------------------- .NET 元数据解析

type dotnetMeta struct {
	pe            *peImage
	rows          map[int]int // 表号 → 行数
	rowOffset     map[int]int // 表号 → 首行文件偏移
	rowWidth      map[int]int // 表号 → 行宽
	colWidths     map[int][]int
	strIdxWidth   int
	blobIdxWidth  int
	guidIdxWidth  int
	strHeapOffset int
	us            map[int]string // #US 堆内偏移 → 字符串
	assemblyName  string
	assemblyVer   [4]int
}

// parseDotnetDll 解析 .NET 程序集元数据；程序集名不是 survivalcraft 时返回错误
func parseDotnetDll(path string) (*dotnetMeta, error) {
	pe, err := openPE(path)
	if err != nil {
		return nil, err
	}
	d := pe.data

	// 数据目录 14 = CLR Header
	if pe.ddBase+15*8 > len(d) {
		return nil, fmt.Errorf("PE 数据目录不完整")
	}
	clrRva := binary.LittleEndian.Uint32(d[pe.ddBase+14*8:])
	cliOff, ok := pe.rvaToOffset(clrRva)
	if !ok || cliOff+16 > len(d) {
		return nil, fmt.Errorf("不是 .NET 程序集（无 CLR 头）")
	}
	mdRva := binary.LittleEndian.Uint32(d[cliOff+8:])
	mdOff, ok := pe.rvaToOffset(mdRva)
	if !ok || mdOff+16 > len(d) {
		return nil, fmt.Errorf("元数据根不可达")
	}
	if binary.LittleEndian.Uint32(d[mdOff:]) != 0x424A5342 { // "BSJB"
		return nil, fmt.Errorf("元数据签名错误")
	}

	// 流头
	verLen := int(binary.LittleEndian.Uint32(d[mdOff+12:]))
	p := mdOff + 16 + verLen
	if p+4 > len(d) {
		return nil, fmt.Errorf("元数据版本区越界")
	}
	nStreams := int(binary.LittleEndian.Uint16(d[p+2:]))
	p += 4
	streams := map[string][2]int{} // name → {文件偏移, 大小}
	for i := 0; i < nStreams; i++ {
		if p+8 > len(d) {
			return nil, fmt.Errorf("流头越界")
		}
		off := int(binary.LittleEndian.Uint32(d[p:]))
		size := int(binary.LittleEndian.Uint32(d[p+4:]))
		p += 8
		end := bytes.IndexByte(d[p:], 0)
		if end < 0 {
			return nil, fmt.Errorf("流名未终止")
		}
		name := string(d[p : p+end])
		streams[name] = [2]int{mdOff + off, size}
		p = (p + end + 1 + 3) & ^3 // 名字按 4 字节对齐
	}

	tbl, ok := streams["#~"]
	if !ok {
		tbl, ok = streams["#-"]
	}
	if !ok {
		return nil, fmt.Errorf("缺少元数据表流")
	}
	tblOff := tbl[0]
	if tblOff+24 > len(d) {
		return nil, fmt.Errorf("表流头越界")
	}

	// 表流头: +6 HeapSizes, +8 Valid(8B), +24 起每表 4 字节行数
	heapSizes := d[tblOff+6]
	valid := binary.LittleEndian.Uint64(d[tblOff+8:])
	m := &dotnetMeta{
		pe:           pe,
		rows:         map[int]int{},
		rowOffset:    map[int]int{},
		rowWidth:     map[int]int{},
		colWidths:    map[int][]int{},
		strIdxWidth:  2,
		blobIdxWidth: 2,
		guidIdxWidth: 2,
		us:           map[int]string{},
	}
	if heapSizes&1 != 0 {
		m.strIdxWidth = 4
	}
	if heapSizes&2 != 0 {
		m.guidIdxWidth = 4
	}
	if heapSizes&4 != 0 {
		m.blobIdxWidth = 4
	}
	q := tblOff + 24
	var tableNums []int
	for t := 0; t < 64; t++ {
		if valid>>t&1 == 0 {
			continue
		}
		if q+4 > len(d) {
			return nil, fmt.Errorf("行数区越界")
		}
		m.rows[t] = int(binary.LittleEndian.Uint32(d[q:]))
		q += 4
		tableNums = append(tableNums, t)
	}

	// 各列宽度与行偏移
	for _, t := range tableNums {
		spec, ok := metaTables[t]
		if !ok {
			return nil, fmt.Errorf("表 0x%02X 未定义", t)
		}
		var widths []int
		total := 0
		for _, c := range spec {
			w := 0
			switch c.typ {
			case colFixed:
				w = c.fixed
			case colString:
				w = m.strIdxWidth
			case colBlob:
				w = m.blobIdxWidth
			case colGuid:
				w = m.guidIdxWidth
			case colSimple:
				w = 2
				if m.rows[c.tables[0]] > 0xFFFF {
					w = 4
				}
			case colCoded:
				maxRows := 0
				for _, tt := range c.tables {
					if r := m.rows[tt]; r > maxRows {
						maxRows = r
					}
				}
				w = 2
				if maxRows >= 1<<(16-c.tagBits) {
					w = 4
				}
			}
			widths = append(widths, w)
			total += w
		}
		m.colWidths[t] = widths
		m.rowWidth[t] = total
	}
	sort.Ints(tableNums)
	for _, t := range tableNums {
		m.rowOffset[t] = q
		q += m.rowWidth[t] * m.rows[t]
	}

	strs, ok := streams["#Strings"]
	if !ok {
		return nil, fmt.Errorf("缺少 #Strings 堆")
	}
	m.strHeapOffset = strs[0]

	// Assembly(32): 0=HashAlgId 1..4=版本 5=Flags 6=PublicKey 7=Name 8=Culture
	if m.rows[32] == 0 {
		return nil, fmt.Errorf("缺少 Assembly 表")
	}
	m.assemblyName = m.cellString(32, 0, 7)
	for i := 0; i < 4; i++ {
		m.assemblyVer[i] = m.cellInt(32, 0, 1+i)
	}
	if !strings.EqualFold(m.assemblyName, "Survivalcraft") {
		return nil, fmt.Errorf("程序集名是 %s 而非 Survivalcraft", m.assemblyName)
	}
	if m.assemblyVer == [4]int{} {
		return nil, fmt.Errorf("程序集版本为 0.0.0.0")
	}

	// #US 用户字符串堆
	if us, ok := streams["#US"]; ok {
		m.parseUS(us[0], us[1])
	}
	return m, nil
}

func (m *dotnetMeta) cellInt(table, row, col int) int {
	widths := m.colWidths[table]
	off := m.rowOffset[table] + row*m.rowWidth[table]
	for _, w := range widths[:col] {
		off += w
	}
	w := widths[col]
	if off < 0 || off+w > len(m.pe.data) {
		return 0
	}
	return leUint(m.pe.data[off : off+w])
}

// leUint 读取 1~4 字节小端无符号整数
func leUint(b []byte) int {
	var v uint32
	for i := 0; i < len(b); i++ {
		v |= uint32(b[i]) << (8 * i)
	}
	return int(v)
}

func (m *dotnetMeta) cellString(table, row, col int) string {
	idx := m.cellInt(table, row, col)
	if idx == 0 {
		return ""
	}
	off := m.strHeapOffset + idx
	if off >= len(m.pe.data) {
		return ""
	}
	end := bytes.IndexByte(m.pe.data[off:], 0)
	if end < 0 {
		return ""
	}
	return string(m.pe.data[off : off+end])
}

// parseUS 解析 #US 堆：堆内偏移 → UTF-16 字符串
func (m *dotnetMeta) parseUS(off, size int) {
	d := m.pe.data
	q := off + 1
	end := off + size
	for q < end {
		b0 := d[q]
		if b0 == 0 {
			q++
			continue
		}
		var blen, hdr int
		switch {
		case b0 < 0x80:
			blen, hdr = int(b0), 1
		case b0 < 0xC0:
			if q+2 > end {
				return
			}
			blen, hdr = int(b0&0x3F)<<8|int(d[q+1]), 2
		default:
			if q+4 > end {
				return
			}
			blen = int(b0&0x1F)<<24 | int(d[q+1])<<16 | int(d[q+2])<<8 | int(d[q+3])
			hdr = 4
		}
		if blen > 0 && q+hdr+blen <= end {
			m.us[q-off] = utf16LE(m.pe.data[q+hdr : q+hdr+blen-1])
		}
		q += hdr + blen
	}
}

func utf16LE(b []byte) string {
	units := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		units = append(units, binary.LittleEndian.Uint16(b[i:]))
	}
	return string(utf16.Decode(units))
}

// methodLdstr 返回 类型.方法() 方法体中的全部 ldstr 字符串；未找到类型/方法返回 false
func (m *dotnetMeta) methodLdstr(typeName, namespace, methodName string) ([]string, bool) {
	nTypes, ok := m.rows[2]
	if !ok || m.rows[6] == 0 {
		return nil, false
	}
	nMethods := m.rows[6]
	for i := 0; i < nTypes; i++ {
		if m.cellString(2, i, 1) != typeName || m.cellString(2, i, 2) != namespace {
			continue
		}
		mlist := m.cellInt(2, i, 5)
		nxt := nMethods + 1
		if i+1 < nTypes {
			nxt = m.cellInt(2, i+1, 5)
		}
		for v := mlist; v < nxt; v++ {
			mrow := v - 1
			// MethodPtr(5) 间接引用（若存在）
			if mp := m.rows[5]; mp > 0 && mrow >= 0 && mrow < mp {
				mrow = m.cellInt(5, mrow, 0) - 1
			}
			if mrow < 0 || mrow >= nMethods {
				continue
			}
			if m.cellString(6, mrow, 3) != methodName {
				continue
			}
			return m.scanLdstr(m.cellInt(6, mrow, 0))
		}
		return nil, false // 类型存在但无此方法
	}
	return nil, false
}

// scanLdstr 解析方法体并收集 ldstr (0x72) 引用的 #US 字符串
func (m *dotnetMeta) scanLdstr(rva int) ([]string, bool) {
	if rva == 0 {
		return nil, false
	}
	off, ok := m.pe.rvaToOffset(uint32(rva))
	if !ok || off >= len(m.pe.data) {
		return nil, false
	}
	d := m.pe.data
	b0 := d[off]
	var codeOff, codeLen int
	switch b0 & 3 {
	case 2: // tiny
		codeOff, codeLen = off+1, int(b0>>2)
	case 3: // fat：低 12 位标志 + 高 4 位头大小（32 位字数），后接 maxstack/codesize/localtok
		if off+12 > len(d) {
			return nil, false
		}
		flags16 := binary.LittleEndian.Uint16(d[off:])
		codeLen = int(binary.LittleEndian.Uint32(d[off+4:]))
		codeOff = off + 4*int(flags16>>12&0xF)
	default:
		return nil, false
	}
	if codeOff < 0 || codeLen < 0 || codeOff+codeLen > len(d) {
		return nil, false
	}
	var out []string
	for k := 0; k+5 <= codeLen; k++ {
		if d[codeOff+k] == 0x72 {
			tok := binary.LittleEndian.Uint32(d[codeOff+k+1:])
			if s, ok := m.us[int(tok&0xFFFFFF)]; ok {
				out = append(out, s)
			}
		}
	}
	return out, true
}

// apiVersionFromCctor 从 ModsManager..cctor() 中识别 API 版本字符串
func (m *dotnetMeta) apiVersionFromCctor() (string, bool) {
	strs, ok := m.methodLdstr("ModsManager", "", ".cctor")
	if !ok {
		return "", false
	}
	var best []int
	bestStr := ""
	for _, s := range strs {
		t := parseVersionTuple(s)
		if t == nil || len(t) < 2 {
			continue
		}
		if allZero(t) {
			continue
		}
		// 剔除游戏版本本身及其 2/3 段短形式（2.4.0.0 / 2.4 / 2.4.0）
		if len(t) <= 4 && tuplePrefixOf(t, m.assemblyVer) {
			continue
		}
		if best == nil || betterTuple(t, best) {
			best, bestStr = t, s
		}
	}
	if best == nil {
		return "", false
	}
	return bestStr, true
}

// parseVersionTuple 解析 "1.9.5.1" 样式版本为整数切片；至少两段、全数字，否则 nil
func parseVersionTuple(s string) []int {
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return nil
	}
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		if p == "" || len(p) > 9 {
			return nil
		}
		v := 0
		for _, c := range p {
			if c < '0' || c > '9' {
				return nil
			}
			v = v*10 + int(c-'0')
		}
		out = append(out, v)
	}
	return out
}

func allZero(t []int) bool {
	for _, v := range t {
		if v != 0 {
			return false
		}
	}
	return true
}

// tuplePrefixOf 判断 t 是否为 game 的前缀（用于识别游戏版本短形式）
func tuplePrefixOf(t []int, game [4]int) bool {
	for i, v := range t {
		if v != game[i] {
			return false
		}
	}
	return true
}

// betterTuple 段数多者优先，同段数按数值从高到高（1.9.5.1 > 1.9）
func betterTuple(a, b []int) bool {
	if len(a) != len(b) {
		return len(a) > len(b)
	}
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

// ---------------------------------------------------------------- 对外检测入口

// DetectLocalVersion 检测本地游戏目录的版本信息：
//  1. 有 Survivalcraft.dll 且为 SCAPI .NET 程序集 → 大版本 + API 版本
//  2. dll 非 .NET（或解析失败）→ PE 版本资源，API 版本用 deps.json 兜底
//  3. 无 dll（原版 2.3- 等）→ Survivalcraft.exe 的 PE 版本资源
func DetectLocalVersion(gameDir string) (LocalVersionInfo, error) {
	info := LocalVersionInfo{}

	dllPath := filepath.Join(gameDir, "Survivalcraft.dll")
	exePath := filepath.Join(gameDir, "Survivalcraft.exe")
	dllExists := fileExists(dllPath)
	exeExists := fileExists(exePath)
	if !dllExists && !exeExists {
		if found := findGameExecutable(gameDir); found != "" {
			if strings.EqualFold(filepath.Base(found), "Survivalcraft.dll") {
				dllPath, dllExists = found, true
			} else {
				exePath, exeExists = found, true
			}
		}
	}

	if dllExists {
		if md, err := parseDotnetDll(dllPath); err == nil {
			info.Source = fmt.Sprintf("Survivalcraft.dll .NET 元数据 (%s)", dllPath)
			info.GameVersionFull = fmt.Sprintf("%d.%d.%d.%d", md.assemblyVer[0], md.assemblyVer[1], md.assemblyVer[2], md.assemblyVer[3])
			info.GameVersion = majorVersionOf(info.GameVersionFull)
			if api, ok := md.apiVersionFromCctor(); ok {
				info.SubVersion, info.IsAPI = api, true
			} else if nuget := readDepsAPIVersion(gameDir); nuget != "" {
				// 极少数情况：dll 里没有 ModsManager 但 deps.json 表明是插件版
				info.SubVersion, info.IsAPI = nuget, true
				info.Source += " + deps.json"
			}
			return info, nil
		}
		// .NET 解析失败（非 .NET dll / 程序集名不符）→ 用 PE 版本资源
		if ver, err := readPEVersion(dllPath); err == nil {
			info.Source = fmt.Sprintf("Survivalcraft.dll PE 版本资源 (%s)", dllPath)
			info.GameVersionFull = ver
			info.GameVersion = majorVersionOf(ver)
			if nuget := readDepsAPIVersion(gameDir); nuget != "" {
				info.SubVersion, info.IsAPI = nuget, true
				info.Source += " + deps.json"
			}
			return info, nil
		}
		// dll 双双失败，继续尝试 exe
	}

	if exeExists {
		ver, err := readPEVersion(exePath)
		if err != nil {
			return info, fmt.Errorf("读取 Survivalcraft.exe 版本失败: %w", err)
		}
		info.Source = fmt.Sprintf("Survivalcraft.exe PE 版本资源 (%s)", exePath)
		info.GameVersionFull = ver
		info.GameVersion = majorVersionOf(ver)
		return info, nil
	}

	return info, fmt.Errorf("目录中未找到 Survivalcraft.dll 或 Survivalcraft.exe: %s", gameDir)
}

// majorVersionOf 取版本号前两段（"2.4.0.0" → "2.4"），与服务端清单的 gameVersion 格式对齐
func majorVersionOf(ver string) string {
	parts := strings.Split(ver, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return ver
}

// depsPackageVersionRe 匹配 deps.json 中 "Survivalcraft/1.9.5.1" 样式的包版本
var depsPackageVersionRe = regexp.MustCompile(`"Survivalcraft/([0-9]+(?:\.[0-9]+)*)"`)

// readDepsAPIVersion 从游戏目录的 Survivalcraft.deps.json 读取插件版 NuGet 包版本
func readDepsAPIVersion(gameDir string) string {
	data, err := os.ReadFile(filepath.Join(gameDir, "Survivalcraft.deps.json"))
	if err != nil {
		return ""
	}
	if m := depsPackageVersionRe.FindSubmatch(data); m != nil {
		return string(m[1])
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// findGameExecutable 递归查找游戏目录（含子目录）中的 Survivalcraft.dll / Survivalcraft.exe，
// 优先返回 dll
func findGameExecutable(gameDir string) string {
	found := ""
	_ = filepath.Walk(gameDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		switch {
		case strings.EqualFold(name, "Survivalcraft.dll"):
			found = path
			return filepath.SkipAll
		case strings.EqualFold(name, "Survivalcraft.exe") && found == "":
			found = path
		}
		return nil
	})
	return found
}
