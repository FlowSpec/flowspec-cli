# FlowSpec 产品分层规范文档

## 概述

本文档定义了 FlowSpec CLI 的产品分层标准，确保不同版本间的功能边界清晰，代码组织合理，商业模式可持续。

## 产品分层定义

### Community Edition (CE) - 开源社区版

**定位**: 本地运行、零遥测的基石版，提供核心的规约解析和校验能力。

**价值主张**: 确保 FlowSpec 作为开源工具的独立性、透明性和可验证性，是整个生态的流量入口和信任基础。

#### 功能包含范围
✅ **核心功能**
- ServiceSpec 注解解析 (Java/TypeScript/Go)
- OpenTelemetry trace 数据摄取
- 规约与执行轨迹对齐验证
- 本地报告生成 (Human/JSON格式)
- YAML 独立合约文件支持
- 流量探索和合约自动生成

✅ **用户体验功能**
- 完整的国际化支持 (8种语言)
- CLI 命令行界面
- 详细的错误提示和帮助信息
- 示例项目和文档

✅ **技术基础设施**
- 跨平台支持 (Windows/macOS/Linux)
- Docker 容器化支持
- CI/CD 集成能力
- 插件化架构基础

#### 严格禁止的功能
❌ **遥测和数据收集**
- 任何形式的使用统计收集
- 错误报告自动上传
- 性能指标遥测
- 用户行为分析

❌ **网络依赖功能**
- 在线更新检查
- 云端服务调用
- 远程配置获取
- 在线帮助或文档

❌ **商业增值功能**
- 高级报告模板
- 性能优化建议
- AI 驱动的洞察
- 企业级集成接口

### Pro Edition - 个人开发者增值服务

Pro 版本分为两个子版本，满足不同隐私偏好的用户需求。

#### Pro Free - 数据计划版

**定位**: 通过"数据换取功能"的自愿协议，为用户提供增值功能。

**价值主张**: 为 FlowSpec 产品改进和 AI 模型训练提供数据养料，同时为用户提供增强体验。

##### 新增功能
✅ **智能增强**
- AI 驱动的规约建议
- 自动化最佳实践推荐
- 智能错误诊断和修复建议
- 代码质量评分

✅ **使用分析**
- 个人使用统计和趋势
- 项目健康度评估
- 性能基准对比
- 自定义仪表板

✅ **社区功能**
- 匿名化经验分享
- 最佳实践案例库
- 社区问答和反馈
- 在线更新和通知

##### 数据收集范围
📊 **自愿提供的数据**
- 使用模式统计 (匿名化)
- 性能指标 (匿名化)
- 错误日志 (去敏化)
- 功能使用反馈

##### 隐私保护措施
🔒 **数据保护**
- 严格的数据匿名化
- 本地数据预处理
- 用户可控的数据分享级别
- 随时撤回数据授权

#### Pro Privacy - 付费隐私版

**定位**: 为隐私敏感型用户提供不贡献数据也能使用高级功能的商业选项。

**价值主张**: 展示对用户选择的尊重，为希望获得增值功能但不愿分享数据的用户提供解决方案。

##### 功能特性
✅ **零数据收集**
- 完全本地运行
- 无任何遥测或统计
- 离线 AI 模型
- 本地化高级功能

✅ **性能增强**
- 高级性能优化算法
- 更快的解析和验证速度
- 内存优化
- 并行处理优化

✅ **专业功能**
- 高级报告模板
- 自定义规则引擎
- 批量处理能力
- 专业技术支持

### Cloud Edition - 企业级解决方案

**定位**: 一站式、合规、企业级解决方案。

**价值主张**: 提供云端托管、组织级协作以及强大的 AI 赋能，帮助企业将规约治理能力扩展到整个团队和复杂的生产环境中。数据主权和合规性是其核心卖点。

#### 功能特性
✅ **云端服务**
- SaaS 托管服务
- 弹性计算资源
- 全球 CDN 加速
- 高可用性保证

✅ **企业协作**
- 团队权限管理
- 项目协作工作流
- 集中化配置管理
- 审计日志和合规报告

✅ **AI 增强**
- 云端 AI 模型服务
- 大规模数据分析
- 预测性洞察
- 智能化运维建议

✅ **企业集成**
- SSO 单点登录
- API 网关集成
- CI/CD 平台连接
- 企业级监控告警

## 代码组织规范

### 目录结构标准

```
flowspec-cli/
├── cmd/flowspec-cli/           # CE: CLI 入口
├── internal/                   # CE: 核心功能
│   ├── engine/                 # CE: 校验引擎
│   ├── parser/                 # CE: 解析器
│   ├── ingestor/              # CE: 数据摄取
│   ├── renderer/              # CE: 报告渲染
│   ├── models/                # CE: 数据模型
│   └── i18n/                  # CE: 国际化
├── internal/pro/              # Pro: 增值功能 (构建时排除)
│   ├── analytics/             # Pro Free: 分析功能
│   ├── ai/                    # Pro: AI 增强
│   ├── telemetry/             # Pro Free: 遥测
│   └── premium/               # Pro Privacy: 付费功能
├── internal/cloud/            # Cloud: 企业功能 (构建时排除)
│   ├── collaboration/         # Cloud: 协作功能
│   ├── enterprise/            # Cloud: 企业集成
│   └── saas/                  # Cloud: SaaS 服务
├── pkg/                       # 公共库 (所有版本共享)
├── examples/                  # CE: 示例项目
├── docs/                      # CE: 文档
└── scripts/                   # CE: 构建脚本
```

### 构建标签规范

```go
// CE 版本 - 无标签或 +build community
//go:build community
// +build community

// Pro Free 版本
//go:build pro && !privacy
// +build pro,!privacy

// Pro Privacy 版本  
//go:build pro && privacy
// +build pro,privacy

// Cloud 版本
//go:build cloud
// +build cloud
```

### 功能开关机制

```go
// internal/config/features.go
package config

type FeatureFlags struct {
    // CE 功能 - 始终启用
    CoreParsing     bool // 始终 true
    TraceIngestion  bool // 始终 true
    LocalReporting  bool // 始终 true
    
    // Pro 功能 - 根据许可证控制
    Analytics       bool // Pro Free + Pro Privacy
    AIEnhancements  bool // Pro Free + Pro Privacy  
    Telemetry       bool // 仅 Pro Free
    PremiumFeatures bool // 仅 Pro Privacy
    
    // Cloud 功能 - 根据订阅控制
    Collaboration   bool // 仅 Cloud
    EnterpriseSSO   bool // 仅 Cloud
    SaaSIntegration bool // 仅 Cloud
}
```

## 许可证和分发策略

### 开源许可证 (CE)
- **许可证**: Apache 2.0
- **分发方式**: GitHub Releases, Package Managers
- **构建**: 公开构建流程
- **代码**: 完全开源

### 商业许可证 (Pro/Cloud)
- **许可证**: 商业许可证 + 使用条款
- **分发方式**: 官方下载 + 许可证验证
- **构建**: 私有构建流程
- **代码**: 闭源增值模块

## 开发工作流规范

### 分支策略
```
main              # CE 版本主分支 (公开)
├── feature/*     # CE 功能开发 (公开)
├── bugfix/*      # CE 问题修复 (公开)
└── release/*     # CE 版本发布 (公开)

pro/main          # Pro 版本主分支 (私有)
├── pro/feature/* # Pro 功能开发 (私有)
└── pro/release/* # Pro 版本发布 (私有)

cloud/main        # Cloud 版本主分支 (私有)
├── cloud/feature/* # Cloud 功能开发 (私有)
└── cloud/release/* # Cloud 版本发布 (私有)
```

### 代码审查标准

#### CE 版本审查重点
✅ **必须检查**
- 无遥测或数据收集代码
- 无网络依赖 (除必要的依赖下载)
- 无商业功能代码
- Apache 2.0 许可证头部
- 完整的测试覆盖

#### Pro/Cloud 版本审查重点  
✅ **必须检查**
- 许可证验证机制
- 数据隐私保护 (Pro Free)
- 企业安全标准 (Cloud)
- 功能开关正确性
- 向后兼容性

## 质量保证标准

### 测试策略

#### CE 版本测试
```bash
# 单元测试 - 80%+ 覆盖率
go test ./internal/... -coverprofile=coverage.out

# 集成测试 - 核心工作流
go test ./tests/integration/...

# 端到端测试 - CLI 功能
go test ./tests/e2e/...

# 性能测试 - 基准测试
go test -bench=. ./internal/engine/...
```

#### Pro/Cloud 版本测试
```bash
# 许可证测试
go test ./internal/pro/license/...

# 功能开关测试  
go test -tags=pro ./internal/pro/...

# 企业集成测试
go test -tags=cloud ./internal/cloud/...
```

### 性能基准

#### CE 版本性能要求
- **内存使用**: < 100MB (典型项目)
- **启动时间**: < 2 秒
- **解析速度**: > 1000 文件/秒
- **验证速度**: > 100 轨迹/秒

#### Pro/Cloud 版本性能要求
- **内存使用**: < 500MB (大型项目)
- **并发处理**: 支持多项目并行
- **云端响应**: < 500ms API 响应
- **可扩展性**: 支持 1000+ 用户并发

## 发布流程规范

### CE 版本发布
1. **代码冻结**: 功能开发完成
2. **质量检查**: 通过所有测试和审查
3. **文档更新**: README、CHANGELOG、API 文档
4. **构建验证**: 跨平台构建测试
5. **社区预发布**: Beta 版本社区测试
6. **正式发布**: GitHub Release + Package Manager

### Pro/Cloud 版本发布
1. **CE 版本同步**: 确保基础功能一致
2. **许可证测试**: 验证许可证机制
3. **安全审查**: 第三方安全扫描
4. **合规检查**: 数据处理合规性
5. **内部测试**: 完整功能测试
6. **分阶段发布**: 逐步向用户推出

## 合规性和安全标准

### 数据处理合规性
- **GDPR 合规**: 欧盟数据保护法规
- **CCPA 合规**: 加州消费者隐私法案
- **SOC 2 Type II**: 安全运营控制
- **ISO 27001**: 信息安全管理

### 安全开发生命周期
- **威胁建模**: 每个新功能进行威胁分析
- **安全编码**: 遵循 OWASP 安全编码标准
- **依赖扫描**: 第三方依赖安全漏洞扫描
- **渗透测试**: 定期第三方安全测试

## 文档和支持标准

### CE 版本文档
- **开源文档**: 完整的使用和开发文档
- **社区支持**: GitHub Issues, Discussions
- **示例项目**: 完整的演示项目
- **视频教程**: 基础使用教程

### Pro/Cloud 版本文档
- **专业文档**: 详细的配置和集成指南  
- **专业支持**: 邮件、聊天、电话支持
- **培训服务**: 在线和现场培训
- **咨询服务**: 实施和最佳实践咨询

## 商业模式实施

### 定价策略
```
Community (CE):    免费
Pro Free:         免费 (数据换功能)
Pro Privacy:      $9.99/月 (个人开发者)
Cloud Starter:    $29.99/月 (小团队)
Cloud Business:   $99.99/月 (中型企业)
Cloud Enterprise: 联系销售 (大型企业)
```

### 许可证验证机制
```go
// 许可证验证流程
type License struct {
    Type        LicenseType   // CE, ProFree, ProPrivacy, Cloud
    UserId      string        // 用户标识
    ExpiresAt   time.Time     // 过期时间
    Features    []Feature     // 启用的功能
    Signature   string        // 数字签名
}

func (l *License) Validate() error {
    // 1. 验证数字签名
    // 2. 检查过期时间
    // 3. 验证用户标识
    // 4. 启用相应功能
}
```

---

**版本**: v1.0  
**生效日期**: 2025-08-30  
**维护团队**: FlowSpec Architecture Team  
**审查周期**: 季度审查，年度更新