# canvas-semantics

动态数据编译语义（表达式求值 / 扫描切分 / hydrate 填充 / flow 流装箱 / paginate 分页）的**中立规格仓**：五份 JSON contract 语料在这里升格为手维护的 executable spec（语料先行），PHP / Go / TS 三端降为平等消费实现。背景裁决见工作区 `docs/adr/0024-corpus-semantic-authority-flip.md`。

**语料管语义，runner 管求值**：本仓钉死「什么是对的」（语料用例 + 薄规格正文）；三端实现仓各自维护一层数据驱动 runner，从公共管线入口消费同一批语料文件、逐字段比对结果——runner 里没有任何语义，语义漂移 = 语料改了而实现没跟上 = runner 变红。第四语言接入 = clone 本仓 + 写薄 runner 对红变绿 + 签署支持面声明，接入成本与语言数量无关。

## 顶层分区（五区）

| 分区 | 位置 | 内容 |
|---|---|---|
| 治理 | `GOVERNANCE.md`、`SUPPORT-SURFACE.md` | MUST / MUST NOT / MAY 清单、contract 版本政策、提案生命周期、支持面声明格式与模块注册表 |
| 薄规格 | `spec/` | 语料钉不住的裁决（优先级、诊断发射顺序、未定义区）+ 错误 code 注册表 `spec/error-codes.json` |
| 语料 | `corpus/` | 五契约用例集，按契约分目录、版本目录化（executable spec 本体） |
| 提案 | `proposals/` | 语义演进的唯一通道（意图文档 + 提案期用例双轨，proposal → integrated） |
| 工具 | `canvasctl/` | Go 单二进制治理工具（当前仅 `validate` 子命令；lint / diff / release 随后续工票） |

## 语料布局与版本政策

```
corpus/<contract-id>/<version>/<module-file>.json
         expression-scan   v1.0    scan
```

- **双写互检**：目录名与文件内 `meta.contract`（形如 `"expression-scan v1.0"`）精确互为镜像，`canvasctl validate` 强制（`GOVERNANCE.md` §4）。
- **主版本 = 语义变更**（新目录），**次版本 = 纯增用例或勘误**（新目录 + `meta.errata` 注记）；已发布版本目录整体冻结，用例期望值禁改、永不回移、永不删除。
- 模块文件 = 版本目录下的 `*.json`；文件名即模块 id（与 `SUPPORT-SURFACE.md` 模块注册表对表）。`corpus/proposals/<提案编号>-<slug>/` 是提案期用例区，volatile、不占 contract 版本。
- `meta` 只保留 contract 标识与出处注记（`notes` / `errata`）；语义注记正文一律写进薄规格，不在 `meta` 里长语义（`GOVERNANCE.md` §9）。

## canvasctl（Go 单二进制、纯 stdlib）

```sh
go run ./canvasctl validate .    # 校验当前仓（corpus/ + spec/error-codes.json）
go build -o bin/ ./canvasctl     # 构建单二进制（产物进 bin/；根级 canvasctl 与源码目录同名，勿裸 build）
go test ./canvasctl/             # 工具自身的表驱动测试
```

`validate` 承担：语料形状校验（**严格 typed 解码——缺字段、未知字段即红**，「语料是法，法必须严」）、contract↔目录双写互检、错误 code 白名单（active `codes` 之外一律红，`reserved` 区永不复用）、用例名唯一（契约版本作用域）、版本目录规范形（`v<主>.<次>`）与模块文件布局检查。发现即红、逐条带定位（文件 / 用例名 / part 序号），exit 1；全绿输出计数摘要。

> **非规范性声明**：canvasctl 只执行 `GOVERNANCE.md` 已钉死的形状与治理规则，**永不参与「什么是对的」裁决**。工具与语料结论冲突时，以语料 + 薄规格为准并视为工具 bug。语义争议走 `proposals/` 提案通道，不通过改工具解决。

## 消费方式（三实现仓）

以 git submodule pin 消费本仓，指针移动即语义升级动作（可审可回滚的 PR）：

1. clone / submodule 本仓 → 对着语料写薄 runner 对红变绿；
2. 在实现仓根签署 `support-surface.json`（格式与模块注册表见 `SUPPORT-SURFACE.md`；缺席 = 不支持，声明支持 ⇒ 该模块 runner 套件 CI 必须全绿）；
3. CI 断言所跑语料的 `meta.contract` 与声明钉住的版本一致。

## 语义演进

改语义 = 先提 `proposals/` 提案（意图文档 + 提案期用例出红）→ 转正三步一个 PR（语料新主版本目录 + 薄规格合入 + 注册表 / 声明登记）。直接改已发布语料的 PR 会被拒绝；`canvasctl` 拦得住形状与治理错误，拦不住语义裁决——那归提案通道。
