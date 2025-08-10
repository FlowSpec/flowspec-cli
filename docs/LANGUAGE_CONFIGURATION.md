# 语言配置指南 / Language Configuration Guide

## 概述 / Overview

FlowSpec CLI 支持多种语言的输出，本文档说明如何配置和使用不同的语言设置。

FlowSpec CLI supports multiple languages for output. This document explains how to configure and use different language settings.

## 支持的语言 / Supported Languages

- `en` - English (英语)
- `zh` - 简体中文 (Simplified Chinese)
- `zh-TW` - 繁体中文 (Traditional Chinese)
- `ja` - 日本語 (Japanese)
- `ko` - 한국어 (Korean)
- `fr` - Français (French)
- `de` - Deutsch (German)
- `es` - Español (Spanish)

## 语言设置优先级 / Language Setting Priority

语言设置按以下优先级确定：
Language settings are determined by the following priority:

1. **命令行参数 / Command Line Parameter**: `--lang` 或 `--language`
2. **环境变量 / Environment Variable**: `FLOWSPEC_LANG`
3. **系统语言 / System Language**: `LANG` 环境变量
4. **默认语言 / Default Language**: English (`en`)

## 使用方法 / Usage

### 1. 命令行参数 / Command Line Parameter

```bash
# 使用中文输出
flowspec-cli align --path=./src --trace=./trace.json --lang=zh

# 使用英文输出
flowspec-cli align --path=./src --trace=./trace.json --lang=en

# 使用日文输出
flowspec-cli align --path=./src --trace=./trace.json --lang=ja
```

### 2. 环境变量 / Environment Variable

```bash
# 设置环境变量（临时）
export FLOWSPEC_LANG=zh
flowspec-cli align --path=./src --trace=./trace.json

# 设置环境变量（永久，添加到 ~/.bashrc 或 ~/.zshrc）
echo 'export FLOWSPEC_LANG=zh' >> ~/.bashrc
```

### 3. 测试环境语言设置 / Test Environment Language Setting

在测试环境中，可以使用 `FLOWSPEC_LANG` 环境变量来确保语言一致性：

For testing environments, use the `FLOWSPEC_LANG` environment variable to ensure language consistency:

```bash
# 测试时强制使用英文
export FLOWSPEC_LANG=en
go test ./...

# 测试时强制使用中文
export FLOWSPEC_LANG=zh
go test ./...
```

## 系统集成 / System Integration

### CI/CD 环境 / CI/CD Environment

在 CI/CD 环境中，建议明确设置语言以确保输出一致性：

In CI/CD environments, it's recommended to explicitly set the language for consistent output:

```yaml
# GitHub Actions 示例
env:
  FLOWSPEC_LANG: en

# Docker 示例
ENV FLOWSPEC_LANG=en
```

### 脚本集成 / Script Integration

在脚本中使用时，可以临时设置语言：

When using in scripts, you can temporarily set the language:

```bash
#!/bin/bash
# 保存原始语言设置
ORIGINAL_LANG=${FLOWSPEC_LANG:-}

# 设置脚本使用的语言
export FLOWSPEC_LANG=en

# 执行 FlowSpec CLI
flowspec-cli align --path=./src --trace=./trace.json

# 恢复原始设置
if [ -n "$ORIGINAL_LANG" ]; then
    export FLOWSPEC_LANG="$ORIGINAL_LANG"
else
    unset FLOWSPEC_LANG
fi
```

## 故障排除 / Troubleshooting

### 语言不一致问题 / Language Inconsistency Issues

如果遇到不同模块输出语言不一致的问题：

If you encounter language inconsistency issues across different modules:

1. **检查环境变量 / Check Environment Variables**:
   ```bash
   echo $FLOWSPEC_LANG
   echo $LANG
   ```

2. **使用明确的语言参数 / Use Explicit Language Parameter**:
   ```bash
   flowspec-cli align --lang=en --path=./src --trace=./trace.json
   ```

3. **清理环境变量 / Clean Environment Variables**:
   ```bash
   unset FLOWSPEC_LANG
   unset LANG
   ```

### 不支持的语言 / Unsupported Language

如果设置了不支持的语言，系统会：

If an unsupported language is set, the system will:

1. 显示警告信息 / Show a warning message
2. 回退到环境检测 / Fall back to environment detection
3. 最终使用英文作为默认语言 / Finally use English as the default language

## 开发者指南 / Developer Guide

### 添加新语言支持 / Adding New Language Support

1. 在 `internal/i18n/i18n.go` 中添加新的语言常量
2. 创建对应的消息文件（如 `messages_xx.go`）
3. 在 `loadMessages()` 函数中添加新语言的处理
4. 更新 `IsSupported()` 和 `GetSupportedLanguages()` 函数

### 测试语言功能 / Testing Language Features

```go
// 在测试中设置语言
func TestWithChineseLanguage(t *testing.T) {
    // 设置测试语言
    SetTestLanguage(i18n.LanguageChinese)
    defer ClearTestLanguage()
    
    // 执行测试
    // ...
}
```

## 最佳实践 / Best Practices

1. **明确设置语言 / Explicitly Set Language**: 在生产环境和测试环境中明确设置语言
2. **一致性检查 / Consistency Check**: 确保所有模块使用相同的语言设置
3. **环境隔离 / Environment Isolation**: 在测试中使用独立的语言环境
4. **文档更新 / Documentation Update**: 添加新语言时更新相关文档

## 更新历史 / Change History

- **v0.2.0**: 重构语言设置系统，统一语言管理
- **v0.1.0**: 初始多语言支持