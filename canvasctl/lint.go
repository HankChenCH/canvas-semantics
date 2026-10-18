// 治理 lint（GOVERNANCE §8）：提案期用例 ↔ 提案文档反链互检。
//
// 两态提案生命周期（GOVERNANCE §7、proposals/README）钉死的反链规则：
//
//   - 提案期用例目录 corpus/proposals/<编号>-<slug>/ 须有提案文档
//     proposals/<编号>-<slug>.md，且文档头部须反链该目录（正文中出现目录路径）；
//   - 状态为 proposal 的提案文档须有用例目录——除非「提案期用例」行显式写明待补
//     （GOVERNANCE §7「用例已出（或声明待补）」）；
//   - 用例目录不可为空（至少一个 *.json 用例文件）。
//
// 状态 integrated 的提案文档不要求用例目录（用例已迁入正式契约或声明未单建，
// 文档保留为历史）。工具非规范性：只执行已钉死的反链规则，不裁提案内容。

package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// 提案编号目录/文档名：三位起步编号顺延（自 001）+ 小写 slug。
	proposalIDRe = regexp.MustCompile(`^[0-9]{3,}-[a-z0-9-]+$`)
	// 文档状态行（模板：`- 状态：proposal ｜ integrated（…）`，取首个分隔符前的状态词）。
	statusLineRe  = regexp.MustCompile(`(?m)^\s*[-*]?\s*状态：(.*)$`)
	pendingLineRe = regexp.MustCompile(`(?m)^\s*[-*]?\s*提案期用例：`)
)

// Lint 校验 root（canvas-semantics 仓根）下提案期用例与提案文档的反链互检。
func Lint(root string) Result {
	var res Result

	// 方向一：提案文档 → 用例目录（状态 proposal 者须有目录或显式待补）。
	docsDir := filepath.Join(root, "proposals")
	docEntries, err := readSortedEntries(docsDir)
	if err != nil {
		// proposals/ 缺失 = 当前无提案文档（如最小仓），与 corpus/proposals/ 同待遇。
		if !os.IsNotExist(err) {
			res.errf("proposals/ 目录读取失败：%v", err)
		}
	} else {
		for _, e := range docEntries {
			name := e.Name()
			switch {
			case strings.HasPrefix(name, "."):
				continue
			case name == "README.md" && !e.IsDir():
				continue // 提案索引（生命周期细则），非提案文档
			case name == "0000-template.md":
				continue // 模板，编号 0000 保留、不参与生命周期
			}
			if e.IsDir() || !strings.HasSuffix(name, ".md") || !proposalIDRe.MatchString(strings.TrimSuffix(name, ".md")) {
				res.errf("proposals/：意外条目 %q（应为 <编号>-<slug>.md 提案文档，编号三位起步、slug 仅小写字母/数字/连字符）", name)
				continue
			}
			res.ProposalDocs++
			lintProposalDoc(root, docsDir, strings.TrimSuffix(name, ".md"), &res)
		}
	}

	// 方向二：提案期用例目录 → 提案文档（目录须有文档、须非空）。
	// 语料区可选：corpus/proposals/ 不存在 = 当前无提案期用例，合法。
	caseDirs := filepath.Join(root, "corpus", "proposals")
	dirEntries, err := readSortedEntries(caseDirs)
	if err != nil {
		if !os.IsNotExist(err) {
			res.errf("corpus/proposals/ 目录读取失败：%v", err)
		}
		return res
	}
	for _, e := range dirEntries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !e.IsDir() || !proposalIDRe.MatchString(name) {
			res.errf("corpus/proposals/：意外条目 %q（应为 <编号>-<slug>/ 提案期用例目录）", name)
			continue
		}
		res.ProposalCaseDirs++
		lintProposalCaseDir(root, name, &res)
	}
	return res
}

// lintProposalDoc 方向一：状态解析 + proposal 态的用例目录存在性（或显式待补）。
func lintProposalDoc(root, docsDir, id string, res *Result) {
	docRel := "proposals/" + id + ".md"
	data, err := os.ReadFile(filepath.Join(docsDir, id+".md"))
	if err != nil {
		res.errf("%s：读取失败：%v", docRel, err)
		return
	}
	body := string(data)

	status := proposalStatus(body, docRel, res)
	dirExists := dirExists(filepath.Join(root, "corpus", "proposals", id))

	if status == "proposal" && !dirExists && !declaresCasesPending(body) {
		res.errf("%s：状态 proposal 须有提案期用例目录 corpus/proposals/%s/（或「提案期用例」行写明待补，GOVERNANCE §7）", docRel, id)
	}
	// 反链（proposals/README：提案文档头部必须反链其用例目录）：目录存在即要求
	// 文档出现目录路径——含 status integrated 但目录尚存（迁走前的历史形态）。
	if dirExists && !strings.Contains(body, "corpus/proposals/"+id) {
		res.errf("%s：提案期用例目录 corpus/proposals/%s/ 存在，但文档未反链该目录路径（GOVERNANCE §7）", docRel, id)
	}
}

// lintProposalCaseDir 方向二：用例目录须有提案文档、须非空。
func lintProposalCaseDir(root, id string, res *Result) {
	dirRel := "corpus/proposals/" + id
	docRel := "proposals/" + id + ".md"
	if _, err := os.Stat(filepath.Join(root, docRel)); err != nil {
		res.errf("%s/：提案期用例目录无对应提案文档 %s（GOVERNANCE §7 反链）", dirRel, docRel)
	}

	entries, err := os.ReadDir(filepath.Join(root, "corpus", "proposals", id))
	if err != nil {
		res.errf("%s/：目录读取失败：%v", dirRel, err)
		return
	}
	hasCase := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			res.errf("%s/：意外条目 %q（提案期用例目录下只允许 *.json 用例文件）", dirRel, e.Name())
			continue
		}
		hasCase = true
	}
	if !hasCase {
		res.errf("%s/：提案期用例目录下没有用例文件（空目录；提不出用例的语义改动不成立，GOVERNANCE §7）", dirRel)
	}
}

// proposalStatus 解析文档状态行：取「状态：」后、首个分隔符（｜/（/(）前的状态词。
// 只认 proposal / integrated 两态（GOVERNANCE §7）；缺行或不可识别即红。
func proposalStatus(body, docRel string, res *Result) string {
	m := statusLineRe.FindStringSubmatch(body)
	if m == nil {
		res.errf("%s：缺少「状态：」行（模板 0000-template.md 头部键行）", docRel)
		return ""
	}
	value := strings.TrimSpace(m[1])
	if idx := strings.IndexAny(value, "｜（("); idx >= 0 {
		value = value[:idx]
	}
	value = strings.TrimSpace(value)
	if value != "proposal" && value != "integrated" {
		res.errf("%s：状态 %q 不可识别（两态生命周期：proposal | integrated）", docRel, value)
		return ""
	}
	return value
}

// declaresCasesPending 「提案期用例」行是否显式写明待补（GOVERNANCE §7「用例已出（或声明待补）」）。
func declaresCasesPending(body string) bool {
	loc := pendingLineRe.FindStringIndex(body)
	if loc == nil {
		return false
	}
	lineEnd := strings.IndexByte(body[loc[0]:], '\n')
	if lineEnd < 0 {
		lineEnd = len(body) - loc[0]
	}
	return strings.Contains(body[loc[0]:loc[0]+lineEnd], "待补")
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
