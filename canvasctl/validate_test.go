package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- 测试脚手架 ----------------------------------------------------------

// writeTree 把 files（相对 repo root 的路径 → 内容）落进临时目录，返回 root。
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func registryJSON(mutate func(m map[string]any)) string {
	m := map[string]any{
		"registry": "canvas-semantics/error-codes v1",
		"codes":    []any{"expression_syntax_error"},
		"reserved": []any{},
	}
	if mutate != nil {
		mutate(m)
	}
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// scanDoc 生成一份最小合法的 expression-scan 语料文档，mutate 逐字段注入要测的坏形态。
func scanDoc(mutate func(m map[string]any)) string {
	m := map[string]any{
		"meta": map[string]any{"contract": "expression-scan v1.0"},
		"cases": []any{
			map[string]any{
				"name":        "plain-literal",
				"description": "纯字面",
				"template":    "纯文本",
				"expect": map[string]any{
					"parts": []any{map[string]any{"kind": "literal", "text": "纯文本"}},
				},
			},
			map[string]any{
				"name":        "frag",
				"description": "片段",
				"template":    "{{orderNo}}",
				"expect": map[string]any{
					"parts": []any{map[string]any{"kind": "fragment", "raw": "orderNo", "expr": "orderNo", "error": nil}},
				},
			},
		},
	}
	if mutate != nil {
		mutate(m)
	}
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// baseTree 最小合法语料仓：注册表 + 一个契约版本目录 + 一个模块文件。
func baseTree(doc string) map[string]string {
	return map[string]string{
		"spec/error-codes.json":                 registryJSON(nil),
		"corpus/expression-scan/v1.0/scan.json": doc,
	}
}

// assertResult 钉住错误条数与逐条消息子串。
func assertResult(t *testing.T, res Result, wantCount int, wantSubstrs []string) {
	t.Helper()
	if len(res.Errors) != wantCount {
		t.Fatalf("错误条数 = %d， want %d；全部错误：\n%s", len(res.Errors), wantCount, joinErrs(res.Errors))
	}
	joined := joinErrs(res.Errors)
	for _, sub := range wantSubstrs {
		if !strings.Contains(joined, sub) {
			t.Fatalf("错误消息缺少子串 %q；全部错误：\n%s", sub, joined)
		}
	}
}

func joinErrs(errs []error) string {
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	return strings.Join(msgs, "\n")
}

// ---- 表驱动：四类坏用例逐一红 + 绿用例 -------------------------------------

func TestValidateTable(t *testing.T) {
	tests := []struct {
		name         string
		files        map[string]string
		wantErrCount int
		wantSubstrs  []string
	}{
		{
			name:         "绿：最小合法仓",
			files:        baseTree(scanDoc(nil)),
			wantErrCount: 0,
		},
		{
			name: "绿：meta 可选字段 notes/errata",
			files: baseTree(scanDoc(func(m map[string]any) {
				m["meta"] = map[string]any{
					"contract": "expression-scan v1.0",
					"notes":    []any{"出处注记"},
					"errata":   []any{"勘误 cases[0]：依据薄规格 §x"},
				}
			})),
			wantErrCount: 0,
		},
		{
			// 坏用例类 1：缺字段（用例顶层 + part 内层各钉一条）
			name: "红：用例缺字段 template",
			files: baseTree(scanDoc(func(m map[string]any) {
				cases := m["cases"].([]any)
				delete(cases[0].(map[string]any), "template")
			})),
			wantErrCount: 1,
			wantSubstrs:  []string{`用例 cases[0]（plain-literal）`, `缺少键 "template"`},
		},
		{
			name: "红：fragment part 缺字段 raw",
			files: baseTree(scanDoc(func(m map[string]any) {
				cases := m["cases"].([]any)
				parts := cases[1].(map[string]any)["expect"].(map[string]any)["parts"].([]any)
				delete(parts[0].(map[string]any), "raw")
			})),
			wantErrCount: 1,
			wantSubstrs:  []string{`parts[0]（fragment）`, `缺少键 "raw"`},
		},
		{
			// 坏用例类 2：多余键（顶层 / meta / 用例 / part 四层严格解码各钉一层）
			name: "红：文档顶层多余键",
			files: baseTree(scanDoc(func(m map[string]any) {
				m["extra"] = true
			})),
			wantErrCount: 1,
			wantSubstrs:  []string{`未知键 "extra"`},
		},
		{
			name: "红：meta 多余键 generator（迁移后不再携带）",
			files: baseTree(scanDoc(func(m map[string]any) {
				m["meta"] = map[string]any{"contract": "expression-scan v1.0", "generator": "x"}
			})),
			wantErrCount: 1,
			wantSubstrs:  []string{`未知键 "generator"`},
		},
		{
			name: "红：part 多余键（literal 带了 expr）",
			files: baseTree(scanDoc(func(m map[string]any) {
				cases := m["cases"].([]any)
				parts := cases[0].(map[string]any)["expect"].(map[string]any)["parts"].([]any)
				parts[0].(map[string]any)["expr"] = "oops"
			})),
			wantErrCount: 1,
			wantSubstrs:  []string{`parts[0]（literal）`, `未知键 "expr"`},
		},
		{
			// 坏用例类 3：contract 与目录名不一致
			name: "红：contract 版本与目录名不一致",
			files: baseTree(scanDoc(func(m map[string]any) {
				m["meta"] = map[string]any{"contract": "expression-scan v1.1"}
			})),
			wantErrCount: 1,
			wantSubstrs:  []string{`双写互检`},
		},
		{
			name: "红：contract 契约 id 与目录名不一致",
			files: map[string]string{
				"spec/error-codes.json": registryJSON(nil),
				"corpus/expression-scan/v1.0/scan.json": scanDoc(func(m map[string]any) {
					m["meta"] = map[string]any{"contract": "other-contract v1.0"}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`双写互检`},
		},
		{
			name: "红：非规范版本目录名（存量 v1 形态必须归一）",
			files: map[string]string{
				"spec/error-codes.json":            registryJSON(nil),
				"corpus/expression-scan/v1/x.json": scanDoc(nil),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`非规范版本目录名 "v1"`},
		},
		{
			// 坏用例类 4：错误 code 不在注册表
			name: "红：错误 code 未注册",
			files: baseTree(scanDoc(func(m map[string]any) {
				cases := m["cases"].([]any)
				parts := cases[1].(map[string]any)["expect"].(map[string]any)["parts"].([]any)
				parts[0].(map[string]any)["error"] = "expression_unknown_error"
			})),
			wantErrCount: 1,
			wantSubstrs:  []string{`错误 code "expression_unknown_error" 不在注册表`},
		},
		{
			name: "红：错误 code 落在 reserved 区（退役 code 不可复用）",
			files: map[string]string{
				"spec/error-codes.json": registryJSON(func(m map[string]any) {
					m["reserved"] = []any{"retired_legacy_error"}
				}),
				"corpus/expression-scan/v1.0/scan.json": scanDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					parts := cases[1].(map[string]any)["expect"].(map[string]any)["parts"].([]any)
					parts[0].(map[string]any)["error"] = "retired_legacy_error"
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`已退役保留`},
		},
		{
			// 治理 lint（GOVERNANCE §8）：用例名唯一
			name: "红：用例名重复",
			files: baseTree(scanDoc(func(m map[string]any) {
				cases := m["cases"].([]any)
				cases[1].(map[string]any)["name"] = "plain-literal"
			})),
			wantErrCount: 1,
			wantSubstrs:  []string{`用例名重复 "plain-literal"`},
		},
		{
			name: "红：用例名为空",
			files: baseTree(scanDoc(func(m map[string]any) {
				cases := m["cases"].([]any)
				cases[0].(map[string]any)["name"] = ""
			})),
			wantErrCount: 1,
			wantSubstrs:  []string{`用例名不可为空`},
		},
		{
			// 形状补充面
			name: "红：cases 为空数组",
			files: baseTree(scanDoc(func(m map[string]any) {
				m["cases"] = []any{}
			})),
			wantErrCount: 1,
			wantSubstrs:  []string{`用例集不可为空`},
		},
		{
			name: "红：未知 part kind",
			files: baseTree(scanDoc(func(m map[string]any) {
				cases := m["cases"].([]any)
				parts := cases[0].(map[string]any)["expect"].(map[string]any)["parts"].([]any)
				parts[0].(map[string]any)["kind"] = "cursor"
			})),
			wantErrCount: 1,
			wantSubstrs:  []string{`未知 part kind "cursor"`},
		},
		{
			name: "红：JSON 语法坏",
			files: map[string]string{
				"spec/error-codes.json":                 registryJSON(nil),
				"corpus/expression-scan/v1.0/scan.json": `{"meta":`,
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`JSON 解析失败`},
		},
		{
			name: "红：字段类型坏（template 为数字）",
			files: baseTree(scanDoc(func(m map[string]any) {
				cases := m["cases"].([]any)
				cases[0].(map[string]any)["template"] = 42
			})),
			wantErrCount: 1,
			wantSubstrs:  []string{`"template" 应为字符串`},
		},
		{
			name: "红：注册表文件缺失",
			files: map[string]string{
				"corpus/expression-scan/v1.0/scan.json": scanDoc(nil),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`spec/error-codes.json`},
		},
		{
			name: "红：注册表 registry 标识不符",
			files: map[string]string{
				"spec/error-codes.json":                 registryJSON(func(m map[string]any) { m["registry"] = "other v9" }),
				"corpus/expression-scan/v1.0/scan.json": scanDoc(nil),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`registry 标识`},
		},
		{
			name: "红：errata 空数组",
			files: baseTree(scanDoc(func(m map[string]any) {
				m["meta"] = map[string]any{"contract": "expression-scan v1.0", "errata": []any{}}
			})),
			wantErrCount: 1,
			wantSubstrs:  []string{`errata`},
		},
		{
			name: "红：版本目录下无模块文件",
			files: map[string]string{
				"spec/error-codes.json":             registryJSON(nil),
				"corpus/expression-scan/v1.0/.keep": "",
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`没有模块文件`},
		},
		{
			name: "绿：corpus/proposals 提案期用例区暂不校验（volatile）",
			files: map[string]string{
				"spec/error-codes.json":                      registryJSON(nil),
				"corpus/expression-scan/v1.0/scan.json":      scanDoc(nil),
				"corpus/proposals/0001-fn-custom/cases.json": `完全不是 JSON`,
			},
			wantErrCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writeTree(t, tt.files)
			res := Validate(root)
			assertResult(t, res, tt.wantErrCount, tt.wantSubstrs)
		})
	}
}

// ---- expression-eval 契约形状（工票 03）：按契约分派的用例形状 + dateTokens 转译表 ----

// evalDoc 生成一份最小合法的 expression-eval 语料文档（shell 模块单用例，字符串 expect）。
func evalDoc(mutate func(m map[string]any)) string {
	m := map[string]any{
		"meta": map[string]any{"contract": "expression-eval v2.0"},
		"cases": []any{
			map[string]any{
				"name":        "bare-name-reads-root-key",
				"description": "裸名只读全局根键",
				"template":    "{{orderNo}}",
				"context":     map[string]any{"orderNo": "A2026-001"},
				"expect":      "A2026-001",
			},
		},
	}
	if mutate != nil {
		mutate(m)
	}
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// evalRegistry 含表达式契约全部三个 code 的注册表。
func evalRegistry() string {
	return registryJSON(func(m map[string]any) {
		m["codes"] = []any{"expression_syntax_error", "expression_type_mismatch", "expression_row_outside_loop"}
	})
}

// dateTokens 最小合法转译表。
func dateTokensJSON(mutate func(m map[string]any)) string {
	m := map[string]any{
		"sample": "2026-10-11 14:30:05",
		"rows": []any{
			map[string]any{"token": "YYYY", "semantics": "四位年，零填充 4 位", "expect": "2026"},
		},
	}
	if mutate != nil {
		mutate(m)
	}
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestValidateEvalTable(t *testing.T) {
	tests := []struct {
		name         string
		files        map[string]string
		wantErrCount int
		wantSubstrs  []string
	}{
		{
			name: "绿：eval 最小仓（shell.json 字符串 expect）",
			files: map[string]string{
				"spec/error-codes.json":                  evalRegistry(),
				"corpus/expression-eval/v2.0/shell.json": evalDoc(nil),
			},
			wantErrCount: 0,
		},
		{
			name: "绿：expect = {errorCode} 对象（注册表白名单内）",
			files: map[string]string{
				"spec/error-codes.json": evalRegistry(),
				"corpus/expression-eval/v2.0/path.json": evalDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["expect"] = map[string]any{"errorCode": "expression_type_mismatch"}
				}),
			},
			wantErrCount: 0,
		},
		{
			name: "绿：fn-date.json 携带 dateTokens 转译表",
			files: map[string]string{
				"spec/error-codes.json": evalRegistry(),
				"corpus/expression-eval/v2.0/fn-date.json": evalDoc(func(m map[string]any) {
					m["dateTokens"] = json.RawMessage(dateTokensJSON(nil))
				}),
			},
			wantErrCount: 0,
		},
		{
			name: "绿：corpus/README.md 语料编写规范放行（文档非语料）",
			files: map[string]string{
				"spec/error-codes.json":                  evalRegistry(),
				"corpus/README.md":                       "# 语料编写规范",
				"corpus/expression-eval/v2.0/shell.json": evalDoc(nil),
			},
			wantErrCount: 0,
		},
		{
			// 用例名唯一作用域 = 契约版本、跨模块文件（scan/eval 同规）。
			name: "红：eval 用例名跨文件重复",
			files: map[string]string{
				"spec/error-codes.json":                  evalRegistry(),
				"corpus/expression-eval/v2.0/shell.json": evalDoc(nil),
				"corpus/expression-eval/v2.0/path.json":  evalDoc(nil),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`用例名重复 "bare-name-reads-root-key"`},
		},
		{
			name: "红：eval 用例缺 context",
			files: map[string]string{
				"spec/error-codes.json": evalRegistry(),
				"corpus/expression-eval/v2.0/shell.json": evalDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					delete(cases[0].(map[string]any), "context")
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`缺少键 "context"`},
		},
		{
			// v2 存量载体每例带 group（value|date|number）；模块文件拆分后归属由文件承载，
			// group 字段退役——严格解码钉死迁移纪律（残留即红）。
			name: "红：eval 用例残留 group 键（模块拆分后退役）",
			files: map[string]string{
				"spec/error-codes.json": evalRegistry(),
				"corpus/expression-eval/v2.0/shell.json": evalDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["group"] = "value"
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`未知键 "group"`},
		},
		{
			name: "红：context 非对象",
			files: map[string]string{
				"spec/error-codes.json": evalRegistry(),
				"corpus/expression-eval/v2.0/shell.json": evalDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["context"] = []any{"row"}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`"context" 应为对象`},
		},
		{
			name: "红：expect 类型坏（数字）",
			files: map[string]string{
				"spec/error-codes.json": evalRegistry(),
				"corpus/expression-eval/v2.0/shell.json": evalDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["expect"] = 42
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`expect 应为字符串或 {errorCode} 对象`},
		},
		{
			name: "红：expect 对象多余键",
			files: map[string]string{
				"spec/error-codes.json": evalRegistry(),
				"corpus/expression-eval/v2.0/shell.json": evalDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["expect"] = map[string]any{"errorCode": "expression_syntax_error", "msg": "x"}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`未知键 "msg"`},
		},
		{
			name: "红：errorCode 未注册",
			files: map[string]string{
				"spec/error-codes.json": evalRegistry(),
				"corpus/expression-eval/v2.0/shell.json": evalDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["expect"] = map[string]any{"errorCode": "expression_unknown_error"}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`错误 code "expression_unknown_error" 不在注册表`},
		},
		{
			name: "红：dateTokens 落在非 fn-date 模块文件",
			files: map[string]string{
				"spec/error-codes.json": evalRegistry(),
				"corpus/expression-eval/v2.0/shell.json": evalDoc(func(m map[string]any) {
					m["dateTokens"] = json.RawMessage(dateTokensJSON(nil))
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`未知键 "dateTokens"`},
		},
		{
			name: "红：dateTokens 行缺 semantics",
			files: map[string]string{
				"spec/error-codes.json": evalRegistry(),
				"corpus/expression-eval/v2.0/fn-date.json": evalDoc(func(m map[string]any) {
					m["dateTokens"] = json.RawMessage(dateTokensJSON(func(m map[string]any) {
						rows := m["rows"].([]any)
						delete(rows[0].(map[string]any), "semantics")
					}))
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`缺少键 "semantics"`},
		},
		{
			name: "红：dateTokens 行多余键",
			files: map[string]string{
				"spec/error-codes.json": evalRegistry(),
				"corpus/expression-eval/v2.0/fn-date.json": evalDoc(func(m map[string]any) {
					m["dateTokens"] = json.RawMessage(dateTokensJSON(func(m map[string]any) {
						rows := m["rows"].([]any)
						rows[0].(map[string]any)["note"] = "x"
					}))
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`未知键 "note"`},
		},
		{
			name: "红：dateTokens 缺 sample",
			files: map[string]string{
				"spec/error-codes.json": evalRegistry(),
				"corpus/expression-eval/v2.0/fn-date.json": evalDoc(func(m map[string]any) {
					m["dateTokens"] = json.RawMessage(dateTokensJSON(func(m map[string]any) {
						delete(m, "sample")
					}))
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`缺少键 "sample"`},
		},
		{
			name: "红：未知契约目录（validate 无形状定义）",
			files: map[string]string{
				"spec/error-codes.json": evalRegistry(),
				"corpus/expression-foo/v1.0/x.json": evalDoc(func(m map[string]any) {
					m["meta"] = map[string]any{"contract": "expression-foo v1.0"}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`未知契约 "expression-foo"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writeTree(t, tt.files)
			res := Validate(root)
			assertResult(t, res, tt.wantErrCount, tt.wantSubstrs)
		})
	}
}

// ---- 管线三契约形状（工票 04）：expand / paginate / flow --------------------
//
// 形状裁决（corpus/README.md「用例形状」同步）：
//   - 信封层严格解码（用例顶层键集 / meta / expect 键集，未知字段即红）；
//   - 图 payload（graph / frames / dataset）为不透明 wire 数据——语料故意携带
//     「实现必须拒绝/透传」的负形态，validate 只钉「是对象/是数组」的骨架；
//   - flowChain 节点同理：骨架 frame(int)+mode(string) 必出、quota/omitIfEmpty
//     类型钉死，未知键放行（钉例 flow:unknown-key-rejected 携带 pageHeight）。

// minGraph 最小合法 wire 图（画布 + 一个无标记文本层，键序对齐导出载体惯例）。
func minGraph() map[string]any {
	return map[string]any{
		"canvas": map[string]any{"width": 320, "height": 100},
		"layers": []any{},
	}
}

// expandDoc 生成一份最小合法的 expand-semantics 语料文档（单用例，graph expect）。
func expandDoc(mutate func(m map[string]any)) string {
	m := map[string]any{
		"meta": map[string]any{"contract": "expand-semantics v1.0"},
		"cases": []any{
			map[string]any{
				"name":        "independent-layers-root-context",
				"description": "独立图层根上下文求值",
				"graph":       minGraph(),
				"dataset":     map[string]any{"orderNo": "A2026-001"},
				"expect":      map[string]any{"graph": minGraph()},
			},
		},
	}
	if mutate != nil {
		mutate(m)
	}
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// pipelineRegistry 含管线三契约语料实际落到的 code 的注册表（全量注册表见 spec/error-codes.json）。
func pipelineRegistry() string {
	return registryJSON(func(m map[string]any) {
		m["codes"] = []any{
			"expression_syntax_error", "expression_type_mismatch", "expression_row_outside_loop",
			"rows_path_invalid", "content_overflow", "flow_chain_invalid",
		}
	})
}

// paginateDoc 生成一份最小合法的 paginate-semantics 语料文档（options 空数组形态）。
func paginateDoc(mutate func(m map[string]any)) string {
	m := map[string]any{
		"meta": map[string]any{"contract": "paginate-semantics v1.0"},
		"cases": []any{
			map[string]any{
				"name":        "passthrough-no-page-height",
				"description": "分页关直通",
				"graph":       minGraph(),
				"options":     []any{},
				"expect":      map[string]any{"pages": []any{minGraph()}},
			},
		},
	}
	if mutate != nil {
		mutate(m)
	}
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// flowDoc 生成一份最小合法的 flow-semantics 语料文档（flowChain null 形态）。
func flowDoc(mutate func(m map[string]any)) string {
	m := map[string]any{
		"meta": map[string]any{"contract": "flow-semantics v1.0"},
		"cases": []any{
			map[string]any{
				"name":        "capacity-split-by-row-height",
				"description": "容量切分",
				"frames":      []any{minGraph()},
				"dataset":     map[string]any{"order": map[string]any{"items": []any{}}},
				"flowChain":   nil,
				"expect":      map[string]any{"canvases": []any{minGraph()}},
			},
		},
	}
	if mutate != nil {
		mutate(m)
	}
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestValidatePipelineTable(t *testing.T) {
	tests := []struct {
		name         string
		files        map[string]string
		wantErrCount int
		wantSubstrs  []string
	}{
		{
			name: "绿：expand 最小仓（graph expect + dataset 对象）",
			files: map[string]string{
				"spec/error-codes.json":                    pipelineRegistry(),
				"corpus/expand-semantics/v1.0/expand.json": expandDoc(nil),
			},
			wantErrCount: 0,
		},
		{
			name: "绿：expand dataset null（未绑数据集恒等直通用例形态）",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/expand-semantics/v1.0/expand.json": expandDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["dataset"] = nil
				}),
			},
			wantErrCount: 0,
		},
		{
			name: "绿：expand expect = {errorCode}（注册表白名单内）",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/expand-semantics/v1.0/expand.json": expandDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["expect"] = map[string]any{"errorCode": "rows_path_invalid"}
				}),
			},
			wantErrCount: 0,
		},
		{
			name: "红：expand 用例缺 dataset",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/expand-semantics/v1.0/expand.json": expandDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					delete(cases[0].(map[string]any), "dataset")
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`缺少键 "dataset"`},
		},
		{
			name: "红：expand 用例多余键（载体的 group 残留即红，同 eval 纪律）",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/expand-semantics/v1.0/expand.json": expandDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["group"] = "resource"
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`未知键 "group"`},
		},
		{
			name: "红：expand graph 非对象",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/expand-semantics/v1.0/expand.json": expandDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["graph"] = []any{minGraph()}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`"graph" 应为对象`},
		},
		{
			name: "红：expand expect.graph 非对象",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/expand-semantics/v1.0/expand.json": expandDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["expect"] = map[string]any{"graph": "not-a-graph"}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`expect：键 "graph" 应为对象`},
		},
		{
			name: "红：expand expect 双键（graph 与 errorCode 同现）",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/expand-semantics/v1.0/expand.json": expandDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["expect"] = map[string]any{
						"graph": minGraph(), "errorCode": "rows_path_invalid",
					}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`expect 应为 {graph} 或 {errorCode} 二选一`},
		},
		{
			name: "红：expand expect 空（无键）",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/expand-semantics/v1.0/expand.json": expandDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["expect"] = map[string]any{}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`expect 应为 {graph} 或 {errorCode} 二选一`},
		},
		{
			name: "绿：paginate options 对象形态（pageHeight + truncate）",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/paginate-semantics/v1.0/paginate.json": paginateDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["options"] = map[string]any{"pageHeight": 250, "truncate": true}
				}),
			},
			wantErrCount: 0,
		},
		{
			name: "绿：paginate expect = {errorCode}",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/paginate-semantics/v1.0/paginate.json": paginateDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["expect"] = map[string]any{"errorCode": "content_overflow"}
				}),
			},
			wantErrCount: 0,
		},
		{
			name: "红：paginate 用例缺 options",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/paginate-semantics/v1.0/paginate.json": paginateDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					delete(cases[0].(map[string]any), "options")
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`缺少键 "options"`},
		},
		{
			name: "红：paginate options 类型坏（字符串）",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/paginate-semantics/v1.0/paginate.json": paginateDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["options"] = "pageHeight=250"
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`options 应为空数组或对象`},
		},
		{
			name: "红：paginate options 对象多余键",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/paginate-semantics/v1.0/paginate.json": paginateDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["options"] = map[string]any{"pageHeight": 250, "mode": "fixed"}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`未知键 "mode"`},
		},
		{
			name: "红：paginate options.pageHeight 类型坏",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/paginate-semantics/v1.0/paginate.json": paginateDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["options"] = map[string]any{"pageHeight": "250"}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`"pageHeight" 应为整数`},
		},
		{
			name: "红：paginate expect.pages 元素非对象",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/paginate-semantics/v1.0/paginate.json": paginateDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["expect"] = map[string]any{"pages": []any{"page-1"}}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`expect.pages[0] 应为对象`},
		},
		{
			name: "绿：flow flowChain 数组形态（quota + omitIfEmpty）",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/flow-semantics/v1.0/flow.json": flowDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["flowChain"] = []any{
						map[string]any{"frame": 0, "mode": "fixed", "quota": 2},
						map[string]any{"frame": 1, "mode": "paged", "omitIfEmpty": true},
					}
				}),
			},
			wantErrCount: 0,
		},
		{
			// 语料故意携带「实现必须拒绝」的负形态（flow:unknown-key-rejected）：
			// flowChain 是 payload 面，validate 只钉骨架、未知键放行。
			name: "绿：flowChain 节点未知键放行（payload 面负形态载体）",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/flow-semantics/v1.0/flow.json": flowDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["flowChain"] = []any{
						map[string]any{"frame": 0, "mode": "fixed", "pageHeight": 100},
					}
					cases[0].(map[string]any)["expect"] = map[string]any{"errorCode": "flow_chain_invalid"}
				}),
			},
			wantErrCount: 0,
		},
		{
			name: "红：flow 用例缺 flowChain",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/flow-semantics/v1.0/flow.json": flowDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					delete(cases[0].(map[string]any), "flowChain")
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`缺少键 "flowChain"`},
		},
		{
			name: "红：flow frames 非数组",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/flow-semantics/v1.0/flow.json": flowDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["frames"] = minGraph()
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`"frames" 应为数组`},
		},
		{
			name: "红：flow frames 元素非对象",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/flow-semantics/v1.0/flow.json": flowDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["frames"] = []any{"frame-0"}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`frames[0] 应为对象`},
		},
		{
			name: "红：flowChain 节点缺 mode",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/flow-semantics/v1.0/flow.json": flowDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["flowChain"] = []any{map[string]any{"frame": 0}}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`flowChain[0]`, `缺少键 "mode"`},
		},
		{
			name: "红：flowChain 节点 frame 类型坏",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/flow-semantics/v1.0/flow.json": flowDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["flowChain"] = []any{map[string]any{"frame": "0", "mode": "fixed"}}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`"frame" 应为整数`},
		},
		{
			name: "红：flowChain 节点 quota 类型坏",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/flow-semantics/v1.0/flow.json": flowDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["flowChain"] = []any{
						map[string]any{"frame": 0, "mode": "fixed", "quota": "2"},
					}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`"quota" 应为整数`},
		},
		{
			name: "红：flow expect = {errorCode} 未注册 code",
			files: map[string]string{
				"spec/error-codes.json": pipelineRegistry(),
				"corpus/flow-semantics/v1.0/flow.json": flowDoc(func(m map[string]any) {
					cases := m["cases"].([]any)
					cases[0].(map[string]any)["expect"] = map[string]any{"errorCode": "flow_unknown_error"}
				}),
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`错误 code "flow_unknown_error" 不在注册表`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writeTree(t, tt.files)
			res := Validate(root)
			assertResult(t, res, tt.wantErrCount, tt.wantSubstrs)
		})
	}
}

// 管线三契约 + 表达式两契约同仓：计数面与用例名唯一作用域（= 契约版本，跨契约可重名）。
func TestValidatePipelineCounts(t *testing.T) {
	root := writeTree(t, map[string]string{
		"spec/error-codes.json":                        pipelineRegistry(),
		"corpus/expression-scan/v1.0/scan.json":        scanDoc(nil),
		"corpus/expand-semantics/v1.0/expand.json":     expandDoc(nil),
		"corpus/paginate-semantics/v1.0/paginate.json": paginateDoc(nil),
		"corpus/flow-semantics/v1.0/flow.json":         flowDoc(nil),
	})
	res := Validate(root)
	if len(res.Errors) != 0 {
		t.Fatalf("应全绿：\n%s", joinErrs(res.Errors))
	}
	if res.Contracts != 4 || res.Versions != 4 || res.ModuleFiles != 4 || res.Cases != 5 {
		t.Fatalf("计数不符： %+v", res)
	}
}

// 绿用例顺带钉住计数面（CI 摘要的数字来源）。
func TestValidateCounts(t *testing.T) {
	root := writeTree(t, baseTree(scanDoc(nil)))
	res := Validate(root)
	if len(res.Errors) != 0 {
		t.Fatalf("应全绿：\n%s", joinErrs(res.Errors))
	}
	if res.Contracts != 1 || res.Versions != 1 || res.ModuleFiles != 1 || res.Cases != 2 {
		t.Fatalf("计数不符： %+v", res)
	}
}

// 端到端：仓内真实语料（本工票迁入的扫描契约）必须全绿。
func TestRealCorpusGreen(t *testing.T) {
	res := Validate("..")
	if len(res.Errors) != 0 {
		t.Fatalf("真实语料应全绿：\n%s", joinErrs(res.Errors))
	}
	if res.Contracts < 1 || res.Cases < 1 {
		t.Fatalf("真实语料计数异常： %+v", res)
	}
}
