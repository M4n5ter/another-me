# 基于文本格式的ReAct Agent示例

这个示例展示了如何使用不依赖于function call能力的ReAct Agent实现。适用于那些不支持工具调用API的模型，通过解析模型生成的文本格式来执行工具调用。

## 功能特点

- 支持不同的文本格式解析器，可以自定义模型输出的格式
- 与现有的ReAct Agent共享相同的工具生态系统
- 提供流式输出支持
- 可配置的最大迭代次数和系统提示
- 特别适用于不支持function calling的模型（如deepseek-reasoner）

## 可用的文本格式解析器

该实现提供了几种常见的文本格式解析器：

1. `JSONFormatParser` - 解析JSON格式的工具调用，例如：
   ```json
   {"action": "fetch", "action_input": {"url": "https://example.com"}}
   ```

2. `MarkdownFormatParser` - 解析Markdown代码块格式的工具调用，例如：
   ```
   ```fetch
   {"url": "https://example.com"}
   ```
   ```

3. `PredefinedPatternParser` - 解析自定义模式的工具调用，例如：
   ```
   工具: fetch
   参数: {"url": "https://example.com"}
   ```

4. `GuiAgentFormatParser` - 解析类似GUI Agent格式的工具调用，例如：
   ```
   Thought: 我需要获取天气信息
   Action: fetch(url="https://weather.example.com")
   ```

## 使用方法

1. 选择适合的LLM适配器（支持或不支持function calling的模型）
2. 创建一个文本格式解析器实例
3. 构建TextBasedAgent，提供解析器、LLM适配器和工具注册表
4. 运行Agent并处理输出

## 特别说明：适配不支持function calling的模型

对于不支持function calling的模型（如本示例中使用的deepseek-reasoner），需要使用`NoToolChatAdapter`来避免向模型发送工具调用信息。

示例代码：

```go
// 创建模型
chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
    APIKey:      apiKey,
    Model:       "deepseek-reasoner", // 不支持 function call
    MaxTokens:   4096,
    Temperature: 0.7,
})

// 使用NoToolChatAdapter适配器
llmAdapter, err := eino.NewNoToolChatAdapter(ctx, chatModel)

// 选择一个解析器
parser := &reactagent.MarkdownFormatParser{}

// 构建基于文本的ReAct Agent
agent, err := agentBuilder.
    WithLLMAdapter(llmAdapter).
    WithToolRegistry(registry).
    WithSystemPrompt(systemPrompt).
    WithTextFormatParser(parser).
    Build()
```

## 提示词设计

为了让模型返回符合特定格式的内容，需要在系统提示中明确指定输出格式，并且最好提供具体的例子。例如：

```
对于所有任务，你必须使用以下Markdown代码块格式来调用工具：

首先，分析问题：
我需要解决的问题是...

然后，必须使用Markdown代码块来调用工具，格式如下：
```tool_name
{"参数1": "值1", "参数2": "值2"}
```

例如，要获取某网站的内容，应该这样调用：
```fetch
{"url": "https://example.com"}
```
```

## 常见问题解决

1. **模型不按预期格式输出**：
   - 提高temperature参数（如设置为0.7）以增加输出多样性
   - 修改系统提示，添加更详细的格式示例和更明确的指令
   - 在用户输入中明确要求使用特定格式

2. **返回"不支持function calling"错误**：
   - 确保使用`NoToolChatAdapter`而不是标准的`ChatAdapter`
   - 这表明API请求中仍包含了function calling信息

3. **无法解析工具调用**：
   - 检查解析器是否与模型输出格式匹配
   - 打印并检查模型原始输出，看看格式是否正确

## 自定义解析器

你可以通过实现`TextFormatParser`接口来创建自定义的文本格式解析器：

```go
type TextFormatParser interface {
    // ParseToolCalls 从文本中解析出工具调用请求
    // 返回解析出的工具调用列表和解析后的剩余文本
    ParseToolCalls(text string) ([]llminterface.ToolCall, string)
}
``` 

## 如何运行示例

1. 设置环境变量 `DEEPSEEK_API_KEY` 为你的DeepSeek API密钥
2. 执行命令：
   ```bash
   go run examples/text_agent/main.go
   ```

## 实现架构与流程

### 架构解析

我们的实现由两个关键部分组成：

1. **NoToolChatAdapter**: 
   - 专门为不支持function calling的模型设计的适配器
   - 使用普通的ChatModel而非ToolCallingChatModel
   - 在API请求中不包含任何工具定义和工具调用数据结构
   - 仅发送纯文本消息给LLM模型

2. **文本格式解析器**:
   - 实现TextFormatParser接口
   - 从LLM返回的纯文本输出中提取工具调用指令
   - 支持多种格式（JSON、Markdown、自定义格式等）

### 执行流程

1. **初始化**:
   - 创建工具注册表并注册工具（如fetch工具）
   - 创建纯聊天模型（非工具调用模型）
   - 创建NoToolChatAdapter适配器（不发送工具信息）
   - 创建文本格式解析器（如MarkdownFormatParser）
   - 构建TextBasedAgent

2. **执行过程**:
   - 向模型发送系统提示和用户输入
   - 模型以纯文本形式返回包含工具调用格式的输出
   - 文本解析器从输出中提取工具调用
   - 执行工具调用并将结果返回给模型
   - 重复以上步骤直到任务完成或达到最大迭代次数

3. **与标准Agent的区别**:
   - 标准Agent: 通过模型API的特殊数据结构传递工具调用
   - 文本Agent: 通过分析模型生成的文本内容解析工具调用

这种实现方式的优势在于它可以使用几乎任何LLM模型构建ReAct Agent，即使该模型不支持官方的function calling API。只要模型能够理解指令并按照特定格式生成输出，我们就能通过文本解析的方式实现Agent功能。 