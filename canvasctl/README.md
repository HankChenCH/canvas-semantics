# canvasctl — 治理工具

Go 单二进制、纯 stdlib（零第三方依赖）。四命令覆盖 GOVERNANCE §8 的工具面全量：`validate`（语料形状 + 支持面互检）、`lint`（提案反链）、`diff`（语义变更人审摘要）、`release`（版本化语料产物）。命令清单与实现一致以本文件为准。

**非规范性声明**（GOVERNANCE §8）：canvasctl 只执行 `GOVERNANCE.md` 已钉死的形状与治理规则，**永不参与「什么是对的」裁决**。工具与语料结论冲突时，以语料 + 薄规格为准并视为工具 bug；语义争议走 `proposals/` 提案通道，不通过改工具解决。

## 命令清单

```sh
canvasctl validate [root] [--surface <file>]
canvasctl lint [root]
canvasctl diff [root] <契约> <版本A> <版本B>
canvasctl release [root] --out <dir>
```

`[root]` 为语料仓根，缺省当前目录（实现仓 CI 在 submodule 内以 `.` 或 `../` 消费）。

### validate — 语料形状 + 治理校验（CI 门禁）

严格 typed struct 解码（**缺字段、未知字段即红**，「语料是法，法必须严」）、contract↔目录双写互检（`meta.contract` 与目录名精确互为镜像，GOVERNANCE §4）、错误 code 白名单（active `codes` 之外一律红，`reserved` 区永不复用）、用例名唯一（契约版本作用域）、版本目录规范形（`v<主>.<次>`）与模块文件布局检查。发现即红、逐条带定位，exit 1；全绿输出计数摘要。

`--surface <file>` 追加支持面声明三互检（SUPPORT-SURFACE §5.2）：形状（声明同「未知字段即红」纪律）、注册表（契约 id / 模块文件 id 必须在声明钉住的语料版本目录内）、版本（声明版本目录必须存在）。实现仓 CI 以 `canvasctl validate --surface ../support-surface.json .` 校验自身声明。

### lint — 提案期用例 ↔ 提案文档反链互检（CI 门禁）

两态提案生命周期（GOVERNANCE §7）钉死的反链规则：提案期用例目录 `corpus/proposals/<编号>-<slug>/` 须有同名提案文档且文档正文反链该目录路径，反之状态 proposal 的文档须有用例目录（或「提案期用例」行显式写明待补）；用例目录不可为空；状态 integrated 的文档豁免；`README.md` 索引与 `0000-template.md` 模板不参与生命周期。exit 语义同 validate。

### diff — 语料 diff 的人审摘要（评审输入，非门禁）

`diff <契约> <版本A> <版本B>` 比对契约两版本目录，产出「语料 diff 即语义变更」的人审清单（GOVERNANCE §8）：

- **逐用例按名比对**，四类差异：`+` 新增（仅版本 B）/ `-` 删除（仅版本 A）/ `~` 期望值变化（语义面不同——模板、上下文、期望产物等，摘要列出变化字段名）/ `≈` 仅注记变化（`description` 人读描述不同、语义面逐字节相同）。
- **文档级语义附加键**（`meta`/`cases` 之外，如 fn-date 的 `dateTokens` 转译表）变化单列——转译表是语义面（GOVERNANCE §3.3），不得吞进「仅注记变化」。
- **meta 注记**（`notes`/`errata` 出处注记）变化单列，标记者非语义。
- 整个模块文件只在一侧（函数族新模块文件 / 模块退役，GOVERNANCE §3.1/§3.6）记文件级新增/删除，用例计入总数。
- 相等性基准 = 规范形比较：对象键序与排版不敏感、数值字面敏感（语料数值纪律为十进制字符串，`1` 与 `1.0` 是不同字面）。

**exit 语义与 validate/lint 相反**：找到差异不是错误（exit 0）——版本间有差异是常态，摘要只做呈现、不裁语义（版本政策裁决归语料与提案通道）；坏版本目录对（契约/版本不存在、版本对相同、非规范形）、语料不可严格解码（含双写不一致、用例名重复）才 exit 1——人审材料不可来自不可信语料。

### release — 版本化语料产物（发版即门禁）

`release --out <dir>` 产出供消费仓 pin 与审计存档的**目录快照包**：

```
<out>/
  MANIFEST.json        # 格式标识 canvas-semantics/release-manifest v1
                       # + 契约/版本/模块文件/用例数清点 + 逐文件 sha256 checksum
  corpus/<契约>/<版本>/<模块>.json   # 语料快照（提案期用例区不入产物：volatile、不占版本）
  spec/                # 薄规格正文 + 错误 code 注册表（语义权威的一部分）
```

三道门卫，任何失败路径不覆盖既有产物：**落点守卫**（`--out` 不可位于仓根、`corpus/`、`spec/` 内——产物写进语料 = 自污染）；**发版前置**（语料 `validate` 全绿，不合只报计数不出产物）；**产物自检**（落盘后复验：磁盘文件集与清单逐一相符、checksum 逐文件重验、快照内每个模块文件 `meta.contract` 与快照路径双写一致）。快照语料自含——对 `<out>` 再跑 `validate` 必须全绿；checksum 可用系统 `sha256sum -c` 独立复核（清单 `files[]` 即 `sha256sum` 格式的路径清单来源）。

## 开发

```sh
go test ./canvasctl/       # 表驱动测试（校验规则钉在这里，行为变化先写失败测试）
go vet ./... && gofmt -l . # 与 CI 同款静态检查
go build -o bin/ ./canvasctl   # 产物进 bin/；根级 canvasctl 与源码目录同名，勿裸 build
```

代码导引：`main.go`（CLI 接线 / 参数解析，空串与未知旗标一律退 usage）、`validate.go`（语料与声明的严格解码面 + 共享解码辅助）、`surface.go`（支持面三互检）、`lint.go`（提案反链）、`corpusdoc.go`（diff/release 共用的模块文档最小加载面 + 规范形 JSON 比较）、`diff.go`（逐用例比对 + 摘要渲染）、`release.go`（快照收集 / 清单 / 产物自检）。
