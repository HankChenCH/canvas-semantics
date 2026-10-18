# spec/ — 薄规格正文

薄规格只写语料钉不住的东西：优先级裁决、诊断发射顺序保证、未定义区显式声明（`GOVERNANCE.md` §0）。语义的权威 = 语料（`corpus/`）+ 本目录正文，冲突时先修订语料与文本、再改实现。

本目录正文：

- `expression.md` — 表达式求值薄规格（已落笔，工票 03）：外壳文法 / 路径与求值上下文 / fn-date / fn-number 四模块的冻结面系统表述、优先级裁决与显式不保证清单；
- `pipeline.md` — 管线薄规格（填充/流装箱/分页三步的输入输出不变量）与诊断发射顺序保证（随工票 04 落笔）；
- `error-codes.json` — **错误 code 注册表**（canvasctl validate 的白名单数据源；active `codes` + 退役 `reserved` 区，退役 code 永不复用，`GOVERNANCE.md` §2.2）。当前登记表达式两契约的三个 code（`expression_syntax_error` / `expression_type_mismatch` / `expression_row_outside_loop`），五阶段全量注册表随工票 04 落笔。

编写纪律：语料用例钉得住的行为不进正文写第二遍；正文条目须能指认钉住它的语料用例（或显式声明未定义区）。
