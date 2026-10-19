// canvasctl release — 版本化语料产物（目录快照包 + checksum，GOVERNANCE §8）。
//
// 产物形态：一个自含目录——corpus/ 快照（契约版本目录的模块文件，提案期用例区除外：
// volatile、不占 contract 版本，GOVERNANCE §7）+ spec/ 快照（薄规格正文与错误 code
// 注册表，语义权威的一部分）+ MANIFEST.json（格式标识 + 契约/版本/用例清点 + 逐文件
// sha256 checksum）。供消费仓 pin 与审计存档：快照内语料自含且 validate 全绿，
// 任何比特变化都与清单冲突。
//
// 版本一致性自检（发版即门禁，与 diff 的「呈现不裁断」不同）：语料 validate 全绿是
// 发版前置；产物落盘后复算——清单文件集与磁盘逐一相符、checksum 逐文件重验、快照内
// 每个模块文件 meta.contract 与快照路径双写一致（GOVERNANCE §4）。

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// releaseManifest 产物清单格式标识：格式演进入新版本字符串，不原地改义
// （同错误 code 注册表与支持面声明的 ident 纪律）。
const releaseIdent = "canvas-semantics/release-manifest v1"

type releaseManifest struct {
	Manifest  string             `json:"manifest"`
	Contracts []manifestContract `json:"contracts"`
	Files     []manifestFile     `json:"files"`
}

type manifestContract struct {
	ID       string            `json:"id"`
	Versions []manifestVersion `json:"versions"`
}

type manifestVersion struct {
	Version     string           `json:"version"`
	ModuleFiles []manifestModule `json:"moduleFiles"`
}

type manifestModule struct {
	File  string `json:"file"`
	Cases int    `json:"cases"`
}

type manifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

// ReleaseResult release 全量结果：Result 携带错误与清点计数。
type ReleaseResult struct {
	Result
	Out string
}

// Release 校验 root（canvas-semantics 仓根）语料后，向 out 产出目录快照包。
// 任何失败路径都不覆盖既有产物；validate 不合时不落任何文件。
func Release(root, out string) ReleaseResult {
	var res ReleaseResult
	res.Out = out

	// 门卫一：产物落点不可自污染——仓根、corpus/、spec/ 及其子树内禁止。
	if err := checkOutPlacement(root, out); err != nil {
		res.err(err)
		return res
	}
	// 门卫二（版本一致性自检的入门票）：语料 validate 全绿才发版。逐条转发会让
	// release 输出淹没在 validate 噪声里——只报计数，修语料归 validate 命令。
	vres := Validate(root)
	if len(vres.Errors) > 0 {
		res.errf("release 前置自检未过：语料 validate %d 处不合（先跑 canvasctl validate 修复后再发版）", len(vres.Errors))
		return res
	}
	// 门卫三：out 不覆盖既有产物。
	switch info, err := os.Stat(out); {
	case err == nil && !info.IsDir():
		res.errf("产物目录 %s 不是目录", out)
		return res
	case err == nil:
		if entries, rerr := os.ReadDir(out); rerr == nil && len(entries) > 0 {
			res.errf("产物目录 %s 已存在且非空（不覆盖既有产物）", out)
			return res
		}
	case !os.IsNotExist(err):
		res.errf("产物目录 %s 状态不可判：%v", out, err)
		return res
	}

	// 收集快照源（先全量读入再落盘：任何收集失败发生在建 out 之前，无半成品目录）。
	files, inventory, err := collectSnapshot(root)
	if err != nil {
		res.errf("语料快照收集失败：%v", err)
		return res
	}
	manifest := buildManifest(inventory, files)

	if err := writeArtifact(root, out, files, manifest); err != nil {
		os.RemoveAll(out)
		res.errf("产物写入失败（已回滚）：%v", err)
		return res
	}

	// 产物自检（版本一致性）：文件集相符 + checksum 复验 + 快照内双写互检。
	if errs := verifyArtifact(out, manifest); len(errs) > 0 {
		res.Errors = append(res.Errors, errs...)
		res.errf("产物自检未过（%d 处；产物保留于 %s 供排查，勿分发）", len(errs), out)
		return res
	}

	for _, c := range manifest.Contracts {
		res.ReleaseContracts++
		for _, v := range c.Versions {
			res.ReleaseVersions++
			res.ReleaseModuleFiles += len(v.ModuleFiles)
			for _, mf := range v.ModuleFiles {
				res.ReleaseCases += mf.Cases
			}
		}
	}
	res.ReleaseFiles = len(manifest.Files)
	for _, f := range manifest.Files {
		res.ReleaseBytes += f.Bytes
	}
	return res
}

// checkOutPlacement 产物落点守卫：out 不得是仓根本身、不得落在 corpus/ 或 spec/ 内
// （产物写进语料 = 自污染语料面）。root 外路径（相对路径出仓等）不限制。
func checkOutPlacement(root, out string) error {
	rel, err := filepath.Rel(root, out)
	if err != nil {
		return nil // 不同盘等不可相对化场景，落点必然在仓外，放行
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return fmt.Errorf("产物目录 %s 不可位于仓根、corpus/ 或 spec/ 内", out)
	}
	if rel == "corpus" || rel == "spec" || strings.HasPrefix(rel, "corpus/") || strings.HasPrefix(rel, "spec/") {
		return fmt.Errorf("产物目录 %s 不可位于仓根、corpus/ 或 spec/ 内", out)
	}
	return nil
}

// collectSnapshot 收集快照源文件与契约清点。语料部分：validate 全绿后 walk corpus/
// （跳过点条目、corpus/README.md、提案期用例区）；这里再次逐文件严格解码与双写互检
// 属防御纵深，正常不可达。薄规格部分：spec/ 下全部非点文件。
func collectSnapshot(root string) ([]manifestFile, map[string]map[string]map[string]int, error) {
	var files []manifestFile
	// contract → version → module file → cases
	inventory := map[string]map[string]map[string]int{}

	corpusRoot := filepath.Join(root, "corpus")
	err := filepath.WalkDir(corpusRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			if name == "proposals" && path == filepath.Join(corpusRoot, "proposals") {
				// 提案期用例区：volatile、不占 contract 版本（GOVERNANCE §7），不入产物。
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return rerr
		}
		relSlash := filepath.ToSlash(rel)
		if relSlash == "corpus/README.md" {
			return nil // 语料编写规范文档（非语料），与 validate 同待遇
		}
		if !strings.HasSuffix(name, ".json") {
			return fmt.Errorf("意外条目 %s（版本目录下只允许 *.json 模块文件）", relSlash)
		}
		parts := strings.Split(relSlash, "/")
		if len(parts) != 4 {
			return fmt.Errorf("语料布局异常 %s（应为 corpus/<契约>/<版本>/<模块>.json）", relSlash)
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		md, rerr := loadModuleDoc(relSlash, parts[1], parts[2], data)
		if rerr != nil {
			return rerr
		}
		cases := inventory[parts[1]]
		if cases == nil {
			cases = map[string]map[string]int{}
			inventory[parts[1]] = cases
		}
		moduleFiles := cases[parts[2]]
		if moduleFiles == nil {
			moduleFiles = map[string]int{}
			cases[parts[2]] = moduleFiles
		}
		moduleFiles[parts[3]] = len(md.names)
		files = append(files, manifestFile{Path: relSlash})
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	specRoot := filepath.Join(root, "spec")
	err = filepath.WalkDir(specRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return rerr
		}
		files = append(files, manifestFile{Path: filepath.ToSlash(rel)})
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	for i := range files {
		data, rerr := os.ReadFile(filepath.Join(root, filepath.FromSlash(files[i].Path)))
		if rerr != nil {
			return nil, nil, rerr
		}
		sum := sha256.Sum256(data)
		files[i].SHA256 = hex.EncodeToString(sum[:])
		files[i].Bytes = int64(len(data))
	}
	return files, inventory, nil
}

// buildManifest 清单组装：契约 id 升序、版本主次序、模块文件名升序。
func buildManifest(inventory map[string]map[string]map[string]int, files []manifestFile) releaseManifest {
	manifest := releaseManifest{Manifest: releaseIdent}
	ids := make([]string, 0, len(inventory))
	for id := range inventory {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		mc := manifestContract{ID: id}
		versions := make([]string, 0, len(inventory[id]))
		for v := range inventory[id] {
			versions = append(versions, v)
		}
		sort.Slice(versions, func(i, j int) bool { return versionLess(versions[i], versions[j]) })
		for _, v := range versions {
			mv := manifestVersion{Version: v}
			fnames := make([]string, 0, len(inventory[id][v]))
			for f := range inventory[id][v] {
				fnames = append(fnames, f)
			}
			sort.Strings(fnames)
			for _, f := range fnames {
				mv.ModuleFiles = append(mv.ModuleFiles, manifestModule{File: f, Cases: inventory[id][v][f]})
			}
			mc.Versions = append(mc.Versions, mv)
		}
		manifest.Contracts = append(manifest.Contracts, mc)
	}
	manifest.Files = files
	return manifest
}

// writeArtifact 落盘：快照文件（源路径即产物路径）+ MANIFEST.json。快照字节重读自源
// （root 相对路径）——checksum 与落盘字节强一致，不受收集时的内存副本影响。
func writeArtifact(root, out string, files []manifestFile, manifest releaseManifest) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	for _, f := range files {
		dst := filepath.Join(out, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f.Path)))
		if err != nil {
			return err
		}
		if err := os.WriteFile(dst, src, 0o644); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "MANIFEST.json"), append(data, '\n'), 0o644)
}

// verifyArtifact 产物自检：磁盘文件集（MANIFEST.json 除外）与清单逐一相符、checksum
// 复验、快照内每个语料模块文件 meta.contract 与快照路径双写一致（GOVERNANCE §4）。
func verifyArtifact(out string, manifest releaseManifest) []error {
	var errs []error
	want := map[string]manifestFile{}
	for _, f := range manifest.Files {
		want[f.Path] = f
	}
	got := map[string]bool{}
	err := filepath.WalkDir(out, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(out, path)
		if rerr != nil {
			return rerr
		}
		relSlash := filepath.ToSlash(rel)
		if relSlash == "MANIFEST.json" {
			return nil
		}
		got[relSlash] = true
		w, ok := want[relSlash]
		if !ok {
			errs = append(errs, fmt.Errorf("产物出现清单外文件 %s", relSlash))
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			errs = append(errs, fmt.Errorf("产物文件 %s 读取失败：%v", relSlash, rerr))
			return nil
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != w.SHA256 || int64(len(data)) != w.Bytes {
			errs = append(errs, fmt.Errorf("产物文件 %s checksum 复验不符", relSlash))
		}
		parts := strings.Split(relSlash, "/")
		if len(parts) == 4 && parts[0] == "corpus" {
			if _, rerr := loadModuleDoc(relSlash, parts[1], parts[2], data); rerr != nil {
				errs = append(errs, rerr)
			}
		}
		return nil
	})
	if err != nil {
		errs = append(errs, fmt.Errorf("产物目录遍历失败：%v", err))
		return errs
	}
	for p := range want {
		if !got[p] {
			errs = append(errs, fmt.Errorf("清单内文件 %s 未落盘", p))
		}
	}
	return errs
}

// versionLess 版本主次序比较（v<主>.<次>，规范形已由 validate 保证）。
func versionLess(a, b string) bool {
	am, an := versionKey(a)
	bm, bn := versionKey(b)
	return am < bm || (am == bm && an < bn)
}

func versionKey(v string) (int, int) {
	major, minor, _ := strings.Cut(strings.TrimPrefix(v, "v"), ".")
	m, _ := strconv.Atoi(major)
	n, _ := strconv.Atoi(minor)
	return m, n
}

// SummaryLine CLI 成功摘要行（产物路径 + 清点 + 自检结论）。
func (r ReleaseResult) SummaryLine() string {
	return fmt.Sprintf("✓ release 产物 %s：%d 契约 / %d 版本目录 / %d 模块文件 / %d 用例；%d 文件 / %d 字节；自检通过（MANIFEST.json + sha256）",
		r.Out, r.ReleaseContracts, r.ReleaseVersions, r.ReleaseModuleFiles, r.ReleaseCases,
		r.ReleaseFiles, r.ReleaseBytes)
}

// parseReleaseArgs 解析 release 参数：[root] --out <dir>（--out 必填；root 位置参数
// 至多一个，缺省句点）；空串 / 未知旗标 / --out 缺值或空值 / 多余位置参数退 usage。
func parseReleaseArgs(args []string) (root, out string, ok bool) {
	root = "."
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--out":
			if i+1 >= len(args) || args[i+1] == "" {
				return "", "", false
			}
			i++
			out = args[i]
		case args[i] == "-h" || args[i] == "--help":
			return "", "", false
		case args[i] == "" || strings.HasPrefix(args[i], "-"):
			return "", "", false
		case root != ".":
			return "", "", false
		default:
			root = args[i]
		}
	}
	if out == "" {
		return "", "", false
	}
	return root, out, true
}
