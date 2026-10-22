# 003: 模板表逐行拆分分页模式（paginate options `split`，paginate-semantics v2.0）

- 状态：proposal
- 提案期用例：`corpus/proposals/003-row-split-pagination/`（paginate.json，8 用例 = 正例 6 + 负例 2，已建齐）
- 关联：工作区 `.scratch/row-split-pagination/`（map + 工票 01 可行性 / 02 断行信息双端现状 / 03 治理路径 / 04 拆分语义九项拍定 / 06 业界六系统对照）；薄规格 `spec/pipeline.md` §3/§4/§7；`GOVERNANCE.md` §3.2（分页选项扩充通道）
- 摘要：paginate options 开键面新增已知键 `split`（枚举 v1 仅 `"lines"`），开启模板表**按内容行逐行拆分**的第二种装箱模式——可拆文本行在页界按行切开、两片段独立边框盒续排，`content_overflow` 在该模式收窄为「不可拆残余」；未知枚举值 fail-fast 新 code `paginate_option_invalid`。现役「行是唯一装箱原子、永不切开」不变量在拆分模式下改写，原子模式（省略 `split`）语义零变化；语义变更走 paginate-semantics 主版本 v2.0（`GOVERNANCE.md` §3.2/§4）。

## 动机

证书批量等场景存在单格超长文本行（一份申请表的单格履历、一段长备注）。现役行原子装箱下只有两难：`truncate` 关 = 单行超页容量即 `content_overflow` fail-fast（paginate:single-row-oversize-throws），整批产出失败；`truncate` 开 = 该行独占一页、超出部分渲染期位图裁切（paginate:truncate-option-isolates-row），信息静默丢失且前一页大量留白。两种终态都无法表达「这一行太长，跨页续排」。

「行是唯一装箱原子、永不切开」现钉三处，本提案显式改写其**拆分模式分支**（原子模式分支原样保留）：

1. `php-canvas-next/src/Compiler/CanvasPaginator.php:16`（类注释语义块）；
2. `go-canvas/paginate/paginator.go:54`（Paginator 注释）;
3. `canvas-semantics/spec/pipeline.md` §3 装箱条目（paginate:row-packing-split 的「行保序装箱、不切开」表述）。

前提已具备（工票 02）：行高是编译期定稿测量值，断行信息核心自带（`Text::getLines()` = UAX14 断行器 + 启发式度量器，零渲染依赖），四个渲染后端只画单行——拆分落点在双端 paginate/Compiler 纯函数，渲染后端零改动。业界对照（工票 06）：Word/LibreOffice、css break、iText、Typst 等六系统的行拆分语义高度趋同（续行 = 空格独立边框盒、单体内容先移后切、页界即拆为默认触发），但拆分全部发生在排版/打印期，编译期拆分无先例——故适用域与反例域措辞（见薄规格影响）必须随本提案钉死，不承诺跨端拆分点逐字节一致。

## 语料影响

- 涉及契约与模块文件：`paginate-semantics`；**主版本 v1.0 → v2.0**（语义变更：新装箱模式 + `content_overflow` 语义收窄，`GOVERNANCE.md` §4）；v1.0 目录冻结不动，v2.0 = 12 例逐字节承入 + 提案期 8 例迁入的全量快照，既有用例期望值零变化；
- 用例变化：提案期用例 8 例（`corpus/proposals/003-row-split-pagination/paginate.json`），逐条钉点——
  | 用例 | 钉点 |
  |---|---|
  | split-lines-row-at-page-boundary | 页界即拆默认触发；K = floor(页剩余/lineHeightPx) 向下取整；片段行高与内容值按 `\n` 切片 |
  | split-lines-mixed-cells-empty-continuation | 混合格：短格全内容进前段、续段 = 空内容格且格盒画满续行高；格独立分片（各格装入 min(自身行数, K)）；两片段独立边框盒 |
  | split-lines-monolithic-row-atomic-move | 单体格（图片）行不拆、整行移下页（与原子模式同果的对照钉） |
  | split-lines-row-across-three-pages | 超整页可拆行多次拆分：片段重入装箱（[200][200][80] 三页），过程收敛 |
  | split-lines-truncate-narrows-to-unsplittable | truncate 收窄：截断开时可拆行照拆不截断，不可拆单体行独占一页（现役 truncate 语义） |
  | split-lines-degenerate-line-overflow | 退化线：不可拆行超单页可用区 → `content_overflow`，行是最终原子（与原子模式同果） |
  | split-lines-key-ineffective-when-pagination-off | 分页关（缺 pageHeight）时 `split` 键无效：直通 + 溢出判定不变 |
  | split-unknown-value-paginate-option-invalid | `split` 键认识而值不认识 → 构造期 fail-fast `paginate_option_invalid` |
- wire 键面：paginate options 为**开键面**（未知键忽略纪律不变），新增已知键 `split`，枚举闭集 v1 仅 `"lines"`；「键认识而值不认识 → fail-fast」为本提案新裁决（开键面下的枚举闭面，工票 03 指认）；
- 注册表：**新增 code `paginate_option_invalid`**（分页期，载体 PHP `PaginateException` / Go sentinel error；追加永不复用，`GOVERNANCE.md` §2.2/§3.4；转正时登记 `spec/error-codes.json` active 区——提案期不占注册表）；无新保留名、无新模块文件；
- 不涉表头/页尾固定行（`spec/pipeline.md` §4.2.5 预留位维持「只立约束不实现」，工票 04 拍定 v1 不支持重复，本集无用例）；不涉 flow 组合（拆分选项只在 paginate 直调面生效，flow 链/帧不暴露，v1 非组合面）；
- 转正后登记：`SUPPORT-SURFACE.md` §2 paginate-semantics 版本规范形随升 v2.0；php-canvas-next 与 go-canvas 双仓 `support-surface.json` 重签；canvas-web 不声明 paginate 契约、零改动。

## 薄规格影响

`spec/pipeline.md` 需正文承接（转正时合入）：

- **§3 options 行**：`{pageHeight: int, truncate?: bool, split?: "lines"}`；`split` 在分页关时无效（无装箱、直通 + 溢出判定不变）；值 ∉ 闭集 = `paginate_option_invalid` 构造期 fail-fast。
- **§3 装箱条目改写**（不变量改写声明第三处落点）：原子模式（省略 `split`）「行是唯一装箱原子」原样保留；拆分模式（`split: "lines"`）下——可拆行（全文本格且内容行数 ≥ 2）在页界即拆（不整行下移，SplitLate 触发语义记演进位）；拆分界对全行统一：K = min(floor(页剩余空间 / lineHeightPx), 最大内容行数 − 1)，K < 1 整行下移；各文本格前段装入 min(自身行数, K) 行、续段承余下行（格独立分片，css-break「一格断行不强制他格断行」先例）；两片段各画独立边框盒，格盒画满所在片段行高，续段短格 = 空内容格，长格余段顶对齐续排；片段重入装箱（可跨多页）。行高仍以用例给定值为准，片段内容值切片以显式 `\n` 硬断行为钉面（用例行数不依赖断行度量）。
- **§3 溢出条目收窄**：`content_overflow` 在拆分模式 = 「不可拆残余」（单体格行、退化线），原子模式语义不变；残余不新增 code。truncate 在拆分模式收窄为仅对不可拆行生效。
- **§4 code 表**：分页期行追加 `paginate_option_invalid`。
- **§7 适用域与反例域**（措辞随本提案钉死）：拆分语义的**适用域** = 默认启发式度量栈（行数与片段内容由定稿图给定，用例行数经显式 `\n` 硬断行给定、不依赖断行度量）；**反例域** = 增强度量注入（Go `image-renderer/typography` OpenType、JS `measureText` 工厂）与字素切分缝（PHP ext-intl 图素簇 vs Go 码点）——编译期行数与渲染端实际行数漂移时**末行可被渲染面位图裁剪**，兜底 = 现役「页可视区外由渲染面位图裁剪自理」先例，不 fail-fast、不做软诊断。**未钉组合显式不保证**：格间 lineHeightPx 异质行的拆分界、非零 padding 的片段算术、浮点行高累加误差——不承诺跨端一致（§7 既有先例同构）。

## 备选与取舍

- **备选一（被否）：方向 1「死循环守卫」**——现役装箱是单次前向、每行恰消费一次，无循环可守；运行时计数器属死代码且无行为面（工票 01 评估，2026-10-21 否决）。
- **备选二（被否）：渲染后端内切页**——分页权在编译期、后端只消费页序列（pdf-renderer spec 明文）；后端重画目的地 = 装箱数学双份实现，违背「编译期纯结构管线」定向（工票 02 缝评估）。
- **备选三（被否）：测高回调契约 / 模板声明行数**——paginate 消费注入度量破坏「编译期纯函数、无 I/O」纪律，语料断言变环境相关；模板声明行数与「内容定稿行高」方向相反、数据变长即失效（工票 02 缝评估表）。落地取「拆分纯函数下沉核心 + 渲染端裁切兜底 + 测高注入仅留可选缝」组合。
- **备选四（不进 v1，记演进位）：iText `SplitLate` 触发语义**（先整行下移、新页仍装不下才拆）、行级 `allowRowSplit` 覆盖（Word/LO 两级开关粒度）、表头/页尾固定行重复（§4.2.5 预留位维持；工票 04 用户拍定不支持，Word 连续性校验形态记未来参照）。
- **备选五（被否）：页界裁切混合格**（Word 行内裁切行为）——整行统一 K 的拆分界语义更可钉可测；格独立分片由页界几何自然涌现（css-break 先例），无需 per-cell 裁切语义。

## 转正记录

（proposal 态——转正三步一个 PR 后回填：语料落点 `corpus/paginate-semantics/v2.0/` 全量快照与提案期目录删除、`spec/pipeline.md` §3/§4/§7 条目合入、`spec/error-codes.json` 登记 `paginate_option_invalid`、`SUPPORT-SURFACE.md` §2 升版与双仓重签状态。）
