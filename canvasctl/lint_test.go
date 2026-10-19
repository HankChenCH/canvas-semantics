package main

import (
	"testing"
)

// proposalDoc 生成提案文档（模板 0000 头部键行形态）。
func proposalDoc(status, casesLine string) string {
	return "# 002: 测试提案\n\n- 状态：" + status + "\n- 提案期用例：" + casesLine + "\n- 摘要：测试。\n"
}

// lintTree 基础树：空语料仓骨架（lint 不依赖语料形状，注册表可缺）。
func lintTree(extra map[string]string) map[string]string {
	files := map[string]string{}
	for k, v := range extra {
		files[k] = v
	}
	return files
}

func TestLintTable(t *testing.T) {
	tests := []struct {
		name             string
		files            map[string]string
		wantErrCount     int
		wantSubstrs      []string
		wantDocs         int
		wantProposalDirs int
	}{
		{
			name:  "绿：无提案区无提案文档",
			files: lintTree(nil),
		},
		{
			name: "绿：proposal 文档 + 反链用例目录（双向反链齐备）",
			files: lintTree(map[string]string{
				"proposals/002-quarter-token.md":                  proposalDoc("proposal", "`corpus/proposals/002-quarter-token/`"),
				"corpus/proposals/002-quarter-token/fn-date.json": `{}`,
			}),
			wantDocs:         1,
			wantProposalDirs: 1,
		},
		{
			name: "绿：proposal 文档写明待补、无用例目录",
			files: lintTree(map[string]string{
				"proposals/002-quarter-token.md": proposalDoc("proposal", "待补（转正前建齐）"),
			}),
			wantDocs: 1,
		},
		{
			name: "绿：integrated 文档无用例目录（0001 形态，文档保留为历史）",
			files: lintTree(map[string]string{
				"proposals/001-merged.md": proposalDoc("integrated（2026-10-18 转正）", "未单建"),
			}),
			wantDocs: 1,
		},
		{
			name: "绿：README 索引与 0000 模板不参与生命周期",
			files: lintTree(map[string]string{
				"proposals/README.md":        "# 索引\n",
				"proposals/0000-template.md": proposalDoc("proposal ｜ integrated", "`corpus/proposals/NNN-<slug>/`"),
			}),
			wantDocs: 0,
		},
		{
			// 方向一：文档 → 目录
			name: "红：proposal 文档无用例目录且未写待补",
			files: lintTree(map[string]string{
				"proposals/002-quarter-token.md": proposalDoc("proposal", "`corpus/proposals/002-quarter-token/`"),
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"须有提案期用例目录"},
			wantDocs:     1,
		},
		{
			// 方向二：目录 → 文档
			name: "红：提案期用例目录无提案文档",
			files: lintTree(map[string]string{
				"corpus/proposals/002-quarter-token/fn-date.json": `{}`,
			}),
			wantErrCount:     1,
			wantSubstrs:      []string{"无对应提案文档"},
			wantProposalDirs: 1,
		},
		{
			name: "红：提案期用例目录为空",
			files: lintTree(map[string]string{
				"proposals/002-quarter-token.md":           proposalDoc("proposal", "`corpus/proposals/002-quarter-token/`"),
				"corpus/proposals/002-quarter-token/.keep": "",
			}),
			wantErrCount:     1,
			wantSubstrs:      []string{"没有用例文件"},
			wantDocs:         1,
			wantProposalDirs: 1,
		},
		{
			// 反链：目录存在但文档未引用目录路径
			name: "红：文档未反链用例目录路径",
			files: lintTree(map[string]string{
				"proposals/002-quarter-token.md":                  proposalDoc("proposal", "待补（转正前建齐）"),
				"corpus/proposals/002-quarter-token/fn-date.json": `{}`,
			}),
			wantErrCount:     1,
			wantSubstrs:      []string{"未反链该目录路径"},
			wantDocs:         1,
			wantProposalDirs: 1,
		},
		{
			name: "红：状态行缺失",
			files: lintTree(map[string]string{
				"proposals/002-quarter-token.md": "# 002\n\n- 摘要：无状态行\n",
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"缺少「状态：」行"},
			wantDocs:     1,
		},
		{
			name: "红：状态词不可识别",
			files: lintTree(map[string]string{
				"proposals/002-quarter-token.md": proposalDoc("draft", "待补"),
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"不可识别"},
			wantDocs:     1,
		},
		{
			name: "红：proposals/ 意外条目（编号文档命名不合规范）",
			files: lintTree(map[string]string{
				"proposals/notes.md": "随手记",
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"意外条目"},
		},
		{
			name: "红：corpus/proposals/ 意外条目（文件而非编号目录）",
			files: lintTree(map[string]string{
				"corpus/proposals/loose.json": `{}`,
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"意外条目"},
		},
		{
			name: "红：用例目录内混入非 json 条目",
			files: lintTree(map[string]string{
				"proposals/002-quarter-token.md":                  proposalDoc("proposal", "`corpus/proposals/002-quarter-token/`"),
				"corpus/proposals/002-quarter-token/fn-date.json": `{}`,
				"corpus/proposals/002-quarter-token/notes.md":     "笔记",
			}),
			wantErrCount:     1,
			wantSubstrs:      []string{"只允许 *.json"},
			wantDocs:         1,
			wantProposalDirs: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := writeTree(t, tc.files)
			res := Lint(root)
			assertResult(t, res, tc.wantErrCount, tc.wantSubstrs)
			if res.ProposalDocs != tc.wantDocs || res.ProposalCaseDirs != tc.wantProposalDirs {
				t.Fatalf("计数不符：docs = %d (want %d), dirs = %d (want %d)",
					res.ProposalDocs, tc.wantDocs, res.ProposalCaseDirs, tc.wantProposalDirs)
			}
		})
	}
}

// 端到端：仓内真实提案区必须全绿（README 索引 + 0000 模板不参与生命周期；存量
// 0001/0002 两提案均已 integrated）。计数随提案增长，此处只钉下界与全绿。
func TestRealProposalsGreen(t *testing.T) {
	res := Lint("..")
	if len(res.Errors) != 0 {
		t.Fatalf("真实提案区应全绿：\n%s", joinErrs(res.Errors))
	}
	if res.ProposalDocs < 1 {
		t.Fatalf("提案文档计数异常：%+v", res)
	}
}
