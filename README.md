# 元宝2API - Yuanbao to OpenAI API
将元宝网页版转换为 OpenAI 兼容的 API 接口。

## 功能特性
- OpenAI 兼容的 `/v1/chat/completions` 接口
- Anthropic 兼容的 `/v1/messages` 接口
- 支持流式和非流式响应
- 自动格式转换（OpenAI ↔ 元宝 / Anthropic ↔ 元宝）
- 支持多个模型（DeepSeek V3.2、Hunyuan T1）
- 支持多轮对话（通过 messages 数组传递历史）
- 自动使用临时对话，无需手动管理会话 ID
- 完整的 `/v1/models` 接口
- 深度思考模式（`reasoning_content` 输出）
- 联网搜索功能
- **工具调用（Tool Calling）**：OpenAI 和 Anthropic 格式均支持
- Web 管理面板（黑白简约风格）

## 快速开始
### 1. 获取 Cookie（只需一次）
打开 https://yuanbao.tencent.com ，登录后按 F12 打开控制台，粘贴运行：
```js
document.cookie
```
复制输出的完整字符串。

> Cookie 跟你的元宝账号绑定，有效期通常几天到几周，过期后重新提取即可。

### 2. 配置环境变量
创建 `.env` 文件：
```bash
# 必需：从浏览器复制的完整 Cookie
YUANBAO_COOKIE="your_cookie_here"

# 可选：Agent ID（默认: naQivTmsDa）
YUANBAO_AGENT_ID="naQivTmsDa"

# 可选：服务端口（默认: 3000）
PORT=3000
```

### 3. 安装并启动
```bash
# 安装依赖
go mod download

# 开发模式（自动重启）
go run .

# 生产模式（使用 Docker）
docker build -t yuanbao2api .
docker run -p 3000:3000 --env-file .env yuanbao2api
```

### 4. 测试
```bash
go test ./...
```

## 支持的模型
| 模型 ID | 名称 | 说明 |
|---------|------|------|
| `deep_seek_v3` / `deepseek` | DeepSeek V3.2 | 适合深度推理、代码生成 |
| `hunyuan` | Hunyuan T1 | 腾讯混元，日常对话、创意写作 |
| `gpt_175B_0404` | GPT 175B | 元宝内部模型标识 |

不指定模型时默认使用 DeepSeek V3.2。

## API 使用
### 基础聊天
```bash
curl http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deep_seek_v3",
    "messages": [{"role": "user", "content": "你好"}]
  }'
```

### 深度思考
```bash
curl http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deep_seek_v3",
    "messages": [{"role": "user", "content": "解释量子纠缠"}],
    "deep_thinking": true
  }'
```
思考过程通过响应中的 `reasoning_content` 字段返回。也可以在管理面板全局开启。

### 联网搜索
```bash
curl http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deep_seek_v3",
    "messages": [{"role": "user", "content": "今天的新闻"}],
    "internet_search": true
  }'
```

### 流式响应
加 `"stream": true` 即可。

### 多轮对话
通过 `messages` 数组传递对话历史：
```bash
curl http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deep_seek_v3",
    "messages": [
      {"role": "user", "content": "我叫小明"},
      {"role": "assistant", "content": "你好小明！"},
      {"role": "user", "content": "我叫什么名字？"}
    ]
  }'
```
每次请求创建新的临时会话，完整 messages 历史格式化后发送给元宝，不会在元宝界面留下记录。
支持 `system` 角色设置系统提示。对话历史过长（>20 轮）可能影响性能，建议定期清理或总结。

#### 模型列表
```bash
curl http://localhost:3000/v1/models
```

## 工具调用（Tool Calling）
支持 OpenAI 格式的工具调用。当请求中包含 `tools` 参数时，模型会根据需要生成工具调用：
```bash
curl http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deep_seek_v3",
    "messages": [{"role": "user", "content": "北京今天天气怎么样？"}],
    "tools": [{
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "获取指定城市的天气信息",
        "parameters": {
          "type": "object",
          "properties": {
            "city": {"type": "string", "description": "城市名称"}
          },
          "required": ["city"]
        }
      }
    }]
  }'
```
当模型决定调用工具时，响应中 `finish_reason` 为 `tool_calls`，`message.tool_calls` 包含调用信息：
```json
{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": null,
      "tool_calls": [{
        "id": "call_abc123...",
        "type": "function",
        "function": {
          "name": "get_weather",
          "arguments": "{\"city\": \"北京\"}"
        }
      }]
    },
    "finish_reason": "tool_calls"
  }]
}
```
多轮工具对话时，将工具结果以 `tool` 角色回传：
```json
{
  "messages": [
    {"role": "user", "content": "北京今天天气怎么样？"},
    {"role": "assistant", "content": null, "tool_calls": [{"id": "call_abc123", "type": "function", "function": {"name": "get_weather", "arguments": "{\"city\": \"北京\"}"}}]},
    {"role": "tool", "tool_call_id": "call_abc123", "name": "get_weather", "content": "北京今天晴，25°C"},
    {"role": "user", "content": "谢谢"}
  ]
}
```
流式模式下工具调用同样支持，`finish_reason` 为 `tool_calls`。

### Anthropic Messages API
兼容 Anthropic Messages API 格式，支持使用 Claude SDK 或其他 Anthropic 兼容客户端：
```bash
curl http://localhost:3000/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: dummy" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "deep_seek_v3",
    "max_tokens": 4096,
    "messages": [{"role": "user", "content": "你好"}]
  }'
```
**系统提示词**：支持 `system` 参数（字符串或 content block 数组）。
**工具调用**：
```bash
curl http://localhost:3000/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: dummy" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "deep_seek_v3",
    "max_tokens": 4096,
    "tools": [{
      "name": "get_weather",
      "description": "获取指定城市的天气信息",
      "input_schema": {
        "type": "object",
        "properties": {
          "city": {"type": "string", "description": "城市名称"}
        },
        "required": ["city"]
      }
    }],
    "messages": [{"role": "user", "content": "北京今天天气怎么样？"}]
  }'
```
工具调用响应中 `stop_reason` 为 `tool_use`，`content` 包含 `tool_use` block：
```json
{
  "id": "msg_abc123...",
  "type": "message",
  "role": "assistant",
  "content": [{"type": "tool_use", "id": "call_abc123...", "name": "get_weather", "input": {"city": "北京"}}],
  "stop_reason": "tool_use"
}
```
工具结果回传使用 `tool_result` content block：
```json
{
  "messages": [
    {"role": "user", "content": "北京今天天气怎么样？"},
    {"role": "assistant", "content": [{"type": "tool_use", "id": "call_abc123", "name": "get_weather", "input": {"city": "北京"}}]},
    {"role": "user", "content": [{"type": "tool_result", "tool_use_id": "call_abc123", "content": "北京今天晴，25°C"}]}
  ]
}
```
**深度思考**：请求中传 `thinking` 或 `deep_thinking: true` 启用，思考过程以 `thinking` content block 返回。
**流式输出**：`"stream": true`，遵循 Anthropic SSE 事件格式（`message_start` → `content_block_start/delta/stop` → `message_delta` → `message_stop`）。

### Python SDK 示例
```python
from openai import OpenAI

client = OpenAI(
    api_key="dummy",
    base_url="http://localhost:3000/v1"
)

# 单轮对话
response = client.chat.completions.create(
    model="deep_seek_v3",
    messages=[{"role": "user", "content": "你好"}]
)
print(response.choices[0].message.content)

# 工具调用
response = client.chat.completions.create(
    model="deep_seek_v3",
    messages=[{"role": "user", "content": "北京天气如何？"}],
    tools=[{
        "type": "function",
        "function": {
            "name": "get_weather",
            "description": "获取天气",
            "parameters": {
                "type": "object",
                "properties": {"city": {"type": "string"}},
                "required": ["city"]
            }
        }
    }]
)
if response.choices[0].finish_reason == "tool_calls":
    for tc in response.choices[0].message.tool_calls:
        print(f"调用: {tc.function.name}({tc.function.arguments})")
```

### Anthropic Python SDK 示例
```python
import anthropic

client = anthropic.Anthropic(
    api_key="dummy",
    base_url="http://localhost:3000"
)

response = client.messages.create(
    model="deep_seek_v3",
    max_tokens=4096,
    messages=[{"role": "user", "content": "你好"}]
)
print(response.content[0].text)
```

## 工作原理
```
你的应用 → 调用 OpenAI 兼容 API → 元宝2API 服务 → 转发到元宝服务器 → 返回响应
```
- **Cookie**：证明你是已登录用户，提取一次可持续使用
- **会话 ID**：每次请求自动生成，无需手动管理
- **临时对话**：设置 `isTemporary: true`，不会在元宝界面留记录

### API 映射
| OpenAI | Anthropic | 元宝 |
|--------|-----------|------|
| `/v1/models` | — | 返回支持的模型列表 |
| `/v1/chat/completions` | `/v1/messages` | `/api/chat/{conversationId}` |
| `messages[].content` | `messages[].content` | `prompt` |
| `stream: true` | `stream: true` | SSE 流式响应 |
| `model` | `model` | `chatModelId` |
| `tools` | `tools` | 注入系统提示词 |
| `tool_calls` | `tool_use` | 标记解析转换 |
| `tool` role | `tool_result` | 格式化为工具结果文本 |
| — | `system` | 系统提示词 |
| — | `thinking` | 深度思考模式 |

## 安全提示
- Cookie 是敏感信息，不要分享或提交到公开仓库
- `.gitignore` 已配置忽略 `.env` 文件

## 注意事项
- Cookie 过期后重新从浏览器提取
- 遵守元宝的使用限制
- 本项目仅用于技术研究和学习

## License
MIT