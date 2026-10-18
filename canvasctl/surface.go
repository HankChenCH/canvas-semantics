// 支持面声明校验（SUPPORT-SURFACE.md）：实现仓 CI 拉取（submodule）语料仓后，
// 以 canvasctl validate --surface 校验自身 support-surface.json 的三互检——
// 形状（严格 typed 解码，未知字段即红）、注册表（契约 id / 模块文件 id 必须在
// 声明钉住的语料版本目录内）、版本（声明版本目录必须存在）。注册表的机器权威 =
// 语料目录本身（SUPPORT-SURFACE §2：模块文件集合以该版本语料目录的实际文件为准）。
//
// 声明是承诺（§4.3）：supported 条目的 runner 套件由实现仓 CI 保证全绿，本工具只查
// 声明与语料的一致性，不跑 runner；缺席 = 不支持（§4.2），未列出的契约/模块不校验。

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// surfaceIdent 支持面声明格式标识：格式演进入新版本字符串，不原地改义。
const surfaceIdent = "canvas-semantics/support-surface v1"

// ValidateSupportSurface 校验 surfacePath 声明文件与 root 语料注册表的三互检，
// 返回独立 Result（错误与计数；由 CLI 与语料 validate 汇总）。
func ValidateSupportSurface(root, surfacePath string) Result {
	var res Result

	data, err := os.ReadFile(surfacePath)
	if err != nil {
		res.errf("支持面声明 %s 读取失败：%v", surfacePath, err)
		return res
	}
	doc, err := decodeObject(data, surfacePath)
	if err != nil {
		res.err(err)
		return res
	}
	if err := checkKeys(doc, []string{"declaration", "implementation", "contracts"}, nil, surfacePath); err != nil {
		res.err(err)
		return res
	}

	ident, err := decodeStringField(doc, "declaration", surfacePath)
	if err != nil {
		res.err(err)
		return res
	}
	if ident != surfaceIdent {
		res.errf("%s：declaration 标识 %q 不符（应为 %q；格式演进入新版本字符串，不原地改义）", surfacePath, ident, surfaceIdent)
	}

	validateImplementation(doc["implementation"], surfacePath, &res)
	validateDeclaredContracts(root, doc["contracts"], surfacePath, &res)
	return res
}

// validateImplementation 实现画像块：四键全字符串，repo/language/profile 非空
// （profile 是说明性画像、不参与校验语义，SUPPORT-SURFACE §3 字段表）。
func validateImplementation(raw json.RawMessage, ctx string, res *Result) {
	m, err := decodeObject(raw, ctx+"：implementation")
	if err != nil {
		res.err(err)
		return
	}
	if err := checkKeys(m, []string{"repo", "language", "profile", "note"}, nil, ctx+"：implementation"); err != nil {
		res.err(err)
		return
	}
	for _, k := range []string{"repo", "language", "profile"} {
		v, err := decodeStringField(m, k, ctx+"：implementation")
		if err != nil {
			res.err(err)
			continue
		}
		if v == "" {
			res.errf("%s：implementation.%s 不可为空串", ctx, k)
		}
	}
	if _, err := decodeStringField(m, "note", ctx+"：implementation"); err != nil {
		res.err(err)
	}
}

// validateDeclaredContracts 契约条目数组：逐条与语料目录三互检；数组不可为空
// （声明是支持承诺，零契约的签署无意义）。
func validateDeclaredContracts(root string, raw json.RawMessage, ctx string, res *Result) {
	entries, err := decodeArrayRaw(raw, ctx+"：contracts")
	if err != nil {
		res.errf("%s：键 %q 应为数组", ctx, "contracts")
		return
	}
	if len(entries) == 0 {
		res.errf("%s：contracts 不可为空数组（声明是支持承诺，缺席 = 不支持，无需签署）", ctx)
		return
	}

	seenContracts := map[string]string{}
	for i, entryRaw := range entries {
		base := fmt.Sprintf("%s：contracts[%d]", ctx, i)
		m, err := decodeObject(entryRaw, base)
		if err != nil {
			res.err(err)
			continue
		}
		if err := checkKeys(m, []string{"id", "version", "moduleFiles"}, nil, base); err != nil {
			res.err(err)
			continue
		}

		id, err := decodeStringField(m, "id", base)
		if err != nil {
			res.err(err)
			continue
		}
		entryCtx := fmt.Sprintf("%s（%s）", base, id)
		if !contractIDRe.MatchString(id) {
			res.errf("%s：契约 id %q 不合规范（小写字母开头，仅小写字母、数字、连字符）", entryCtx, id)
			continue
		}
		if prev, dup := seenContracts[id]; dup {
			res.errf("%s：契约 %q 重复声明（与 %s 冲突）", entryCtx, id, prev)
			continue
		}
		seenContracts[id] = entryCtx

		if _, err := os.Stat(filepath.Join(root, "corpus", id)); err != nil {
			res.errf("%s：契约 %q 不在语料注册表（corpus/ 下无此目录；新契约走提案转正 + 注册表登记后接入）", entryCtx, id)
			continue
		}

		version, err := decodeStringField(m, "version", entryCtx)
		if err != nil {
			res.err(err)
			continue
		}
		if !versionDirRe.MatchString(version) {
			res.errf("%s：版本 %q 非规范形 v<主>.<次>（GOVERNANCE §4）", entryCtx, version)
			continue
		}
		if _, err := os.Stat(filepath.Join(root, "corpus", id, version)); err != nil {
			res.errf("%s：声明版本 %q 在语料 %s 下不存在（声明须与 submodule pin 的语料一致）", entryCtx, version, id)
			continue
		}

		validateDeclaredModuleFiles(root, id, version, m["moduleFiles"], entryCtx, res)
		res.SurfaceContracts++
	}
}

// validateDeclaredModuleFiles 模块条目数组：id 必须是声明版本目录内的实际模块文件
// （含 .json 后缀），support 两值枚举，契约内不可重复；缺席 = 不支持（§4.2），
// 未列出的模块文件不在校验面。
func validateDeclaredModuleFiles(root, contractID, version string, raw json.RawMessage, ctx string, res *Result) {
	base := ctx + "：moduleFiles"
	entries, err := decodeArrayRaw(raw, base)
	if err != nil {
		res.errf("%s 应为数组", base)
		return
	}
	if len(entries) == 0 {
		res.errf("%s：moduleFiles 不可为空数组（该契约不在支持面则整条契约不应出现在声明中）", base)
		return
	}

	seenModules := map[string]string{}
	for i, entryRaw := range entries {
		entryBase := fmt.Sprintf("%s[%d]", base, i)
		m, err := decodeObject(entryRaw, entryBase)
		if err != nil {
			res.err(err)
			continue
		}
		if err := checkKeys(m, []string{"id", "support", "note"}, nil, entryBase); err != nil {
			res.err(err)
			continue
		}

		id, err := decodeStringField(m, "id", entryBase)
		if err != nil {
			res.err(err)
			continue
		}
		entryCtx := fmt.Sprintf("%s（%s）", entryBase, id)
		if prev, dup := seenModules[id]; dup {
			res.errf("%s：模块文件 %q 重复声明（与 %s 冲突）", entryCtx, id, prev)
			continue
		}
		seenModules[id] = entryCtx

		if _, err := os.Stat(filepath.Join(root, "corpus", contractID, version, id+".json")); err != nil {
			res.errf("%s：模块文件 %q 在语料 %s %s 下不存在（模块 id = 语料目录内实际模块文件名）", entryCtx, id, contractID, version)
			continue
		}

		support, err := decodeStringField(m, "support", entryCtx)
		if err != nil {
			res.err(err)
			continue
		}
		if support != "supported" && support != "unsupported" {
			res.errf("%s：support %q 非法（仅 supported / unsupported 两值，SUPPORT-SURFACE §3）", entryCtx, support)
			continue
		}
		if _, err := decodeStringField(m, "note", entryCtx); err != nil {
			res.err(err)
			continue
		}
		res.SurfaceModuleFiles++
	}
}
