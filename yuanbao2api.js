const express = require('express');
const fetch = require('node-fetch');
const { v4: uuidv4 } = require('uuid');
const path = require('path');
require('dotenv').config();

const app = express();
app.use(express.json());

// 提供静态文件（管理面板）
app.use(express.static(path.join(__dirname, 'public')));

// 元宝 API 配置
const YUANBAO_CONFIG = {
  baseUrl: 'https://yuanbao.tencent.com',
  chatEndpoint: '/api/chat',
  // 需要从浏览器中提取的认证信息
  headers: {
    'x-device-id': '',
    'x-language': 'zh-CN',
    'x-requested-with': 'XMLHttpRequest',
    'content-type': 'text/plain;charset=UTF-8',
    'x-platform': 'win',
    'x-source': 'web',
    'x-webversion': '2.63.0',
    'user-agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
    'origin': 'https://yuanbao.tencent.com',
    'referer': 'https://yuanbao.tencent.com/chat'
  },
  cookies: ''
};

// 服务端全局默认配置（管理面板可修改）
const serverConfig = {
  deepThinking: false,
  internetSearch: false,
  defaultModel: 'deep_seek_v3'
};

// 会话管理（用于支持多轮对话）
const sessions = new Map();

// 生成临时会话 ID
function generateConversationId() {
  return uuidv4();
}

// 模型映射配置
const MODEL_MAPPING = {
  'DeepSeek-V3.2': {
    chatModelId: 'deep_seek_v3',
    model: 'gpt_175B_0404',
    name: 'DeepSeek V3.2',
    description: '适合深度思考和复杂推理任务'
  },
  'deep_seek_v3': {
    chatModelId: 'deep_seek_v3',
    model: 'gpt_175B_0404',
    name: 'DeepSeek V3.2',
    description: '适合深度思考和复杂推理任务'
  },
  'deepseek': {
    chatModelId: 'deep_seek_v3',
    model: 'gpt_175B_0404',
    name: 'DeepSeek V3.2',
    description: '适合深度思考和复杂推理任务'
  },
  'hunyuan-t1': {
    chatModelId: 'hunyuan',
    model: 'gpt_175B_0404',
    name: 'Hunyuan T1',
    description: '腾讯混元大模型，全能处理'
  },
  'hunyuan': {
    chatModelId: 'hunyuan',
    model: 'gpt_175B_0404',
    name: 'Hunyuan T1',
    description: '腾讯混元大模型，全能处理'
  },
  'gpt_175B_0404': {
    chatModelId: 'deep_seek_v3',
    model: 'gpt_175B_0404',
    name: 'GPT 175B',
    description: '元宝内部模型'
  }
};

// 获取模型配置
function getModelConfig(modelName) {
  const normalizedName = modelName.toLowerCase().replace(/[-_]/g, '');

  // 精确匹配
  if (MODEL_MAPPING[modelName]) {
    return MODEL_MAPPING[modelName];
  }

  // 模糊匹配
  for (const [key, config] of Object.entries(MODEL_MAPPING)) {
    if (key.replace(/[-_]/g, '') === normalizedName) {
      return config;
    }
  }

  // 默认使用 DeepSeek V3.2
  return MODEL_MAPPING['DeepSeek-V3.2'];
}

// 获取服务端配置
app.get('/api/config', (req, res) => {
  res.json(serverConfig);
});

// 更新服务端配置
app.post('/api/config', (req, res) => {
  const { deepThinking, internetSearch, defaultModel } = req.body;
  if (typeof deepThinking === 'boolean') serverConfig.deepThinking = deepThinking;
  if (typeof internetSearch === 'boolean') serverConfig.internetSearch = internetSearch;
  if (typeof defaultModel === 'string') serverConfig.defaultModel = defaultModel;
  console.log('配置已更新:', serverConfig);
  res.json(serverConfig);
});

// OpenAI 兼容的聊天接口
app.post('/v1/chat/completions', async (req, res) => {
  try {
    const {
      messages,
      stream = false,
      model = serverConfig.defaultModel,
      deep_thinking,
      internet_search
    } = req.body;

    // 请求没有显式传参时，使用服务端全局默认值
    const useDeepThinking = deep_thinking !== undefined ? deep_thinking : serverConfig.deepThinking;
    const useInternetSearch = internet_search !== undefined ? internet_search : serverConfig.internetSearch;

    // 构建 prompt
    // 对于多轮对话，将历史消息格式化后一起发送
    let prompt = '';

    if (messages.length === 1) {
      // 单轮对话
      prompt = messages[0].content;
    } else {
      // 多轮对话：格式化所有历史消息
      // 格式：用户: xxx\n助手: xxx\n用户: xxx
      for (let i = 0; i < messages.length; i++) {
        const msg = messages[i];

        if (msg.role === 'system') {
          // 系统消息放在最前面
          prompt = `[系统提示: ${msg.content}]\n\n` + prompt;
        } else if (msg.role === 'user') {
          prompt += `用户: ${msg.content}\n`;
        } else if (msg.role === 'assistant') {
          prompt += `助手: ${msg.content}\n`;
        }
      }

      // 添加提示，让 AI 继续对话
      prompt += '\n请作为助手继续回复：';
    }

    // 生成或获取会话 ID（每次请求使用新的临时会话）
    const conversationId = generateConversationId();
    const agentId = process.env.YUANBAO_AGENT_ID || 'naQivTmsDa';

    // 获取模型配置
    const modelConfig = getModelConfig(model);

    // 根据功能配置设置 supportFunctions
    let supportFunctions = [];
    if (useInternetSearch) {
      supportFunctions.push('openInternetSearch');
    } else {
      supportFunctions.push('closeInternetSearch');
    }

    // 构建元宝 API 请求
    // 根据模型和深度思考配置选择正确的模型ID
    let chatModelId = modelConfig.chatModelId;
    let modelId = modelConfig.chatModelId;
    let subModelId = '';

    if (useDeepThinking) {
      // 深度思考模式：根据不同模型使用不同配置
      if (model.includes('hunyuan')) {
        // Hunyuan模型的深度思考
        chatModelId = 'hunyuan_t1';
        modelId = 'hunyuan_t1';
        subModelId = 'hunyuan_t1';
      } else {
        // DeepSeek模型的深度思考
        // 注意：chatModelId 使用 deep_seek，而 modelId 和 subModelId 使用 deep_seek_v3
        chatModelId = 'deep_seek';
        modelId = 'deep_seek_v3';
        subModelId = 'deep_seek';
      }
    }

    const yuanbaoRequest = {
      model: modelConfig.model,
      prompt: prompt,
      plugin: useDeepThinking ? '' : 'Adaptive',
      displayPrompt: prompt,
      displayPromptType: 1,
      agentId: agentId,
      isTemporary: true,
      projectId: '',
      chatModelId: chatModelId,
      supportFunctions: supportFunctions,
      docOpenid: '',
      options: {
        imageIntention: {
          needIntentionModel: true,
          backendUpdateFlag: 2,
          intentionStatus: true
        }
      },
      multimedia: [],
      supportHint: 1,
      chatModelExtInfo: JSON.stringify({
        modelId: modelId,
        subModelId: subModelId,
        supportFunctions: {
          internetSearch: useInternetSearch ? 'openInternetSearch' : 'closeInternetSearch'
        }
      }),
      applicationIdList: [],
      version: 'v2',
      extReportParams: null,
      isAtomInput: false,
      offsetOfHour: 8,
      offsetOfMinute: 0
    };

    const yuanbaoUrl = `${YUANBAO_CONFIG.baseUrl}${YUANBAO_CONFIG.chatEndpoint}/${conversationId}`;

    // 调试日志：打印发送给元宝的请求
    console.log('=== 发送给元宝的请求 ===');
    console.log('URL:', yuanbaoUrl);
    console.log('deep_thinking:', useDeepThinking);
    console.log('chatModelId:', chatModelId);
    console.log('modelId:', modelId);
    console.log('subModelId:', subModelId);
    console.log('Request Body:', JSON.stringify(yuanbaoRequest, null, 2));
    console.log('========================');

    // 发送请求到元宝 API
    const response = await fetch(yuanbaoUrl, {
      method: 'POST',
      headers: {
        ...YUANBAO_CONFIG.headers,
        'cookie': process.env.YUANBAO_COOKIE || YUANBAO_CONFIG.cookies,
        'x-agentid': `${agentId}/${conversationId}`,
        'x-timestamp': Date.now().toString(),
        'referer': `https://yuanbao.tencent.com/chat/${agentId}/${conversationId}`
      },
      body: JSON.stringify(yuanbaoRequest)
    });

    if (!response.ok) {
      throw new Error(`Yuanbao API error: ${response.status}`);
    }

    if (stream) {
      // 流式响应
      res.setHeader('Content-Type', 'text/event-stream');
      res.setHeader('Cache-Control', 'no-cache');
      res.setHeader('Connection', 'keep-alive');

      let fullText = '';
      let thinkingText = '';
      let isFirstThinkChunk = true;
      let isFirstTextChunk = true;
      let buffer = '';
      const reader = response.body;

      function processLine(line) {
        const trimmed = line.trim();
        if (!trimmed.startsWith('data:')) return;
        const payload = trimmed.startsWith('data: ') ? trimmed.substring(6) : trimmed.substring(5);
        if (payload === '[DONE]') return;

        let data;
        try { data = JSON.parse(payload); } catch { return; }

        if (data.type === 'think' && data.content) {
          thinkingText += data.content;
          const delta = { reasoning_content: data.content };
          if (isFirstThinkChunk) { delta.role = 'assistant'; isFirstThinkChunk = false; }
          res.write(`data: ${JSON.stringify({
            id: `chatcmpl-${Date.now()}`,
            object: 'chat.completion.chunk',
            created: Math.floor(Date.now() / 1000),
            model: model,
            choices: [{ index: 0, delta, finish_reason: null }]
          })}\n\n`);
        }

        if (data.type === 'text' && data.msg) {
          fullText += data.msg;
          const delta = { content: data.msg };
          if (isFirstTextChunk && isFirstThinkChunk) { delta.role = 'assistant'; }
          isFirstTextChunk = false;
          res.write(`data: ${JSON.stringify({
            id: `chatcmpl-${Date.now()}`,
            object: 'chat.completion.chunk',
            created: Math.floor(Date.now() / 1000),
            model: model,
            choices: [{ index: 0, delta, finish_reason: null }]
          })}\n\n`);
        }
      }

      reader.on('data', (chunk) => {
        buffer += chunk.toString();
        const parts = buffer.split('\n');
        buffer = parts.pop();
        for (const line of parts) {
          processLine(line);
        }
      });

      reader.on('end', () => {
        if (buffer.trim()) processLine(buffer);

        res.write(`data: ${JSON.stringify({
          id: `chatcmpl-${Date.now()}`,
          object: 'chat.completion.chunk',
          created: Math.floor(Date.now() / 1000),
          model: model,
          choices: [{ index: 0, delta: {}, finish_reason: 'stop' }]
        })}\n\n`);
        res.write('data: [DONE]\n\n');
        res.end();
      });

    } else {
      // 非流式响应
      const text = await response.text();
      const lines = text.split('\n');
      let fullText = '';
      let thinkingText = '';

      for (const rawLine of lines) {
        const line = rawLine.trim();
        if (!line.startsWith('data:')) continue;
        const payload = line.startsWith('data: ') ? line.substring(6) : line.substring(5);
        if (payload === '[DONE]') continue;

        let data;
        try { data = JSON.parse(payload); } catch { continue; }

        if (data.type === 'think' && data.content) {
          thinkingText += data.content;
        }
        if (data.type === 'text' && data.msg) {
          fullText += data.msg;
        }
      }

      const openaiResponse = {
        id: `chatcmpl-${Date.now()}`,
        object: 'chat.completion',
        created: Math.floor(Date.now() / 1000),
        model: model,
        choices: [{
          index: 0,
          message: {
            role: 'assistant',
            content: fullText
          },
          finish_reason: 'stop'
        }],
        usage: {
          prompt_tokens: 0,
          completion_tokens: 0,
          total_tokens: 0
        }
      };

      if (thinkingText) {
        openaiResponse.choices[0].message.reasoning_content = thinkingText;
      }

      res.json(openaiResponse);
    }

  } catch (error) {
    console.error('Error:', error);
    res.status(500).json({ error: error.message });
  }
});

// 模型列表接口
app.get('/v1/models', (req, res) => {
  const models = [
    {
      id: 'DeepSeek-V3.2',
      object: 'model',
      created: 1704067200,
      owned_by: 'yuanbao',
      permission: [],
      root: 'DeepSeek-V3.2',
      parent: null,
      description: 'DeepSeek V3.2 - 适合深度思考和复杂推理任务'
    },
    {
      id: 'hunyuan-t1',
      object: 'model',
      created: 1704067200,
      owned_by: 'yuanbao',
      permission: [],
      root: 'hunyuan-t1',
      parent: null,
      description: 'Hunyuan T1 - 腾讯混元大模型，全能处理'
    },
    {
      id: 'gpt_175B_0404',
      object: 'model',
      created: 1704067200,
      owned_by: 'yuanbao',
      permission: [],
      root: 'gpt_175B_0404',
      parent: null,
      description: '元宝内部模型标识'
    }
  ];

  res.json({
    object: 'list',
    data: models
  });
});

// 健康检查
app.get('/health', (req, res) => {
  res.json({ status: 'ok' });
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`Yuanbao2API server running on port ${PORT}`);
  console.log(`\n📊 管理面板: http://localhost:${PORT}`);
  console.log(`\n配置说明：`);
  console.log(`1. 设置环境变量 YUANBAO_COOKIE（从浏览器复制）`);
  console.log(`2. 可选：设置环境变量 YUANBAO_AGENT_ID（默认: naQivTmsDa）`);
  console.log(`\n✨ 使用临时对话模式，每次请求自动创建新会话`);
  console.log(`\n功能特性：`);
  console.log(`- 深度思考模式（deep_thinking: true）`);
  console.log(`- 联网搜索（internet_search: true）`);
  console.log(`\n使用示例：`);
  console.log(`curl http://localhost:${PORT}/v1/chat/completions \\`);
  console.log(`  -H "Content-Type: application/json" \\`);
  console.log(`  -d '{"model":"deep_seek_v3","messages":[{"role":"user","content":"你好"}],"deep_thinking":true}'`);
});
