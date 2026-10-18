# 001: 软诊断（warnings）语料面落地与 expression_fallback_empty 注册

- 状态：integrated（2026-10-18 随工票 05 转正，同一 PR 完成转正三步，记录见文末）
- 提案期用例：未单建——本提案钉的是 PHP 权威端已发布行为（php-canvas-next `CanvasHydrator` 发射面，spec/pipeline.md §4 已声明「现役发射面唯一」），正式用例直接随转正落地（钉既有行为非新语义，test-first 的红绿由 Go/TS 实现端的 runner 对红变绿承担）
- 关联：工作区工票 `.scratch/canvas-semantics/issues/05-go-warnings-seam.md`；薄规格 `spec/pipeline.md` §4/§5/§7
- 摘要：把软诊断（warnings）的可执行语料面补齐——`expect` 增可选 `warnings` 期望序列键，软诊断 code `expression_fallback_empty` 入注册表，expand/flow 次版本补用例钉 code、path 与发射序。

## 动机

`spec/pipeline.md` §4 已声明软诊断面（硬失败走异常、策略性软失败走 `Diagnostic{code, message, path?}` 收集通道，现役发射面唯一 = 文案类表达式兜底空串），但标注 **[未钉]**：三契约 `expect` 无诊断序列字段、软诊断 code 不在注册表、发射序无跨端保证。后果：Go 端（工票 05 前的 go-canvas）`CompileResult.warnings` 恒空而无人察觉，「全量支持」声明在诊断面不成立——诊断语义只有 PHP 单端行为、无三端可执行的合钉载体。

## 语料影响

- 涉及契约与模块文件：`expand-semantics` / `expand`、`flow-semantics` / `flow`（paginate 输入为 hydrate 产物、无发射面，不涉及）；**次版本**（纯增用例钉既有语义，GOVERNANCE §4）：`v1.0` → `v1.1`，v1.0 目录冻结不动；
- 用例变化：expand 增 3 例（`text-fallback-single-warning-path` / `text-fallback-template-row-sequence` / `text-fallback-v1-row-path`）、flow 增 1 例（`text-fallback-warnings-full-hydrate`）；既有用例期望值零变化（省略 `warnings` 键 = 空序列，反向钉「命中不发射」的负面）；
- expect 形状：`{graph|canvases, warnings?: [{code, path}]}`，warnings 省略 = 空序列；断言面只落 code + path + 序列序，消息人读可本地化不入断言面（`spec/pipeline.md` §5）；errorCode 分支不携带 warnings（硬失败即无软诊断汇入）；
- 注册表：`spec/error-codes.json` 追加 `expression_fallback_empty`（21st code，填充期软诊断；追加永不复用，GOVERNANCE §2.2/§3.4）。

## 薄规格影响

`spec/pipeline.md` 三处 [未钉] 升格：§4 软诊断面（语料可执行面落 v1.1、code 入注册表）；§5 诊断 `path` 字段进语料断言面；§7 诊断发射序钉定为「帧序 → 图层文档序 → 行/格/内容序」并由语料期望序列逐用例锁住（flow 路径 = 分配前全量 hydrate 的发射面、按帧序累积、不带帧标识——帧标识升格走 spec 版本）。

## 备选与取舍

- **备选一（被否）：v1.0 目录就地补 warnings 字段**——违反冻结纪律（GOVERNANCE §2.1/§4：已发布目录不接收新用例、期望值禁改），否。
- **备选二（被否）：只加注册表不落语料**——code 入表而无用例可执行 = 合钉载体缺位，三端一致性无从断言；注册表登记必须与钉例同 PR（GOVERNANCE §7 转正第 1/3 步），否。
- **备选三（被否）：warnings 断言面带 message**——消息人读可本地化（三端无需字节一致），入断言面会把实现文案钉死、阻断本地化；code 是唯一锚点（先例：错误断言只落 code，spec/pipeline.md §4）。

## 转正记录

- 转正 PR：工票 05 实现提交（canvas-semantics 仓，2026-10-18）。
- 语料落点：`corpus/expand-semantics/v1.1/expand.json`（29 例）、`corpus/flow-semantics/v1.1/flow.json`（23 例）；载体对账 = 与 go-canvas testdata 导出载体共有用例逐字段深比较一致。
- 注册表登记：`spec/error-codes.json` + `expression_fallback_empty`（active 区，填充期）。
- 模块注册表：`SUPPORT-SURFACE.md` §2 expand-semantics / flow-semantics 版本规范形随升至 `v1.1`（模块文件集合不变）。
- 实现仓重签状态：各实现仓尚未签署 `support-surface.json`（签署面随工票 07 submodule 接入落地），无存量签名需重签；go-canvas 本工票内完成 v1.1 runner 对齐（hydrate 发射缝 + warnings 比对全绿）。
