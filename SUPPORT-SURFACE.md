# SUPPORT-SURFACE — 支持面声明格式

每个消费实现仓签署一份**机器可校验的支持面声明**，钉死「支持到什么程度」——支持面是声明出来的事实，不是口头印象。本仓不设 conformance 等级（唯一等级 + 模块级支持面声明，见 `GOVERNANCE.md` §5）；渲染端全量、编辑器仅扫描等消费者画像体现在声明内容里。声明是 protobuf per-实现 failure list 的**反向**：failure list 记账「预期失败」，本声明记账「承诺支持」。

## 1. 格式裁决：机器可校验 JSON（不是人读 markdown 清单）

二选一钉死：**可校验 schema 形态**（单一 JSON 文件，`canvas-semantics/support-surface v1`）。理由：

1. **第一消费者是机器**：runner 的模块跳过与记账、CI 门禁「声明支持 ⇒ 该模块 runner 套件必须绿」、canvasctl 对模块注册表与语料版本的互检，全部需要可枚举数据；人读是第二需求。人读清单（otel spec-compliance-matrix.md 形态）靠手工维护、与事实漂移无机器可查，恰是要避免的形态。
2. **先例的本来面目**：protobuf failure list 虽是文本文件，但本质是 runner 数据——conformance runner 第一步即向被测实现拉取 `FailureSet`；JSON Schema 的 optional/ 与 CEL 的用例文件选集也都是结构化选集。成熟先例没有一家把支持面钉在纯散文里。
3. **零新增工具面**：canvasctl 已按「语料是法、法必须严」承担严格 typed struct 解码（未知字段即红），声明校验落同一工具、同一纪律、同一套表驱动测试。
4. **人读面不丢**：声明的每个模块条目带 `note` 人读字段；将来若需汇总视图，以 canvasctl 渲染子命令生成（非规范性，v1 不做）。

## 2. 模块注册表（v1）

声明与校验的对照基准。契约（contract）即 `GOVERNANCE.md` §0 的五域；求值契约内部按模块文件再分（CEL testdata 文件级 prescribed subset 先例），其余四契约整文件即单模块。扩展函数永远单列模块文件（`fn-<族名>`，`GOVERNANCE.md` §3.1）。

| 契约 id | 语义域 | 版本规范形 | 模块文件 | 模块文件说明 |
|---|---|---|---|---|
| `expression-eval` | 表达式求值 | `v3.0`（季度 `Q` 随提案 0002 收编，2026-10-19；`v2.0` 目录冻结） | `shell` | mustache 外壳与片段切分语义 |
| | | | `path` | 路径求值与保留名 |
| | | | `fn-date` | 日期函数与 dateTokens 转译 |
| | | | `fn-number` | 数字函数族 number/fixed/percent（共享格式化底座故共文件） |
| `expression-scan` | 表达式片段扫描（编辑器扫描面） | `v1.0` | `scan` | 整文件迁移，单模块 |
| `expand-semantics` | hydrate 数据填充 | `v1.1`（软诊断语料面，工票 05） | `expand` | 整文件迁移，单模块 |
| `paginate-semantics` | 分页 | `v1.0` | `paginate` | 整文件迁移，单模块 |
| `flow-semantics` | 内容流装箱 | `v1.1`（软诊断语料面，工票 05） | `flow` | 整文件迁移，单模块 |

注册表演进规则：提案转正新增契约或模块文件时，本表增行并随语料发布（`GOVERNANCE.md` §7 转正第 3 步）；每契约每版本的模块文件集合以该版本语料目录的实际文件为准，本表跟随。tooling/渲染等非语料能力（layout、渲染原语）不在注册表、不在声明面——本仓不裁 wire 字节面。

## 3. 声明 schema（v1）

实现仓在自身仓根提交 `support-surface.json`：

```json
{
  "declaration": "canvas-semantics/support-surface v1",
  "implementation": {
    "repo": "hankchen/php-canvas-next",
    "language": "php",
    "profile": "renderer",
    "note": "渲染端，全量支持（five contracts 全模块）"
  },
  "contracts": [
    {
      "id": "expression-eval",
      "version": "v3.0",
      "moduleFiles": [
        { "id": "shell",     "support": "supported", "note": "" },
        { "id": "path",      "support": "supported", "note": "" },
        { "id": "fn-date",   "support": "supported", "note": "" },
        { "id": "fn-number", "support": "supported", "note": "" }
      ]
    },
    {
      "id": "expression-scan",
      "version": "v1.0",
      "moduleFiles": [
        { "id": "scan", "support": "supported", "note": "" }
      ]
    },
    {
      "id": "expand-semantics",
      "version": "v1.1",
      "moduleFiles": [
        { "id": "expand", "support": "supported", "note": "" }
      ]
    },
    {
      "id": "paginate-semantics",
      "version": "v1.0",
      "moduleFiles": [
        { "id": "paginate", "support": "supported", "note": "" }
      ]
    },
    {
      "id": "flow-semantics",
      "version": "v1.1",
      "moduleFiles": [
        { "id": "flow", "support": "supported", "note": "" }
      ]
    }
  ]
}
```

字段表：

| 字段 | 类型 | 约束 |
|---|---|---|
| `declaration` | string | 恒为 `canvas-semantics/support-surface v1`；后续格式演进入 v2，不原地改义 |
| `implementation.repo` | string | 实现仓标识 |
| `implementation.language` | string | `php` / `go` / `ts` / …（第四语言自取名，与 runner 形态一致即可） |
| `implementation.profile` | string | 说明性画像：`renderer` / `editor` / `tool`；不参与校验语义，真实支持面以 `contracts` 为准 |
| `implementation.note` | string | 人读备注，可为空 |
| `contracts[].id` | string | 必须在 §2 模块注册表内 |
| `contracts[].version` | string | 声明钉住的语料版本（规范形 `v<主>.<次>`），必须存在于该契约的版本目录，且与实现仓 submodule pin 的语料一致（实现仓 CI 断言） |
| `contracts[].moduleFiles[].id` | string | 必须在 §2 中该契约该版本的模块文件集合内 |
| `contracts[].moduleFiles[].support` | string | 仅 `supported` / `unsupported` 两值 |
| `contracts[].moduleFiles[].note` | string | 人读备注（如 unsupported 原因），可为空 |

## 4. 校验与语义规则

1. **未知字段即红**：canvasctl validate 对声明做严格 typed struct 解码，与语料同纪律——声明是承诺，承诺必须严。
2. **缺席 = 不支持**：`contracts` 未列出的契约、`moduleFiles` 未列出的模块文件，一律视为 `unsupported`。新增模块文件后，存量声明**无需改动即自动视为未声明支持**——沉默不产生合标宣称，虚报才需要动作。
3. **声明即承诺**：凡 `supported` 条目，实现仓 CI 的对应 runner 套件 MUST 全绿；不绿时要么修复实现、要么显式把该条目改回 `unsupported`（一次可见可审的声明变更，MUST NOT 静默跳过记账）。
4. **声明无行为开关**：本 schema 刻意不含任何行为字段（未知输入处理、错误分类等）——行为一律由语料与 `GOVERNANCE.md` 钉死；声明只回答「支持哪些模块」。
5. **全量支持的定义**：五契约全部模块文件均 `supported` 且版本钉住最新发布版本。「全量」是清单内容达成的状态，不是规格等级（`GOVERNANCE.md` §5）。

## 5. 签署与消费流程

1. **签署时机**：实现仓接入语料仓时首次签署；提案转正新增模块文件或语料主版本升级后，实现仓跟进重签（`GOVERNANCE.md` §7 转正第 3 步通告），重签 = 声明文件变更的可审 PR。
2. **消费路径**：实现仓 CI 拉（submodule）语料仓后，以 canvasctl validate 校验自身 `support-surface.json`（注册表、版本、形状三互检），再按声明跑 runner：`supported` 模块套件必须绿，`unsupported` 模块用例跳过并计数记账。
3. **第四语言接入**：照抄本流程——clone 语料仓 → 写薄 runner 对红变绿 → 签署 `support-surface.json` → 接入 CI；接入成本与语言数量无关地保持最小。
