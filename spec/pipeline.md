# pipeline.md — 编译管线薄规格（expand / flow / paginate v1.0）

语义权威 = `corpus/expand-semantics/v1.0/` + `corpus/flow-semantics/v1.0/` + `corpus/paginate-semantics/v1.0/` 三契约语料 + 本正文（`GOVERNANCE.md` §0）。表达式求值（语料内部消费的求值器语义）归 `expression.md`，本正文只钉三步管线的输入输出不变量、组合时序与错误分类。条目以「契约缩写:用例名」指认钉住它的用例（expand: / flow: / paginate:）；标注 **[未钉]** 的条目是语料暂无用例的现状裁决，可随次版本补用例升格。

## 0. 总览：管线组合与阶段模型

- **编译管线** = 填充（hydrate）→ 文档编译（flow 流装箱）→ 分页（paginate）；表达式求值是填充内部消费的求值器语义（`expression.md`），不是独立管线步。
- **组合时序**：flow 自模板态声明 graph 起跑、hydrate 在管线内（flow:global-index-across-frames 钉「全量 hydrate 在前、分配在后」——跨帧 `$index` 全局连续是观测点）；paginate 吃 hydrate 后产物（无标记、无模板、高度定稿的具体树），**分页不重演填充**（paginate 段输入不变式，载体 meta 注记 + 全部用例形态）；expand 既可独立驱动（`render(hydrate(canvas, dataset))`）也可被 flow 内嵌消费。
- **纯结构步骤**：三步均零 I/O、无渲染依赖；填充永不回写源 graph（产物与源分离，expand 全部直通用例的对称面）；流分配回写只落在分配产物上（flow:image-fit-fidelity-through-flow 钉非驱动图层逐页回写不触碰其声明面）。
- **五阶段错误模型**：解码期（wire 形状，`fromGraph`）→ 填充期（数据形状）→ 文档期（链与分配）→ 分页期（页几何与装箱）→ 绘制期（资源物化，渲染面）。每错误一个稳定 code，注册表见 `spec/error-codes.json`（§4）。

## 1. 填充（expand-semantics v1.0）

- **签名形态** `hydrate(Canvas, dataset): Canvas`，纯函数；`dataset = null` = 未绑数据集恒等直通（模板表不展开，渲染按零行空壳呈现，expand:null-dataset-identity）；无标记无模板画布恒等直通（expand:no-bindings-identity，字面 `{{` 不误伤）。
- **hydrate 不触碰 spec（管线级不变量）**：静态声明字段不进填充改写面——图片层 `spec.fit` 逐字段透传（expand:image-fit-spec-passthrough，ADR 0023）、画布 `canvas.background` 双分量逐字段透传（expand:canvas-background-passthrough，ADR 0022）。graph 解析—重建的规范化往返不构成触碰；实现不得在填充路径改写、补全或剔除任何 spec 声明键。
- **标记解除与条件写键**：求值成功即解除标记——`valueType → StaticValue`、`value` 落求值结果；TextLayer 恒写 `expression: ''`，Image/Qr 移除 `expression` 键（对齐未标记两键形态，expand:qr-marked-content、expand:resource-single-bare-path-legal）。禁倒退为恒落键（`GOVERNANCE.md` §2.4）。
- **行展开**：模板表按 rowsPath 定位行数组实例化——每行深拷贝模板子树、行上下文求值、V1 高度耦合重放（行高取最高格、格 auto 采纳/固定压平，expand:template-table-instantiation）；行命名空间 `row` + `$index` **自 1 起**（expand:template-table-instantiation、expand:resource-index-path-legal-row-context）；嵌套模板表内层 rowsPath 行相对取数、内层 `$index` 重新自 1 起（expand:nested-template-table-row-relative）；V1 表（rows only）标记内容在根作用域求值（expand:v1-table-marked-content-root-scope）。
- **空与缺**：rows 为空数组 = 合法零行、template/data 消失（空壳渲染，expand:empty-row-array-empty-shell）；rowsPath 指向缺键/null/非数组 = `rows_path_invalid` fail-fast（expand:rows-path-missing-key、expand:rows-path-non-array）。
- **保留名门**：数据集根级键名 `row` 或 `$` 前缀 = `reserved_root_key`（expand:reserved-root-key-row、expand:reserved-root-key-dollar）；全量保留名注册表见 §6。
- **资源单值门（ADR 0020）**：已标记 Image/Qr 的 expression 必须恰为一个 `{{...}}` 闭合片段占满整串、片段内容为裸点路径（`$index`/`$root.` 前缀合法，expand:resource-root-escape-path-legal）；多片段拼接 / 片段外字面 / 函数调用 / 空片段 = `expression_resource_single_ref`；**门先于求值**——违例即使求值必空也报 single_ref（expand:resource-violation-precedes-empty-check），资源求值空串 = `expression_empty_resource`（空 URL 无意义，expand:empty-resource-expression）。
- **值分轴**（宽松值面 / fail-fast 文法面）与全部求值语义归 `expression.md`；文案类求值空串 = 兜底空串 + 软诊断 `expression_fallback_empty`（§4 诊断面）。
- **解码期负形态同管线表达**：template ⊕ rows 双键同现 = `template_rows_conflict`、模板态缺 rowsPath = `rows_path_missing`（wire 非法，expand:decode-template-rows-conflict、expand:decode-rows-path-missing）。

## 2. 文档编译 / 流装箱（flow-semantics v1.0）

- **输入不变式**：frames 一律为模板态声明 graph 列表（驱动表含 `data.rowsPath` + `template`）；flowChain 节点骨架 = `{frame: int, mode: "fixed"|"paged", quota?: int, omitIfEmpty?: bool}`，**显式 `[]` 与 null 同义**；链驱动表 rowsPath 全链一致，链外同 rowsPath 表 = 分配权威被架空（flow:rows-path-collides-off-chain）。
- **链校验（全部 `flow_chain_invalid`，校验先于分配）**：frame 越界（flow:chain-frame-out-of-range）、frame 重复（flow:chain-frame-duplicated）、paged 非末位（flow:paged-not-last——paged 是链尾终止符，至多一个）、paged 带 quota（flow:paged-with-quota——paged 吃尽剩余，无配额位）、链上帧无模板表（flow:chain-frame-without-template-table）、链序乱序回跳（flow:chain-order-violated）、未知键（封闭键面，flow:unknown-key-rejected）、**未知 mode 枚举值**（闭集 `fixed|paged` 外即报，flow:unknown-mode-rejected——未知输入 fail-fast，`GOVERNANCE.md` §1.5）。
- **rowsPath 一致性（`flow_rows_path_inconsistent`）**：链内不一致（flow:rows-path-mismatch-in-chain）、链外碰撞（flow:rows-path-collides-off-chain）。
- **分配语义**：容量切分——行高定稿后按帧可用区（画布高 − 表顶 y）装箱，切分点是装箱产物（flow:capacity-split-by-row-height：可用区 400、行高 30 → 首帧 13 行）；fixed+quota 显式切分与容量无关、退化为声明式切分（flow:quota-explicit-split）；`quota: 0` 显式不吃行（flow:quota-zero-declares-no-intake）；`omitIfEmpty` 分得零行的帧跳帧——不产页、不进产物序列（flow:omit-if-empty-skips-frame）；paged 链尾帧内按帧画布高多页出图（flow:paged-terminal-multi-page）；`$index` 全局连续不逐帧重计（flow:global-index-across-frames）。
- **聚合语义**：根键绑定（如「共 {{itemsTotal}} 条」）每帧都取全数据集聚合值，不因行被切走而失真（flow:cross-frame-aggregation）——聚合面吃 hydrate 全量产物，不吃分配切片。
- **链外共存**：无表帧、异 rowsPath 模板帧、V1 表帧与链帧同文档编译，产物按帧序扁平拼接；链外模板帧吃自己的 rowsPath（flow:non-chain-frames-coexist）。
- **错误 code 复用裁决**：分配期超容（容量型首行超可用区）= paginate 段 `content_overflow`（同一装箱数学同一失败语义，不换载体不换段，flow:first-row-oversize-throws）；任意帧 ≥2 顶层表 = paginate 段 `paginate_target_invalid`（flow:multi-table-invalid）。文档段自立 code 仅 `flow_chain_invalid` / `flow_rows_path_inconsistent` 两个。
- **空数据矩阵**：全链 0 行 → 每链帧各产 1 页零行空壳，omitIfEmpty 逐节点可关（flow:empty-dataset-matrix）。
- **`frames_empty` 显式不立 code**：frames 空数组是入口守卫（调用方编程错误），消息可报、不进稳定 code 集——语料不收此形态（validate 钉 frames 非空），实现不得将其升格为锚点 code。

## 3. 分页（paginate-semantics v1.0）

- **options** = `{pageHeight: int, truncate?: bool}`（wire 空[] = 无选项）；`pageHeight ≤ 0` = `page_geometry_invalid` 构造期 fail-fast（paginate:page-geometry-invalid）；缺 pageHeight = 分页关——单元素直通、页 graph 与输入字节等价，溢出判定仍执行（paginate:passthrough-no-page-height）。
- **页几何**：页宽恒 = 源画布宽、全页等高、末页留白不收缩（paginate:single-page-fit）；页高 = options.pageHeight（flow 链尾 paged 帧例外：页高 = 帧画布高，flow:paged-terminal-multi-page）。
- **装箱**：行序列按行高保序装箱，可用区 = 页高 − 表顶 y（paginate:row-packing-split：页高 250、表顶 50 → [150+50][100+100]）；行恰好填满不产尾随空页（paginate:exact-fill-no-trailing-page）；表壳每页按声明重画、行切片从壳顶起（paginate:row-packing-split）；非驱动图层（页眉/落款）逐页按源坐标原样复制，页可视区外位图裁剪自理（paginate:non-table-layers-replicated）。
- **溢出（`content_overflow`）**：分页关 + 行序列底边越画布底（paginate:overflow-off-throws）；分页开 + 单行超页容量且截断关（paginate:single-row-oversize-throws）。**截断选项**开启时不报错——超容行独占一页、超出部分位图裁剪自理（paginate:truncate-option-isolates-row）。
- **分页目标**：顶层 TableLayer 恰 0 或 1 张——0 张 + 分页开 = 无内容可分、单页直通（paginate:no-table-passthrough）；≥2 张 = `paginate_target_invalid`（paginate:multi-table-invalid）。零行驱动表 = 1 页空壳、壳按声明画、行区零高（paginate:zero-rows-empty-shell）。

## 4. 错误分类与 code 注册表

注册表 `spec/error-codes.json`（`canvas-semantics/error-codes v1`）：active `codes` + 退役 `reserved` 区；**退役 code 永不复用、任何 code 语义一经发布不可改**（`GOVERNANCE.md` §2.2）；扩充走提案通道（§3.4）。语料错误断言只落 code 不落消息文案；code 是三端一致性锚点。五阶段全量：

| 阶段 | 载体（PHP 异常类，三端各映射本地类型） | code |
|---|---|---|
| 解码期（wire） | `DecodeException` | `unknown_layer_type` / `template_rows_conflict` / `rows_path_missing` |
| 填充期（数据形状） | `HydrateException` | `rows_path_invalid` / `reserved_root_key` / `expression_empty_resource` / `expression_resource_single_ref` |
| ↳ 求值器（经填充管线表达） | `HydrateException` 承载 | `expression_syntax_error` / `expression_type_mismatch` / `expression_row_outside_loop`（语义钉于 `expression.md` §5） |
| 文档期（链与分配） | `FlowException` | `flow_chain_invalid` / `flow_rows_path_inconsistent` |
| 分页期（页几何与装箱） | `PaginateException` | `content_overflow` / `paginate_target_invalid` / `page_geometry_invalid` |
| 绘制期（资源物化，渲染面） | `MaterializeException` | `resource_download_failed` / `resource_save_failed` / `qr_generate_failed` / `render_cancelled` / `render_deadline_exceeded` |

- **reserved 区**：首轮为空（尚无退役 code）；入区即冻结、永不复用。
- **注册表无阶段维度（工具面现状）**：validate 的 code 白名单对全表一视同仁、不区分阶段——绘制期 code 在语料中不可达是 §7 的显式声明而非工具强制；若未来需要「阶段 × 契约可达面」的更严校验，走提案 + 工具演进（仍非规范性）。
- **绘制期 code 不进语料**（显式声明）：物化是渲染面 I/O（网络/磁盘/QR 生成/取消控制），不在编译语义产物内、语料不可执行钉定——其行为面由实现仓测试钉。`render_cancelled` / `render_deadline_exceeded` 与下载流超时（归 `resource_download_failed`）的边界裁决见工作区 `.scratch/canvas-compiler/spec.md` §5.2。
- **`frames_empty` 不是 code**：入口守卫（调用方编程错误），永不进注册表、实现不得私升为锚点（§2）。
- **软诊断（warnings）面**：硬失败走异常（上表），策略性软失败走 `Diagnostic{code, message, path?}` 收集通道（不中断管线）。现役发射面唯一：文案类表达式兜底空串 = `expression_fallback_empty`。其语料可执行面（期望诊断序列）随工票 05（Go warnings 缝）落笔——本版语料三契约 `expect` 无诊断序列字段，**[未钉]**：软诊断 code 暂不在注册表、发射序无跨端保证。

## 5. 诊断路径中立记法

诊断与消息文案中定位产物树节点的**中立记法**（沿用实现既有形态升格，不另造新记法，`GOVERNANCE.md` §9）：

- **形态**：产物树根相对点路径 + 数组下标方括号——`layers[0].rows[1].cells[0]`、`layers[0].content`、`layers[0].data.value`、`layers[0].children[1][0]`（children 为「子表键.实例序」两级下标）；帧面根 `frames[i]`。
- **字符集**：段为裸键名（不做转义、不引号）或十进制数组下标；分隔符 `.`；下标括号 `[n]`。不采用 JSON Pointer（`/` 分隔）等异形态。
- **落点**：诊断 `path` 字段与消息文案定位段；错误断言在语料中只落 code、消息与 path 不进断言面（用例形状钉定），path 字段的语料断言随 warnings 语料面（工票 05）落地。

## 6. 保留名注册表（首轮盘点）

保留名是求值上下文形状的封闭集合（`GOVERNANCE.md` §1.2）；**扩充走提案通道**（§3.4），实现不得私留第三命名空间。首轮盘点四个，每名至少一条语料用例钉死：

| 保留名 | 层面 | 语义 | 钉例 |
|---|---|---|---|
| `row` | 求值 + 填充 | 行命名空间：行上下文内 `row.` 前缀取当前行字段；数据集根级键名 `row` 禁用（`reserved_root_key`） | `path:row-field-and-literal-mix`、`path:row-prefix-outside-row-context`、`expand:reserved-root-key-row` |
| `$` 前缀 | 求值 + 填充 | 保留前缀机制：`$` 开头的名字是保留名空间，不是数据键；数据集根级 `$` 前缀键名禁用（`reserved_root_key`） | `expand:reserved-root-key-dollar` |
| `$index` | 填充供给 + 求值读取 | 行序号：填充面自 1 起、逐行供给（嵌套表内层重计、flow 跨帧全局连续）；求值面按裸键读上下文供给值 | `path:index-stringified`、`expand:template-table-instantiation`、`flow:global-index-across-frames` |
| `$root` | 求值 | 显式取根：与裸名等价的逃生门（行上下文内取根键不经过 `row`） | `path:root-escape-equals-bare-name`、`expand:resource-root-escape-path-legal` |

## 7. 显式不保证与范围外

- **绘制期（物化）行为不入语料**：资源下载/落盘/QR 生成/渲染取消与截止——渲染面 I/O，错误行为由实现仓测试钉（§4）；编译管线对其唯一承诺 = 求值产物为字面资源引用、填充 fail-fast 先于物化。
- **layout / 排版语义不入语料**：断行度量、行高定稿的度量来源、位图裁剪渲染——wire 字节面权威不动（`GOVERNANCE.md` §0）；语料只钉装箱与回写的结构行为面（行高以用例给定值为准）。
- **具体装箱边界数值**：切分点（每页行数）是装箱产物、逐用例钉定；未钉的边界组合（浮点行高累加误差等）不承诺跨端一致。
- **诊断发射序**：warnings 收集序在多节点场景无跨端保证 **[未钉]**（现役发射面单一，工票 05 引入序列断言时一并钉）。
