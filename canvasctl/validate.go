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
	"strconv"
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
		if name == "README.md" && !ce.IsDir() {
			// 语料编写规范文档（非语料），与点条目同待遇。
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
		validateModuleFile(fileRel, data, contractID, version, fname, reg, seenCaseNames, res)
	}
}

// corpusMeta 语料 meta：contract 标识 + 出处注记（notes）+ 勘误注记（errata）。
// GOVERNANCE §9：语义注记正文一律迁薄规格，不在 meta 里长语义。
type corpusMeta struct {
	contract string
	errata   []string
	notes    []string
}

// contractSchema 契约级语料形状：用例顶层键集与用例校验器按契约分派
// （模块注册表五契约；SUPPORT-SURFACE.md §2）。
type contractSchema struct {
	caseKeys     []string
	validateCase func(relPath string, idx int, raw json.RawMessage, reg *codeRegistry, seenCaseNames map[string]string, res *Result)
	// moduleExtraKeys 模块文件名 → 文档级额外允许键（严格解码，未列模块出现即红）。
	moduleExtraKeys map[string][]string
}

// 五契约的用例顶层键集（严格解码，未知字段即红）。
var (
	scanCaseKeys = []string{"name", "description", "template", "expect"}
	evalCaseKeys = []string{"name", "description", "template", "context", "expect"}
	// 管线三契约（工票 04）：图 payload（graph/frames/dataset）为不透明 wire 数据，
	// validate 只钉信封键集与 expect 形状——语料故意携带「实现必须拒绝/透传」的负形态。
	expandCaseKeys   = []string{"name", "description", "graph", "dataset", "expect"}
	paginateCaseKeys = []string{"name", "description", "graph", "options", "expect"}
	flowCaseKeys     = []string{"name", "description", "frames", "dataset", "flowChain", "expect"}
)

var contractSchemas = map[string]*contractSchema{
	"expression-scan": {
		caseKeys:     scanCaseKeys,
		validateCase: validateScanCase,
	},
	"expression-eval": {
		caseKeys:     evalCaseKeys,
		validateCase: validateEvalCase,
		// dateTokens 转译表随语料发布（GOVERNANCE §3.3），钉在日期模块文件一处。
		moduleExtraKeys: map[string][]string{"fn-date.json": {"dateTokens"}},
	},
	"expand-semantics": {
		caseKeys:     expandCaseKeys,
		validateCase: validateExpandCase,
	},
	"paginate-semantics": {
		caseKeys:     paginateCaseKeys,
		validateCase: validatePaginateCase,
	},
	"flow-semantics": {
		caseKeys:     flowCaseKeys,
		validateCase: validateFlowCase,
	},
}

func validateModuleFile(relPath string, data []byte, contractID, version, fname string, reg *codeRegistry, seenCaseNames map[string]string, res *Result) {
	schema, ok := contractSchemas[contractID]
	if !ok {
		res.errf("%s：未知契约 %q（validate 无其形状定义；新契约走提案转正 + 注册表登记后接入）", relPath, contractID)
		return
	}

	doc, err := decodeObject(data, relPath)
	if err != nil {
		res.err(err)
		return
	}
	docKeys := append([]string{"meta", "cases"}, schema.moduleExtraKeys[fname]...)
	if err := checkKeys(doc, docKeys, nil, relPath); err != nil {
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

	for _, extra := range schema.moduleExtraKeys[fname] {
		if raw, ok := doc[extra]; ok {
			validateDateTokens(raw, relPath+"："+extra, res)
		}
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
		schema.validateCase(relPath, i, caseRaw, reg, seenCaseNames, res)
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

// caseIdentity 用例公共前置：解码对象、取非空名、钉用例名唯一（作用域 = 契约版本，跨模块文件）。
// 返回 nil 对象时错误已记，调用方直接返回。
func caseIdentity(relPath string, idx int, raw json.RawMessage, seenCaseNames map[string]string, res *Result) (map[string]json.RawMessage, string) {
	base := fmt.Sprintf("%s：用例 cases[%d]", relPath, idx)
	m, err := decodeObject(raw, base)
	if err != nil {
		res.err(err)
		return nil, ""
	}
	name, err := decodeStringField(m, "name", base)
	if err != nil {
		res.err(err)
		return nil, ""
	}
	ctx := fmt.Sprintf("%s（%s）", base, name)

	if name == "" {
		res.errf("%s：用例名不可为空", ctx)
	} else if prev, dup := seenCaseNames[name]; dup {
		res.errf("%s：用例名重复 %q（与 %s 冲突）", ctx, name, prev)
	} else {
		seenCaseNames[name] = ctx
	}

	return m, ctx
}

// checkErrorCode 错误 code 白名单（GOVERNANCE §2.2/§3.4）：只认注册表 active 区；
// reserved 区 = 已退役，永不复用。注册表加载失败时跳过（已单独报错）。
func checkErrorCode(ctx, code string, reg *codeRegistry, res *Result) {
	if reg == nil {
		return
	}
	switch {
	case reg.reserved[code]:
		res.errf("%s：错误 code %q 已退役保留，不可复用（GOVERNANCE §2.2）", ctx, code)
	case !reg.codes[code]:
		res.errf("%s：错误 code %q 不在注册表（code 扩充走提案通道，GOVERNANCE §3.4）", ctx, code)
	}
}

// validateScanCase expression-scan 用例形状：expect.parts 逐 kind 切分产物。
func validateScanCase(relPath string, idx int, raw json.RawMessage, reg *codeRegistry, seenCaseNames map[string]string, res *Result) {
	m, ctx := caseIdentity(relPath, idx, raw, seenCaseNames, res)
	if m == nil {
		return
	}

	if err := checkKeys(m, scanCaseKeys, nil, ctx); err != nil {
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

// validateEvalCase expression-eval 用例形状：context 求值上下文 + expect 字符串或 {errorCode}。
func validateEvalCase(relPath string, idx int, raw json.RawMessage, reg *codeRegistry, seenCaseNames map[string]string, res *Result) {
	m, ctx := caseIdentity(relPath, idx, raw, seenCaseNames, res)
	if m == nil {
		return
	}

	if err := checkKeys(m, evalCaseKeys, nil, ctx); err != nil {
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
	// context 形状 = 任意 JSON 对象（行数据/根键值不作约束，值域由用例自钉）。
	var ctxObj map[string]json.RawMessage
	if err := json.Unmarshal(m["context"], &ctxObj); err != nil {
		res.errf("%s：键 %q 应为对象", ctx, "context")
		return
	}

	expectRaw, ok := m["expect"]
	if !ok {
		res.errf("%s：缺少键 %q", ctx, "expect")
		return
	}
	var strExpect string
	if err := json.Unmarshal(expectRaw, &strExpect); err == nil {
		res.Cases++
		return
	}
	expectM, err := decodeObject(expectRaw, ctx+"：expect")
	if err != nil {
		res.errf("%s：expect 应为字符串或 {errorCode} 对象", ctx)
		return
	}
	if err := checkKeys(expectM, []string{"errorCode"}, nil, ctx+"：expect"); err != nil {
		res.err(err)
		return
	}
	code, err := decodeStringField(expectM, "errorCode", ctx+"：expect")
	if err != nil {
		res.err(err)
		return
	}
	checkErrorCode(ctx+"：expect", code, reg, res)
	res.Cases++
}

// validateExpandCase expand-semantics 用例形状：graph + dataset（任意 JSON，含 null）
// + expect = {graph: 对象} | {errorCode}。
func validateExpandCase(relPath string, idx int, raw json.RawMessage, reg *codeRegistry, seenCaseNames map[string]string, res *Result) {
	m, ctx := caseIdentity(relPath, idx, raw, seenCaseNames, res)
	if m == nil {
		return
	}

	if err := checkKeys(m, expandCaseKeys, nil, ctx); err != nil {
		res.err(err)
		return
	}
	if _, err := decodeStringField(m, "description", ctx); err != nil {
		res.err(err)
		return
	}
	if err := requireObjectField(m, "graph", ctx); err != nil {
		res.err(err)
		return
	}
	// dataset 形状 = 任意 JSON（对象/null/标量不作约束，值域由用例自钉）。
	validatePipelineExpect(ctx, m["expect"], "graph", false, true, reg, res)
	res.Cases++
}

// validatePaginateCase paginate-semantics 用例形状：graph + options（[] 空数组 = 无选项，
// PHP 空关联数组 wire 形态；对象形态键集 {pageHeight?, truncate?} 严格解码）
// + expect = {pages: [页 graph…]} | {errorCode}。
func validatePaginateCase(relPath string, idx int, raw json.RawMessage, reg *codeRegistry, seenCaseNames map[string]string, res *Result) {
	m, ctx := caseIdentity(relPath, idx, raw, seenCaseNames, res)
	if m == nil {
		return
	}

	if err := checkKeys(m, paginateCaseKeys, nil, ctx); err != nil {
		res.err(err)
		return
	}
	if _, err := decodeStringField(m, "description", ctx); err != nil {
		res.err(err)
		return
	}
	if err := requireObjectField(m, "graph", ctx); err != nil {
		res.err(err)
		return
	}
	if err := validatePaginateOptions(m["options"], ctx+"：options", res); err != nil {
		res.err(err)
		return
	}
	validatePipelineExpect(ctx, m["expect"], "pages", true, false, reg, res)
	res.Cases++
}

// validatePaginateOptions options 信封形状：[] （空数组，与无选项同义）| 对象
// {pageHeight?: 整数, truncate?: 布尔}（严格解码，未知键即红；对象内容语义归实现）。
func validatePaginateOptions(raw json.RawMessage, ctx string, res *Result) error {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "[]" {
		return nil
	}
	m, err := decodeObject(raw, ctx)
	if err != nil {
		return fmt.Errorf("%s 应为空数组或对象（[] = 无选项）", ctx)
	}
	if err := checkKeys(m, nil, []string{"pageHeight", "truncate"}, ctx); err != nil {
		return err
	}
	if raw, ok := m["pageHeight"]; ok {
		if _, err := decodeIntField(raw, "pageHeight", ctx); err != nil {
			return err
		}
	}
	if raw, ok := m["truncate"]; ok {
		if err := decodeBoolField(raw, "truncate", ctx); err != nil {
			return err
		}
	}
	return nil
}

// validateFlowCase flow-semantics 用例形状：frames（模板态声明 graph 数组，元素皆对象、不可为空）
// + dataset（任意 JSON）+ flowChain（null 与显式 [] 同义；节点骨架 frame(int)+mode(string) 必出、
// quota(int)/omitIfEmpty(bool) 类型钉死、未知键放行——语料携带实现必须拒绝的负形态）
// + expect = {canvases: [canvas graph…]} | {errorCode}。
func validateFlowCase(relPath string, idx int, raw json.RawMessage, reg *codeRegistry, seenCaseNames map[string]string, res *Result) {
	m, ctx := caseIdentity(relPath, idx, raw, seenCaseNames, res)
	if m == nil {
		return
	}

	if err := checkKeys(m, flowCaseKeys, nil, ctx); err != nil {
		res.err(err)
		return
	}
	if _, err := decodeStringField(m, "description", ctx); err != nil {
		res.err(err)
		return
	}
	framesRaw, err := decodeArrayField(m, "frames", ctx)
	if err != nil {
		res.err(err)
		return
	}
	if len(framesRaw) == 0 {
		// frames 空数组 = 实现端入口守卫（frames_empty，调用方编程错误、不立锚点 code，
		// 裁决见 spec/pipeline.md §2/§4）——语义不可由语料钉（错误断言只落 code），语料不收此形态。
		res.errf("%s：frames 不可为空数组（空 frames 是实现端入口守卫，不入语料面）", ctx)
		return
	}
	for i, frameRaw := range framesRaw {
		if _, err := decodeObject(frameRaw, fmt.Sprintf("%s：frames[%d]", ctx, i)); err != nil {
			res.errf("%s：frames[%d] 应为对象", ctx, i)
		}
	}
	// dataset 形状 = 任意 JSON（与 expand 同款：payload 面，值域由用例自钉）。
	validateFlowChain(m["flowChain"], ctx+"：flowChain", res)
	validatePipelineExpect(ctx, m["expect"], "canvases", true, true, reg, res)
	res.Cases++
}

// validateFlowChain 节点骨架校验：frame 整数 + mode 字符串必出；quota 整数、omitIfEmpty
// 布尔出则类型钉死；未知键放行（payload 面负形态，钉例 flow:unknown-key-rejected）。
// 枚举闭面（mode ∈ fixed|paged）、quota ≥ 0、frame 越界等语义校验归实现（flow_chain_invalid），
// 语料负形态必须可达——validate 不预演语义裁决（工具非规范性，GOVERNANCE §8）。
func validateFlowChain(raw json.RawMessage, ctx string, res *Result) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "null" || trimmed == "[]" {
		return
	}
	nodes, err := decodeArrayRaw(raw, ctx)
	if err != nil {
		res.err(err)
		return
	}
	for i, nodeRaw := range nodes {
		nodeCtx := fmt.Sprintf("%s[%d]", ctx, i)
		node, err := decodeObject(nodeRaw, nodeCtx)
		if err != nil {
			res.errf("%s 应为对象", nodeCtx)
			continue
		}
		if raw, ok := node["frame"]; ok {
			if _, err := decodeIntField(raw, "frame", nodeCtx); err != nil {
				res.err(err)
			}
		} else {
			res.errf("%s：缺少键 %q", nodeCtx, "frame")
		}
		if _, ok := node["mode"]; ok {
			if _, err := decodeStringField(node, "mode", nodeCtx); err != nil {
				res.err(err)
			}
		} else {
			res.errf("%s：缺少键 %q", nodeCtx, "mode")
		}
		if raw, ok := node["quota"]; ok {
			if _, err := decodeIntField(raw, "quota", nodeCtx); err != nil {
				res.err(err)
			}
		}
		if raw, ok := node["omitIfEmpty"]; ok {
			if err := decodeBoolField(raw, "omitIfEmpty", nodeCtx); err != nil {
				res.err(err)
			}
		}
	}
}

// validatePipelineExpect 管线三契约共享的 expect 形状：{<payloadKey>: …, warnings?} |
// {errorCode} 二选一。payloadIsArray = false 要求载荷为对象（填充产物单图），= true
// 要求对象数组（页/画布序列，允许空数组——零产物用例形态留口）。errorCode 走注册表
// 白名单。allowWarnings = 该契约 expect 允携带软诊断期望序列 warnings（工票 05：
// expand/flow 有诊断发射面，paginate 输入为 hydrate 产物、无发射面即不允许）。
func validatePipelineExpect(caseCtx string, raw json.RawMessage, payloadKey string, payloadIsArray bool, allowWarnings bool, reg *codeRegistry, res *Result) {
	ctx := caseCtx + "：expect"
	m, err := decodeObject(raw, ctx)
	if err != nil {
		res.errf("%s 应为 {%s} 或 {errorCode} 对象", ctx, payloadKey)
		return
	}
	_, hasPayload := m[payloadKey]
	code, hasCode := m["errorCode"]
	if hasPayload == hasCode {
		res.errf("%s 应为 {%s} 或 {errorCode} 二选一", ctx, payloadKey)
		return
	}
	if hasCode {
		if err := checkKeys(m, []string{"errorCode"}, nil, ctx); err != nil {
			res.err(err)
			return
		}
		var s string
		if err := json.Unmarshal(code, &s); err != nil {
			res.errf("%s：键 %q 应为字符串", ctx, "errorCode")
			return
		}
		checkErrorCode(ctx, s, reg, res)
		return
	}
	// warnings = 期望诊断序列（软诊断语料面，GOVERNANCE §3.4）：可选伴随键，
	// 省略 = 空序列；条目断言面只落 code（注册表白名单）+ path（中立记法），
	// 消息人读可本地化、不入断言面（spec/pipeline.md §4/§5）。
	allowed := []string{payloadKey}
	if allowWarnings {
		allowed = append(allowed, "warnings")
	}
	if err := checkKeys(m, []string{payloadKey}, allowed, ctx); err != nil {
		res.err(err)
		return
	}
	if raw, ok := m["warnings"]; ok {
		validateWarnings(raw, ctx, reg, res)
	}
	if !payloadIsArray {
		if err := requireObjectField(m, payloadKey, ctx); err != nil {
			res.err(err)
		}
		return
	}
	items, err := decodeArrayRaw(m[payloadKey], fmt.Sprintf("%s：键 %q", ctx, payloadKey))
	if err != nil {
		res.errf("%s：键 %q 应为数组", ctx, payloadKey)
		return
	}
	for i, item := range items {
		if _, err := decodeObject(item, fmt.Sprintf("%s.%s[%d]", ctx, payloadKey, i)); err != nil {
			res.errf("%s.%s[%d] 应为对象", ctx, payloadKey, i)
		}
	}
}

// validateWarnings 期望诊断序列形状：逐条目 {code: 字符串（注册表白名单）,
// path?: 字符串（中立记法，语义归 runner 比对）}；序列序 = 期望发射序。
func validateWarnings(raw json.RawMessage, expectCtx string, reg *codeRegistry, res *Result) {
	ctx := expectCtx + "：warnings"
	items, err := decodeArrayRaw(raw, ctx)
	if err != nil {
		res.errf("%s 应为数组", ctx)
		return
	}
	for i, item := range items {
		itemCtx := fmt.Sprintf("%s[%d]", ctx, i)
		m, err := decodeObject(item, itemCtx)
		if err != nil {
			res.errf("%s 应为对象", itemCtx)
			continue
		}
		if err := checkKeys(m, []string{"code"}, []string{"code", "path"}, itemCtx); err != nil {
			res.err(err)
			continue
		}
		code, err := decodeStringField(m, "code", itemCtx)
		if err != nil {
			res.err(err)
			continue
		}
		checkErrorCode(itemCtx, code, reg, res)
		if _, ok := m["path"]; ok {
			if _, err := decodeStringField(m, "path", itemCtx); err != nil {
				res.err(err)
			}
		}
	}
}

// validateDateTokens dateTokens 转译表（仅 expression-eval 的 fn-date.json）：moment 风格
// token 逐行自含语义，sample 钉死样例时刻（GOVERNANCE §3.3，权威数据随语料发布）。
func validateDateTokens(raw json.RawMessage, ctx string, res *Result) {
	m, err := decodeObject(raw, ctx)
	if err != nil {
		res.err(err)
		return
	}
	if err := checkKeys(m, []string{"sample", "rows"}, nil, ctx); err != nil {
		res.err(err)
		return
	}
	if _, err := decodeStringField(m, "sample", ctx); err != nil {
		res.err(err)
		return
	}
	rowsRaw, err := decodeArrayField(m, "rows", ctx)
	if err != nil {
		res.err(err)
		return
	}
	if len(rowsRaw) == 0 {
		res.errf("%s：rows 不可为空", ctx)
		return
	}
	for i, rowRaw := range rowsRaw {
		rowCtx := fmt.Sprintf("%s：rows[%d]", ctx, i)
		rowM, err := decodeObject(rowRaw, rowCtx)
		if err != nil {
			res.err(err)
			continue
		}
		if err := checkKeys(rowM, []string{"token", "semantics", "expect"}, nil, rowCtx); err != nil {
			res.err(err)
			continue
		}
		if _, err := decodeStringField(rowM, "token", rowCtx); err != nil {
			res.err(err)
			continue
		}
		if _, err := decodeStringField(rowM, "semantics", rowCtx); err != nil {
			res.err(err)
			continue
		}
		if _, err := decodeStringField(rowM, "expect", rowCtx); err != nil {
			res.err(err)
			continue
		}
	}
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
		if code != nil {
			checkErrorCode(ctx, *code, reg, res)
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

// intLiteralRe 整数字面量（十进制，无小数/指数；字符串形态 "0" 不合语料整数惯例即红）。
var intLiteralRe = regexp.MustCompile(`^-?[0-9]+$`)

// decodeIntField 整数字段：只认 JSON number 的十进制整数字面量。
func decodeIntField(raw json.RawMessage, key, ctx string) (int64, error) {
	s := strings.TrimSpace(string(raw))
	if !intLiteralRe.MatchString(s) {
		return 0, fmt.Errorf("%s：键 %q 应为整数", ctx, key)
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s：键 %q 应为整数", ctx, key)
	}
	return v, nil
}

// decodeArrayRaw 原始 JSON 数组解码（ctx 为该数组位的定位描述）。
func decodeArrayRaw(raw json.RawMessage, ctx string) ([]json.RawMessage, error) {
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("%s 应为数组", ctx)
	}
	return arr, nil
}

// decodeBoolField 布尔字段。
func decodeBoolField(raw json.RawMessage, key, ctx string) error {
	var b bool
	if err := json.Unmarshal(raw, &b); err != nil {
		return fmt.Errorf("%s：键 %q 应为布尔", ctx, key)
	}
	return nil
}

// requireObjectField 键存在且为 JSON 对象（数组/标量即红，消息统一「键 %q 应为对象」）。
func requireObjectField(m map[string]json.RawMessage, key, ctx string) error {
	raw, ok := m[key]
	if !ok {
		return fmt.Errorf("%s：缺少键 %q", ctx, key)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return fmt.Errorf("%s：键 %q 应为对象", ctx, key)
	}
	return nil
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
