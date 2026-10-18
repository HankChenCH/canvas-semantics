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

两契约两种用例形状，由 validate 按契约分派严格钉定（未知字段即红）：

- **expression-scan**：`{name, description, template, expect:{parts[]}}`，part = `literal{text}` | `fragment{raw,expr,error}` | `open{text,expr}`；不含位置字段（各端索引空间自行派生），空字面一律省略。
- **expression-eval**：`{name, description, template, context, expect}`，`expect` = 输出字符串 | `{errorCode}`；`context` 为任意 JSON 对象。存量载体的 `group` 字段已随模块文件拆分退役（归属由文件承载，残留即红）。
- **dateTokens 转译表**：仅 `fn-date.json` 允许携带（`{sample, rows:[{token, semantics, expect}]}`，sample 钉死样例时刻）。
- 用例名唯一的作用域 = 契约版本、跨模块文件；错误断言只落 code 不落消息文案。

## 语料设计纪律

- **数值十进制字符串表示**：数值语义锚定在值的最短往返十进制串上；数字族可解析输入只收严格十进制串 `-?\d+(\.\d+)?` 与常规区间数值，科学计数法等形态以「不可解析原文直通」用例钉定；超出十进制字符串语义的极端浮点用例**不得入集**（显式不保证清单见 `../spec/expression.md` §6）。
- **用例自含**：同组用例共享 context 样本时逐用例全量复制、逐字节一致，不引入共享 fixture 引用——单条用例可独立摘取运行。
- **迁移对账**：外部契约迁入 = 机械搬运 + 程序对账（迁移前后用例集合按用例名逐字段深比较零差异；跨文件顺序按模块聚合、文件内保持原相对顺序），`meta` 收敛为 contract 标识 + 出处注记，语义注记一律升格 `../spec/` 正文。
