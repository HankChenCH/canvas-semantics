# expression.md — 表达式求值薄规格（expression-eval v2.0）

语义权威 = `corpus/expression-eval/v2.0/` 四模块语料（`shell` / `path` / `fn-date` / `fn-number`）+ 本正文（`GOVERNANCE.md` §0）。正文承担两件事：冻结面的系统表述（语法文法、路径语义、内建函数签名与舍入/格式化语义、求值上下文形状，`GOVERNANCE.md` §1）与语料钉不住的优先级裁决、未钉区显式声明。条目以「模块文件:用例名」指认钉住它的用例；标注 **[未钉]** 的条目是语料暂无用例的裁决/现状，可随次版本补用例升格为语料钉死。

## 0. 总览与分轴哲学

- 求值器 = mustache 外壳 + 受限表达式子集：`{{ }}` 划定表达式片段，其余字面直通；片段 = 点路径取值 ｜ 内建函数调用。v1 内建函数闭集 = `date` / `number` / `fixed` / `percent`；扩充只走提案 + 主版本（`GOVERNANCE.md` §3.1/§3.3），永不触碰外壳文法。
- **分轴哲学**（全契约通用）：
  - **值面宽松**：数据逐行变化，某行缺键/不可解析不炸整批——缺字段/null/空串 = 空串信号，不可解析 = 原文直通（打印物场景数据不丢、「没格式化」肉眼可见）。钉例：`path:missing-row-field-empty-signal`、`path:null-mid-path-empty-signal`、`date:date-unparseable-passthrough`、`fn-number:number-unparseable-passthrough`。
  - **文法/签名面 fail-fast**：模板内写死的、数据无关的确定性错误（未知函数、参数个数、结构错、n 超界、格式串白名单外字母）报错优于整批静默输出错误内容；全部并入 `expression_syntax_error`，零新 code（§5）。钉例：`shell:unknown-function-fails-fast`、`shell:call-structure-arg-garbage`、`fn-number:number-n-out-of-range`、`fn-date:date-unknown-token`。
- **求值原子性**：模板求值是原子操作——任一片段 fail-fast 即整次求值失败、无部分产物（错误用例 `expect = {errorCode}` 的编码即此语义）。
- **范围切分**：资源表达式单值门（ADR 0020）不在求值器——文本插值语义零变化，违例/合法形态的 hydrate 级判定锁在 expand-semantics 契约（`expression_resource_single_ref`，已随工票 04 迁入，钉例见 `pipeline.md` §1）。数据集根键的保留名禁用（`row`、`$` 前缀）属编译/填充面校验，求值面不重钉。行循环 `$index` 自 1 起的供给语义归填充契约，求值器只认上下文供给值。
- 编辑器红标只认结构面（扫描契约 `expression-scan` 的 `error` 字段只承载结构错，未知函数名不在编辑器标红）；函数名单校验是求值端义务。
- 一致性边界：语料锁**默认求值器**（受限插值）语义；实现可经求值器注入口换实现，但声明支持本契约的实现必须四模块全绿。

## 1. 外壳文法（模块 shell）

- **切分**：`{{` 开、**首个 `}}` 收口**、引号不感知、**片段内禁 `}}` 字面**（`shell:close-braces-in-string-truncates`——字符串实参内出现会在引号闭合前被截断）；未闭合 `{{` 剩余全字面（`shell:unclosed-brace-stays-literal`）；不含 `{{` 的标记串 = 整串字面（`shell:fragmentless-marked-string-literal`）。切分产物形状（literal/fragment/open）由 expression-scan 契约钉定，求值端将 open（未闭合）态按字面拼回。
- **trim**：片段内容 trim 六字符集——空格/制表/换行/回车/NUL/垂直制表（PHP `trim` 语义）；全角空白不在字符集。字符集边界用例在 scan 契约（`ascii-whitespace-trimmed` / `fullwidth-space-not-trimmed`），求值侧钉例 `shell:whitespace-inside-braces-tolerated`。
- **片段分类**（文法位消歧）：片段 trim 后，取 `(`、`)`、`,`、`'`、`.`、空白之前的首个 token，其后紧跟可选空白与 `(` 即分类为**调用**，否则为**路径**。名字合法性由白名单校验后置：`foo(bar)` / `日期(bar)` / `foo (bar)` 均按调用分类、白名单外即未知函数 fail-fast（消灭「含 `(` 的怪键名静默输出空串」陷阱，`shell:call-classified-with-space`、`shell:unknown-function-non-ascii`）；`{{date}}`（无 `(`）恒为路径，数据集根键叫 `date` 合法（`shell:bare-name-date-key-is-path`）；首 token 后无左括号仍按路径，路径段内空白属于键名（存量画布字节兼容，`shell:path-with-paren-stays-path`）。
- **函数名空间**：v1 白名单闭集四函数，大小写敏感（`DATE(...)` 是未知函数）、不保留字化（「函数名只在调用位生效，字段名只在路径位」）；未知函数 = `expression_syntax_error`。未知函数恒报错是**终态**（`GOVERNANCE.md` §6：宿主自定义函数 v1 不开放、不得私造注册口）。
- **调用文法**：`调用 := 名字 WS* '(' WS* [实参表] WS* ')'`；`实参 := 路径 | '字符串' | 数字`；实参间 `WS* ',' WS*`，尾逗号与空参数位 = 语法错。字符串字面量仅单引号一种、`''` 翻倍转义（ICU/SQL 风格），串内其余任意字符合法；数字字面量无符号无指数 `数字+ ['.' 数字+]`（负数 n 在文法层即错 `fn-number:number-n-negative`），布尔/null 字面量不进文法；零参调用文法合法、个数归签名层拒（为将来 `now()` 留门，`fn-date:date-zero-args-syntax-error`）。收口后多余内容、括号不平衡、字符串未闭合均 = `expression_syntax_error`（`shell:call-structure-trailing-content` 等）。
- **WS 字符集**：调用内空白 = 空格/制表/换行/回车四字符。**[未钉]** 与片段 trim 六字符集的差异（NUL/垂直制表不作调用空白）暂无用例，双端现行口径一致。
- **优先级裁决（结构错 vs 分类）**：片段按调用分类后，结构错（实参表文法）先于白名单校验之后的首参类型检查；首参文法层钉死为路径（`fn-date:date-first-arg-string-syntax-error`、`fn-number:number-first-arg-string-syntax-error`——「格式化常量」是伪场景，误用更早一层被抓住）。

## 2. 路径语义与求值上下文形状（模块 path）

- **上下文形状**：`row` 键 = 行命名空间（`row.` 前缀出现在非行上下文 = `expression_row_outside_loop`，`path:row-prefix-outside-row-context`）；裸名 = 数据集根键，两命名空间分离无遮蔽（`path:bare-name-reads-root-key`）；`$index` 保留名按裸键取上下文供给值（`path:index-stringified`）；`$root` 显式取根、与裸名等价（`path:root-escape-equals-bare-name`）；`$` 为保留前缀机制。
- **点路径**：空段（前导/中部/尾随点）= `expression_syntax_error`（`path:malformed-path-syntax-error`）；下钻缺键、中途 null、末端 null = 空串信号（`path:missing-row-field-empty-signal`、`path:null-mid-path-empty-signal`、`path:null-terminal-empty-signal`）；标量中途下钻、数组/对象落标量位 = `expression_type_mismatch`（`path:type-mismatch-scalar-drill`、`path:type-mismatch-array-at-scalar`）。
- **字符串化**：null → 空串；字符串原样；整数直书；浮点按最短往返十进制表示（常规区间，`path:numeric-float-stringified`）。
- **[未钉]** 路径末端为 bool 时的输出形态：语料无用例；现实现取 PHP `(string)` 语义（`true`→`"1"`、`false`→`""`），双端一致性不作保证（§6）。函数参数位的 bool 已钉为 `expression_type_mismatch`，无此暴露面。
- **多片段合成**：片段依序求值、结果直接拼接（`shell:multiple-fragments-concatenate`、`shell:calls-mix-with-literals-and-paths`）；调用片段与路径/字面片段可任意混排。

## 3. fn-date 日期函数

- **签名** `date(path [, '格式串'])`，1–2 参：首参文法层钉死为路径；第二参须为字符串字面量（`fn-date:date-second-arg-number-syntax-error`）；≥3 参 = 参数个数不符（`fn-date:date-three-args-syntax-error`）。
- **可解析输入** = 3 种无时区形态的**严格**解析：`YYYY-MM-DD` ｜ `YYYY-MM-DD HH:mm:ss` ｜ `YYYY-MM-DDTHH:mm:ss`（仅大写 T，`fn-date:date-input-iso-t-form`）。位数严格、越界即不可解析（`2026-02-30`、`24:00:00` 均不收，`fn-date:date-overflow-day-passthrough`、`fn-date:date-overflow-hour-passthrough`）；**时区不建模**（wall clock，不转时区；带时区后缀不收）；时间戳数字输入不收——数值 = 字符串化后原文直通（`fn-date:date-numeric-passthrough`）。
- **值分轴**：字段缺失/null/空串 = 空串（`fn-date:date-empty-value-empty-output`、`fn-date:date-missing-field-empty-output`）；非空不可解析 = 原文直通（`fn-date:date-unparseable-passthrough`）。
- **格式**：单参缺省 = `YYYY-MM-DD`（等价预设 `date`，`fn-date:date-default-format`、`fn-date:date-preset-date`）；预设 `datetime` = `YYYY-MM-DD HH:mm:ss`（`fn-date:date-preset-datetime`）；预设是纯糖——第二参**整体等值**命中才展开为 token 串，否则按 token 串解析（`fn-date:date-preset-whole-match`）；要输出字面 `date` 用转义段 `'[date]'`（`fn-date:date-escape-preset-name`）。
- **token 转译**：14 token 权威表随 `fn-date.json` 的 `dateTokens` 发布（sample 钉 `2026-10-11 14:30:05` = 星期日，`GOVERNANCE.md` §3.3）；匹配**最长优先**（`YYYY` 先于 `YY`）、大小写敏感（`MM` 月 ≠ `mm` 分，`fn-date:date-token-case-sensitive`）；非 ASCII 字母字面直通（中文写即所得，`fn-date:date-chinese-literal-mix`）；白名单外 ASCII 字母（含小写 `yyyy` 类无对应 token）= `expression_syntax_error`（`fn-date:date-unknown-token`、`fn-date:date-lowercase-token`）；空格式串 = 零输出（`fn-date:date-empty-format`）。token 子集扩充走提案 + 主版本（本轮明确不收：月名 `MMM`/`MMMM`、12 小时制 `h`/`hh`/`A`/`a`、毫秒、季度、ISO 周数/周几、序数、时间戳输出、时区 token——将来纯新增不破坏）。
- **转义两层正交**：方括号转义段至**首个** `]` 止、段内字面直通，无嵌套无再次转义（`fn-date:date-escape-literal-segments`、`fn-date:date-escape-nested-bracket`）；未配对 `[` 按普通字符字面输出（含 `[` 自身，单遍扫描无回溯），其**后**字符仍走 token 白名单——可 fail-fast（`fn-date:date-escape-unclosed-bracket`、`fn-date:date-escape-then-whitelist`）；孤立 `]` 字面输出（`fn-date:date-escape-orphan-bracket`）；单引号翻倍转义属字符串字面量层，与方括号层正交（`fn-date:date-escape-two-layer-orthogonal`）；格式串内反斜杠是普通字符。
- **优先级裁决 [未钉]**：不可解析输入的原文直通**先于**格式串处理——格式串白名单校验只发生在输入可解析的分支（不可解析 + 坏格式串 = 直通不报错）。语料暂无用例钉此组合，可次版本补用例。

## 4. fn-number 数字函数族

`number` / `fixed` / `percent` 三函数同签名 `fn(path [, n])`、1–2 参，共享十进制字符串格式化底座（SUPPORT-SURFACE 注册表「共享格式化底座故共文件」）。

- **签名与 n**：第二参须为无符号整数数字字面量、范围 0–8（字符串/路径第二参 = 语法错 `fn-number:number-second-arg-string`、`fn-number:number-second-arg-path`；非整数 n = 语法错 `fn-number:number-n-non-integer`；超界 = 语义层 fail-fast `fn-number:number-n-out-of-range`；负号在文法层即错 `fn-number:number-n-negative`）；零参/≥3 参 = 参数个数不符（`fn-number:number-zero-args-syntax-error`、`fn-number:number-three-args-syntax-error`）；首参恒为路径。
- **缺省**：`number` 缺省不定点——小数按最短往返表示原样直出、仅整数部分分组（`fn-number:number-thousands-default`）；`fixed` 缺省 n=2（`fn-number:fixed-default-n2`）；`percent` 缺省 n=0（`fn-number:percent-times-100-default`）。
- **三函数语义**：`number` 只在整数部分插 `,` 分组、小数点固定 `.` 不参数化、负号在分组符之外（`fn-number:number-negative-outside-group`）、零 → `0`；`fixed` 定点补零不分组、n=0 无小数点（`fn-number:fixed-n0-no-decimal-point`）；`percent` 值 ×100 后定点 n 位再追加 `%`，×100 = 十进制字符串小数点右移两位、不引入二进制误差（`fn-number:percent-shift-two-decimals`）。**族间正交**：`fixed` 只补零不加千分位；千分位+定点兼得的唯一可达路径 = `number(x, n)`（`fn-number:number-money-form`）。
- **舍入**（契约级，全族共用）：**half-up 朝 +∞**——丢弃首位 >5 或（=5 且尾部有非零）进位；精确 tie 正数进位、负数朝零（`fn-number:fixed-tie-up`：0.125 n2 → 0.13；`fn-number:fixed-negative-tie-toward-zero`：-2.5 n0 → -2；`fn-number:fixed-negative-tie-fraction`）；舍入定义在值的最短往返**十进制字符串表示**上，不得把二进制浮点直接喂给原生 round（`fn-number:fixed-decimal-rounding`：2.675 → 2.68 而非 2.67；近界 `fn-number:fixed-near-boundary`）；整数进位可跨位（`fn-number:fixed-integer-carry`：9.9999 n3 → 10.000）；定点结果为 0 输出 `0` 不输出 `-0`（`fn-number:fixed-no-negative-zero`）；补零到恰好 n 位（`fn-number:fixed-n3-pads`）；n 上界 8（`fn-number:fixed-n8-upper-bound`）。
- **格式化序**（裁决，钉例 `fn-number:number-with-n-rounds-then-groups`）：`number` 带 n = 先定点 n 位舍入补零、后千分位分组。
- **输入口径**：常规区间 JSON 数值与严格十进制串 `-?\d+(\.\d+)?` 同管线进格式化（`fn-number:number-strict-decimal-string`、`fn-number:fixed-strict-string`），前导零按值归一（`fn-number:number-leading-zeros-normalized`：`"007"` → `7`）；缺失/null/空串 = 空串（`fn-number:number-empty-value-empty-output`、`fn-number:number-missing-field-empty-output`）；非空不可解析（科学计数法 `"1e3"`、千分位、含空白等形态）= 原文直通（`fn-number:number-unparseable-passthrough`）；bool/数组/对象落参数位 = `expression_type_mismatch`（`fn-number:number-bool-type-mismatch`、`fn-number:number-container-type-mismatch`——bool 字符串化不入契约，见 §6）。
- **优先级裁决 [未钉]**：n 的签名校验**先于**首参取值——数据无关错误优先（字段缺失 + 超界 n = `expression_syntax_error` 而非空串）。语料暂无用例钉此组合。

## 5. 错误分类汇总

- 本契约错误 code 全集 = 3 个（注册表 `spec/error-codes.json`）：`expression_syntax_error`（文法结构错、未知函数、参数个数不符、n 超界、格式串白名单外字母）、`expression_type_mismatch`（结构错：标量中途下钻/容器落标量位；参数位类型错：bool/数组/对象）、`expression_row_outside_loop`（`row.` 命名空间门）。零新 code 纪律：同一失败语义不立新 code，扩充走提案（`GOVERNANCE.md` §3.4）。
- 诊断只落 code，消息文案不进契约；code 是三端一致性锚点。

## 6. 显式不保证清单

以下各项语料**不入集、不承诺双端一致**；用例入集须先走提案（含新用例的次版本）钉死：

- **极端浮点**：常规区间（整数/常规小数）之外的浮点——科学计数法阈值两侧的字符串化、二进制表示误差敏感值的舍入——行为由实现自由；证书场景数据源应自备规范字段。
- **bool 字符串化**：函数参数位已钉为 `expression_type_mismatch`（形态不进契约）；路径末端 bool 输出形态未钉（现实现取 PHP `(string)` 语义）。
- **时区与时间戳**：时区整体不建模（wall clock）；时间戳输入、带时区后缀输入不收。
- **`number` 缺省档的直出**依赖输入的最短往返表示——极端精度输入落在极端浮点不保证区。
- **编辑器预览**：预览呈表达式原文是既定语义，浏览器预览允许与终图差异。

## 7. 范围外（雾区观察，未定语义）

`map`（枚举→label，label 由数据源自备）、人民币大写、字符串函数族、`now()`、严格度档位、条件/计算扩展：均为提案通道雾区（`GOVERNANCE.md` §7），本版本文法与语料不为它们预留语义（零参调用的文法合法性除外——为 `now()` 留门，个数归签名层拒）。
