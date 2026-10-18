package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// ---- 测试脚手架 ----------------------------------------------------------

// surfaceJSON 生成一份最小合法支持面声明（对照 SUPPORT-SURFACE §3 schema v1），
// mutate 逐字段注入要测的坏形态。默认声明 expression-scan v1.0 的 scan 模块。
func surfaceJSON(mutate func(m map[string]any)) string {
	m := map[string]any{
		"declaration": surfaceIdent,
		"implementation": map[string]any{
			"repo":     "github.com/HankChenCH/example",
			"language": "go",
			"profile":  "renderer",
			"note":     "",
		},
		"contracts": []any{
			map[string]any{
				"id":      "expression-scan",
				"version": "v1.0",
				"moduleFiles": []any{
					map[string]any{"id": "scan", "support": "supported", "note": ""},
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

// surfaceTree 最小语料仓 + 合法声明文件（声明与语料一致）。
func surfaceTree(mutate func(m map[string]any)) map[string]string {
	files := baseTree(scanDoc(nil))
	files["support-surface.json"] = surfaceJSON(mutate)
	return files
}

// ---- 表驱动：三互检逐一红 + 绿用例 -----------------------------------------

func TestValidateSupportSurfaceTable(t *testing.T) {
	tests := []struct {
		name         string
		files        map[string]string
		wantErrCount int
		wantSubstrs  []string
	}{
		{
			name:  "绿：声明与语料一致",
			files: surfaceTree(nil),
		},
		{
			name: "绿：unsupported 条目 + note 人读备注",
			files: surfaceTree(func(m map[string]any) {
				m["contracts"] = []any{
					map[string]any{
						"id":      "expression-scan",
						"version": "v1.0",
						"moduleFiles": []any{
							map[string]any{"id": "scan", "support": "unsupported", "note": "编辑器画像外"},
						},
					},
				}
			}),
		},
		{
			// 坏用例类 1：形状（严格 typed 解码）
			name: "红：declaration 标识不符",
			files: surfaceTree(func(m map[string]any) {
				m["declaration"] = "canvas-semantics/support-surface v2"
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"declaration 标识"},
		},
		{
			name: "红：未知顶层键（未知字段即红）",
			files: surfaceTree(func(m map[string]any) {
				m["levels"] = []any{"gold"}
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"未知键"},
		},
		{
			name: "红：implementation 缺 language",
			files: surfaceTree(func(m map[string]any) {
				delete(m["implementation"].(map[string]any), "language")
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"缺少键"},
		},
		{
			name: "红：implementation.repo 空串",
			files: surfaceTree(func(m map[string]any) {
				m["implementation"].(map[string]any)["repo"] = ""
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"不可为空串"},
		},
		{
			name: "红：contracts 空数组（零契约无须签署）",
			files: surfaceTree(func(m map[string]any) {
				m["contracts"] = []any{}
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"contracts 不可为空"},
		},
		{
			name: "红：契约条目未知键",
			files: surfaceTree(func(m map[string]any) {
				m["contracts"].([]any)[0].(map[string]any)["status"] = "gold"
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"未知键"},
		},
		{
			// 坏用例类 2：注册表互检（契约/模块必须在语料目录内）
			name: "红：契约 id 不在语料注册表",
			files: surfaceTree(func(m map[string]any) {
				m["contracts"] = []any{
					map[string]any{
						"id":      "paginate-semantics",
						"version": "v1.0",
						"moduleFiles": []any{
							map[string]any{"id": "paginate", "support": "supported", "note": ""},
						},
					},
				}
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"不在语料注册表"},
		},
		{
			name: "红：模块文件 id 不在该版本目录",
			files: surfaceTree(func(m map[string]any) {
				m["contracts"].([]any)[0].(map[string]any)["moduleFiles"] = []any{
					map[string]any{"id": "path", "support": "supported", "note": ""},
				}
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"在语料 expression-scan v1.0 下不存在"},
		},
		{
			// 坏用例类 3：版本互检
			name: "红：版本目录不存在（声明落后于 pin）",
			files: surfaceTree(func(m map[string]any) {
				m["contracts"].([]any)[0].(map[string]any)["version"] = "v1.1"
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"在语料 expression-scan 下不存在"},
		},
		{
			name: "红：版本非规范形 v1",
			files: surfaceTree(func(m map[string]any) {
				m["contracts"].([]any)[0].(map[string]any)["version"] = "v1"
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"非规范形"},
		},
		{
			// 坏用例类 4：声明语义
			name: "红：support 非两值枚举",
			files: surfaceTree(func(m map[string]any) {
				m["contracts"].([]any)[0].(map[string]any)["moduleFiles"] = []any{
					map[string]any{"id": "scan", "support": "partial", "note": ""},
				}
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"非法（仅 supported / unsupported"},
		},
		{
			name: "红：契约重复声明",
			files: surfaceTree(func(m map[string]any) {
				m["contracts"] = []any{
					m["contracts"].([]any)[0],
					m["contracts"].([]any)[0],
				}
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"重复声明"},
		},
		{
			name: "红：模块条目重复声明",
			files: surfaceTree(func(m map[string]any) {
				m["contracts"].([]any)[0].(map[string]any)["moduleFiles"] = []any{
					map[string]any{"id": "scan", "support": "supported", "note": ""},
					map[string]any{"id": "scan", "support": "supported", "note": ""},
				}
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"重复声明"},
		},
		{
			name: "红：moduleFiles 空数组",
			files: surfaceTree(func(m map[string]any) {
				m["contracts"].([]any)[0].(map[string]any)["moduleFiles"] = []any{}
			}),
			wantErrCount: 1,
			wantSubstrs:  []string{"moduleFiles 不可为空"},
		},
		{
			name: "红：声明文件缺失",
			files: func() map[string]string {
				f := surfaceTree(nil)
				delete(f, "support-surface.json")
				return f
			}(),
			wantErrCount: 1,
			wantSubstrs:  []string{"读取失败"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := writeTree(t, tc.files)
			res := ValidateSupportSurface(root, filepath.Join(root, "support-surface.json"))
			assertResult(t, res, tc.wantErrCount, tc.wantSubstrs)
			if tc.wantErrCount == 0 && (res.SurfaceContracts != 1 || res.SurfaceModuleFiles != 1) {
				t.Fatalf("绿用例计数异常：%+v", res)
			}
		})
	}
}

// 端到端：go-canvas 的真实声明（仓外文件，针对真实语料根）必须全绿。
func TestRealGoSurfaceGreen(t *testing.T) {
	res := ValidateSupportSurface("..", "../../go-canvas/support-surface.json")
	if len(res.Errors) != 0 {
		t.Fatalf("go-canvas 真实声明应全绿：\n%s", joinErrs(res.Errors))
	}
	if res.SurfaceContracts != 4 {
		t.Fatalf("go-canvas 声明应含四契约：%+v", res)
	}
}

// 表面汇总输出子串防回归（CLI 摘要与计数联动）。
func TestSurfaceSummaryCounts(t *testing.T) {
	root := writeTree(t, surfaceTree(nil))
	res := ValidateSupportSurface(root, filepath.Join(root, "support-surface.json"))
	if len(res.Errors) != 0 {
		t.Fatalf("应全绿：\n%s", joinErrs(res.Errors))
	}
	if res.SurfaceContracts != 1 || res.SurfaceModuleFiles != 1 {
		t.Fatalf("计数不符：%+v", res)
	}
	if strings.Contains(joinErrs(res.Errors), "unsupported") {
		t.Fatalf("不应有 unsupported 相关错误：%s", joinErrs(res.Errors))
	}
}
