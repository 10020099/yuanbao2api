# 元宝2API - Yuanbao to OpenAI API

将腾讯元宝网页版转换为 OpenAI 和 Anthropic 兼容的 API 接口，支持流式响应、工具调用、深度思考等高级功能。

## 概述

元宝2API 通过代理腾讯元宝网页版，提供标准的 OpenAI 兼容 API 接口。用户可以使用现有的 OpenAI SDK 或 Anthropic SDK 直接调用元宝的 AI 模型，无需修改现有代码。

## 功能特性

- **OpenAI 兼容接口**：`/v1/chat/completions` 和 `/v1/models`
- **Anthropic 兼容接口**：`/v1/messages`
- **流式和非流式响应**：支持 Server-Sent Events (SSE)
- **工具调用（Tool Calling）**：支持 OpenAI 和 Anthropic 格式
- **深度思考模式**：返回推理过程（`reasoning_content`）
- **联网搜索**：实时获取网络信息
- **多轮对话**：通过 messages 数组传递历史
- **临时会话**：自动创建临时对话，不在元宝界面留记录
- **Web 管理面板**：黑白简约风格的可视化控制台

## 技术栈

- **后端语言**：Go 1.19+
- **Web 框架**：Gin
- **前端**：HTML/CSS/JavaScript
- **环境管理**：godotenv
- **容器化**：Docker（多阶段构建）
- **Node.js 工具**：Express、node-fetch、uuid（辅助脚本）

## 项目结构

```
元宝2API/
├── main.go                    # 应用入口，Gin 服务器配置
├── go.mod / go.sum            # Go 依赖管理
├── package.json               # Node.js 项目配置（辅助脚本）
│
├── api/                       # API 处理逻辑
│   ├── openai.go             # OpenAI 兼容接口实现
│   ├── anthropic.go          # Anthropic 兼容接口实现
│   ├── models.go             # 模型配置和映射
│   └── config.go             # API 配置
│
├── config/                    # 配置管理
│   └── config.go             # 环境变量加载
│
├── yuanbao/                   # 元宝 API 交互模块
│   └── client.go             # 元宝 API 客户端
│
├── toolcall/                  # 工具调用处理模块
│   └── parser.go             # 工具调用解析
│
├── session/                   # 会话管理模块
│   └── session.go            # 会话 ID 生成
│
├── public/                    # 静态文件（Web 管理面板）
│   └── index.html            # 管理面板 UI
│
├── Dockerfile                 # Docker 多阶段构建配置
├── .env.example               # 环境变量示例
└── README.md                  # 项目说明文档
```

## 快速开始

### 前置要求

- Go 1.19 或更高版本
- Node.js 14+ （可选，仅用于辅助脚本）
- Docker （可选，用于容器化部署）
- 腾讯元宝账号（https://yuanbao.tencent.com）

### 1. 获取 Cookie

打开 https://yuanbao.tencent.com，登录后按 F12 打开浏览器控制台，粘贴运行：

```javascript
document.cookie
```

复制输出的完整 Cookie 字符串。

> Cookie 与你的元宝账号绑定，有效期通常为几天到几周，过期后重新提取即可。

### 2. 配置环境变量

在项目根目录创建 `.env` 文件：

```bash
# 必需：从浏览器复制的完整 Cookie
YUANBAO_COOKIE="your_cookie_here"

# 可选：Agent ID（默认: naQivTmsDa）
YUANBAO_AGENT_ID="naQivTmsDa"

# 可选：服务端口（默认: 3000）
PORT=3000

# 可选：Gin 运行模式（debug 或 release，默认: debug）
GIN_MODE=debug
```

### 3. 安装并启动

**开发模式**（需要 Go 环境）：

```bash
# 下载 Go 依赖
go mod download

# 运行应用
go run .
```

**生产模式**（使用 Docker）：

```bash
# 构建镜像
docker build -t yuanbao2api .

# 运行容器
docker run -p 3000:3000 --env-file .env yuanbao2api
```

### 4. 验证安装

访问 http://localhost:3000 查看管理面板，或测试 API：

```bash
curl http://localhost:3000/v1/models
```

## 支持的模型

| 模型 ID | 名称 | 说明 |
|---------|------|------|
| `deep_seek_v3` / `deepseek` | DeepSeek V3.2 | 适合深度推理、代码生成 |
| `hunyuan` | Hunyuan T1 | 腾讯混元，日常对话、创意写作 |
| `gpt_175B_0404` | GPT 175B | 元宝内部模型标识 |

不指定模型时默认使用 DeepSeek V3.2。

## API 使用示例

### 基础聊天

```bash
curl http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deep_seek_v3",
    "messages": [{"role": "user", "content": "你好"}]
  }'
```

### 流式响应

```bash
curl http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deep_seek_v3",
    "messages": [{"role": "user", "content": "你好"}],
    "stream": true
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

每次请求创建新的临时会话，完整 messages 历史格式化后发送给元宝，不会在元宝界面留下记录。支持 `system` 角色设置系统提示。

> 对话历史过长（>20 轮）可能影响性能，建议定期清理或总结。

### 工具调用（Tool Calling）

支持 OpenAI 格式的工具调用：

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

当模型决定调用工具时，响应中 `finish_reason` 为 `tool_calls`：

```json
{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": null,
      "tool_calls": [{
        "id": "call_abc123",
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
    {"role": "assistant", "content": null, "tool_calls": [...]},
    {"role": "tool", "tool_call_id": "call_abc123", "name": "get_weather", "content": "北京今天晴，25°C"}
  ]
}
```

### Anthropic Messages API

兼容 Anthropic Messages API 格式：

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

**工具调用**：使用 `input_schema` 定义参数，响应中 `stop_reason` 为 `tool_use`。

**深度思考**：请求中传 `thinking` 或 `deep_thinking: true` 启用，思考过程以 `thinking` content block 返回。

**流式输出**：`"stream": true`，遵循 Anthropic SSE 事件格式。

### Python SDK 示例

**OpenAI SDK**：

```python
from openai import OpenAI

client = OpenAI(
    api_key="dummy",
    base_url="http://localhost:3000/v1"
)

# 基础对话
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

**Anthropic SDK**：

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
你的应用 → OpenAI/Anthropic SDK → 元宝2API → 元宝服务器 → 返回响应
```

### 核心流程

1. **Cookie 认证**：使用浏览器提取的 Cookie 证明已登录
2. **会话管理**：每次请求自动生成临时会话 ID
3. **格式转换**：将 OpenAI/Anthropic 格式转换为元宝格式
4. **响应处理**：解析元宝响应，转换回标准格式
5. **临时对话**：设置 `isTemporary: true`，不在元宝界面留记录

### API 映射表

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

## 开发指南

### 可用脚本

| 脚本 | 命令 | 说明 |
|------|------|------|
| 启动服务 | `go run .` | 开发模式 |
| 构建二进制 | `go build -o main .` | 编译为可执行文件 |
| 运行测试 | `go test ./...` | 执行所有测试 |
| Docker 构建 | `docker build -t yuanbao2api .` | 构建 Docker 镜像 |
| Docker 运行 | `docker run -p 3000:3000 --env-file .env yuanbao2api` | 运行容器 |

### 项目架构

```
请求 → Gin 路由 → API 处理层 → 元宝 API 交互 → 响应格式转换 → 返回客户端
```

### 核心模块

1. **main.go**：应用入口，配置 Gin 服务器和路由
2. **api/openai.go**：OpenAI 兼容接口实现
3. **api/anthropic.go**：Anthropic 兼容接口实现
4. **api/models.go**：模型配置和映射
5. **yuanbao/client.go**：元宝 API 客户端
6. **toolcall/parser.go**：工具调用解析
7. **public/index.html**：Web 管理面板

### 开发流程

1. 修改代码后，应用会自动重启（使用 `go run .`）
2. 在管理面板测试 API 功能
3. 使用 curl 或 SDK 进行集成测试
4. 提交前运行 `go test ./...` 验证

## 配置说明

### 环境变量

| 变量 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `YUANBAO_COOKIE` | ✓ | — | 从浏览器复制的完整 Cookie |
| `YUANBAO_AGENT_ID` | ✗ | `naQivTmsDa` | 元宝 Agent ID |
| `PORT` | ✗ | `3000` | 服务监听端口 |
| `GIN_MODE` | ✗ | `debug` | Gin 运行模式（`debug` 或 `release`） |

### 配置文件

- `.env`：本地环境变量（不提交到 Git）
- `.env.example`：环境变量示例模板
- `config/config.go`：Go 配置结构和加载逻辑

## 安全提示

- **Cookie 敏感性**：Cookie 是敏感信息，不要分享或提交到公开仓库
- **Git 忽略**：`.gitignore` 已配置忽略 `.env` 文件
- **账号安全**：定期更新 Cookie，如发现异常立即重新提取
- **使用限制**：遵守元宝的使用限制和服务条款

## 注意事项

- Cookie 过期后需要重新从浏览器提取
- 遵守腾讯元宝的使用限制和服务条款
- 本项目仅用于技术研究和学习
- 对话历史过长（>20 轮）可能影响性能，建议定期清理或总结
- 流式模式下工具调用同样支持，`finish_reason` 为 `tool_calls`

## 许可证

MIT License

Copyright (c) 2026 utd-sakura

详见 [LICENSE](./LICENSE) 文件。

---

**项目状态**：活跃开发中  
**主要贡献者**：utd-sakura
