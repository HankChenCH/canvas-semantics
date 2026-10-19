# 002: 日期 token 收编季度 Q（expression-eval v3.0）

- 状态：integrated（2026-10-19 转正，同一 PR 完成转正三步，记录见文末）
- 提案期用例：已迁入 `corpus/expression-eval/v3.0/fn-date.json`（提案期目录随之删除）
- 关联：工票 08 验收演示（工作区 `.scratch/canvas-semantics/issues/08-ci-gates-exporter-retirement.md`）；薄规格 `spec/expression.md` §3.1 token 转译（「本轮明确不收」清单显式列名「季度」）
- 摘要：dateTokens 收编季度 token `Q`（输出 1–4，ceil(月/3)），expression-eval v2.0 → **v3.0**——token 子集扩充走提案 + 主版本（`GOVERNANCE.md` §3.3），新增行为 = 语义变更 = 主版本（§4）。

## 动机

证书批量场景需要按季度归档命名（如「2026年第4季度证书」），现役 14 token 无季度表达，只能由宿主预处理数据集再拼模板——把语义推给模板作者、绕开语料权威面。`spec/expression.md` §3.1「本轮明确不收」清单已显式列名「季度」且声明「将来纯新增不破坏」：本提案按预告通道收编，是语料先行流程（工票 08 翻转判据「任何语义变更的落点只有语料仓」）的验收演示载体。

## 语料影响

- 涉及契约与模块文件：`expression-eval` / `fn-date`；**主版本 v2.0 → v3.0**，v2.0 目录冻结不动；v3.0 = 四模块文件全量快照（shell / path / fn-number 的 cases 逐字节承 v2.0，fn-date 增量）；
- 用例变化：fn-date 新增 3 例（`date-quarter-token` 样例时点 Q=4、`date-quarter-january` Q=1、`date-quarter-june` Q=2），用例自含 context；既有用例期望值零变化；
- dateTokens 表：追加行 `{token: "Q", semantics: "季度 1-4（ceil(月/3)）", expect: "4"}`（sample `2026-10-11 14:30:05` → 10 月 → Q4）；小写 `q` 不收（白名单外 ASCII 字母 fail-fast，既有 `date-lowercase-token` / `date-unknown-token` 钉死的泛化形态覆盖，不另立用例）；
- 注册表：无新 code、无新保留名、无新模块文件（`SUPPORT-SURFACE.md` §2 expression-eval 版本规范形随升至 v3.0）。

## 薄规格影响

`spec/expression.md` §3.1 token 转译行：「14 token 权威表」→「15 token 权威表」；「本轮明确不收」清单移除「季度」（其余项保持不收）。

## 备选与取舍

- **备选一（被否）：v2.0 目录就地加 token 行与新用例**——违反冻结纪律（`GOVERNANCE.md` §2.1/§4：已发布目录不接收新用例、期望值禁改），否。
- **备选二（被否）：宿主以 `MM` 输出月份、模板侧自行换算季度**——季度映射（ceil(月/3)）落进每张模板 = 语义漂移面复活，恰是本仓要消灭的形态；且现役函数族无数值取月份面，组合不可行，否。

## 转正记录

- 转正 PR：工票 08 实现提交（canvas-semantics 仓，2026-10-19；验收演示载体——语料先行 → PHP/Go runner 红 → 双端实现 Q → 绿，记录见工票 Comments）。
- 语料落点：`corpus/expression-eval/v3.0/` 全量快照（fn-date 37 例含新增 3 例季度用例 + dateTokens 15 行；shell/path/fn-number 的 cases 逐字节承 v2.0）；提案期目录 `corpus/proposals/0002-fn-date-quarter-token/` 迁入后删除。
- 薄规格登记：`spec/expression.md` §3.1 token 转译行 14 → 15 token，「本轮明确不收」清单移除「季度」。
- 注册表登记：`SUPPORT-SURFACE.md` §2 expression-eval 版本规范形随升至 `v3.0`；无新 code / 保留名 / 模块文件。
- 实现仓重签状态：php-canvas-next 与 go-canvas 随 pin 移动重签 `support-surface.json`（expression-eval v2.0 → v3.0），canvas-web（编辑器画像，仅 scan）零改动。
