# Requirements Document

## Introduction

This feature enhances flowspec-cli with exploration capabilities and optimizes its CI/CD experience. The core objective is to implement "dual-mode governance" by adding exploration mode that can automatically generate contracts from traffic logs, support standalone YAML contract files, and improve the tool's usability in automated pipelines.

## Requirements

### Requirement 1: Traffic Exploration and Contract Auto-Generation

**User Story:** As a developer, I want to explore existing traffic patterns and automatically generate service contracts, so that I can quickly establish baseline specifications without manual contract writing.

#### Acceptance Criteria

1. WHEN I run `flowspec-cli explore --traffic <log_dir> --out <yaml_path>` THEN the system SHALL parse traffic logs and generate a YAML contract file
2. WHEN the system processes Nginx access logs THEN it SHALL extract method, path, status code, and timestamp information
3. WHEN similar URL paths are detected (e.g., /api/users/123, /api/users/456) THEN the system SHALL cluster them into parameterized endpoints (e.g., /api/users/{var})
4. WHEN generating contracts THEN the system SHALL identify HTTP methods, status code ranges, and required fields for each endpoint
5. WHEN parsing fails for a log line THEN the system SHALL log the error and continue processing remaining lines

### Requirement 2: Standalone YAML Contract Support

**User Story:** As a DevOps engineer, I want to use YAML files as independent contract definitions, so that I can manage service specifications separately from source code.

#### Acceptance Criteria

1. WHEN I provide a .yaml or .yml file to the verify command THEN the system SHALL parse it as a ServiceSpec
2. WHEN I run `flowspec-cli verify --path <yaml_file> --trace <trace_file>` THEN the system SHALL validate the trace against the YAML contract
3. WHEN the YAML file contains invalid ServiceSpec format THEN the system SHALL provide clear error messages
4. WHEN I use the verify command alias THEN it SHALL behave identically to the align command
5. WHEN the --path parameter points to a directory THEN the system SHALL process source code files as before

### Requirement 3: CI/CD Experience Optimization

**User Story:** As a CI/CD pipeline maintainer, I want streamlined integration and clear success/failure indicators, so that I can easily incorporate flowspec validation into automated workflows.

#### Acceptance Criteria

1. WHEN I use the GitHub Action with path and trace inputs THEN it SHALL download, install, and execute flowspec-cli verification
2. WHEN verification runs in CI mode with no failures THEN the system SHALL output a concise success summary (e.g., "✅ All 12 checks passed!")
3. WHEN verification runs in CI mode with failures THEN the system SHALL output the full detailed report
4. WHEN CI mode succeeds THEN the system SHALL display an ASCII logo and value proposition tagline in green
5. WHEN the GitHub Action fails THEN it SHALL provide clear exit codes and error messages for pipeline debugging

### Requirement 4: Traffic Parsing Infrastructure

**User Story:** As a system architect, I want extensible traffic parsing capabilities, so that different log formats can be supported in the future.

#### Acceptance Criteria

1. WHEN implementing traffic parsing THEN the system SHALL define a TrafficIngestor interface for extensibility
2. WHEN parsing Nginx access logs THEN the system SHALL use regular expressions to extract structured data
3. WHEN converting log entries THEN the system SHALL normalize them into NormalizedRecord structures
4. WHEN encountering unsupported log formats THEN the system SHALL provide clear error messages
5. WHEN processing large log files THEN the system SHALL handle them efficiently without memory issues

### Requirement 5: Contract Generation Logic

**User Story:** As a service owner, I want intelligent contract generation from traffic patterns, so that generated specifications accurately reflect actual service behavior.

#### Acceptance Criteria

1. WHEN analyzing traffic records THEN the system SHALL group similar endpoints using path pattern matching
2. WHEN multiple HTTP methods are used for an endpoint THEN the system SHALL include all observed methods in the contract
3. WHEN status codes vary for an endpoint THEN the system SHALL define appropriate status code ranges
4. WHEN query parameters or headers appear consistently THEN the system SHALL mark them as required fields
5. WHEN generating ServiceSpec output THEN it SHALL be compatible with existing flowspec validation logic



以下是对该需求文档的中文审阅与改进建议。内容包含总体评价、关键风险/缺口、逐条评审与可执行补充验收标准、接口与数据结构建议、CI/CD 细节建议、测试建议与待澄清问题清单。

总体评价
- 优点：目标清晰，围绕“探索模式 + 独立合约 + CI 体验 + 可扩展解析 + 智能生成”的闭环设计合理，具备落地可行性与演进空间。
- 改进方向：需将若干描述具体化以提升可测试性与一致性，尤其是“相似路径聚类、必填字段判定、状态码范围、YAML 格式兼容性、GitHub Action 行为、日志规模处理与错误可观测性”等。

关键风险与缺口（优先处理）
- “trace”与“ServiceSpec”的格式未定义（或未链接到既有规范），阻碍验收与互操作。
- Nginx 日志格式存在差异（common/combined/自定义 log_format），需要清晰的支持范围与配置方式；否则解析稳定性和可测试性不足。
- 路径参数化聚类算法与阈值未定义，可能过拟合或过度泛化。
- “必填字段”判定口径未明确（何为“出现得足够一致”），需阈值和最小样本量。
- CI 模式输出与退出码未分级定义，影响流水线可诊断性。
- 大文件与压缩日志的处理策略未述及（流式、并发、采样、内存界限）。
- YAML 合约的版本、兼容性与校验机制（Schema/JSON Pointer 报错）未明确。
- 安全与隐私（PII）在日志探索中的处理未提及（脱敏/采样/屏蔽策略）。

逐条评审与建议

Requirement 1: 流量探索与自动合约生成
- 亮点：覆盖了从解析到合约产出的核心路径，并考虑了日志级别的健壮性（出错不中断）。
- 模糊点与风险
  - 仅提到 Nginx access log，但 Nginx 支持多种 log_format；需限定支持版本或允许用户传入正则。
  - “相似 URL 聚类”的算法、阈值（如唯一值比例、最小样本量）未定义。
  - “必填字段”的来源与判定口径不明确（查询参数、头、还是路径变量？一致性阈值多少？）。
  - 输出 YAML 的结构与 flowspec 现有逻辑的映射未具体说明（字段名与语义）。
- 建议补充验收标准（可配置默认值）
  - 支持 --traffic 接受文件、目录、glob；支持按时间窗口与采样率过滤。
  - 支持 --log-format=nginx_combined 或 --regex="<pattern>"；无法匹配时给出明确错误与示例。
  - 路径聚类：基于分段统计，若某分段的唯一值比例高于阈值（默认≥0.8）且样本数≥N（默认20），则抽象为参数；参数名可用通用名{id}或根据上下文推断{id|uuid|num}。
  - 必填字段判定：字段在同一路径+方法下出现比例≥T（默认0.95）且样本数≥N，则标记为required；其余为optional。
  - 状态码聚合：默认同类整合为范围（如2xx），若跨类分布零散则列出枚举；规则可通过 --status-aggregation=range|exact 控制。
  - 解析容错：记录总行数、成功行数、失败行数与失败样例（限流打印），失败率超过阈值（默认>10%）则整体返回非零码并提示检查 log_format。
  - 性能：对≥5GB 日志要求单进程在可配置内存上限内完成（流式读取），并提供进度与 ETA。

Requirement 2: 独立 YAML 合约支持
- 亮点：解耦合约与源码，便于 DevOps 管理与审阅。
- 模糊点与风险
  - ServiceSpec 的 schema、版本与兼容性位于何处（v1alpha1/v1）？
  - verify 与 align 的别名关系描述略含糊，是否完全等价？未来是否保留/废弃其中之一？
  - --path 指向目录时的行为优先级与 YAML 混合场景不明确（目录中既有源码又有 YAML）。
  - trace 文件的格式尚未定义（HAR、OTel、pcap、flowspec 自有格式？）。
- 建议补充验收标准
  - verify 支持 .yaml/.yml 文件；使用内置 JSON Schema 校验并输出精确的行/列与 JSON Pointer。
  - verify --path=<file.yaml> --trace=<file> 成功比对并输出统计摘要（匹配/不匹配用例数量）。
  - verify 与 align 行为等效；文档标注两者是别名，并声明长期保留策略。
  - 当 --path 为目录时：优先查找 YAML（默认 service-spec.yaml），否则回退旧有源码扫描逻辑；二者共存时需明确合并或选择优先级（建议明确优先 YAML）。
  - 对 trace 的支持格式列表与优先级；无法识别时输出清晰错误并建议转换工具。

Requirement 3: CI/CD 体验优化
- 亮点：明确了成功与失败的输出策略，对流水线体验很重要。
- 模糊点与风险
  - “CI 模式”如何开启（--ci 或自动检测 CI 环境变量）？
  - GitHub Action 的缓存策略、安装方式（版本/校验）与多平台支持未述及。
  - 详细报告的输出格式是否可机器读取（如 JSON、JUnit）以便后续步骤消费？
- 建议补充验收标准
  - GitHub Action inputs：path、trace、version、ci、fail-on-warn；支持 Linux/Windows/macOS。
  - 成功时输出：✅ All X checks passed!（包含耗时端到端时长）。
  - 失败时：输出人类可读详细报告，并在 GHA 中使用分组与注释；同时产出 machine-readable 工件（JSON 摘要与可选 JUnit）。
  - ASCII logo 与标语仅在 TTY 且支持颜色时展示；尊重 NO_COLOR、CI 环境变量。
  - 退出码规范：0=成功；1=校验失败；2=合约/YAML 格式错误；3=日志/trace 解析错误；4=运行时/环境错误；64=用法错误。
  - Action 失败时输出错误类别、建议修复步骤与错误定位。

Requirement 4: 流量解析基础设施
- 亮点：通过接口抽象留好了可扩展点。
- 模糊点与风险
  - TrafficIngestor 的最小接口、错误传播与流式迭代器机制未明确。
  - 大文件与压缩文件的策略（.gz/.zst）与并发模型未说明。
- 建议补充验收标准
  - 定义 TrafficIngestor 接口：supports(file)、ingest(paths, options)->迭代器/通道、metrics()、close()；错误通过回调或聚合上报。
  - Nginx ingestor：内置常见 log_format（common/combined）；支持 --regex 自定义。解析提取 method、path、status、ts，可选 host、query、ua、req_size、resp_size。
  - 归一化结构 NormalizedRecord：method、path、status、timestamp、query(map)、headers(map，可选)、host、scheme（可选）、bodyBytes（可选）。
  - 不支持的格式：识别并报错，提示使用 --regex 或更换 ingestor。
  - 性能：行流式读取、可并行解析；内存上限可配置；对超大文件提供分块与背压；对压缩文件支持流式解压。

Requirement 5: 合约生成逻辑
- 亮点：覆盖方法集合、状态码聚合、必填字段提取等关键要素。
- 模糊点与风险
  - 状态码“范围”与“枚举”选择规则不清；可能影响验证严格度。
  - 路径聚类冲突（不同资源树形）与大小写/结尾斜杠/URL 解码细节未定义。
- 建议补充验收标准
  - 路径归一：统一大小写策略（默认大小写敏感）、去除尾部斜杠（可配置）、对 URL 编码进行解码规范化。
  - 聚类冲突处理：若两个模式覆盖高度重叠，保持更具体的模式优先；在输出中附带支持度（样本计数）。
  - 方法集合：为同一路径生成 method 列表；必要时按方法拆分 endpoint 节点以提升精度（可配置）。
  - 状态码聚合策略：同类聚合为 2xx/4xx/5xx，跨类不连续时用枚举；在输出中保留样本数与首次/最近出现时间，方便审阅。
  - 必填字段阈值可配；同时保留“可选字段清单”，便于后续精炼。
  - 生成的 ServiceSpec 包含版本字段（apiVersion/kind），并通过同一 Schema 校验，确保与现有 flowspec 验证逻辑兼容。

接口与数据结构建议（概要）
- TrafficIngestor
  - supports(filePath): bool
  - ingest(inputs, options) -> Iterator<NormalizedRecord>（或回调/通道）；遇错不中断，累计错误计数与样例
  - metrics(): 返回解析行数、错误率、吞吐、耗时
  - close(): 释放资源
- NormalizedRecord（最小集合）
  - method、path、status、timestamp（带时区）
  - query: map<string, string|list>
  - headers: map<string, string|list>（可选，允许采样）
  - host、scheme（可选）、bodyBytes（可选）
- ServiceSpec（关键字段建议）
  - apiVersion、kind、metadata{name, version}
  - service{endpoints[]}
  - endpoint: path（可含参数）、methods[]、responses（statusRanges|codes）、required{query[], headers[]}, optional{query[], headers[]}
  - stats（可选）：supportCount、firstSeen、lastSeen

CI/CD 细节建议
- GitHub Action
  - inputs：path、trace、version（CLI 版本或渠道）、ci（布尔）、status-aggregation、required-threshold、min-samples
  - outputs：summary_json（机器可读）、report_url（如上传到 artifact）
  - 缓存：利用 actions/cache 缓存 CLI 与依赖
- 终端输出
  - 成功：最简一行摘要 + 关键计数（checks、endpoints、duration）
  - 失败：详细报告 + GitHub 注释；同时输出 JSON 摘要
  - 颜色：仅在 TTY 下启用，尊重 NO_COLOR；CI 下默认绿色/红色启用但可关闭
- 退出码
  - 0 成功；1 校验失败；2 合约/YAML 错误；3 解析错误；4 运行时/环境错误；64 用法错误
- 诊断
  - 打印版本与构建信息；打印输入解析统计；在失败时给出“下一步建议”

测试与验收建议（抽样清单）
- Nginx 日志解析：common、combined、带缺失字段、含引号/转义、UTC/本地时区、极端长路径
- 大文件与压缩：10GB 文本与 .gz 流式解析，内存曲线与吞吐达标
- 聚类算法：数值型段、UUID 段、日期段、混合静态/动态段；阈值边界测试（0.79/0.8/0.81）
- 必填字段：存在95%、90%、100%的三组样本，验证 required/optional 判定
- 状态码：单类/跨类/不连续，range 与 exact 切换正确
- YAML Schema：合法/非法样例，报错含行列与 JSON Pointer
- verify/align：命令等价性；目录与 YAML 共存的优先级
- CI 模式：成功最简输出、失败详细输出、JSON 摘要、退出码分层、NO_COLOR 生效
- 跨平台：Linux/macOS/Windows 路径与编码差异
- 隐私：含邮箱/手机号等 PII 的日志，脱敏与输出不泄露

待澄清问题
- trace 的格式与来源是什么？是否支持多种格式（HAR、OTel、pcap、自研）及其优先级？
- ServiceSpec 的正式 Schema、版本与兼容策略？是否与 OpenAPI 做映射或网关导入导出？
- 路径参数命名策略（统一 {var} 还是智能命名 {userId}/{orderId}）？
- 当目录下既有源码也有 YAML 时，优先级如何？是否允许合并？
- 合约生成是否保留统计信息（首次/末次出现、样本数）以便审阅？
- 是否支持日志时间窗口过滤、采样率、去重（防止高频重复请求主导统计）？
- 是否需要机器可读失败报告（JSON/JUnit）作为 CI 工件？
- 安全与隐私策略（脱敏、忽略敏感头部/参数、GDPR 合规）？

总结
- 该文档总体方向正确，建议引入明确的阈值、Schema、退出码与接口约定，以提升可测试性与工程落地性。若需要，我可以据此产出一版“修订后的中文需求文档”（包含更新后的验收标准与参数选项），或提供示例 YAML/日志正则以便团队对齐。