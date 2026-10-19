# AGENTS.md — canvas-semantics

## 包定位

中立规格仓 `canvas-semantics`：动态数据编译语义（表达式求值 / 扫描切分 / hydrate 填充 / flow 流装箱 / paginate 分页）的**唯一权威**——五份 JSON contract 语料升格为手维护的 executable spec，PHP/Go/TS 三端降为平等消费实现（git submodule pin 消费，数据驱动 runner）。背景裁决见根 ADR 0024（工作区 `docs/adr/0024-corpus-semantic-authority-flip.md`），一手调研见工作区 `docs/neutral-spec-governance-research.md`。本仓处于建仓期，工票见工作区 `.scratch/canvas-semantics/issues/`。

## 顶层分区（五区；治理与提案已就位，其余由后续工票落地）

- **治理**（已就位）：`GOVERNANCE.md`（MUST/MUST NOT/MAY、版本政策、提案生命周期、fail-fast 切分、自定义函数终态）、`SUPPORT-SURFACE.md`（支持面声明格式 + 模块注册表）。
- **提案**（已就位）：`proposals/`（README 生命周期 + `0000-template.md`；语义演进的唯一通道）。
- **薄规格正文** `spec/`：`expression.md` 已落笔（工票 03：外壳/路径/fn-date/fn-number 冻结面 + 优先级裁决 + 显式不保证）；`pipeline.md` 已落笔（工票 04：填充/流装箱/分页输入输出不变量 + 五阶段错误分类 + 诊断路径中立记法 + 保留名注册表；工票 05：软诊断 warnings 面语料可执行化 + 发射序钉定）；`spec/error-codes.json` 是错误 code 注册表（validate 白名单数据源，五阶段全量 20 code + 软诊断 expression_fallback_empty 共 21 code + reserved 区）。
- **语料** `corpus/<契约>/<v主.次>/<模块文件>.json`（编写规范见 `corpus/README.md`）：五契约已全量入仓——`expression-scan v1.0`、`expression-eval v3.0`（shell/path/fn-date/fn-number 四模块；v2.0 = 工票 02/03 迁入已冻结，v3.0 = 提案 0002 季度 token Q 收编，2026-10-19）、`expand-semantics v1.1`（软诊断面，工票 05）、`paginate-semantics v1.0`、`flow-semantics v1.1`（软诊断面，工票 05；v1.0 基线两目录均工票 04）。
- **Go 工具** `canvasctl/`：`validate` 子命令已就位（严格 typed 解码 + 按契约分派用例形状 + 双写互检 + code 白名单 + 用例名唯一 + dateTokens 严格解码；`--surface <file>` 追加支持面声明三互检——形状/注册表/版本，SUPPORT-SURFACE §5.2，实现仓 CI 在 submodule 内以 `--surface ../support-surface.json` 校验自身声明）；`lint` 子命令已就位（工票 08：提案期用例 ↔ 提案文档反链互检，GOVERNANCE §7）；`diff` 子命令已就位（工票 09：逐用例四类差异人审摘要——新增/删除/期望值变化/仅注记变化 + dateTokens 等文档级语义键单列；差异非错误、语料不可信才 exit 1）；`release` 子命令已就位（工票 09：corpus + spec 目录快照包 + MANIFEST checksum，validate 全绿是发版前置，产物落盘后自检，提案期用例不入产物）。命令清单权威 = `canvasctl/README.md`。
- 消费端声明文件在各实现仓（`<repo>/support-surface.json`），本仓只定义格式与注册表。

## 红线（改动前先读 `GOVERNANCE.md` 全文）

- **语料 + 薄规格 = 语义权威**；已发布版本目录冻结（用例期望值禁改；勘误走次版本新目录 + errata 注记）；改语义 = 主版本新目录。
- **wire 字节面权威不动**：graph 键位与布局快照仍归 PHP 权威端；本仓不裁 layout、渲染原语、取消控制面、排版语义。
- **未知输入 fail-fast 是终态**：未知函数 / 未知流策略 / 未知主版本恒报错；宿主自定义函数 v1 不开放且不得私造注册口。静态声明字段（fit、画布背景类）另走「契约面宽容透传 + 渲染面显式降级」，两切面不得混（`GOVERNANCE.md` §6）。
- **工具非规范性**：canvasctl 只执行治理规则与形状校验（validate 未知字段即红），永不参与「什么是对的」裁决。
- **声明无行为开关**：`support-surface.json` 只答「支持哪些模块」；缺席 = 不支持；声明支持 ⇒ runner 套件必须绿。
- 扩展（函数族 / 流策略 / options / dateTokens / 诊断 code / 保留名 / 图层绑定面）只走 `proposals/` 提案通道，永不入语法外壳。

## 命令

```sh
cd canvas-semantics
go test ./canvasctl/           # canvasctl 表驱动测试（校验规则钉在这里）
go vet ./... && gofmt -l .     # 与 CI 同款静态检查
go run ./canvasctl validate .  # 语料门禁（CI 同款）：语料形状 + 双写互检 + code 白名单
go run ./canvasctl lint .      # 治理 lint（CI 同款）：提案期用例 ↔ 提案文档反链
go run ./canvasctl diff . expression-eval v2.0 v3.0  # 语料 diff 人审摘要（评审输入，差异非错误）
go run ./canvasctl release . --out dist/rel-<stamp>  # 版本化语料产物（发版即门禁；--out 不可在 corpus/、spec/ 内）
go build -o bin/ ./canvasctl   # 构建单二进制（产物进 bin/；根级 canvasctl 与源码目录同名，勿裸 build）
```

Go module 是仓根 `go.mod`（纯 stdlib，零第三方依赖）；改语料或注册表后先跑 `validate` + `lint` 再提交；动版本目录（新版本 / 勘误）后先 `diff` 出人审摘要、发布动作走 `release`。

## 惯例

- 注释与文档工作语言中文；commit 用 conventional-commit 前缀 + 中文 subject。
- 引用工作区文档（ADR、调研报告）时注明「工作区」路径；独立 clone 场景以仓内文档自含为先。
