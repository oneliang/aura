# Aura Skill System

Skill 系统是一个轻量级的 prompt 模板机制，允许用户通过 Markdown 文件定义自定义技能和行为模式。

## 与 Plugin 系统的区别

| 特性 | Plugin 系统（已移除） | Skill 系统（新） |
|------|---------------------|-----------------|
| 实现方式 | 外部脚本/二进制文件 | Markdown Prompt 模板 |
| 通信协议 | JSON 标准输入输出 | 直接注入 System Prompt |
| 安全性 | 需要执行外部程序 | 仅文本，无执行风险 |
| 复杂度 | 高（需要处理进程管理） | 低（纯文本解析） |
| 适用场景 | 复杂外部工具集成 | 行为模式、工作流程指导 |

## 目录结构

```
modules/skill/
├── pkg/
│   ├── skill/           # Skill 结构定义
│   ├── loader/          # 从目录加载 Skills
│   └── builder/         # 构建 Skills Prompt
├── go.mod
└── README.md
```

## SKILL.md 文件格式

每个 Skill 是一个包含 YAML 元数据的 Markdown 文件：

```markdown
---
name: skill-name
description: 触发此技能的描述（用于 LLM 判断是否需要参考）
---

## 工作流程

1. 第一步...
2. 第二步...

## 注意事项

- 注意项 1
- 注意项 2
```

### 字段说明

- **name**: 技能标识符，用于在 system prompt 中引用
- **description**: 触发描述，LLM 根据此描述判断是否需要参考该技能
- **body**: 技能的完整指令内容，当 LLM 决定使用技能时会参考

## 使用方法

### 1. 创建 Skill 目录

```bash
mkdir -p ~/.aura/skills/my-skill
```

### 2. 编写 SKILL.md

```bash
cat > ~/.aura/skills/my-skill/SKILL.md << 'EOF'
---
name: my-skill
description: 当用户需要 XXX 时使用此技能
---

## 工作流程

1. 第一步...
2. 第二步...
EOF
```

### 3. 配置文件

确保 `~/.aura/config.yaml` 中启用了 skill 系统：

```yaml
skills:
  enabled: true
  directories:
    - ~/.aura/skills
```

### 4. 验证

```bash
./bin/aura "帮我做 XXX"  # 应该触发技能
```

## 技能触发机制

Skill 系统采用 **Progressive Disclosure** 策略：

1. **Metadata 始终在 System Prompt 中**
   - 所有技能的 name + description 始终包含在 system prompt
   - LLM 根据用户请求和 description 判断是否需要参考技能

2. **Body 按需参考**
   - 当 LLM 判断需要某技能时，会参考其完整 body 内容
   - 通过 attention 机制，LLM 自动聚焦相关技能

3. **无显式触发语法**
   - 不需要特殊的触发命令
   - 自然语言匹配即可

## 示例技能

### 数据分析师

```markdown
---
name: data-analyst
description: 分析数据文件并生成统计报告。当用户需要分析 CSV、Excel、JSON 数据或计算统计指标时使用。
---

## 工作流程

1. 使用 file_read 读取数据文件内容
2. 解析数据格式（CSV/JSON/Excel）
3. 使用 calculator 计算统计指标
4. 输出清晰的分析结果
```

### 代码审查员

```markdown
---
name: code-reviewer
description: 审查代码并提供改进建议。当用户请求代码审查、代码质量评估或最佳实践检查时使用。
---

## 工作流程

1. 使用 file_read 读取相关代码文件
2. 使用 code_navigate 查找相关函数和引用
3. 检查代码质量、安全性、性能问题
4. 提供改进建议和示例代码
```

## API 参考

### loader.Loader

```go
import "github.com/oneliang/aura/skill/pkg/loader"

// 创建 loader
loader := loader.NewLoader([]string{"~/.aura/skills"})

// 加载所有技能
skills, err := loader.Load()

// 获取已加载的技能
skills = loader.GetSkills()
```

### builder

```go
import "github.com/oneliang/aura/skill/pkg/builder"

// 生成系统提示段落
promptSection := builder.BuildSystemPromptSection(skills)

// 生成完整技能提示
fullPrompt := builder.BuildFullPrompt(skill)

// 生成技能元数据
metadata := builder.BuildSkillMetadata(skills)
```

## 与 Runtime 集成

Skill 系统已集成到 `AgentRuntime` 中：

```go
import "github.com/oneliang/aura/core/pkg/runtime"

cfg := runtime.DefaultRuntimeConfig()
cfg.Skills.Enabled = true
cfg.Skills.Directories = []string{"~/.aura/skills"}

rt, _ := runtime.New(cfg)
rt.Initialize(ctx)

// Skills 会自动加载并注入到 system prompt
```

## 最佳实践

1. **描述要具体**：description 应清晰说明触发场景
2. **流程要简洁**：步骤不要超过 5-7 步
3. **命名要有意义**：name 应能反映技能用途
4. **避免重复**：相似功能合并到一个技能
5. **测试验证**：创建后实际测试触发效果

## 故障排除

### 技能未触发

- 检查 `description` 是否足够具体
- 确保技能目录在配置文件中
- 验证 `skills.enabled` 为 `true`

### 加载错误

- 检查 `SKILL.md` 格式是否正确
- 确保 YAML frontmatter 以 `---` 开始和结束
- 验证 `name` 和 `description` 字段存在

## 设计哲学

Skill 系统遵循以下设计原则：

1. **轻量级**：基于纯文本，无外部依赖
2. **声明式**：通过描述而非代码定义行为
3. **可组合**：多个技能可以协同工作
4. **易调试**：直接编辑 Markdown 文件
5. **安全**：不执行外部代码，仅作为 prompt
