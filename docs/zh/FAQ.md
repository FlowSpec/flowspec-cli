# 常见问题解答 (FAQ)

本文档回答了使用 FlowSpec CLI 过程中的常见问题。

## 目录

- [安装和配置](#安装和配置)
- [使用问题](#使用问题)
- [ServiceSpec 注解](#servicespec-注解)
- [轨迹数据](#轨迹数据)
- [性能和优化](#性能和优化)
- [故障排除](#故障排除)

## 安装和配置

### Q: 支持哪些 Go 版本？

A: FlowSpec CLI 需要 Go 1.21 或更高版本。我们建议使用最新的稳定版本。

```bash
# 检查 Go 版本
go version

# 如果版本过低，请升级 Go
```

### Q: 如何在 CI/CD 中使用 FlowSpec CLI？

A: 可以通过以下方式在 CI/CD 中集成：

```yaml
# GitHub Actions 示例
- name: Install FlowSpec CLI
  run: go install github.com/FlowSpec/flowspec-cli/cmd/flowspec-cli@latest

- name: Run FlowSpec Validation
  run: |
    flowspec-cli align \
      --path=./src \
      --trace=./traces/integration-test.json \
      --output=json > flowspec-report.json

- name: Upload Report
  uses: actions/upload-artifact@v3
  with:
    name: flowspec-report
    path: flowspec-report.json
```

### Q: 如何配置日志级别？

A: 使用 `--log-level` 参数：

```bash
# 调试模式
flowspec-cli align --path=./src --trace=./trace.json --log-level=debug

# 静默模式
flowspec-cli align --path=./src --trace=./trace.json --log-level=error
```

## 使用问题

### Q: 为什么没有找到任何 ServiceSpec？

A: 可能的原因：

1. **文件路径错误**: 确保 `--path` 指向正确的源代码目录
2. **注解格式错误**: 检查 ServiceSpec 注解的语法
3. **文件类型不支持**: 确保文件扩展名为 `.java`、`.ts` 或 `.go`

```bash
# 使用详细输出查看解析过程
flowspec-cli align --path=./src --trace=./trace.json --verbose
```

### Q: 如何处理大型项目？

A: 对于大型项目，建议：

1. **分批处理**: 将项目分成多个子目录分别验证
2. **并行处理**: FlowSpec CLI 自动使用多核并行处理
3. **内存监控**: 使用 `--verbose` 查看内存使用情况

```bash
# 分批处理示例
flowspec-cli align --path=./src/user-service --trace=./traces/user-service.json
flowspec-cli align --path=./src/order-service --trace=./traces/order-service.json
```

### Q: 退出码的含义是什么？

A: FlowSpec CLI 使用以下退出码：

- `0`: 验证成功，所有断言通过
- `1`: 验证失败，存在断言不通过的情况
- `2`: 系统错误，如文件不存在、解析错误等

```bash
# 在脚本中检查退出码
flowspec-cli align --path=./src --trace=./trace.json
if [ $? -eq 0 ]; then
    echo "验证成功"
elif [ $? -eq 1 ]; then
    echo "验证失败"
else
    echo "系统错误"
fi
```

## ServiceSpec 注解

### Q: ServiceSpec 注解的基本格式是什么？

A: 基本格式如下：

```java
/**
 * @ServiceSpec
 * operationId: "uniqueOperationId"
 * description: "操作描述"
 * preconditions: {
 *   "condition_name": JSONLogic_Expression
 * }
 * postconditions: {
 *   "condition_name": JSONLogic_Expression
 * }
 */
```

### Q: 如何编写复杂的断言表达式？

A: 使用 JSONLogic 语法编写复杂表达式：

```json
{
  "preconditions": {
    "request_validation": {
      "and": [
        {"!=": [{"var": "span.attributes.http.method"}, null]},
        {"in": [{"var": "span.attributes.http.method"}, ["POST", "PUT"]}},
        {">": [{"var": "span.attributes.request.body.password.length"}, 8]}
      ]
    }
  },
  "postconditions": {
    "response_validation": {
      "or": [
        {"==": [{"var": "span.attributes.http.status_code"}, 200]},
        {"==": [{"var": "span.attributes.http.status_code"}, 201]}
      ]
    }
  }
}
```

### Q: 可以在断言中访问哪些变量？

A: 在断言表达式中可以访问：

**前置条件中**:
- `span.attributes.*`: Span 属性
- `span.startTime`: 开始时间
- `span.name`: Span 名称

**后置条件中**:
- 前置条件的所有变量
- `endTime`: 结束时间
- `status.code`: 状态码
- `status.message`: 状态消息
- `events[*].*`: 事件数组

### Q: 如何处理可选字段？

A: 使用条件判断处理可选字段：

```json
{
  "postconditions": {
    "optional_field_check": {
      "if": [
        {"!=": [{"var": "span.attributes.response.body.optionalField"}, null]},
        {">": [{"var": "span.attributes.response.body.optionalField"}, 0]},
        true
      ]
    }
  }
}
```

## 轨迹数据

### Q: 支持哪些轨迹数据格式？

A: 目前支持 OpenTelemetry JSON 格式的轨迹数据。

### Q: 如何生成 OpenTelemetry 轨迹数据？

A: 可以使用各种 OpenTelemetry SDK：

```java
// Java 示例
import io.opentelemetry.api.OpenTelemetry;
import io.opentelemetry.api.trace.Tracer;

Tracer tracer = OpenTelemetry.getGlobalTracer("my-service");
Span span = tracer.spanBuilder("createUser").startSpan();
// ... 业务逻辑
span.end();
```

### Q: 轨迹文件太大怎么办？

A: FlowSpec CLI 支持大文件处理：

1. **流式处理**: 自动使用流式解析，避免内存溢出
2. **内存限制**: 默认限制峰值内存使用 500MB
3. **分片处理**: 可以将大轨迹文件分割成多个小文件

```bash
# 监控内存使用
flowspec-cli align --path=./src --trace=./large-trace.json --verbose
```

### Q: 如何匹配 ServiceSpec 和 Span？

A: 匹配基于 `operationId`：

1. ServiceSpec 中的 `operationId` 字段
2. Span 中的 `name` 属性或 `operation.name` 属性
3. 支持模糊匹配和正则表达式匹配

## 性能和优化

### Q: 如何提高解析性能？

A: 性能优化建议：

1. **并行处理**: FlowSpec CLI 自动使用多核并行处理
2. **文件过滤**: 只扫描包含 ServiceSpec 的目录
3. **缓存机制**: 启用解析结果缓存

```bash
# 使用性能监控
flowspec-cli align --path=./src --trace=./trace.json --verbose
```

### Q: 内存使用过高怎么办？

A: 内存优化策略：

1. **分批处理**: 将大项目分成小批次处理
2. **调整限制**: 根据系统配置调整内存限制
3. **清理缓存**: 定期清理解析缓存

### Q: 如何进行性能基准测试？

A: 使用内置的性能测试：

```bash
# 运行性能测试
make performance-test

# 运行基准测试
make benchmark

# 运行压力测试
make stress-test
```

## 故障排除

### Q: 解析错误如何调试？

A: 调试步骤：

1. **启用详细输出**: 使用 `--verbose` 参数
2. **检查日志**: 查看详细的错误信息
3. **验证语法**: 确保 ServiceSpec 注解语法正确

```bash
# 调试模式
flowspec-cli align --path=./src --trace=./trace.json --verbose --log-level=debug
```

### Q: JSONLogic 表达式错误怎么办？

A: 常见问题和解决方案：

1. **语法错误**: 检查 JSON 格式是否正确
2. **变量引用错误**: 确保变量路径正确
3. **类型不匹配**: 检查数据类型是否匹配

```bash
# 使用在线 JSONLogic 测试工具验证表达式
# https://jsonlogic.com/play.html
```

### Q: 轨迹数据解析失败怎么办？

A: 检查步骤：

1. **文件格式**: 确保是有效的 JSON 格式
2. **数据结构**: 检查是否符合 OpenTelemetry 规范
3. **文件权限**: 确保有读取文件的权限

```bash
# 验证 JSON 格式
cat trace.json | jq . > /dev/null && echo "JSON 格式正确" || echo "JSON 格式错误"
```

### Q: 如何报告 Bug？

A: 报告 Bug 时请提供：

1. **版本信息**: `flowspec-cli --version`
2. **命令行参数**: 完整的命令行调用
3. **错误信息**: 完整的错误输出
4. **环境信息**: 操作系统、Go 版本等
5. **重现步骤**: 详细的重现步骤

```bash
# 收集环境信息
flowspec-cli --version
go version
uname -a
```

## 更多帮助

如果这里没有找到您问题的答案，请：

1. 查看 [API 文档](./API.md)
2. 查看 [架构文档](./ARCHITECTURE.md)
3. 在 [GitHub Issues](https://github.com/FlowSpec/flowspec-cli/issues) 中搜索
4. 在 [GitHub Discussions](https://github.com/FlowSpec/flowspec-cli/discussions) 中提问
5. 发送邮件到 youming@flowspec.org

---

**提示**: 这个 FAQ 会持续更新，如果您有新的问题或建议，欢迎提交 Issue 或 Pull Request。
