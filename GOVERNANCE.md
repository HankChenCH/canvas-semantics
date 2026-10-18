# GOVERNANCE — canvas-semantics 治理文件

本文件钉死 canvas-semantics 仓的治理规则：语义权威的切分与冻结面 / 禁改面 / 可扩展面三清单、contract 版本政策、两态提案生命周期、未知输入 fail-fast 与静态声明字段宽容透传的显式切分、宿主自定义函数的终态表态。治理规则不依赖任何人的记忆；改动本文件的规则本身走「提案 → 转正」通道（见 §7），背景裁决见根 ADR 0024。

本文 MUST / MUST NOT / MAY 按 RFC 2119 语义使用（MUST = 绝对要求、MUST NOT = 绝对禁止、MAY = 真正可选）。每条清单逐条标注调研依据：一手调研报告《中立规格仓库治理的一手经验调研》（`../docs/neutral-spec-governance-research.md`，下称「调研报告」，工作区根 docs 下；独立 clone 本仓时按报告末尾来源清单取先例 URL），引注形如「调研 §2.3」。

## 0. 权威切分（本仓裁什么、不裁什么）

- **编译语义权威 = 语料 + 薄规格正文**。表达式求值、扫描切分、hydrate 填充、flow 流装箱、paginate 分页五域的行为以语料用例为 executable spec；薄规格正文只写语料钉不住的：优先级裁决、诊断发射顺序保证、未定义区显式声明。
- **冲突消解顺序**：语料用例 > 薄规格正文 > 实现内文档 > 实现行为。冲突时先修订语料与文本、再改实现；实现永远不因「我就是这样写的」而赢。
- **wire 字节面权威不动**：graph 键位、序列化字节面与布局快照的权威仍是 PHP 权威端（php-canvas-next）与共享 fixture 再生成流程；本仓不裁 layout、渲染原语、取消控制面、断行度量等排版语义。语料只钉编译管线必须保持的透传/不回写不变量（如「hydrate 不触碰 spec」），不新增 wire 键位裁决。
- **提案三车道边界**：根 ADR 目录管系统架构裁决；工作区 scratch 特性目录管实现工票；本仓 `proposals/` 只管语义演进。三者不互相越界。

## 1. MUST — 冻结面（任何已发布 contract 版本内，以下为绝对要求）

1. **mustache 外壳文法冻结**：`{{…}}` 分隔符、片段切分、片段 trim 语义（含 trim 字符集边界）、片段外字面直通、语法错误分类，以上只能由语料钉死的行为说话。
   依据：调研 §1.1/§2.2——CEL grammar 与标准函数同文冻结，无「实现私加语法」的合法通道。
2. **路径求值语义、求值上下文形状与保留名注册表**：`row` 行命名空间、数据集根键裸名访问、`$index`/`$root` 保留名、`$` 保留前缀机制构成的求值上下文形状，属必选核，任何实现 MUST 全量支持、不可关闭。
   依据：调研 §2.4——JSON Schema core vocabulary「MUST be supported by any implementation, and cannot be disabled」的必选核做法。
3. **内建函数签名与语义**：v1 四函数（date/number/fixed/percent）的调用形态、参数约束、舍入（朝 +∞ 的 half-up 十进制）、格式化与 moment 风格 dateTokens 转译表语义冻结；函数是闭集，闭集扩充只走 §3 的 MAY 通道。
   依据：调研 §1.1/§2.2——CEL 标准函数闭集同文冻结、宏闭集 opt-in 先例。
4. **管线不变量与错误分类**：hydrate/flow/paginate 各自的输入不变式（如分页吃 hydrate 产物、流装箱吃模板态声明 graph）、输出产物形状、五阶段错误分类与错误 code 全集，全部冻结；所有合规实现 MUST 对同一输入给出同一语义结果（求值结果、错误 code、诊断序列逐字段一致）。
   依据：调研 §1.1——CEL conformance「All CEL implementations which pass the conformance tests should give valid CEL expressions the same meaning」。
5. **未知输入 fail-fast**：未知函数、未知流策略枚举值、未知 contract 主版本，实现 MUST 报错（错误行为本身由语料钉死），MUST NOT 私自宽容或私造错误（详见 §6 切分）。
   依据：调研 §2.4——JSON Schema 对不识别的必选 vocabulary「MUST refuse to process」；行为必须被规格规定而不是被实现发明。
6. **语料钉住的透传不变量**：凡语料用例钉住的「编译管线不触碰的声明字段」（静态声明字段透传、填充不回写源 graph 等），实现 MUST 保持逐字节透传；这些不变量虽触及 wire 邻接面，但其裁决出处仍是语料，不是本仓新立 wire 规则。
   依据：调研 §2.3——protobuf unknown fields preservation 的「保留并随序列化带出」纪律。

## 2. MUST NOT — 禁改面（绝对禁止，无提案豁免）

1. **已发布版本目录内的用例期望值不可修改**：期望产物、期望 code、期望诊断序列一经发布版本目录冻结即不可改；改语义 MUST 升主版本建新目录（唯一例外见 §4 勘误条款）。
   依据：调研 §1.2——JSON Schema 套件旧方言目录冻结（draft-04/03 frozen）；目的同 user story：存量实现的合标状态不被一次合并悄悄改写。
2. **错误 code 语义复用**：退役 code 进入 reserved 区后永不再用；任何 code 的语义一经发布不可改作他用。
   依据：调研 §2.3——protobuf「The same field number must not [be] used again」，后果原文「data corruption, privacy bugs」；配套 reserved 机制使违规在工具期即红。
3. **wire/graph 键名语义挪用**：已被 graph 契约钉死的键名不得在本仓语料中赋予新语义；已删除键名进保留清单（随薄规格错误注册表维护）。
   依据：调研 §2.3——同 protobuf 字段号/enum 值 reserved 纪律。
4. **条件写键纪律倒退**：缺省不落键（如 visible 仅 false 落键、fit 仅非 cover 落键、背景两分量皆空整键不落）的既有纪律不得在语料用例或薄规格中被倒退为恒落键。
   依据：根 CONTEXT.md「graph」词条往返恒等契约 + ADR 0022/0023 条件写键先例；语料透传不变量（§1.6）的对称面。
5. **实现私自宽容或私抛**：实现不得对语料钉死为错误的输入静默宽容，也不得对语料钉死为合法的输入私抛错误；未知输入的处理行为一律以语料为准（§1.5、§6）。
   依据：调研 §2.4/§2.5——JSON Schema「unrecognized keywords → annotations」（行为被规格收编）原则与 JMESPath `unknown-function` 反面教材。
6. **语法外壳扩张**：新函数、新语法构造 MUST NOT 进入 mustache 外壳文法或路径文法；函数族扩张只以新语料模块文件 + 主版本发生（§3.1）。
   依据：调研 §2.1/§2.2——GFM「strict superset… highlighted as such」扩展只增不改核心、CEL 扩展只走函数/库层（doc/extensions/ + cel-go/ext）。

## 3. MAY — 可扩展面（允许演进，一律走提案通道，见 §7）

1. **函数族**：新增内建函数族 = expression-eval 新增 `fn-<族名>` 模块文件 + 主版本升级；永不混入既有模块文件、永不触碰外壳文法。
   依据：调研 §1.1——CEL 扩展函数 `*_ext` 单独成文件、README 核心清单不含扩展文件。
2. **流策略与分页选项**：流装箱策略 mode 枚举扩充、paginate options 扩充，走提案 + 各自契约主版本。
   依据：调研 §2.4——JSON Schema vocabulary 声明层扩展（`$vocabulary`）先例。
3. **dateTokens**：日期格式 token 子集扩充，走提案 + expression-eval 主版本（转译表随语料发布）。
   依据：ADR 0018 自含语义表逐 token 钉死纪律，同 §1.3 闭集扩充通道。
4. **诊断 code 与保留名**：注册表追加新 code、新增保留名，走提案；追加永不复用（§2.2/§2.3 的对称面）。
5. **图层绑定面**：新增 `(layerType, field, 约束)` 绑定行（约束字典 v1 = 插值 / 资源单值门，ADR 0020/0022/0023 先例），走提案 + expand-semantics 主版本；新图层类型进语料用例同通道（用例只可用支持面交集内的图层类型）。
6. **新语料模块**：新增契约或既有契约新增模块文件，走提案 + 注册表登记（`SUPPORT-SURFACE.md` 模块注册表增行）+ 已签署实现重签声明。

## 4. contract 版本政策

- **版本字符串规范形 `v<主>.<次>`**（如 `expression-eval v2.0`）；目录名与 `meta.contract` 字符串精确双写，由 canvasctl validate 互检。存量 `v1`/`v2` 形态在语料迁入本仓时归一为 `v1.0`/`v2.0`（归迁移工票的一次性动作）。
- **主版本 = 语义变更**：任何用例期望值改变、行为改变、code 语义改变 → 新主版本目录。旧目录整体冻结、永不回移、永不删除。
- **次版本 = 非语义修订**，仅含两种形态：纯增用例（钉既有语义的新用例）；**勘误（erratum）**——限转写笔误，即三端 runner 对实际行为一致、且期望值与薄规格正文矛盾可证的错误。勘误 MUST 落在新次版本目录并带 `meta.errata` 注记（指认被勘误用例与依据），MUST NOT 就地修改旧目录；若「笔误」实际构成语义变更，按主版本处理，无例外。
- **已发布版本冻结的含义**：旧版本目录不接收新用例、不接收修复、不因新主版本发布而回移或注释化（对齐 JSON Schema「frozen」语义；bug 修复是否在新版本中重表达由提案决定）。
- **消费端 pin**：实现仓以 submodule pin 消费，runner MUST 断言所跑语料的 `meta.contract` 与自身声明（`SUPPORT-SURFACE.md`）钉住的版本一致；版本升级 = 指针移动的可审 PR，而非目录内容的原地改写。

依据：调研 §1.2/§3.4——JSON Schema 方言↔目录一一对应、`main` 恒稳定、旧方言冻结；目录制优于 branch 制（submodule pin 即路径）。

## 5. 支持面声明（不设 conformance 等级）

本仓不设 conformance 等级：唯一等级 + 模块级支持面声明。消费者画像（渲染端全量 / 编辑器仅扫描）体现在声明内容，不进规格分级。声明格式、模块注册表、校验规则与签署流程见 `SUPPORT-SURFACE.md`（机器可校验形态，protobuf failure list 的反向）；声明只回答「支持哪些模块」，不含任何行为开关——行为一律由语料与本文钉死。

依据：调研 §1.7——CEL 文件选集 / JSON Schema optional/ / protobuf failure list / ECMA-262 normative optional 四先例同构；SQL-92 三档制证伪。

## 6. 未知输入 fail-fast 与静态声明字段宽容透传的显式切分

- **编译语义面（fail-fast）**：求值、填充、流装箱、分页消费的动态语义输入中，未知函数名、未知流策略枚举值、未知 contract 主版本，实现 MUST 报错（code 由语料钉死），MUST NOT 私自宽容、MUST NOT 私改错误分类。判据：未知值使编译语义产物不可确定——静默宽容 = 产出内容错误的图 = data corruption。
- **静态声明字段渲染声明面（宽容）**：已注册的静态声明枚举字段（现例：图片层 `fit`、画布背景，ADR 0022/0023 先例）的未知值，wire 契约面 MUST 宽容透传（保往返恒等），渲染面 MUST 显式降级到缺省档（如 fit 未知值按 cover）。判据：未知值只影响渲染外观、不进入编译语义产物，结构面无损。
- **切分的边界**：后者只适用于「已声明的静态枚举字段的渲染声明面」，不得外溢为「对一切未知输入宽容」的通则；wire 解码面对未知**键**的忽略（往返恒等的既有纪律）与本条正交。
- **宿主自定义函数终态表态**：v1 不开放宿主自定义函数机制；「未知函数恒报错」即现行行为且为**终态**——任何实现 MUST NOT 私造函数注册口或私留注入缝。未来若开放，走提案 + 主版本，注册口形态届时钉进语料，三端一体跟进。
  依据：调研 §2.5——JMESPath 教训：spec 不表态 → 各实现私造注册口、语义互不兼容（issue 链 #100/#119/#202/#290）；v1 显式表态即关闭该漂移面。

## 7. 两态提案生命周期（proposal → integrated）

- **提案 = 语义演进的唯一通道**，形态为「意图文档 + 提案期用例」双轨：意图文档在 `proposals/`（编号文件，模板见 `proposals/0000-template.md`，索引与细则见 `proposals/README.md`）；提案期用例在语料侧 `corpus/proposals/<提案编号>-<slug>/`（工票 02 落地语料区后生效），volatile、可随时改，但**声称支持该提案的实现必须过其用例**。
- **两态**：
  - `proposal`：意图成立、用例已出（或声明待补）、未进正式契约。提案期用例不进合规集、不占 contract 版本。
  - `integrated`：转正三步一个 PR——(1) 语料：提案期用例迁入对应契约正式模块文件 + 新主版本目录；(2) 薄规格：正文条目合入；(3) 登记：contract 版本升级、模块注册表增行（若涉）、支持面声明重签通告、错误/保留名注册表登记（若涉）。转正后提案文档保留为历史，不删除。
- **test-first 纪律**：语义变更 MUST 先在提案期用例出红，再改正文、再进实现；纯实现工票（不涉语义）不进本通道。
- 第一原则（OTEP 原话）：提案是 documents of intent，**只有其内容合入正式规格后才成为 requirements**。

依据：调研 §3.2/§3.3——OTEP「documents of intent… become requirements only after integration」；JSON Schema 套件 `proposals/` 子目录（提案期用例 volatile、声称支持者必须过）；test262 staging 转正形态；tc39 多阶段在 1 人团队收敛为两态。

## 8. 工具非规范性（canvasctl）

canvasctl（Go 单二进制、纯 stdlib）承担语料形状校验（validate：严格 typed struct 解码、未知字段即红）、治理 lint（contract↔目录双写互检、code 白名单、用例名唯一、提案期用例反链提案文档）、语义化 diff 摘要与 release 产物。工具**非规范性**：只执行本文已钉死的形状与治理规则，MUST NOT 参与「什么是对的」裁决；工具与语料结论冲突时，以语料 + 薄规格为准并视为工具 bug。语料是法、法必须严（validate 未知字段即红）；渲染面的宽容读与语料面的严判方向相反且各自成立（§6）。

## 9. 语料编写纪律（约束语料作者，非合规面）

- 数值一律十进制字符串表示；极端浮点显式不保证（超出十进制字符串语义的用例不得入集）。
- 诊断路径沿用既有 `layers[0].rows[1]` 数组索引形态升格为中立记法，不另造新记法。
- 用例输入图只可使用支持面清单交集内的图层类型；新图层类型进语料走 §3.5 提案通道。
- `meta` 只保留 contract 标识与出处注记（含 errata）；语义注记正文一律迁薄规格，不在 `meta` 里长语义。
