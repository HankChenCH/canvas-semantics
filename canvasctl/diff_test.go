package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// ---- 测试脚手架 ----------------------------------------------------------
//
// diff 只依赖语料目录形状（meta + cases），不依赖错误 code 注册表——脚手架最小化：
// 一个契约两个版本目录，用例取 expression-scan 形状（diff 不按契约 schema 钉键集，
// 那是 validate 的职责；diff 只取可比对的原始 JSON）。

// jsonString 序列化测试夹具（键序由 encoding/json 排序，确定性输出）。
func jsonString(m map[string]any) string {
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// diffCase 生成一条最小 scan 用例 JSON（fields 为额外/覆盖字段）。
func diffCase(name, desc, template string, fields map[string]any) string {
	m := map[string]any{
		"name":        name,
		"description": desc,
		"template":    template,
		"expect": map[string]any{
			"parts": []any{map[string]any{"kind": "literal", "text": "x"}},
		},
	}
	for k, v := range fields {
		m[k] = v
	}
	return jsonString(m)
}

// diffDoc 生成一份模块文件 JSON（meta 双写 + cases）。
func diffDoc(contract, version string, cases ...string) string {
	caseList := "[\n" + strings.Join(cases, ",\n") + "\n]"
	return `{"meta": {"contract": "` + contract + ` ` + version + `"}, "cases": ` + caseList + `}`
}

// diffTree 两版本目录树：契约 demo，文件 x.json；vA/vB 为各版本目录下的文件集。
func diffTree(vA, vB map[string]string) map[string]string {
	files := map[string]string{}
	for f, content := range vA {
		files["corpus/demo/v1.0/"+f] = content
	}
	for f, content := range vB {
		files["corpus/demo/v2.0/"+f] = content
	}
	return files
}

// ---- 表驱动：正路径四类差异 + 负路径坏版本目录对 / 期望值不可比形态 -----------

func TestDiffTable(t *testing.T) {
	kept := diffCase("kept-case", "两版本一致", "x", nil)
	removed := diffCase("removed-case", "仅 v1.0", "x", nil)
	changed := diffCase("changed-case", "模板改了", "x", nil)
	changedNew := diffCase("changed-case", "模板改了", "{{y}}", nil)
	noteOld := diffCase("note-case", "旧描述", "x", nil)
	noteNew := diffCase("note-case", "新描述", "x", nil)
	added := diffCase("added-case", "仅 v2.0", "x", nil)

	tests := []struct {
		name         string
		contract     string
		versionA     string
		versionB     string
		files        map[string]string
		wantErrCount int
		wantSubstrs  []string
		want         func(t *testing.T, res DiffResult)
	}{
		{
			name:     "绿：四类差异各一（新增/删除/期望值变化/仅注记变化）",
			contract: "demo", versionA: "v1.0", versionB: "v2.0",
			files: diffTree(
				map[string]string{"x.json": diffDoc("demo", "v1.0", kept, removed, changed, noteOld)},
				map[string]string{"x.json": diffDoc("demo", "v2.0", kept, changedNew, noteNew, added)},
			),
			want: func(t *testing.T, res DiffResult) {
				fd, ok := fileDiffByName(res.Files, "x.json")
				if !ok {
					t.Fatalf("缺 x.json 差异块：%+v", res.Files)
				}
				assertNames(t, "Added", fd.Added, []string{"added-case"})
				assertNames(t, "Removed", fd.Removed, []string{"removed-case"})
				assertNames(t, "ExpectChanged", fd.ExpectChanged, []string{"changed-case"})
				assertNames(t, "NoteOnly", fd.NoteOnly, []string{"note-case"})
				if fd.Unchanged != 1 {
					t.Fatalf("Unchanged = %d，want 1", fd.Unchanged)
				}
				if got := fd.ExpectFields["changed-case"]; len(got) != 1 || got[0] != "template" {
					t.Fatalf("ExpectFields[changed-case] = %v，want [template]", got)
				}
				if res.DiffFiles != 1 || res.DiffAdded != 1 || res.DiffRemoved != 1 ||
					res.DiffExpectChanged != 1 || res.DiffNoteOnly != 1 {
					t.Fatalf("计数不符：%+v", res.Result)
				}
			},
		},
		{
			// 期望值可比性不受键序与空白影响：同一用例两种排版 = 无差异。
			name:     "绿：键序与排版不同但语义相同 = 无差异",
			contract: "demo", versionA: "v1.0", versionB: "v2.0",
			files: diffTree(
				map[string]string{"x.json": diffDoc("demo", "v1.0", kept)},
				map[string]string{"x.json": "{\n  \"cases\": [ {\"expect\": {\"parts\": [{\"text\": \"x\", \"kind\": \"literal\"}]}, \"template\": \"x\", \"description\": \"两版本一致\", \"name\": \"kept-case\"} ],\n  \"meta\": {\"contract\": \"demo v2.0\"}\n}"},
			),
			want: func(t *testing.T, res DiffResult) {
				fd := res.Files[0]
				if len(fd.Added)+len(fd.Removed)+len(fd.ExpectChanged)+len(fd.NoteOnly) != 0 || fd.Unchanged != 1 {
					t.Fatalf("应无差异：%+v", fd)
				}
			},
		},
		{
			// 期望值逐字面可比：数值字面 1 与 1.0 是不同字面（语料数值纪律为十进制字符串，
			// 结构整数字面不重排）——diff 按字面比，不猜数值等价。
			name:     "绿：数值字面 1 与 1.0 记期望值变化",
			contract: "demo", versionA: "v1.0", versionB: "v2.0",
			files: diffTree(
				map[string]string{"x.json": diffDoc("demo", "v1.0", diffCase("n", "d", "x", map[string]any{"width": json.RawMessage("1")}))},
				map[string]string{"x.json": diffDoc("demo", "v2.0", diffCase("n", "d", "x", map[string]any{"width": json.RawMessage("1.0")}))},
			),
			want: func(t *testing.T, res DiffResult) {
				if len(res.Files[0].ExpectChanged) != 1 || res.Files[0].ExpectChanged[0] != "n" {
					t.Fatalf("数值字面变化应记期望值变化：%+v", res.Files[0])
				}
			},
		},
		{
			// 文档级语义附加键（dateTokens 转译表，GOVERNANCE §3.3）变化 = 语义变化，
			// 不得吞进「仅注记变化」。
			name:     "绿：dateTokens 转译表变化单独记账",
			contract: "demo", versionA: "v1.0", versionB: "v2.0",
			files: diffTree(
				map[string]string{"fn-date.json": `{"meta": {"contract": "demo v1.0"}, "dateTokens": {"sample": "s", "rows": [{"token": "YYYY", "semantics": "年", "expect": "2026"}]}, "cases": [` + diffCase("kept", "d", "x", nil) + `]}`},
				map[string]string{"fn-date.json": `{"meta": {"contract": "demo v2.0"}, "dateTokens": {"sample": "s", "rows": [{"token": "YYYY", "semantics": "年", "expect": "2026"}, {"token": "Q", "semantics": "季度", "expect": "4"}]}, "cases": [` + diffCase("kept", "d", "x", nil) + `]}`},
			),
			want: func(t *testing.T, res DiffResult) {
				fd := res.Files[0]
				if len(fd.ExtraKeysChanged) != 1 || fd.ExtraKeysChanged[0] != "dateTokens" {
					t.Fatalf("ExtraKeysChanged = %v，want [dateTokens]", fd.ExtraKeysChanged)
				}
				if len(fd.Added)+len(fd.ExpectChanged) != 0 {
					t.Fatalf("用例面应无差异：%+v", fd)
				}
			},
		},
		{
			name:     "绿：meta.notes 注记变化 = 仅注记变化（非语义）",
			contract: "demo", versionA: "v1.0", versionB: "v2.0",
			files: diffTree(
				map[string]string{"x.json": `{"meta": {"contract": "demo v1.0", "notes": ["旧注记"]}, "cases": [` + kept + `]}`},
				map[string]string{"x.json": `{"meta": {"contract": "demo v2.0", "notes": ["旧注记", "新注记"]}, "cases": [` + kept + `]}`},
			),
			want: func(t *testing.T, res DiffResult) {
				if !res.Files[0].MetaNotesChanged {
					t.Fatal("应记 meta 注记变化")
				}
				if len(res.Files[0].ExpectChanged)+len(res.Files[0].NoteOnly) != 0 {
					t.Fatalf("用例面应无差异：%+v", res.Files[0])
				}
			},
		},
		{
			name:     "绿：模块文件新增 / 删除（整文件只在一侧）",
			contract: "demo", versionA: "v1.0", versionB: "v2.0",
			files: diffTree(
				map[string]string{"old.json": diffDoc("demo", "v1.0", removed)},
				map[string]string{"new.json": diffDoc("demo", "v2.0", added)},
			),
			want: func(t *testing.T, res DiffResult) {
				oldFD, ok := fileDiffByName(res.Files, "old.json")
				if !ok || !oldFD.FileRemoved || len(oldFD.Removed) != 1 {
					t.Fatalf("old.json 应记整文件删除：%+v", oldFD)
				}
				newFD, ok := fileDiffByName(res.Files, "new.json")
				if !ok || !newFD.FileAdded || len(newFD.Added) != 1 {
					t.Fatalf("new.json 应记整文件新增：%+v", newFD)
				}
				if res.DiffRemoved != 1 || res.DiffAdded != 1 {
					t.Fatalf("整文件差异应计入用例总数：%+v", res.Result)
				}
			},
		},
		{
			name:     "绿：两版本无差异",
			contract: "demo", versionA: "v1.0", versionB: "v2.0",
			files: diffTree(
				map[string]string{"x.json": diffDoc("demo", "v1.0", kept)},
				map[string]string{"x.json": diffDoc("demo", "v2.0", kept)},
			),
			want: func(t *testing.T, res DiffResult) {
				if res.DiffAdded+res.DiffRemoved+res.DiffExpectChanged+res.DiffNoteOnly != 0 {
					t.Fatalf("应无差异：%+v", res.Result)
				}
			},
		},
		// ---- 负路径：坏版本目录对 ----
		{
			name:     "红：契约目录不存在",
			contract: "no-such", versionA: "v1.0", versionB: "v2.0",
			files:        diffTree(map[string]string{"x.json": diffDoc("demo", "v1.0", kept)}, map[string]string{"x.json": diffDoc("demo", "v2.0", kept)}),
			wantErrCount: 1,
			wantSubstrs:  []string{`契约目录 "no-such"`},
		},
		{
			name:     "红：版本 A 目录不存在",
			contract: "demo", versionA: "v1.5", versionB: "v2.0",
			files:        diffTree(map[string]string{"x.json": diffDoc("demo", "v1.0", kept)}, map[string]string{"x.json": diffDoc("demo", "v2.0", kept)}),
			wantErrCount: 1,
			wantSubstrs:  []string{`版本目录 corpus/demo/v1.5 不存在`},
		},
		{
			name:     "红：版本 B 目录不存在",
			contract: "demo", versionA: "v1.0", versionB: "v9.9",
			files:        diffTree(map[string]string{"x.json": diffDoc("demo", "v1.0", kept)}, map[string]string{"x.json": diffDoc("demo", "v2.0", kept)}),
			wantErrCount: 1,
			wantSubstrs:  []string{`版本目录 corpus/demo/v9.9 不存在`},
		},
		{
			name:     "红：版本对相同",
			contract: "demo", versionA: "v2.0", versionB: "v2.0",
			files:        diffTree(map[string]string{"x.json": diffDoc("demo", "v1.0", kept)}, map[string]string{"x.json": diffDoc("demo", "v2.0", kept)}),
			wantErrCount: 1,
			wantSubstrs:  []string{`版本对相同`},
		},
		{
			name:     "红：版本名非规范形（含路径逃逸形态）",
			contract: "demo", versionA: "v1.0", versionB: "../../spec",
			files:        diffTree(map[string]string{"x.json": diffDoc("demo", "v1.0", kept)}, map[string]string{"x.json": diffDoc("demo", "v2.0", kept)}),
			wantErrCount: 1,
			wantSubstrs:  []string{`非规范版本目录名`},
		},
		{
			name:     "红：契约 id 非规范形",
			contract: "../spec", versionA: "v1.0", versionB: "v2.0",
			files:        diffTree(map[string]string{"x.json": diffDoc("demo", "v1.0", kept)}, map[string]string{"x.json": diffDoc("demo", "v2.0", kept)}),
			wantErrCount: 1,
			wantSubstrs:  []string{`非规范契约 id`},
		},
		// ---- 负路径：期望值不可比形态（diff 拒绝为不可严格解码的语料出摘要）----
		{
			name:     "红：cases 为对象非数组（期望值不可比形态）",
			contract: "demo", versionA: "v1.0", versionB: "v2.0",
			files: diffTree(
				map[string]string{"x.json": diffDoc("demo", "v1.0", kept)},
				map[string]string{"x.json": `{"meta": {"contract": "demo v2.0"}, "cases": {"name": "x"}}`},
			),
			wantErrCount: 1,
			wantSubstrs:  []string{`cases`},
		},
		{
			name:     "红：用例条目非对象",
			contract: "demo", versionA: "v1.0", versionB: "v2.0",
			files: diffTree(
				map[string]string{"x.json": diffDoc("demo", "v1.0", kept)},
				map[string]string{"x.json": diffDoc("demo", "v2.0", `"stray-case"`)},
			),
			wantErrCount: 1,
			wantSubstrs:  []string{`应为对象`},
		},
		{
			name:     "红：用例缺 name（无从按名索引比对）",
			contract: "demo", versionA: "v1.0", versionB: "v2.0",
			files: diffTree(
				map[string]string{"x.json": diffDoc("demo", "v1.0", kept)},
				map[string]string{"x.json": diffDoc("demo", "v2.0", diffCase("", "无名", "x", nil))},
			),
			wantErrCount: 1,
			wantSubstrs:  []string{`用例名不可为空`},
		},
		{
			name:     "红：JSON 语法坏",
			contract: "demo", versionA: "v1.0", versionB: "v2.0",
			files: diffTree(
				map[string]string{"x.json": diffDoc("demo", "v1.0", kept)},
				map[string]string{"x.json": `{"meta":`},
			),
			wantErrCount: 1,
			wantSubstrs:  []string{`JSON 解析失败`},
		},
		{
			name:     "红：meta.contract 双写不一致（版本 B 文件冒认他版本）",
			contract: "demo", versionA: "v1.0", versionB: "v2.0",
			files: diffTree(
				map[string]string{"x.json": diffDoc("demo", "v1.0", kept)},
				map[string]string{"x.json": diffDoc("demo", "v1.0", kept)},
			),
			wantErrCount: 1,
			wantSubstrs:  []string{`双写互检`},
		},
		{
			name:     "红：版本内用例名跨文件重复（按名索引不可信）",
			contract: "demo", versionA: "v1.0", versionB: "v2.0",
			files: diffTree(
				map[string]string{
					"x.json": diffDoc("demo", "v1.0", kept),
					"y.json": diffDoc("demo", "v1.0", kept),
				},
				map[string]string{"x.json": diffDoc("demo", "v2.0", kept)},
			),
			wantErrCount: 1,
			wantSubstrs:  []string{`用例名重复 "kept-case"`},
		},
		{
			name:     "红：版本目录下非 json 条目（validate 面，diff 拒绝出摘要）",
			contract: "demo", versionA: "v1.0", versionB: "v2.0",
			files: func() map[string]string {
				files := diffTree(
					map[string]string{"x.json": diffDoc("demo", "v1.0", kept)},
					map[string]string{"x.json": diffDoc("demo", "v2.0", kept)},
				)
				files["corpus/demo/v2.0/notes.txt"] = "杂条目"
				return files
			}(),
			wantErrCount: 1,
			wantSubstrs:  []string{`只允许 *.json 模块文件`},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := writeTree(t, tc.files)
			res := Diff(root, tc.contract, tc.versionA, tc.versionB)
			assertResult(t, res.Result, tc.wantErrCount, tc.wantSubstrs)
			if tc.wantErrCount == 0 && tc.want != nil {
				tc.want(t, res)
			}
		})
	}
}

// ---- 人可审摘要渲染 ---------------------------------------------------------

func TestDiffSummary(t *testing.T) {
	kept := diffCase("kept-case", "两版本一致", "x", nil)
	files := diffTree(
		map[string]string{
			"x.json":    diffDoc("demo", "v1.0", kept, diffCase("removed-case", "仅 A", "x", nil), diffCase("changed-case", "旧", "a", nil), diffCase("note-case", "旧描述", "x", nil)),
			"gone.json": diffDoc("demo", "v1.0", diffCase("gone-case", "整文件删除", "x", nil)),
		},
		map[string]string{
			"x.json":   diffDoc("demo", "v2.0", kept, diffCase("added-case", "仅 B", "x", nil), diffCase("changed-case", "旧", "b", nil), diffCase("note-case", "新描述", "x", nil)),
			"new.json": diffDoc("demo", "v2.0", diffCase("new-case", "整文件新增", "x", nil)),
		},
	)
	root := writeTree(t, files)
	res := Diff(root, "demo", "v1.0", "v2.0")
	if len(res.Errors) != 0 {
		t.Fatalf("应全绿：\n%s", joinErrs(res.Errors))
	}
	summary := res.Summary()
	for _, want := range []string{
		"语料 diff：demo v1.0 → v2.0",
		"比对文件 3：新增 2 用例 / 删除 2 用例 / 期望值变化 1 用例 / 仅注记变化 1 用例",
		"x.json：新增 1 / 删除 1 / 期望值变化 1 / 仅注记变化 1",
		"+ added-case",
		"- removed-case",
		"~ changed-case（template）",
		"≈ note-case（仅注记变化）",
		"gone.json：模块文件删除",
		"- gone-case",
		"new.json：模块文件新增",
		"+ new-case",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("摘要缺少 %q：\n%s", want, summary)
		}
	}
}

// 无差异版本的摘要：总数行明示两版本无差异（人审的「确认无语义变化」结论形态）。
func TestDiffSummaryNoDifferences(t *testing.T) {
	kept := diffCase("kept-case", "两版本一致", "x", nil)
	root := writeTree(t, diffTree(
		map[string]string{"x.json": diffDoc("demo", "v1.0", kept)},
		map[string]string{"x.json": diffDoc("demo", "v2.0", kept)},
	))
	res := Diff(root, "demo", "v1.0", "v2.0")
	if len(res.Errors) != 0 {
		t.Fatalf("应全绿：\n%s", joinErrs(res.Errors))
	}
	if !strings.Contains(res.Summary(), "两版本无差异") {
		t.Fatalf("无差异摘要缺失明示：\n%s", res.Summary())
	}
}

// 防回归（评审发现）：仅 meta 注记变化时，头部不得宣称「两版本无差异」——
// 头部结论与文件块（≈ meta 注记变化）必须一致，呈现不可自相矛盾。
func TestDiffSummaryMetaNotesOnly(t *testing.T) {
	root := writeTree(t, diffTree(
		map[string]string{"x.json": `{"meta": {"contract": "demo v1.0", "notes": ["旧注记"]}, "cases": [` + diffCase("kept-case", "两版本一致", "x", nil) + `]}`},
		map[string]string{"x.json": `{"meta": {"contract": "demo v2.0", "notes": ["旧注记", "新注记"]}, "cases": [` + diffCase("kept-case", "两版本一致", "x", nil) + `]}`},
	))
	res := Diff(root, "demo", "v1.0", "v2.0")
	if len(res.Errors) != 0 {
		t.Fatalf("应全绿：\n%s", joinErrs(res.Errors))
	}
	summary := res.Summary()
	if strings.Contains(summary, "两版本无差异") {
		t.Fatalf("仅 meta 注记变化不得宣称两版本无差异：\n%s", summary)
	}
	if !strings.Contains(summary, "≈ meta 注记变化") {
		t.Fatalf("注记变化未呈现：\n%s", summary)
	}
}

// CLI 参数解析：位置参数 3 或 4 个（[root] 契约 版本A 版本B），空串/未知旗标退 usage。
func TestParseDiffArgs(t *testing.T) {
	if _, _, _, _, ok := parseDiffArgs([]string{""}); ok {
		t.Fatal("空串位置参数应退 usage")
	}
	if _, _, _, _, ok := parseDiffArgs([]string{"--unknown"}); ok {
		t.Fatal("未知旗标应退 usage")
	}
	if _, _, _, _, ok := parseDiffArgs([]string{"a", "b", "c", "d", "e"}); ok {
		t.Fatal("多余位置参数应退 usage")
	}
	root, contract, va, vb, ok := parseDiffArgs([]string{"root/dir", "expression-eval", "v2.0", "v3.0"})
	if !ok || root != "root/dir" || contract != "expression-eval" || va != "v2.0" || vb != "v3.0" {
		t.Fatalf("四位置参数解析不符：%q %q %q %q %v", root, contract, va, vb, ok)
	}
	root, contract, va, vb, ok = parseDiffArgs([]string{"expression-eval", "v2.0", "v3.0"})
	if !ok || root != "." || contract != "expression-eval" || va != "v2.0" || vb != "v3.0" {
		t.Fatalf("三位置参数解析不符（root 缺省句点）：%q %q %q %q %v", root, contract, va, vb, ok)
	}
}

// 端到端：真实语料的提案 0002 版本对（expression-eval v2.0 → v3.0，季度 token Q 收编）。
// 计数即该次语义变更的人审清单：fn-date 新增 3 例季度用例 + date-unknown-token 换示例
// 字母（template 变化）+ dateTokens 转译表加 Q 行 + meta 出处注记追加；其余三文件冻结。
func TestDiffRealCorpusProposal0002(t *testing.T) {
	res := Diff("..", "expression-eval", "v2.0", "v3.0")
	if len(res.Errors) != 0 {
		t.Fatalf("真实版本对应可Diff：\n%s", joinErrs(res.Errors))
	}
	if res.DiffFiles != 4 || res.DiffAdded != 3 || res.DiffRemoved != 0 ||
		res.DiffExpectChanged != 1 || res.DiffNoteOnly != 0 {
		t.Fatalf("提案 0002 版本对计数不符：%+v", res.Result)
	}
	fd, ok := fileDiffByName(res.Files, "fn-date.json")
	if !ok {
		t.Fatal("缺 fn-date.json 差异块")
	}
	assertNames(t, "Added", fd.Added, []string{"date-quarter-january", "date-quarter-june", "date-quarter-token"})
	assertNames(t, "ExpectChanged", fd.ExpectChanged, []string{"date-unknown-token"})
	if len(fd.ExtraKeysChanged) != 1 || fd.ExtraKeysChanged[0] != "dateTokens" {
		t.Fatalf("dateTokens 转译表变化应单独记账：%+v", fd.ExtraKeysChanged)
	}
	if !fd.MetaNotesChanged {
		t.Fatal("fn-date v3.0 出处注记追加应记 meta 注记变化")
	}
	for _, name := range []string{"shell.json", "path.json", "fn-number.json"} {
		other, ok := fileDiffByName(res.Files, name)
		if !ok {
			t.Fatalf("缺 %s 差异块", name)
		}
		if other.FileAdded || other.FileRemoved || len(other.Added)+len(other.Removed)+
			len(other.ExpectChanged)+len(other.NoteOnly) != 0 || len(other.ExtraKeysChanged) != 0 {
			t.Fatalf("%s 应无差异（v2.0 冻结承 v3.0）：%+v", name, other)
		}
	}
}

// ---- 断言辅助 --------------------------------------------------------------

func fileDiffByName(files []FileDiff, name string) (FileDiff, bool) {
	for _, fd := range files {
		if fd.ModuleFile == name {
			return fd, true
		}
	}
	return FileDiff{}, false
}

func assertNames(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %v，want %v", label, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s = %v，want %v", label, got, want)
		}
	}
}
