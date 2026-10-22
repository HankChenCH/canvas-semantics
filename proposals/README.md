# proposals — 语义演进提案

提案是本仓语义演进的**唯一通道**。提案是 documents of intent——**只有其内容合入正式规格后才成为 requirements**（OTEP 原话，第一原则）。

三车道边界（`GOVERNANCE.md` §0）：根 ADR 管系统架构裁决；工作区 scratch 特性目录管实现工票；本目录只管语义演进——一个改动若只是「实现某已钉语义」，不进这里。

## 生命周期（两态：proposal → integrated）

```
proposal ── 转正三步一个 PR ──> integrated（文档保留为历史，不删除）
```

### proposal（意图态）

- 按模板建 `NNN-<slug>.md`（编号自 001 顺延，模板见 `0000-template.md`），状态 `proposal`，并登记到下方索引表。
- 出提案期用例：语料侧 `corpus/proposals/<提案编号>-<slug>/`（语料区由工票 02 落地后生效），volatile、可随时增改——但**声称支持该提案的实现必须过其用例**（JSON Schema 套件 `proposals/` 先例）。提案文档头部必须反链其用例目录；canvasctl lint 校验该反链（提案期用例须有提案文档，反之亦然）。
- test-first 纪律：语义变更先在提案期用例出红，再改正文、再进实现；提不出用例的语义改动不成立（GCC 纪律：不带测试需书面说明为什么不能带）。
- 提案期不占 contract 版本、不进合规集。

### integrated（转正态）

转正三步一个 PR（`GOVERNANCE.md` §7）：

1. **语料**：提案期用例迁入对应契约正式模块文件 + 新主版本目录（语义变更）或次版本目录（纯增用例 / 勘误，见版本政策）；
2. **薄规格**：正文条目合入；
3. **登记**：contract 版本升级、模块注册表增行（若涉，`SUPPORT-SURFACE.md` §2）、支持面声明重签通告、错误/保留名注册表登记（若涉）。

转正后在本文档头部回填状态与转正记录（PR、落点、版本），文档保留为历史。

## 索引

| 编号 | 标题 | 状态 | 提案期用例 |
|---|---|---|---|
| [001](0001-expression-fallback-warning.md) | 软诊断（warnings）语料面落地与 expression_fallback_empty 注册 | integrated（2026-10-18，工票 05） | 未单建（钉 PHP 权威端既有行为，正式用例随转正落 expand/flow v1.1） |
| [002](0002-fn-date-quarter-token.md) | 日期 token 收编季度 Q（expression-eval v3.0） | integrated（2026-10-19，工票 08 验收演示） | 已迁入 `corpus/expression-eval/v3.0/fn-date.json`（提案期目录随之删除） |
| [003](003-row-split-pagination.md) | 模板表逐行拆分分页模式（paginate options `split`，paginate-semantics v2.0） | proposal（2026-10-22，工作区 row-split-pagination 工票 05） | `corpus/proposals/003-row-split-pagination/paginate.json`（8 用例） |
