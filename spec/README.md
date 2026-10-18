# spec/ — 薄规格正文

薄规格只写语料钉不住的东西：优先级裁决、诊断发射顺序保证、未定义区显式声明（`GOVERNANCE.md` §0）。语义的权威 = 语料（`corpus/`）+ 本目录正文，冲突时先修订语料与文本、再改实现。

本目录随工票 03/04 起落正文：

- `expression.md` — 表达式求值薄规格（外壳/路径/fn-date/fn-number 四模块的优先级与未定义区裁决）；
- `pipeline.md` — 管线薄规格（填充/流装箱/分页三步的输入输出不变量）与诊断发射顺序保证；
- `error-codes.json` — **错误 code 注册表**（canvasctl validate 的白名单数据源；active `codes` + 退役 `reserved` 区，退役 code 永不复用，`GOVERNANCE.md` §2.2）。当前仅登记扫描契约已用的 `expression_syntax_error`，五阶段全量注册表随工票 04 落笔。

编写纪律：语料用例钉得住的行为不进正文写第二遍；正文条目须能指认钉住它的语料用例（或显式声明未定义区）。
