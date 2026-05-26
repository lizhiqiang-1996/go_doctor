# 🏥 Go Doctor

AI 生成 Go 代码质量评估工具。从完整性、惯用性、整洁性、实现度、类型安全 5 个维度评估 AI 生成代码的质量，同时提供 33 条规则覆盖错误处理、安全、并发等常规代码质量问题。

灵感来源于 [react-doctor](https://github.com/millionco/react-doctor) 和 [goreporter](https://github.com/qax-os/goreporter)。

## 特性

- **5 维 AI 质量评估** — Completeness / Idiomatic / Cleanliness / Implementation / Type Safety
- **33 条检测规则** — 覆盖错误处理、安全、并发、性能、代码风格、正确性、死代码
- **0-100 评分系统** — 分类加权 + 密度归一化 + 对数衰减，大型项目评分合理
- **MR/PR 代码审查** — `--diff` 模式只扫描变更行
- **Commit 代码审查** — `--commit` 模式只扫描指定提交的变更行
- **JSON 输出** — 结构化报告，便于集成 CI/CD

## 安装

```bash
go install github.com/lizhiqiang-1996/go_doctor/cmd/go-doctor@latest
```

## 快速开始

```bash
# 评估代码质量
go-doctor . --verbose

# 只看评分
go-doctor . --score

# JSON 格式输出
go-doctor . --json

# MR/PR 代码审查（最常用）
go-doctor . --diff master --verbose
```

## AI 质量评估

Go Doctor 的核心功能是评估 AI 生成代码的质量，从 5 个维度进行检测：

| 维度 | 规则 | 检测内容 | 示例 |
|------|------|---------|------|
| **Completeness** 完整性 | `completeness/placeholder-comment` | TODO/FIXME/HACK 占位符 — 实现不完整 | `// TODO: implement this` |
| **Idiomatic** 惯用性 | `idiomatic/snake-case-naming` | snake_case 命名 — 不符合 Go 惯例 | `user_id` → `userID` |
| **Cleanliness** 整洁性 | `cleanliness/debug-print` | fmt.Println 调试代码 — 遗留调试语句 | `fmt.Println("debug")` → `log.Printf` |
| **Implementation** 实现度 | `implementation/empty-func-body` | 空函数体/存根 — 未真正实现 | `func Foo() { return nil }` |
| **Type Safety** 类型安全 | `type-safety/overly-broad-interface` | interface{}/any — 类型安全不足 | `func Process(data any) any` |

5 条 AI 质量规则默认包含在每次扫描中，无需额外参数。

## 使用方式

### 全量扫描

```bash
go-doctor /path/to/project --verbose
```

扫描所有 Go 文件，输出完整的代码质量报告（包含 5 维 AI 质量评估）。

### MR/PR 代码审查

```bash
# 对比 main 分支
go-doctor . --diff main --verbose

# 对比远程分支
go-doctor . --diff origin/main --verbose
```

只扫描与目标分支有差异的变更行，适合 MR/PR 审查。

### Commit 代码审查

```bash
# 扫描最新提交
go-doctor . --commit HEAD --verbose

# 扫描指定提交
go-doctor . --commit abc1234 --verbose
```

只扫描指定提交中变更的行。

## 命令行参数

```
go-doctor [directory] [options]

Options:
  --verbose         显示每个规则的详细问题
  --score           只输出评分
  --json            输出 JSON 格式报告
  --no-lint         跳过 lint 检查
  --no-dead-code    跳过死代码检测
  --diff [branch]   只扫描与目标分支有差异的变更行（默认: main）
  --commit <hash>   只扫描指定提交中变更的行
  -v, --version     显示版本号
  -h, --help        显示帮助信息
```

## 评分系统

### 评分等级

| 评分 | 等级 | 含义 |
|------|------|------|
| 75-100 | Good | 代码质量可接受 |
| 50-74 | Needs Work | 存在需要修复的问题 |
| 0-49 | Critical | 存在严重问题，需立即处理 |

### 评分算法

1. 每个问题按类别和严重级别加权计分
2. 按文件数量密度归一化，避免大型项目评分过低
3. 使用对数衰减 `math.Log1p(penalty) * 13` 平滑惩罚

### 类别权重

| 类别 | Error 惩罚 | Warning 惩罚 |
|------|-----------|-------------|
| Security | 5 | 2 |
| Error Handling | 5 | 2 |
| Concurrency | 4 | 1.5 |
| AI Quality | 3 | 1.5 |
| Correctness | 3 | 1 |
| Performance | 2 | 0.5 |
| Code Style | 1 | 0.2 |
| Dead Code | 1 | 0.1 |

## 检测规则

### AI 质量评估（5 条）

| 维度 | 规则 | 严重级别 | 检测内容 |
|------|------|---------|---------|
| Completeness | `completeness/placeholder-comment` | warning | TODO/FIXME/HACK 占位符 |
| Idiomatic | `idiomatic/snake-case-naming` | warning | snake_case 非惯用命名 |
| Cleanliness | `cleanliness/debug-print` | warning | fmt.Println 调试代码 |
| Implementation | `implementation/empty-func-body` | warning | 空函数体/存根实现 |
| Type Safety | `type-safety/overly-broad-interface` | warning | interface{}/any 过度泛化 |

### 错误处理（3 条）

| 规则 | 严重级别 | 检测内容 |
|------|---------|---------|
| `error-handling/unchecked-error` | error | 函数返回 error 但未检查 |
| `error-handling/swallowed-error` | warning | error 被赋值但未处理 |
| `error-handling/panic-in-library` | warning | 库代码中使用 panic |

### 安全（4 条）

| 规则 | 严重级别 | 检测内容 |
|------|---------|---------|
| `security/sql-injection` | error | SQL 拼接注入风险 |
| `security/command-injection` | error | 命令注入风险 |
| `security/weak-crypto` | warning | 弱加密算法（MD5/SHA1） |
| `security/hardcoded-credentials` | error | 硬编码密码/密钥 |

### 并发（4 条）

| 规则 | 严重级别 | 检测内容 |
|------|---------|---------|
| `concurrency/defer-in-loop` | warning | 循环中使用 defer |
| `concurrency/range-var-capture` | warning | range 变量捕获问题 |
| `concurrency/missing-mutex-unlock` | error | Mutex 未配对 Unlock |
| `concurrency/goroutine-leak` | warning | Goroutine 泄漏风险 |

### 性能（3 条）

| 规则 | 严重级别 | 检测内容 |
|------|---------|---------|
| `performance/string-concat-in-loop` | warning | 循环中字符串拼接 |
| `performance/unnecessary-conversion` | warning | 不必要的类型转换 |
| `performance/large-struct-copy` | warning | 大结构体值拷贝 |

### 代码风格（3 条）

| 规则 | 严重级别 | 检测内容 |
|------|---------|---------|
| `style/exported-without-comment` | warning | 导出标识符缺少注释 |
| `style/package-naming` | warning | 包名不符合规范 |
| `style/function-complexity` | warning | 函数圈复杂度过高 |

### 正确性（4 条）

| 规则 | 严重级别 | 检测内容 |
|------|---------|---------|
| `correctness/empty-interface` | warning | 空 interface{} 缺乏类型安全 |
| `correctness/unused-label` | warning | 未使用的 label |
| `correctness/redundant-return` | warning | 冗余的 return |
| `correctness/error-check-without-handling` | warning | 检查了 error 但未处理 |

### 长度与复杂度（4 条）

| 规则 | 严重级别 | 检测内容 |
|------|---------|---------|
| `complexity/function-depth` | warning | 函数嵌套深度过深 |
| `length/function-length` | warning | 函数过长（>80 行） |
| `length/file-length` | warning | 文件过长（>500 行） |
| `length/line-length` | warning | 行过长（>120 字符） |

### 变量与死代码（3 条）

| 规则 | 严重级别 | 检测内容 |
|------|---------|---------|
| `shadow/variable-shadow` | warning | 变量遮蔽 |
| `deadcode/unused-global-var` | warning | 未使用的全局变量 |
| `deadcode/unused-struct-field` | warning | 未使用的结构体字段 |

## 配置文件

在项目根目录创建 `go-doctor.config.json`：

```json
{
  "ignore": {
    "rules": ["style/exported-without-comment"],
    "files": ["generated/**", "vendor/**"]
  },
  "lint": true,
  "deadCode": true,
  "verbose": false
}
```

## AI 工具集成

### Skill

将 `.trae/skills/go-doctor/` 复制到项目根目录，AI 编码助手会自动识别并调用。

### .cursorrules / CLAUDE.md

将项目自带的 `.cursorrules` 或 `CLAUDE.md` 复制到目标项目根目录，AI 编码工具会在生成代码后自动运行质量检查。

## CI/CD 集成

### GitHub Actions

```yaml
name: Code Quality
on: [pull_request]
jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: go install github.com/lizhiqiang-1996/go_doctor/cmd/go-doctor@latest
      - run: go-doctor . --verbose
      - run: |
          SCORE=$(go-doctor . --score | grep -oE '[0-9]+' | head -1)
          [ "$SCORE" -lt 50 ] && exit 1
```

### Pre-commit Hook

```bash
cp hooks/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

## 输出示例

```
  ┌─────┐
  │ • • │   55 / 100 Needs Work
  │  ─  │   ████████████████░░░░░░░░░░░░░░
  └─────┘   Go Doctor

  Project: myproject
  Go Version: 1.21.6
  Framework: gin
  Source Files: 42

  AI Quality 15 issues
    ⚠ idiomatic/snake-case-naming ×6
        Variable 'user_id' uses snake_case — not idiomatic Go
        → Use camelCase in Go: rename 'user_id' to 'userID'
        main.go:10  main.go:22

    ⚠ type-safety/overly-broad-interface ×4
        Function parameter uses overly broad type (interface{}/any) — lacks type safety
        → Define a specific interface with required methods
        handler.go:15  handler.go:23

    ⚠ completeness/placeholder-comment ×3
        Placeholder comment detected: // TODO: implement this
        → Replace placeholder with actual implementation
        service.go:8  service.go:45

    ⚠ implementation/empty-func-body ×1
        Function 'Validate' has a minimal body — likely a stub
        → Implement the function with actual logic
        validator.go:12

    ⚠ cleanliness/debug-print ×1
        Debug print statement: fmt.Println()
        → Remove or replace with proper logging: log.Printf(...)
        main.go:42

  Error Handling 3 issues
    ✗ error-handling/unchecked-error ×3
        Function call returns an error but the error is not checked
        → Assign the error return value and check it
        main.go:42  main.go:67  handler.go:15

  Summary: 3 errors, 15 warnings across 8 files
```

## 项目结构

```
go-doctor/
├── cmd/
│   └── go-doctor/          # CLI 入口
├── pkg/
│   ├── analyzer/
│   │   ├── analyzer.go     # AST 分析引擎
│   │   └── rules/          # 检测规则
│   │       ├── ai_code.go  # AI 质量评估（5 维度）
│   │       ├── complexity.go
│   │       ├── concurrency.go
│   │       ├── correctness.go
│   │       ├── deadcode.go
│   │       ├── error_handling.go
│   │       ├── length.go
│   │       ├── performance.go
│   │       ├── registry.go
│   │       ├── security.go
│   │       ├── shadow.go
│   │       └── style.go
│   ├── config/             # 配置加载
│   ├── git/                # Git diff/commit 集成
│   ├── project/            # 项目发现（框架/版本检测）
│   ├── reporter/           # 报告输出（终端 + JSON）
│   ├── scanner/            # 扫描编排器
│   ├── scorer/             # 加权评分算法
│   └── types/              # 核心类型定义
├── .trae/skills/           # Trae Skill
├── .github/workflows/      # CI/CD 模板
├── hooks/                  # Pre-commit Hook
├── .cursorrules            # Cursor 规则
└── CLAUDE.md               # Claude Code 规则
```

## License

MIT
