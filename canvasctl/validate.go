package main

// canvasctl validate — 语料形状校验 + 治理校验（contract↔目录双写互检、错误 code 白名单、用例名唯一）。
//
// 工具非规范性（GOVERNANCE §8）：只执行 GOVERNANCE.md 已钉死的形状与治理规则，
// 永不参与「什么是对的」裁决；工具与语料结论冲突时，以语料 + 薄规格为准并视为工具 bug。
//
// 语料布局（双写互检的对象）：
//
//	corpus/<contract-id>/<version>/<module-file>.json   契约 / 版本 / 模块文件三级目录
//	spec/error-codes.json                               错误 code 注册表（codes + reserved）
//
// 严格 typed 解码：逐层显式键集检查，缺键与未知键一律红——「语料是法，法必须严」；
// 与渲染面对静态声明字段未知值的宽容读方向相反且各自成立（GOVERNANCE §6）。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// registryIdent 注册表格式标识：格式演进入新版本字符串，不原地改义（同支持面声明的 declaration 字段纪律）。
const registryIdent = "canvas-semantics/error-codes v1"

var (
	// 版本目录规范形 v<主>.<次>（GOVERNANCE §4）；存量 v1/v2 短形态须在迁移时归一。
	versionDirRe = regexp.MustCompile(`^v[0-9]+\.[0-9]+$`)
	// 契约 id 字符集：小写字母开头，仅小写字母、数字、连字符（模块注册表现有五契约同构）。
	contractIDRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

// Result 汇总校验错误与计数（计数供 CLI 摘要输出）。
type Result struct {
	Errors      []error
	Contracts   int
	Versions    int
	ModuleFiles int
	Cases       int
}

func (r *Result) errf(format string, args ...any) {
	r.Errors = append(r.Errors, fmt.Errorf(format, args...))
}

func (r *Result) err(err error) {
	r.Errors = append(r.Errors, err)
}

// codeRegistry 错误 code 注册表（spec/error-codes.json）的内存形态。
type codeRegistry struct {
	codes    map[string]bool
	reserved map[string]bool
}

// Validate 校验 root（canvas-semantics 仓根）下的语料形状与治理规则，返回全部错误（不短路）。
func Validate(root string) Result {
	var res Result

	// 注册表先行：加载失败只报一处，code 白名单检查跳过（避免级联噪声掩盖形状问题）。
	reg, regErr := loadRegistry(filepath.Join(root, "spec", "error-codes.json"))
	if regErr != nil {
		res.err(regErr)
	}

	corpusDir := filepath.Join(root, "corpus")
	contractEntries, err := readSortedEntries(corpusDir)
	if err != nil {
		res.errf("corpus/ 目录读取失败：%v", err)
		return res
	}

	for _, ce := range contractEntries {
		name := ce.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if name == "proposals" && ce.IsDir() {
			// 提案期用例区（GOVERNANCE §7）：volatile、不占 contract 版本；
			// validate 暂不校验，「提案期用例反链提案文档」等治理 lint 由后续工票接入。
			continue
		}
		if !ce.IsDir() {
			res.errf("corpus/：意外条目 %q（应为契约目录）", name)
			continue
		}
		if !contractIDRe.MatchString(name) {
			res.errf("corpus/：契约目录名 %q 不合规范（小写字母开头，仅小写字母、数字、连字符）", name)
			continue
		}
		res.Contracts++
		validateContractDir(corpusDir, name, reg, &res)
	}
	return res
}

func validateContractDir(corpusDir, contractID string, reg *codeRegistry, res *Result) {
	entries, err := readSortedEntries(filepath.Join(corpusDir, contractID))
	if err != nil {
		res.errf("corpus/%s：目录读取失败：%v", contractID, err)
		return
	}
	nonDot := 0
	for _, ve := range entries {
		vname := ve.Name()
		if strings.HasPrefix(vname, ".") {
			continue
		}
		nonDot++
		if !ve.IsDir() {
			res.errf("corpus/%s：意外条目 %q（应为版本目录）", contractID, vname)
			continue
		}
		if !versionDirRe.MatchString(vname) {
			res.errf("corpus/%s：非规范版本目录名 %q（规范形 v<主>.<次>，如 v1.0）", contractID, vname)
			continue
		}
		res.Versions++
		validateVersionDir(corpusDir, contractID, vname, reg, res)
	}
	if nonDot == 0 {
		res.errf("corpus/%s：契约目录下没有版本目录", contractID)
	}
}

func validateVersionDir(corpusDir, contractID, version string, reg *codeRegistry, res *Result) {
	dirRel := "corpus/" + contractID + "/" + version
	entries, err := readSortedEntries(filepath.Join(corpusDir, contractID, version))
	if err != nil {
		res.errf("%s：目录读取失败：%v", dirRel, err)
		return
	}

	var moduleFiles []string
	for _, e := range entries {
		fname := e.Name()
		if strings.HasPrefix(fname, ".") {
			continue
		}
		if e.IsDir() || !strings.HasSuffix(fname, ".json") {
			res.errf("%s：意外条目 %q（版本目录下只允许 *.json 模块文件）", dirRel, fname)
			continue
		}
		moduleFiles = append(moduleFiles, fname)
	}
	if len(moduleFiles) == 0 {
		res.errf("%s：版本目录下没有模块文件", dirRel)
		return
	}

	// 用例名唯一的作用域 = 契约版本（跨模块文件；GOVERNANCE §8 治理校验项）。
	seenCaseNames := map[string]string{}
	for _, fname := range moduleFiles {
		res.ModuleFiles++
		fileRel := dirRel + "/" + fname
		data, err := os.ReadFile(filepath.Join(corpusDir, contractID, version, fname))
		if err != nil {
			res.errf("%s：读取失败：%v", fileRel, err)
			continue
		}
		validateModuleFile(fileRel, data, contractID, version, reg, seenCaseNames, res)
	}
}

// corpusMeta 语料 meta：contract 标识 + 出处注记（notes）+ 勘误注记（errata）。
// GOVERNANCE §9：语义注记正文一律迁薄规格，不在 meta 里长语义。
type corpusMeta struct {
	contract string
	errata   []string
	notes    []string
}

func validateModuleFile(relPath string, data []byte, contractID, version string, reg *codeRegistry, seenCaseNames map[string]string, res *Result) {
	doc, err := decodeObject(data, relPath)
	if err != nil {
		res.err(err)
		return
	}
	if err := checkKeys(doc, []string{"meta", "cases"}, nil, relPath); err != nil {
		res.err(err)
		return
	}

	meta, err := decodeMeta(doc["meta"], relPath)
	if err != nil {
		res.err(err)
		return
	}
	// 双写互检（GOVERNANCE §4）：目录名与 meta.contract 字符串精确互为镜像。
	if want := contractID + " " + version; meta.contract != want {
		res.errf("%s：meta.contract %q 与目录名不一致（双写互检，应为 %q）", relPath, meta.contract, want)
	}

	casesRaw, err := decodeArrayField(doc, "cases", relPath)
	if err != nil {
		res.err(err)
		return
	}
	if len(casesRaw) == 0 {
		res.errf("%s：用例集不可为空", relPath)
		return
	}
	for i, caseRaw := range casesRaw {
		validateCase(relPath, i, caseRaw, reg, seenCaseNames, res)
	}
}

func decodeMeta(raw json.RawMessage, relPath string) (corpusMeta, error) {
	var meta corpusMeta
	ctx := relPath + "：meta"
	m, err := decodeObject(raw, ctx)
	if err != nil {
		return meta, err
	}
	if err := checkKeys(m, []string{"contract"}, []string{"contract", "errata", "notes"}, ctx); err != nil {
		return meta, err
	}
	meta.contract, err = decodeStringField(m, "contract", ctx)
	if err != nil {
		return meta, err
	}
	if raw, ok := m["errata"]; ok {
		if meta.errata, err = decodeStringArray(raw, "errata", ctx); err != nil {
			return meta, err
		}
		if len(meta.errata) == 0 {
			return meta, fmt.Errorf("%s：meta.errata 存在时不可为空数组（勘误必须指认被勘误用例与依据，GOVERNANCE §4）", ctx)
		}
	}
	if raw, ok := m["notes"]; ok {
		if meta.notes, err = decodeStringArray(raw, "notes", ctx); err != nil {
			return meta, err
		}
	}
	return meta, nil
}

func validateCase(relPath string, idx int, raw json.RawMessage, reg *codeRegistry, seenCaseNames map[string]string, res *Result) {
	base := fmt.Sprintf("%s：用例 cases[%d]", relPath, idx)
	m, err := decodeObject(raw, base)
	if err != nil {
		res.err(err)
		return
	}
	name, err := decodeStringField(m, "name", base)
	if err != nil {
		res.err(err)
		return
	}
	ctx := fmt.Sprintf("%s（%s）", base, name)

	// 用例名唯一 + 非空（GOVERNANCE §8；唯一作用域 = 契约版本，跨模块文件）。
	if name == "" {
		res.errf("%s：用例名不可为空", ctx)
	} else if prev, dup := seenCaseNames[name]; dup {
		res.errf("%s：用例名重复 %q（与 %s 冲突）", ctx, name, prev)
	} else {
		seenCaseNames[name] = ctx
	}

	if err := checkKeys(m, []string{"name", "description", "template", "expect"}, nil, ctx); err != nil {
		res.err(err)
		return
	}
	if _, err := decodeStringField(m, "description", ctx); err != nil {
		res.err(err)
		return
	}
	if _, err := decodeStringField(m, "template", ctx); err != nil {
		res.err(err)
		return
	}

	expectM, err := decodeObjectField(m, "expect", ctx)
	if err != nil {
		res.err(err)
		return
	}
	if err := checkKeys(expectM, []string{"parts"}, nil, ctx+"：expect"); err != nil {
		res.err(err)
		return
	}
	partsRaw, err := decodeArrayField(expectM, "parts", ctx+"：expect")
	if err != nil {
		res.err(err)
		return
	}
	for j, partRaw := range partsRaw {
		validatePart(ctx, j, partRaw, reg, res)
	}
	res.Cases++
}

// validatePart 逐 kind 钉死 part 键集（与 PHP 侧 scanFixtureExportShape 的逐 kind 键集断言同构）：
// literal{kind,text} | fragment{kind,raw,expr,error} | open{kind,text,expr}。
func validatePart(caseCtx string, idx int, raw json.RawMessage, reg *codeRegistry, res *Result) {
	base := fmt.Sprintf("%s：expect.parts[%d]", caseCtx, idx)
	m, err := decodeObject(raw, base)
	if err != nil {
		res.err(err)
		return
	}
	kind, err := decodeStringField(m, "kind", base)
	if err != nil {
		res.err(err)
		return
	}
	ctx := fmt.Sprintf("%s（%s）", base, kind)

	var required []string
	switch kind {
	case "literal":
		required = []string{"kind", "text"}
	case "open":
		required = []string{"kind", "text", "expr"}
	case "fragment":
		required = []string{"kind", "raw", "expr", "error"}
	default:
		res.errf("%s：未知 part kind %q（合法形态：literal | fragment | open）", base, kind)
		return
	}
	if err := checkKeys(m, required, nil, ctx); err != nil {
		res.err(err)
		return
	}

	if kind == "literal" || kind == "open" {
		if _, err := decodeStringField(m, "text", ctx); err != nil {
			res.err(err)
			return
		}
	}
	if kind == "open" || kind == "fragment" {
		if _, err := decodeStringField(m, "expr", ctx); err != nil {
			res.err(err)
			return
		}
	}
	if kind == "fragment" {
		if _, err := decodeStringField(m, "raw", ctx); err != nil {
			res.err(err)
			return
		}
		code, err := decodeStringOrNullField(m, "error", ctx)
		if err != nil {
			res.err(err)
			return
		}
		// 错误 code 白名单（GOVERNANCE §2.2/§3.4）：只认注册表 active 区；
		// reserved 区 = 已退役，永不复用。注册表加载失败时跳过（已单独报错）。
		if code != nil && reg != nil {
			switch {
			case reg.reserved[*code]:
				res.errf("%s：错误 code %q 已退役保留，不可复用（GOVERNANCE §2.2）", ctx, *code)
			case !reg.codes[*code]:
				res.errf("%s：错误 code %q 不在注册表（code 扩充走提案通道，GOVERNANCE §3.4）", ctx, *code)
			}
		}
	}
}

// ---- 严格解码辅助 ----------------------------------------------------------

// loadRegistry 读取并严格解码错误 code 注册表（工具吃自己的狗粮：声明面同「未知字段即红」纪律）。
func loadRegistry(path string) (*codeRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("错误 code 注册表 spec/error-codes.json 读取失败：%v", err)
	}
	const ctx = "spec/error-codes.json"
	m, err := decodeObject(data, ctx)
	if err != nil {
		return nil, err
	}
	if err := checkKeys(m, []string{"registry", "codes", "reserved"}, nil, ctx); err != nil {
		return nil, err
	}
	ident, err := decodeStringField(m, "registry", ctx)
	if err != nil {
		return nil, err
	}
	if ident != registryIdent {
		return nil, fmt.Errorf("%s：registry 标识 %q 不符（应为 %q；格式演进入新版本字符串，不原地改义）", ctx, ident, registryIdent)
	}
	codes, err := decodeStringArray(m["codes"], "codes", ctx)
	if err != nil {
		return nil, err
	}
	reserved, err := decodeStringArray(m["reserved"], "reserved", ctx)
	if err != nil {
		return nil, err
	}
	reg := &codeRegistry{codes: map[string]bool{}, reserved: map[string]bool{}}
	for _, c := range codes {
		if c == "" {
			return nil, fmt.Errorf("%s：codes 含空字符串", ctx)
		}
		reg.codes[c] = true
	}
	for _, c := range reserved {
		if c == "" {
			return nil, fmt.Errorf("%s：reserved 含空字符串", ctx)
		}
		if reg.codes[c] {
			return nil, fmt.Errorf("%s：code %q 同时出现在 codes 与 reserved（注册表自相矛盾）", ctx, c)
		}
		reg.reserved[c] = true
	}
	return reg, nil
}

func readSortedEntries(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

func decodeObject(data []byte, ctx string) (map[string]json.RawMessage, error) {
	m := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("%s：JSON 解析失败：%v", ctx, err)
	}
	return m, nil
}

// checkKeys 键集检查：required 全在、无 allowed 之外的键（allowed 为 nil 时 = required）。
func checkKeys(m map[string]json.RawMessage, required, allowed []string, ctx string) error {
	for _, k := range required {
		if _, ok := m[k]; !ok {
			return fmt.Errorf("%s：缺少键 %q", ctx, k)
		}
	}
	if allowed == nil {
		allowed = required
	}
	okSet := make(map[string]bool, len(allowed))
	for _, k := range allowed {
		okSet[k] = true
	}
	for k := range m {
		if !okSet[k] {
			return fmt.Errorf("%s：未知键 %q（严格解码：未知字段即红）", ctx, k)
		}
	}
	return nil
}

func decodeStringField(m map[string]json.RawMessage, key, ctx string) (string, error) {
	raw, ok := m[key]
	if !ok {
		return "", fmt.Errorf("%s：缺少键 %q", ctx, key)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", fmt.Errorf("%s：键 %q 应为字符串", ctx, key)
	}
	return s, nil
}

func decodeStringOrNullField(m map[string]json.RawMessage, key, ctx string) (*string, error) {
	raw, ok := m[key]
	if !ok {
		return nil, fmt.Errorf("%s：缺少键 %q", ctx, key)
	}
	if string(raw) == "null" {
		return nil, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("%s：键 %q 应为字符串或 null", ctx, key)
	}
	return &s, nil
}

func decodeObjectField(m map[string]json.RawMessage, key, ctx string) (map[string]json.RawMessage, error) {
	raw, ok := m[key]
	if !ok {
		return nil, fmt.Errorf("%s：缺少键 %q", ctx, key)
	}
	return decodeObject(raw, ctx+"："+key)
}

func decodeArrayField(m map[string]json.RawMessage, key, ctx string) ([]json.RawMessage, error) {
	raw, ok := m[key]
	if !ok {
		return nil, fmt.Errorf("%s：缺少键 %q", ctx, key)
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("%s：键 %q 应为数组", ctx, key)
	}
	return arr, nil
}

func decodeStringArray(raw json.RawMessage, key, ctx string) ([]string, error) {
	var arr []string
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("%s：键 %q 应为字符串数组", ctx, key)
	}
	return arr, nil
}
