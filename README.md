# 元宝2API - Yuanbao to OpenAI API

将元宝网页版转换为 OpenAI 兼容的 API 接口。

## 功能特性

- OpenAI 兼容的 `/v1/chat/completions` 接口
- 支持流式和非流式响应
- 自动格式转换（OpenAI ↔ 元宝）
- 支持多个模型（DeepSeek V3.2、Hunyuan T1）
- 支持多轮对话（通过 messages 数组传递历史）
- 自动使用临时对话，无需手动管理会话 ID
- 完整的 `/v1/models` 接口
- 深度思考模式（`reasoning_content` 输出）
- 联网搜索功能
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
npm install
npm start
```

服务启动后，访问 http://localhost:3000 打开管理面板，可以：
- 查看服务状态
- 开启/关闭深度思考模式
- 开启/关闭联网搜索
- 选择默认模型
- 测试 API 功能

### 4. 测试

```bash
npm test
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

### 模型列表

```bash
curl http://localhost:3000/v1/models
```

## SDK 集成

### Python

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

# 多轮对话
messages = [
    {"role": "user", "content": "我叫小明"},
]
response = client.chat.completions.create(model="deep_seek_v3", messages=messages)
print(response.choices[0].message.content)

messages.append({"role": "assistant", "content": response.choices[0].message.content})
messages.append({"role": "user", "content": "我叫什么名字？"})
response = client.chat.completions.create(model="deep_seek_v3", messages=messages)
print(response.choices[0].message.content)
```

### Node.js

```javascript
const OpenAI = require('openai');

const client = new OpenAI({
  apiKey: 'dummy',
  baseURL: 'http://localhost:3000/v1'
});

const response = await client.chat.completions.create({
  model: 'deep_seek_v3',
  messages: [{ role: 'user', content: '你好' }]
});
console.log(response.choices[0].message.content);
```

## 工作原理

```
你的应用 → 调用 OpenAI 兼容 API → 元宝2API 服务 → 转发到元宝服务器 → 返回响应
```

- **Cookie**：证明你是已登录用户，提取一次可持续使用
- **会话 ID**：每次请求自动生成，无需手动管理
- **临时对话**：设置 `isTemporary: true`，不会在元宝界面留记录

### API 映射

| OpenAI | 元宝 |
|--------|------|
| `/v1/models` | 返回支持的模型列表 |
| `/v1/chat/completions` | `/api/chat/{conversationId}` |
| `messages[].content` | `prompt` |
| `stream: true` | SSE 流式响应 |
| `model` | `chatModelId` |

## 安全提示

- Cookie 是敏感信息，不要分享或提交到公开仓库
- `.gitignore` 已配置忽略 `.env` 文件

## 注意事项

- Cookie 过期后重新从浏览器提取
- 遵守元宝的使用限制
- 本项目仅用于技术研究和学习

## License

MIT
