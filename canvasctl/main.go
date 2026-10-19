// canvasctl — canvas-semantics 治理工具（Go 单二进制、纯 stdlib）。
//
// 非规范性（GOVERNANCE §8）：只执行治理文件已钉死的形状与治理规则，
// 永不参与「什么是对的」裁决——语料管语义，runner 管求值，本工具只管门禁。
package main

import (
	"fmt"
	"os"
	"strings"
)

const usage = `canvasctl — canvas-semantics 治理工具（非规范性，见 GOVERNANCE.md §8）

用法:
  canvasctl validate [root] [--surface <file>]
      语料形状 + 治理校验（contract↔目录双写互检、code 白名单、用例名唯一）；
      --surface 追加支持面声明三互检（形状/注册表/版本，SUPPORT-SURFACE §5.2——
      实现仓 CI 在 submodule 内以 --surface ../support-surface.json 校验自身声明）
  canvasctl lint [root]
      治理 lint：提案期用例 ↔ 提案文档反链互检（GOVERNANCE §7）

语料布局:
  corpus/<contract-id>/<version>/<module-file>.json   契约 / 版本 / 模块文件
  corpus/proposals/<编号>-<slug>/                     提案期用例（volatile，lint 反链）
  spec/error-codes.json                               错误 code 注册表
`

// parseArgs 解析子命令后的参数：--surface 取值，其余为位置参数（root，至多一个）。
// 未知旗标 / 空串与多余位置参数返回 ok=false（usage 退出）。
func parseArgs(args []string) (root string, surface string, ok bool) {
	root = "."
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--surface":
			if i+1 >= len(args) {
				return "", "", false
			}
			i++
			surface = args[i]
		case args[i] == "-h" || args[i] == "--help":
			return "", "", false
		case strings.HasPrefix(args[i], "-"):
			return "", "", false
		case args[i] == "":
			return "", "", false
		case root != ".":
			return "", "", false
		default:
			root = args[i]
		}
	}
	return root, surface, true
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	sub, args := os.Args[1], os.Args[2:]

	switch sub {
	case "validate":
		root, surface, ok := parseArgs(args)
		if !ok {
			fmt.Fprint(os.Stderr, usage)
			os.Exit(2)
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

		if surface != "" {
			sres := ValidateSupportSurface(root, surface)
			for _, e := range sres.Errors {
				fmt.Fprintf(os.Stderr, "✗ %s\n", e)
			}
			if len(sres.Errors) > 0 {
				fmt.Fprintf(os.Stderr, "支持面声明未通过：%d 处不合\n", len(sres.Errors))
				os.Exit(1)
			}
			fmt.Printf("✓ 支持面声明通过（三互检）：%d 契约 / %d 模块条目\n",
				sres.SurfaceContracts, sres.SurfaceModuleFiles)
		}

	case "lint":
		root, surface, ok := parseArgs(args)
		if !ok || surface != "" {
			fmt.Fprint(os.Stderr, usage)
			os.Exit(2)
		}
		res := Lint(root)
		for _, e := range res.Errors {
			fmt.Fprintf(os.Stderr, "✗ %s\n", e)
		}
		if len(res.Errors) > 0 {
			fmt.Fprintf(os.Stderr, "lint 未通过：%d 处不合\n", len(res.Errors))
			os.Exit(1)
		}
		fmt.Printf("✓ lint 通过：反链互检 %d 提案文档 / %d 提案期用例目录\n",
			res.ProposalDocs, res.ProposalCaseDirs)

	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}
