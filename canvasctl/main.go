// canvasctl — canvas-semantics 治理工具（Go 单二进制、纯 stdlib）。
//
// 非规范性（GOVERNANCE §8）：只执行治理文件已钉死的形状与治理规则，
// 永不参与「什么是对的」裁决——语料管语义，runner 管求值，本工具只管门禁。
package main

import (
	"fmt"
	"os"
)

const usage = `canvasctl — canvas-semantics 治理工具（非规范性，见 GOVERNANCE.md §8）

用法:
  canvasctl validate [root]    校验 root（默认当前目录）下的语料形状与治理规则

语料布局:
  corpus/<contract-id>/<version>/<module-file>.json   契约 / 版本 / 模块文件
  spec/error-codes.json                               错误 code 注册表
`

func main() {
	if len(os.Args) < 2 || os.Args[1] != "validate" {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	root := "."
	if len(os.Args) > 2 {
		root = os.Args[2]
	}

	res := Validate(root)
	for _, e := range res.Errors {
		fmt.Fprintf(os.Stderr, "✗ %s\n", e)
	}
	if len(res.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "validate 未通过：%d 处不合\n", len(res.Errors))
		os.Exit(1)
	}
	fmt.Printf("✓ validate 通过：%d 契约 / %d 版本目录 / %d 模块文件 / %d 用例\n",
		res.Contracts, res.Versions, res.ModuleFiles, res.Cases)
}
