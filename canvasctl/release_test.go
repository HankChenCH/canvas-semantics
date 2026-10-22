package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- 测试脚手架 ----------------------------------------------------------
//
// release 的前置门卫是 validate 全绿，夹具复用 validate_test.go 的 baseTree/scanDoc/
// registryJSON（最小合法语料仓）。产物目录取 root 内非语料路径（validate 先行，无自污染）。

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// testManifest 产物清单的测试侧解码形态（与 release.go 的结构体互为镜像，防漂移）。
type testManifest struct {
	Manifest  string `json:"manifest"`
	Contracts []struct {
		ID       string `json:"id"`
		Versions []struct {
			Version     string `json:"version"`
			ModuleFiles []struct {
				File  string `json:"file"`
				Cases int    `json:"cases"`
			} `json:"moduleFiles"`
		} `json:"versions"`
	} `json:"contracts"`
	Files []struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Bytes  int64  `json:"bytes"`
	} `json:"files"`
}

func releaseOut(root string) string {
	return filepath.Join(root, "release-out")
}

// ---- 表驱动：正路径产物形态 + 负路径 ----------------------------------------

func TestReleaseTable(t *testing.T) {
	tests := []struct {
		name         string
		files        map[string]string
		out          func(root string) string
		wantErrCount int
		wantSubstrs  []string
		// wantArtifactAbsent 失败路径不得建出产物目录（前置自检失败连目录都不落）。
		wantArtifactAbsent bool
		want               func(t *testing.T, root string, res ReleaseResult)
	}{
		{
			name:  "绿：最小仓出产物（out 不存在则创建）",
			files: baseTree(scanDoc(nil)),
			out:   releaseOut,
			want: func(t *testing.T, root string, res ReleaseResult) {
				if res.ReleaseContracts != 1 || res.ReleaseVersions != 1 ||
					res.ReleaseModuleFiles != 1 || res.ReleaseCases != 2 {
					t.Fatalf("清点计数不符：%+v", res.Result)
				}
				// files = 语料模块文件 + spec/error-codes.json
				if res.ReleaseFiles != 2 {
					t.Fatalf("ReleaseFiles = %d，want 2", res.ReleaseFiles)
				}
				if _, err := os.Stat(filepath.Join(releaseOut(root), "MANIFEST.json")); err != nil {
					t.Fatalf("MANIFEST.json 缺失：%v", err)
				}
			},
		},
		{
			name:  "绿：out 已存在但为空目录（可复用）",
			files: baseTree(scanDoc(nil)),
			out: func(root string) string {
				out := releaseOut(root)
				if err := os.MkdirAll(out, 0o755); err != nil {
					t.Fatal(err)
				}
				return out
			},
			want: func(t *testing.T, root string, res ReleaseResult) {
				if res.ReleaseFiles != 2 {
					t.Fatalf("ReleaseFiles = %d，want 2", res.ReleaseFiles)
				}
			},
		},
		{
			// 版本一致性自检的入门票：语料不合不发版，且不落任何产物文件。
			name: "红：语料 validate 不合（release 前置自检未过）",
			files: baseTree(scanDoc(func(m map[string]any) {
				m["extra"] = true
			})),
			out:                releaseOut,
			wantErrCount:       1,
			wantSubstrs:        []string{`release 前置自检未过`, `validate`},
			wantArtifactAbsent: true,
		},
		{
			name:  "红：out 已存在且非空（不覆盖既有产物）",
			files: baseTree(scanDoc(nil)),
			out: func(root string) string {
				out := releaseOut(root)
				if err := os.MkdirAll(out, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(out, "previous.txt"), []byte("x"), 0o644); err != nil {
					t.Fatal(err)
				}
				return out
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`已存在且非空`},
		},
		{
			name:  "红：out 落在语料区内（产物自污染禁止）",
			files: baseTree(scanDoc(nil)),
			out: func(root string) string {
				return filepath.Join(root, "corpus", "release-out")
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`不可位于仓根、corpus/ 或 spec/ 内`},
		},
		{
			name:  "红：out 即仓根",
			files: baseTree(scanDoc(nil)),
			out: func(root string) string {
				return root
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`不可位于仓根、corpus/ 或 spec/ 内`},
		},
		{
			name:  "红：out 是普通文件",
			files: baseTree(scanDoc(nil)),
			out: func(root string) string {
				p := filepath.Join(root, "a-file")
				if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
					t.Fatal(err)
				}
				return p
			},
			wantErrCount: 1,
			wantSubstrs:  []string{`不是目录`},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := writeTree(t, tc.files)
			out := tc.out(root)
			res := Release(root, out)
			assertResult(t, res.Result, tc.wantErrCount, tc.wantSubstrs)
			if tc.wantErrCount > 0 {
				if tc.wantArtifactAbsent {
					if _, err := os.Stat(out); !os.IsNotExist(err) {
						t.Fatalf("失败路径不得建产物目录（%v）", err)
					}
				}
				return
			}
			if tc.want != nil {
				tc.want(t, root, res)
			}
		})
	}
}

// ---- 产物形态：目录快照包 + checksum 清单 + 版本一致性 -----------------------

func TestReleaseArtifactShape(t *testing.T) {
	files := baseTree(scanDoc(nil))
	// 提案期用例：volatile、不占 contract 版本（GOVERNANCE §7）——不入产物。
	files["corpus/proposals/0002-x/fn-date.json"] = `{}`
	// 点条目与语料编写规范文档：validate 不校验，release 不快照。
	files["corpus/expression-scan/v1.0/.DS_Store"] = "junk"
	files["corpus/README.md"] = "# 语料编写规范"
	// 薄规格正文：语义权威的一部分（语料钉不住的裁决）——入产物。
	files["spec/expression.md"] = "# expression 薄规格"
	root := writeTree(t, files)

	res := Release(root, releaseOut(root))
	if len(res.Errors) != 0 {
		t.Fatalf("应全绿：\n%s", joinErrs(res.Errors))
	}

	manifestData, err := os.ReadFile(filepath.Join(releaseOut(root), "MANIFEST.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest testManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Manifest != releaseIdent {
		t.Fatalf("清单标识 %q 不符（应为 %q）", manifest.Manifest, releaseIdent)
	}

	// 文件集 = 快照语料 + spec/（proposals、点条目、corpus/README.md 排除）
	wantFiles := map[string]string{
		"corpus/expression-scan/v1.0/scan.json": sha256Hex([]byte(files["corpus/expression-scan/v1.0/scan.json"])),
		"spec/error-codes.json":                 sha256Hex([]byte(files["spec/error-codes.json"])),
		"spec/expression.md":                    sha256Hex([]byte(files["spec/expression.md"])),
	}
	gotFiles := map[string]string{}
	for _, f := range manifest.Files {
		gotFiles[f.Path] = f.SHA256
	}
	if len(gotFiles) != len(wantFiles) {
		t.Fatalf("清单文件集不符：%v", gotFiles)
	}
	for p, wantSHA := range wantFiles {
		gotSHA, ok := gotFiles[p]
		if !ok {
			t.Fatalf("清单缺 %s", p)
		}
		if gotSHA != wantSHA {
			t.Fatalf("%s checksum 不符：got %s want %s", p, gotSHA, wantSHA)
		}
		// 快照文件与源逐字节一致
		snap, err := os.ReadFile(filepath.Join(releaseOut(root), filepath.FromSlash(p)))
		if err != nil {
			t.Fatal(err)
		}
		if sha256Hex(snap) != wantSHA {
			t.Fatalf("%s 快照字节与源不符", p)
		}
	}
	for _, banned := range []string{"corpus/proposals/0002-x/fn-date.json", "corpus/README.md", "corpus/expression-scan/v1.0/.DS_Store"} {
		if _, ok := gotFiles[banned]; ok {
			t.Fatalf("%s 不应入产物", banned)
		}
		if _, err := os.Stat(filepath.Join(releaseOut(root), filepath.FromSlash(banned))); !os.IsNotExist(err) {
			t.Fatalf("%s 不应出现在快照目录", banned)
		}
	}

	// 契约清点：expression-scan v1.0 / scan.json / 2 用例
	if len(manifest.Contracts) != 1 || manifest.Contracts[0].ID != "expression-scan" {
		t.Fatalf("契约清点不符：%+v", manifest.Contracts)
	}
	ver := manifest.Contracts[0].Versions
	if len(ver) != 1 || ver[0].Version != "v1.0" || len(ver[0].ModuleFiles) != 1 ||
		ver[0].ModuleFiles[0].File != "scan.json" || ver[0].ModuleFiles[0].Cases != 2 {
		t.Fatalf("版本清点不符：%+v", ver)
	}

	// 产物是自含语料仓：validate 对快照根全绿
	if vres := Validate(releaseOut(root)); len(vres.Errors) != 0 {
		t.Fatalf("产物内语料应自含全绿：\n%s", joinErrs(vres.Errors))
	}
}

// 端到端：真实语料整仓发版。清点计数钉住当前语料面（5 契约 / 10 版本目录 /
// 16 模块文件 / 390 用例 + spec/ 4 文件）；产物内语料自含 validate 全绿。
// （paginate-semantics v2.0 转正后 8→9 / 14→15 / 350→370 / 18→19；
//
//	v2.1 勘误版新增目录后 9→10 / 15→16 / 370→390 / 19→20）
func TestReleaseRealCorpus(t *testing.T) {
	out := filepath.Join(t.TempDir(), "release")
	res := Release("..", out)
	if len(res.Errors) != 0 {
		t.Fatalf("真实语料应可发版：\n%s", joinErrs(res.Errors))
	}
	if res.ReleaseContracts != 5 || res.ReleaseVersions != 10 || res.ReleaseModuleFiles != 16 ||
		res.ReleaseCases != 390 || res.ReleaseFiles != 20 {
		t.Fatalf("真实语料清点不符：%+v", res.Result)
	}
	if res.ReleaseBytes <= 0 {
		t.Fatalf("ReleaseBytes 异常：%d", res.ReleaseBytes)
	}
	if vres := Validate(out); len(vres.Errors) != 0 {
		t.Fatalf("产物内语料应自含全绿：\n%s", joinErrs(vres.Errors))
	}
	// 契约按 id 升序、版本按主次序（v1.0 < v1.1 < v2.0）排序可预期
	manifestData, err := os.ReadFile(filepath.Join(out, "MANIFEST.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest testManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Contracts) != 5 {
		t.Fatalf("契约数 %d，want 5", len(manifest.Contracts))
	}
	for i := 1; i < len(manifest.Contracts); i++ {
		if manifest.Contracts[i-1].ID >= manifest.Contracts[i].ID {
			t.Fatalf("契约未按 id 升序：%s ≥ %s", manifest.Contracts[i-1].ID, manifest.Contracts[i].ID)
		}
	}
	for _, c := range manifest.Contracts {
		for i := 1; i < len(c.Versions); i++ {
			if !versionLess(c.Versions[i-1].Version, c.Versions[i].Version) {
				t.Fatalf("%s 版本未按主次序：%s ≥ %s", c.ID, c.Versions[i-1].Version, c.Versions[i].Version)
			}
		}
	}
}

// CLI 参数解析：--out 必带值且必填，位置参数至多一个（root），空串/未知旗标退 usage。
func TestParseReleaseArgs(t *testing.T) {
	if _, _, ok := parseReleaseArgs(nil); ok {
		t.Fatal("缺 --out 应退 usage")
	}
	if _, _, ok := parseReleaseArgs([]string{"--out"}); ok {
		t.Fatal("--out 缺值应退 usage")
	}
	if _, _, ok := parseReleaseArgs([]string{"--out", ""}); ok {
		t.Fatal("--out 空值应退 usage")
	}
	if _, _, ok := parseReleaseArgs([]string{""}); ok {
		t.Fatal("空串位置参数应退 usage")
	}
	if _, _, ok := parseReleaseArgs([]string{"--unknown"}); ok {
		t.Fatal("未知旗标应退 usage")
	}
	if _, _, ok := parseReleaseArgs([]string{"a", "b"}); ok {
		t.Fatal("多余位置参数应退 usage")
	}
	root, out, ok := parseReleaseArgs([]string{"root/dir", "--out", "dist/rel"})
	if !ok || root != "root/dir" || out != "dist/rel" {
		t.Fatalf("合法参数解析不符：%q %q %v", root, out, ok)
	}
	root, out, ok = parseReleaseArgs([]string{"--out", "dist/rel"})
	if !ok || root != "." || out != "dist/rel" {
		t.Fatalf("缺省 root 解析不符：%q %q %v", root, out, ok)
	}
}

// 防回归：release 摘要行（CLI 成功输出）含产物路径与自检结论。
func TestReleaseSummaryLine(t *testing.T) {
	root := writeTree(t, baseTree(scanDoc(nil)))
	res := Release(root, releaseOut(root))
	if len(res.Errors) != 0 {
		t.Fatalf("应全绿：\n%s", joinErrs(res.Errors))
	}
	line := res.SummaryLine()
	for _, want := range []string{"release 产物", "自检通过", "MANIFEST.json"} {
		if !strings.Contains(line, want) {
			t.Fatalf("摘要行缺少 %q：%s", want, line)
		}
	}
}
