# 元宝2API - Yuanbao to OpenAI API

将腾讯元宝网页版转换为 OpenAI 和 Anthropic 兼容的 API 接口，支持流式响应、工具调用、深度思考等高级功能。

## 概述

元宝2API 是一个混合型应用（Go + Node.js），通过代理腾讯元宝网页版，提供标准的 OpenAI 兼容 API 接口。用户可以使用现有的 OpenAI SDK 或 Anthropic SDK 直接调用元宝的 AI 模型，无需修改现有代码。

项目支持多种高级功能：
- **OpenAI 兼容接口**：`/v1/chat/completions` 和 `/v1/models`
- **Anthropic 兼容接口**：`/v1/messages`
- **流式和非流式响应**
- **工具调用（Tool Calling）**：支持 OpenAI 和 Anthropic 格式
- **深度思考模式**：返回推理过程
- **联网搜索**：实时获取网络信息
- **多轮对话**：通过 messages 数组传递历史
- **Web 管理面板**：黑白简约风格的控制台

## 技术栈

- **后端语言**：Go 1.19
- **Web 框架**：Gin（Go）
- **前端**：HTML/CSS/JavaScript（管理面板）
- **环境管理**：godotenv
- **容器化**：Docker（多阶段构建）
- **依赖管理**：Go Modules
- **Node.js 工具**：Express、node-fetch、uuid（辅助脚本）

## 项目结构

```
元宝/
├── main.go                    # 应用入口，Gin 服务器配置
├── go.mod                     # Go 模块定义
├── go.sum                     # Go 依赖锁定文件
├── package.json               # Node.js 项目配置（辅助脚本）
├── package-lock.json          # Node.js 依赖锁定
│
├── api/                       # API 处理逻辑
│   └── openai.go             # OpenAI 兼容接口实现
│
├── config/                    # 配置管理
│   └── config.go             # 环境变量加载和配置结构
│
├── public/                    # 静态文件（Web 管理面板）
│   └── index.html            # 管理面板 UI
│
├── yuanbao/                   # 元宝 API 交互模块（推测）
├── toolcall/                  # 工具调用处理模块（推测）
├── session/                   # 会话管理模块（推测）
│
├── Dockerfile                 # Docker 多阶段构建配置
├── .dockerignore              # Docker 忽略文件
├── .env.example               # 环境变量示例
├── .env                       # 环境变量（本地，不提交）
├── .gitignore                 # Git 忽略规则
│
├── README.md                  # 项目说明文档
├── LICENSE                    # MIT 许可证
│
├── extract-config.js          # 配置提取脚本（Node.js）
├── yuanbao2api.js             # Node.js 版本实现（参考）
├── test.js                    # JavaScript 测试文件
├── test.go                    # Go 测试文件
│
└── .github/                   # GitHub 配置（CI/CD 等）
```

## 主要功能

- **OpenAI 兼容 API**：完全兼容 OpenAI `/v1/chat/completions` 接口
- **Anthropic 兼容 API**：支持 Anthropic Messages API 格式
- **多模型支持**：DeepSeek V3.2、Hunyuan T1、GPT 175B
- **流式响应**：支持 Server-Sent Events (SSE) 流式输出
- **工具调用**：OpenAI 和 Anthropic 格式均支持
- **深度思考**：返回模型的推理过程（`reasoning_content`）
- **联网搜索**：实时网络信息检索
- **多轮对话**：完整的对话历史管理
- **临时会话**：自动创建临时对话，不在元宝界面留记录
- **Web 管理面板**：可视化配置和测试界面
- **模型列表接口**：`/v1/models` 返回支持的模型信息

## 快速开始

### 前置要求

- Go 1.19 或更高版本
- Node.js 14+ （可选，仅用于辅助脚本）
- Docker （可选，用于容器化部署）
- 腾讯元宝账号（https://yuanbao.tencent.com）

### 安装步骤

#### 1. 获取 Cookie（仅需一次）

打开 https://yuanbao.tencent.com，登录后按 F12 打开浏览器控制台，粘贴运行：

```javascript
document.cookie
```

复制输出的完整 Cookie 字符串。

> Cookie 与你的元宝账号绑定，有效期通常为几天到几周，过期后重新提取即可。

#### 2. 配置环境变量

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

#### 3. 安装依赖并启动

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

#### 4. 测试

```bash
# 运行 Go 测试
go test ./...

# 或使用 Node.js 测试脚本
node test.js
```

### 验证安装

访问 http://localhost:3000 查看管理面板，或测试 API：

```bash
curl http://localhost:3000/v1/models
```

## 可用脚本

| 脚本 | 命令 | 说明 |
|------|------|------|
| 启动服务 | `go run .` | 开发模式，自动重启 |
| 构建二进制 | `go build -o main .` | 编译为可执行文件 |
| 运行测试 | `go test ./...` | 执行所有测试 |
| Docker 构建 | `docker build -t yuanbao2api .` | 构建 Docker 镜像 |
| Docker 运行 | `docker run -p 3000:3000 --env-file .env yuanbao2api` | 运行容器 |

## 开发工作流

### 项目架构

```
请求 → Gin 路由 → API 处理层 → 元宝 API 交互 → 响应格式转换 → 返回客户端
```

### 核心模块

1. **main.go**：应用入口，配置 Gin 服务器和路由
2. **api/openai.go**：OpenAI 兼容接口实现，处理请求转换和响应格式化
3. **config/config.go**：环境变量加载和配置管理
4. **public/index.html**：Web 管理面板，提供可视化测试界面

### 开发流程

1. 修改代码后，应用会自动重启（如使用 `go run .`）
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

### 工具调用

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
          "properties": {"city": {"type": "string"}},
          "required": ["city"]
        }
      }
    }]
  }'
```

### Python SDK 示例

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

### Anthropic SDK 示例

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

**最后更新**：2026-04-21

**项目状态**：活跃开发中

**主要贡献者**：utd-sakura
