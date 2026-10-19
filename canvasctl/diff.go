// canvasctl diff — 语料 diff 的语义化审阅摘要（GOVERNANCE §8）。
//
// 「语料 diff 即语义变更」的人审纪律从这份摘要进入：两版本目录间逐用例按名比对，
// 变化分四类——新增（仅版本 B）/ 删除（仅版本 A）/ 期望值变化（语义面不同，含模板、
// 上下文、期望产物、文档级语义附加键如 dateTokens 转译表）/ 仅注记变化（description
// 与 meta.notes/errata 人读注记不同、语义面逐字节相同）。摘要只做呈现、不裁语义：
// 版本政策（GOVERNANCE §4）的裁决仍归语料与提案通道，本命令是评审输入而非门禁——
// 找到差异不是错误（exit 0），操作失败（坏版本目录对、语料不可严格解码）才 exit 1。

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// FileDiff 单个模块文件（版本 A ∪ 版本 B 文件名并集）的差异块。
type FileDiff struct {
	ModuleFile string
	// 整文件只在一侧（函数族新模块文件 / 模块退役，GOVERNANCE §3.1/§3.6）
	FileAdded   bool
	FileRemoved bool
	// 用例级四类差异（用例名升序）
	Added         []string
	Removed       []string
	ExpectChanged []string
	NoteOnly      []string
	Unchanged     int
	// ExpectChanged 用例 → 语义面变化字段名集（升序；人审定位用）
	ExpectFields map[string][]string
	// ExtraKeysChanged 文档级语义附加键（meta/cases 之外，如 dateTokens）发生变化者
	ExtraKeysChanged []string
	// MetaNotesChanged meta.notes/errata 出处注记变化（非语义，仅注记）
	MetaNotesChanged bool
}

// DiffResult diff 全量结果：Result 携带错误与计数，Files 为逐文件差异块（文件名序）。
type DiffResult struct {
	Result
	Contract string
	VersionA string
	VersionB string
	Files    []FileDiff
}

// Diff 比对契约的两版本目录，产出逐用例差异。任何一侧不可加载（目录缺失、语料不可
// 严格解码、双写不一致、用例名重复）即只报错不出摘要——人审材料不可来自不可信语料。
func Diff(root, contractID, versionA, versionB string) DiffResult {
	var res DiffResult
	res.Contract, res.VersionA, res.VersionB = contractID, versionA, versionB

	// 形态门卫：契约 id 与版本名先过规范形（防路径拼接逃逸），版本对相同无 diff 对象。
	if !contractIDRe.MatchString(contractID) {
		res.errf("非规范契约 id %q（小写字母开头，仅小写字母、数字、连字符）", contractID)
		return res
	}
	for _, v := range [2]string{versionA, versionB} {
		if !versionDirRe.MatchString(v) {
			res.errf("非规范版本目录名 %q（规范形 v<主>.<次>，如 v1.0）", v)
			return res
		}
	}
	if versionA == versionB {
		res.errf("版本对相同（%s）：diff 无对象", versionA)
		return res
	}
	if info, err := os.Stat(filepath.Join(root, "corpus", contractID)); err != nil || !info.IsDir() {
		res.errf("契约目录 %q 不存在（corpus/%s）", contractID, contractID)
		return res
	}

	docsA, errs := loadVersionDir(root, contractID, versionA)
	if len(errs) > 0 {
		res.Errors = append(res.Errors, errs...)
		return res
	}
	docsB, errs := loadVersionDir(root, contractID, versionB)
	if len(errs) > 0 {
		res.Errors = append(res.Errors, errs...)
		return res
	}

	for _, fname := range sortedUnion(mapKeys(docsA), mapKeys(docsB)) {
		fd := diffModuleFile(fname, docsA[fname], docsB[fname])
		res.Files = append(res.Files, fd)
		res.DiffFiles++
		res.DiffAdded += len(fd.Added)
		res.DiffRemoved += len(fd.Removed)
		res.DiffExpectChanged += len(fd.ExpectChanged)
		res.DiffNoteOnly += len(fd.NoteOnly)
		res.DiffExtras += len(fd.ExtraKeysChanged)
	}
	return res
}

// diffModuleFile 单文件比对。两侧文件名并集逐用例按名分类；文档级语义附加键与
// meta 注记另记账。
func diffModuleFile(fname string, docA, docB *moduleDoc) FileDiff {
	fd := FileDiff{ModuleFile: fname, ExpectFields: map[string][]string{}}
	switch {
	case docA == nil:
		fd.FileAdded = true
		fd.Added = append(fd.Added, docB.names...)
	case docB == nil:
		fd.FileRemoved = true
		fd.Removed = append(fd.Removed, docA.names...)
	default:
		setA, setB := nameSet(docA), nameSet(docB)
		for _, n := range docA.names {
			if !setB[n] {
				fd.Removed = append(fd.Removed, n)
			}
		}
		for _, n := range docB.names {
			if !setA[n] {
				fd.Added = append(fd.Added, n)
			}
		}
		for _, n := range docA.names {
			if !setB[n] {
				continue
			}
			fields, semanticDiffers := diffCaseBody(docA.bodies[n], docB.bodies[n])
			switch {
			case semanticDiffers:
				fd.ExpectChanged = append(fd.ExpectChanged, n)
				fd.ExpectFields[n] = fields
			case caseDescription(docA.bodies[n]) != caseDescription(docB.bodies[n]):
				fd.NoteOnly = append(fd.NoteOnly, n)
			default:
				fd.Unchanged++
			}
		}
		for _, k := range sortedUnion(extraKeys(docA.doc), extraKeys(docB.doc)) {
			aRaw, okA := docA.doc[k]
			bRaw, okB := docB.doc[k]
			if okA != okB || (okA && !sameRaw(aRaw, bRaw)) {
				fd.ExtraKeysChanged = append(fd.ExtraKeysChanged, k)
			}
		}
		fd.MetaNotesChanged = !reflect.DeepEqual(docA.meta.notes, docB.meta.notes) ||
			!reflect.DeepEqual(docA.meta.errata, docB.meta.errata)
	}
	sort.Strings(fd.Added)
	sort.Strings(fd.Removed)
	sort.Strings(fd.ExpectChanged)
	sort.Strings(fd.NoteOnly)
	sort.Strings(fd.ExtraKeysChanged)
	return fd
}

// diffCaseBody 用例语义面比对：全字段（description 除外——人读描述是注记非断言面）。
// 返回语义面变化的字段名集（升序）。
func diffCaseBody(a, b json.RawMessage) ([]string, bool) {
	ma, errA := decodeObject(a, "")
	mb, errB := decodeObject(b, "")
	if errA != nil || errB != nil {
		return nil, string(a) != string(b)
	}
	delete(ma, "description")
	delete(mb, "description")
	var changed []string
	for k, av := range ma {
		bv, ok := mb[k]
		if !ok || !sameRaw(av, bv) {
			changed = append(changed, k)
		}
	}
	for k := range mb {
		if _, ok := ma[k]; !ok {
			changed = append(changed, k)
		}
	}
	sort.Strings(changed)
	return changed, len(changed) > 0
}

func nameSet(md *moduleDoc) map[string]bool {
	s := make(map[string]bool, len(md.names))
	for _, n := range md.names {
		s[n] = true
	}
	return s
}

func mapKeys(m map[string]*moduleDoc) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func extraKeys(doc map[string]json.RawMessage) []string {
	var out []string
	for k := range doc {
		if k == "meta" || k == "cases" {
			continue
		}
		out = append(out, k)
	}
	return out
}

// caseDescription 用例的 description 人读描述（缺失 = 空串；body 已由 loader 保证为对象）。
func caseDescription(body json.RawMessage) string {
	m, err := decodeObject(body, "")
	if err != nil {
		return ""
	}
	s, err := decodeStringField(m, "description", "")
	if err != nil {
		return ""
	}
	return s
}

// Summary 渲染人可审摘要（diff 的交付物本体）。
func (r DiffResult) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "语料 diff：%s %s → %s\n", r.Contract, r.VersionA, r.VersionB)
	fmt.Fprintf(&b, "比对文件 %d：新增 %d 用例 / 删除 %d 用例 / 期望值变化 %d 用例 / 仅注记变化 %d 用例",
		r.DiffFiles, r.DiffAdded, r.DiffRemoved, r.DiffExpectChanged, r.DiffNoteOnly)
	if r.DiffExtras > 0 {
		fmt.Fprintf(&b, "；文档级语义键变化 %d 处", r.DiffExtras)
	}
	if !r.hasDifference() {
		b.WriteString("\n两版本无差异")
	}

	for _, fd := range r.Files {
		b.WriteString("\n")
		switch {
		case fd.FileAdded:
			fmt.Fprintf(&b, "%s：模块文件新增", fd.ModuleFile)
		case fd.FileRemoved:
			fmt.Fprintf(&b, "%s：模块文件删除", fd.ModuleFile)
		default:
			if fd.quiet() {
				fmt.Fprintf(&b, "%s：无变化", fd.ModuleFile)
				continue
			}
			fmt.Fprintf(&b, "%s：新增 %d / 删除 %d / 期望值变化 %d / 仅注记变化 %d",
				fd.ModuleFile, len(fd.Added), len(fd.Removed), len(fd.ExpectChanged), len(fd.NoteOnly))
		}
		for _, n := range fd.Added {
			fmt.Fprintf(&b, "\n  + %s", n)
		}
		for _, n := range fd.Removed {
			fmt.Fprintf(&b, "\n  - %s", n)
		}
		for _, n := range fd.ExpectChanged {
			fmt.Fprintf(&b, "\n  ~ %s（%s）", n, strings.Join(fd.ExpectFields[n], "、"))
		}
		for _, n := range fd.NoteOnly {
			fmt.Fprintf(&b, "\n  ≈ %s（仅注记变化）", n)
		}
		for _, k := range fd.ExtraKeysChanged {
			fmt.Fprintf(&b, "\n  ！%s（文档级语义附加键变化）", k)
		}
		if fd.MetaNotesChanged {
			b.WriteString("\n  ≈ meta 注记变化（notes/errata，非语义）")
		}
	}
	return b.String()
}

// hasDifference 摘要级差异判据：任一文件块非 quiet（含 meta 注记变化与文档级语义键）
// 或任一文件整文件只在一侧。仅 meta 注记变化时不得宣称「两版本无差异」——
// 头部结论与文件块必须一致（评审发现的呈现自相矛盾）。
func (r DiffResult) hasDifference() bool {
	for _, fd := range r.Files {
		if fd.FileAdded || fd.FileRemoved || !fd.quiet() {
			return true
		}
	}
	return false
}

// quiet 文件块无任何差异（无变化行）。
func (fd FileDiff) quiet() bool {
	return len(fd.Added)+len(fd.Removed)+len(fd.ExpectChanged)+len(fd.NoteOnly)+
		len(fd.ExtraKeysChanged) == 0 && !fd.MetaNotesChanged && fd.Unchanged > 0
}

// parseDiffArgs 解析 diff 参数：[root] <契约> <版本A> <版本B>（root 缺省句点，
// 位置参数 3 或 4 个）；空串 / 未知旗标 / 参数不足或多余返回 ok=false（usage 退出）。
func parseDiffArgs(args []string) (root, contract, versionA, versionB string, ok bool) {
	root = "."
	var pos []string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "-h" || args[i] == "--help":
			return "", "", "", "", false
		case args[i] == "" || strings.HasPrefix(args[i], "-"):
			return "", "", "", "", false
		default:
			pos = append(pos, args[i])
		}
	}
	switch len(pos) {
	case 3:
		return ".", pos[0], pos[1], pos[2], true
	case 4:
		return pos[0], pos[1], pos[2], pos[3], true
	}
	return "", "", "", "", false
}
