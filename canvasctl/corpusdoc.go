// corpusdoc — diff / release 共用的语料模块文档最小加载面。
//
// 严格解码到「可比对 / 可清点」为止：文档对象 + meta（复用 validate 的 decodeMeta，
// 含 contract 双写互检）+ cases 非空数组且元素皆对象、用例名非空字符串。用例体与
// 文档级附加键不按契约 schema 钉键集——那是 validate 的职责（语料形状的权威只有一处）；
// 本文件只产出 diff 的比对单元与 release 的清点单元，皆为原始 JSON（工具非规范性，
// GOVERNANCE §8）。

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// moduleDoc 语料模块文档的最小内存形态。
type moduleDoc struct {
	relPath string
	meta    corpusMeta
	names   []string                   // 用例名，文件内出现序
	bodies  map[string]json.RawMessage // 用例名 → 用例对象原始 JSON
	doc     map[string]json.RawMessage // 完整文档（meta/cases 之外的文档级语义附加键，如 dateTokens，在此）
}

// loadModuleDoc 严格解码模块文档。contractID + version 用于 meta.contract 双写互检
// （GOVERNANCE §4，与 validate 同规）——版本目录里的文件冒认他版本即红。
func loadModuleDoc(relPath, contractID, version string, data []byte) (*moduleDoc, error) {
	doc, err := decodeObject(data, relPath)
	if err != nil {
		return nil, err
	}
	for _, k := range []string{"meta", "cases"} {
		if _, ok := doc[k]; !ok {
			return nil, fmt.Errorf("%s：缺少键 %q", relPath, k)
		}
	}
	meta, err := decodeMeta(doc["meta"], relPath)
	if err != nil {
		return nil, err
	}
	if want := contractID + " " + version; meta.contract != want {
		return nil, fmt.Errorf("%s：meta.contract %q 与目录名不一致（双写互检，应为 %q）", relPath, meta.contract, want)
	}
	casesRaw, err := decodeArrayField(doc, "cases", relPath)
	if err != nil {
		return nil, err
	}
	if len(casesRaw) == 0 {
		return nil, fmt.Errorf("%s：用例集不可为空", relPath)
	}
	md := &moduleDoc{relPath: relPath, meta: meta, bodies: map[string]json.RawMessage{}, doc: doc}
	for i, caseRaw := range casesRaw {
		ctx := fmt.Sprintf("%s：用例 cases[%d]", relPath, i)
		var m map[string]json.RawMessage
		if err := json.Unmarshal(caseRaw, &m); err != nil {
			return nil, fmt.Errorf("%s 应为对象", ctx)
		}
		name, err := decodeStringField(m, "name", ctx)
		if err != nil {
			return nil, err
		}
		if name == "" {
			return nil, fmt.Errorf("%s：用例名不可为空", ctx)
		}
		md.names = append(md.names, name)
		md.bodies[name] = caseRaw
	}
	return md, nil
}

// loadVersionDir 加载契约版本目录下全部模块文件（文件名序）。目录缺失、非 json 条目、
// 任一文件不可严格解码、用例名跨文件重复（按名索引的前提，作用域 = 契约版本，
// 与 validate 同规）即错——diff/release 拒绝为不可信语料出产物。
func loadVersionDir(root, contractID, version string) (map[string]*moduleDoc, []error) {
	dirRel := "corpus/" + contractID + "/" + version
	dir := filepath.Join(root, "corpus", contractID, version)
	entries, err := readSortedEntries(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, []error{fmt.Errorf("版本目录 %s 不存在（坏版本目录对）", dirRel)}
		}
		return nil, []error{fmt.Errorf("%s：目录读取失败：%v", dirRel, err)}
	}
	var errs []error
	docs := map[string]*moduleDoc{}
	var fnames []string
	for _, e := range entries {
		fname := e.Name()
		if strings.HasPrefix(fname, ".") {
			continue
		}
		if e.IsDir() || !strings.HasSuffix(fname, ".json") {
			errs = append(errs, fmt.Errorf("%s：意外条目 %q（版本目录下只允许 *.json 模块文件）", dirRel, fname))
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, fname))
		if err != nil {
			errs = append(errs, fmt.Errorf("%s/%s：读取失败：%v", dirRel, fname, err))
			continue
		}
		md, err := loadModuleDoc(dirRel+"/"+fname, contractID, version, data)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		docs[fname] = md
		fnames = append(fnames, fname)
	}
	seen := map[string]string{}
	for _, fname := range fnames {
		for _, name := range docs[fname].names {
			if prev, dup := seen[name]; dup {
				errs = append(errs, fmt.Errorf("%s：用例名重复 %q（与 %s 冲突，按名索引不可信）", dirRel, name, prev))
			} else {
				seen[name] = fname
			}
		}
	}
	if len(errs) > 0 {
		return nil, errs
	}
	return docs, nil
}

// canonicalJSON 规范形：解码（数字保留字面）后重序列化——对象键全序、空白压缩、
// 数组序保持。语料 diff 的相等性基准：键序与排版不敏感、数值字面敏感
// （语料数值纪律为十进制字符串，1 与 1.0 是不同字面，diff 不猜数值等价）。
func canonicalJSON(raw json.RawMessage) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return "", err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// sameRaw 两段原始 JSON 语义相等（规范形比较；任一侧不可解码时退字节比较）。
func sameRaw(a, b json.RawMessage) bool {
	sa, errA := canonicalJSON(a)
	sb, errB := canonicalJSON(b)
	if errA != nil || errB != nil {
		return string(a) == string(b)
	}
	return sa == sb
}

// sortedUnion 两侧键集并集，升序。
func sortedUnion(keysA, keysB []string) []string {
	seen := map[string]bool{}
	for _, k := range append(append([]string{}, keysA...), keysB...) {
		seen[k] = true
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
