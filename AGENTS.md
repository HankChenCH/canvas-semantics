# AGENTS.md — canvas-semantics

## 包定位

中立规格仓 `canvas-semantics`：动态数据编译语义（表达式求值 / 扫描切分 / hydrate 填充 / flow 流装箱 / paginate 分页）的**唯一权威**——五份 JSON contract 语料升格为手维护的 executable spec，PHP/Go/TS 三端降为平等消费实现（git submodule pin 消费，数据驱动 runner）。背景裁决见根 ADR 0024（工作区 `docs/adr/0024-corpus-semantic-authority-flip.md`），一手调研见工作区 `docs/neutral-spec-governance-research.md`。本仓处于建仓期，工票见工作区 `.scratch/canvas-semantics/issues/`。

## 顶层分区（五区；治理与提案已就位，其余由后续工票落地）

- **治理**（已就位）：`GOVERNANCE.md`（MUST/MUST NOT/MAY、版本政策、提案生命周期、fail-fast 切分、自定义函数终态）、`SUPPORT-SURFACE.md`（支持面声明格式 + 模块注册表）。
- **提案**（已就位）：`proposals/`（README 生命周期 + `0000-template.md`；语义演进的唯一通道）。
- **薄规格正文** `spec/`、**语料** `corpus/`（按契约分目录、版本目录化）、**Go 工具** `canvasctl/`：工票 02 起落地，目录形态以届时实现为准。
- 消费端声明文件在各实现仓（`<repo>/support-surface.json`），本仓只定义格式与注册表。

## 红线（改动前先读 `GOVERNANCE.md` 全文）

- **语料 + 薄规格 = 语义权威**；已发布版本目录冻结（用例期望值禁改；勘误走次版本新目录 + errata 注记）；改语义 = 主版本新目录。
- **wire 字节面权威不动**：graph 键位与布局快照仍归 PHP 权威端；本仓不裁 layout、渲染原语、取消控制面、排版语义。
- **未知输入 fail-fast 是终态**：未知函数 / 未知流策略 / 未知主版本恒报错；宿主自定义函数 v1 不开放且不得私造注册口。静态声明字段（fit、画布背景类）另走「契约面宽容透传 + 渲染面显式降级」，两切面不得混（`GOVERNANCE.md` §6）。
- **工具非规范性**：canvasctl 只执行治理规则与形状校验（validate 未知字段即红），永不参与「什么是对的」裁决。
- **声明无行为开关**：`support-surface.json` 只答「支持哪些模块」；缺席 = 不支持；声明支持 ⇒ runner 套件必须绿。
- 扩展（函数族 / 流策略 / options / dateTokens / 诊断 code / 保留名 / 图层绑定面）只走 `proposals/` 提案通道，永不入语法外壳。

## 命令

（建仓期暂无；工票 02 落地 Go module 后此处补 `go build ./canvasctl` / `go test ./...` 等入口。）

## 惯例

- 注释与文档工作语言中文；commit 用 conventional-commit 前缀 + 中文 subject。
- 引用工作区文档（ADR、调研报告）时注明「工作区」路径；独立 clone 场景以仓内文档自含为先。
