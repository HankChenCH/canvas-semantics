# corpus/ — 语料编写规范

约束语料作者的操作面规范；治理规则见 `../GOVERNANCE.md`（§9 语料编写纪律），工具执行面见 `../canvasctl`（validate 严格解码，未知字段即红）。语义权威 = 本目录语料 + `../spec/` 薄规格正文。

## 目录与双写

- 三级布局 `corpus/<契约id>/<v主.次>/<模块>.json`；目录名与各模块文件 `meta.contract` 字符串精确互为镜像（validate 双写互检，如 `expression-eval v2.0`）。
- 版本目录规范形 `v<主>.<次>`；已发布版本目录整体冻结（`GOVERNANCE.md` §4）：用例期望值禁改、不回移，勘误走新次版本目录 + `meta.errata`。

## 模块划分与命名惯例

- **单模块契约整文件**：模块文件名 = 支持面注册表的模块 id（`scan` / `expand` / `paginate` / `flow`，`../SUPPORT-SURFACE.md` §2）。
- **求值契约按语义域拆分**：`shell`（mustache 外壳与片段切分）/ `path`（路径求值与保留名）/ `fn-date`（日期函数与 dateTokens 转译）/ `fn-number`（数字函数族 number/fixed/percent——共享格式化底座故共文件）。
- **函数族永远单列文件**：内建/扩展函数族 = `fn-<族名>` 新模块文件 + 主版本，永不混入既有模块文件、永不触碰外壳文法（`GOVERNANCE.md` §3.1；CEL 扩展函数 `*_ext` 单独成文件先例的同构落例）。
- 新契约/新模块文件 = 提案转正（`GOVERNANCE.md` §7）+ 注册表增行 + canvasctl 形状接入，三者一个 PR。

## 用例形状

五契约两种用例形状族，由 validate 按契约分派严格钉定（未知字段即红）：

- **expression-scan**：`{name, description, template, expect:{parts[]}}`，part = `literal{text}` | `fragment{raw,expr,error}` | `open{text,expr}`；不含位置字段（各端索引空间自行派生），空字面一律省略。
- **expression-eval**：`{name, description, template, context, expect}`，`expect` = 输出字符串 | `{errorCode}`；`context` 为任意 JSON 对象。存量载体的 `group` 字段已随模块文件拆分退役（归属由文件承载，残留即红）。
- **expand-semantics**：`{name, description, graph, dataset, expect}`，`expect` = `{graph: 填充后 graph}` | `{errorCode}`；`dataset` 任意 JSON（null = 未绑数据集恒等直通用例形态）；`graph` 为不透明 wire 对象。
- **paginate-semantics**：`{name, description, graph, options, expect}`，`options` = 空数组 `[]`（= 无选项，PHP 空关联数组 wire 形态）| `{pageHeight: int, truncate?: bool}`（信封严格解码）；`expect` = `{pages: [页 graph…]}` | `{errorCode}`。
- **flow-semantics**：`{name, description, frames, dataset, flowChain, expect}`，`frames` = 模板态声明 graph 数组（非空）；`flowChain` = null | 节点数组（节点骨架 `frame: int` + `mode: string` 必出、`quota: int` / `omitIfEmpty: bool` 出则类型钉死；**未知键放行**——语料故意携带实现必须拒绝的负形态如 `flow:unknown-key-rejected`，枚举闭面/quota≥0/frame 越界等校验语义归实现）；`expect` = `{canvases: [canvas graph…]}` | `{errorCode}`。
- **dateTokens 转译表**：仅 `fn-date.json` 允许携带（`{sample, rows:[{token, semantics, expect}]}`，sample 钉死样例时刻）。
- **信封严 / 载荷宽的切分**（本轮成文的工具裁决，validate.go 注释同步）：严格解码（未知字段即红）作用于语料文档自身的结构面——用例顶层键集、`meta`、`expect` 信封；管线三契约的图载荷（`graph` / `frames` / `dataset`）是**被测实现的输入数据**，validate 只钉「是对象/是数组」的骨架——语料故意携带「实现必须拒绝/透传」的负形态（如 `flow:unknown-key-rejected`、`flow:unknown-mode-rejected`），预演语义裁决即越权（`GOVERNANCE.md` §8）。
- 管线三契约的 `expect` 载荷语义（graph/pages/canvases 产物面）不作结构约束——产物逐字段比对归各端 runner；validate 只钉信封与 errorCode 白名单。
- 用例名唯一的作用域 = 契约版本、跨模块文件；错误断言只落 code 不落消息文案。

## 语料设计纪律

- **数值十进制字符串表示**：数值语义锚定在值的最短往返十进制串上；数字族可解析输入只收严格十进制串 `-?\d+(\.\d+)?` 与常规区间数值，科学计数法等形态以「不可解析原文直通」用例钉定；超出十进制字符串语义的极端浮点用例**不得入集**（显式不保证清单见 `../spec/expression.md` §6）。
- **诊断路径中立记法**：诊断与消息文案中定位产物树节点，一律沿用既有**数组索引形态**（升格为中立记法，不另造新记法，`GOVERNANCE.md` §9；权威定义见 `../spec/pipeline.md` §5）：产物树根相对点路径 + 十进制数组下标方括号——`layers[0].rows[1].cells[0]`、`layers[0].content`、`layers[0].data.value`、`layers[0].children[1][0]`，帧面根 `frames[i]`。语料错误断言只落 code，路径只出现在消息文案与诊断 `path` 字段（其断言面随 warnings 语料落地，工票 05）。
- **用例自含**：同组用例共享 context 样本时逐用例全量复制、逐字节一致，不引入共享 fixture 引用——单条用例可独立摘取运行。
- **迁移对账**：外部契约迁入 = 机械搬运 + 程序对账（迁移前后用例集合按用例名逐字段深比较零差异；跨文件顺序按模块聚合、文件内保持原相对顺序），`meta` 收敛为 contract 标识 + 出处注记，语义注记一律升格 `../spec/` 正文。
