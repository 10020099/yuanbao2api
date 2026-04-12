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

// 工具调用标记
const TOOL_CALL_START = '<｜tool▁calls_begin｜>';
const TOOL_CALL_END = '<｜tool▁calls_end｜>';

// 生成临时会话 ID
function generateConversationId() {
  return uuidv4();
}

// 根据 tools 定义生成系统提示词片段
function buildToolSystemPrompt(tools) {
  if (!tools || tools.length === 0) return '';

  const toolDescriptions = tools.map(tool => {
    const func = tool.function;
    const params = func.parameters
      ? JSON.stringify(func.parameters, null, 2)
      : '{}';
    return `### ${func.name}\n${func.description || ''}\n参数:\n\`\`\`json\n${params}\n\`\`\``;
  }).join('\n\n');

  return [
    '',
    '# 可用工具',
    '你可以调用以下工具来完成任务。当你需要调用工具时，必须严格按照以下格式输出，不要输出任何其他内容包裹此标记：',
    TOOL_CALL_START,
    '{"name": "函数名", "arguments": {"参数名": "参数值"}}',
    TOOL_CALL_END,
    '',
    '你可以同时调用多个工具，每个工具调用使用单独的标记对。',
    '如果你不需要调用任何工具，直接回复用户即可，不要输出标记。',
    '',
    '可用工具列表：',
    toolDescriptions,
    ''
  ].join('\n');
}

// 解析模型输出中的工具调用
function parseToolCalls(text) {
  const calls = [];
  const regex = new RegExp(
    escapeRegex(TOOL_CALL_START) + '\\s*([\\s\\S]*?)\\s*' + escapeRegex(TOOL_CALL_END),
    'g'
  );
  let match;
  while ((match = regex.exec(text)) !== null) {
    try {
      const parsed = JSON.parse(match[1].trim());
      calls.push({
        name: parsed.name,
        arguments: typeof parsed.arguments === 'string'
          ? parsed.arguments
          : JSON.stringify(parsed.arguments)
      });
    } catch {
      const raw = match[1].trim();
      calls.push({ name: 'unknown', arguments: raw });
    }
  }
  return calls;
}

// 从文本中移除工具调用标记，返回纯文本内容
function stripToolCalls(text) {
  return text.replace(
    new RegExp(
      escapeRegex(TOOL_CALL_START) + '[\\s\\S]*?' + escapeRegex(TOOL_CALL_END),
      'g'
    ),
    ''
  ).trim();
}

// 转义正则特殊字符
function escapeRegex(str) {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

// 为工具调用生成 OpenAI 格式的 tool_calls 数组
function formatToolCalls(calls, startIndex = 0) {
  return calls.map((call, i) => ({
    id: `call_${uuidv4().replace(/-/g, '').slice(0, 24)}`,
    type: 'function',
    function: {
      name: call.name,
      arguments: typeof call.arguments === 'string'
        ? call.arguments
        : JSON.stringify(call.arguments)
    },
    index: startIndex + i
  }));
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
      internet_search,
      tools,
      tool_choice
    } = req.body;

    // 请求没有显式传参时，使用服务端全局默认值
    const useDeepThinking = deep_thinking !== undefined ? deep_thinking : serverConfig.deepThinking;
    const useInternetSearch = internet_search !== undefined ? internet_search : serverConfig.internetSearch;

    // 生成工具系统提示词
    const toolSystemPrompt = buildToolSystemPrompt(tools);

    // 构建 prompt
    // 对于多轮对话，将历史消息格式化后一起发送
    let prompt = '';
    let systemInjected = false;

    // 如果有工具定义且没有 system 消息，注入工具提示词
    if (toolSystemPrompt && !messages.some(m => m.role === 'system')) {
      prompt = `[系统提示:${toolSystemPrompt}]\n\n`;
      systemInjected = true;
    }

    if (messages.length === 1 && messages[0].role === 'user') {
      // 单轮对话
      prompt += messages[0].content;
    } else {
      // 多轮对话：格式化所有历史消息
      for (let i = 0; i < messages.length; i++) {
        const msg = messages[i];

        if (msg.role === 'system') {
          // 系统消息放在最前面，附带工具提示词
          const toolPart = toolSystemPrompt ? `\n${toolSystemPrompt}` : '';
          prompt = `[系统提示: ${msg.content}${toolPart}]\n\n` + prompt;
          systemInjected = true;
        } else if (msg.role === 'tool') {
          // 工具结果消息
          const toolName = msg.name || 'unknown';
          prompt += `工具 ${toolName} 的执行结果:\n${msg.content}\n\n`;
        } else if (msg.role === 'assistant') {
          // 助手消息，可能包含工具调用
          if (msg.tool_calls && msg.tool_calls.length > 0) {
            const callDescriptions = msg.tool_calls.map(tc =>
              `调用工具 ${tc.function.name}，参数: ${tc.function.arguments}`
            ).join('\n');
            prompt += `助手: 我需要调用工具来完成任务。\n${callDescriptions}\n`;
          } else {
            prompt += `助手: ${msg.content}\n`;
          }
        } else if (msg.role === 'user') {
          prompt += `用户: ${msg.content}\n`;
        }
      }

      // 添加提示，让 AI 继续对话
      prompt += '\n请作为助手继续回复：';
    }

    // 如果工具提示词未注入（没有 system 消息且上面没走到），追加到末尾
    if (toolSystemPrompt && !systemInjected) {
      prompt += toolSystemPrompt;
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
      let textBuffer = '';  // 缓冲未发送的文本，用于检测工具调用标记
      let inToolCall = false;  // 是否正在工具调用标记内
      const reader = response.body;

      function flushTextBuffer() {
        // 发送缓冲的文本内容（排除工具调用标记部分）
        if (textBuffer && !inToolCall) {
          const delta = { content: textBuffer };
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
        textBuffer = '';
      }

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
          textBuffer += data.msg;

          if (tools) {
            // 检测工具调用标记的开始
            const startIdx = textBuffer.indexOf(TOOL_CALL_START);
            if (startIdx !== -1 && !inToolCall) {
              // 先把标记前的文本发出去
              const beforeTag = textBuffer.substring(0, startIdx);
              textBuffer = textBuffer.substring(startIdx);
              if (beforeTag) {
                const delta = { content: beforeTag };
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
              inToolCall = true;
              textBuffer = '';
            }

            // 检测工具调用标记的结束
            if (inToolCall) {
              const endIdx = fullText.indexOf(TOOL_CALL_END);
              if (endIdx !== -1) {
                inToolCall = false;
                // 标记之后的文本继续缓冲
              }
            }

            // 如果不在工具调用标记内，且有足够缓冲，安全地发送文本
            if (!inToolCall) {
              // 保留可能的不完整标记前缀
              const safeLen = textBuffer.length - TOOL_CALL_START.length;
              if (safeLen > 0) {
                const safeText = textBuffer.substring(0, safeLen);
                textBuffer = textBuffer.substring(safeLen);
                const delta = { content: safeText };
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
          } else {
            // 没有工具定义，直接发送
            flushTextBuffer();
          }
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

        // 流式结束后处理剩余内容
        const toolCalls = tools ? parseToolCalls(fullText) : [];
        const hasToolCalls = toolCalls.length > 0;

        if (hasToolCalls) {
          // 发送剩余的纯文本（标记已被过滤）
          const cleanText = stripToolCalls(fullText);
          // 流式中可能已经发送了部分文本，这里只发工具调用
          const formattedCalls = formatToolCalls(toolCalls);

          for (let i = 0; i < formattedCalls.length; i++) {
            res.write(`data: ${JSON.stringify({
              id: `chatcmpl-${Date.now()}`,
              object: 'chat.completion.chunk',
              created: Math.floor(Date.now() / 1000),
              model: model,
              choices: [{
                index: 0,
                delta: {
                  tool_calls: [{
                    index: i,
                    id: formattedCalls[i].id,
                    type: 'function',
                    function: {
                      name: formattedCalls[i].function.name,
                      arguments: formattedCalls[i].function.arguments
                    }
                  }]
                },
                finish_reason: null
              }]
            })}\n\n`);
          }
        } else if (textBuffer) {
          // 没有工具调用，发送剩余缓冲文本
          const delta = { content: textBuffer };
          if (isFirstTextChunk && isFirstThinkChunk) { delta.role = 'assistant'; }
          res.write(`data: ${JSON.stringify({
            id: `chatcmpl-${Date.now()}`,
            object: 'chat.completion.chunk',
            created: Math.floor(Date.now() / 1000),
            model: model,
            choices: [{ index: 0, delta, finish_reason: null }]
          })}\n\n`);
        }

        const finishReason = hasToolCalls ? 'tool_calls' : 'stop';
        res.write(`data: ${JSON.stringify({
          id: `chatcmpl-${Date.now()}`,
          object: 'chat.completion.chunk',
          created: Math.floor(Date.now() / 1000),
          model: model,
          choices: [{ index: 0, delta: {}, finish_reason: finishReason }]
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

      // 解析工具调用
      const toolCalls = tools ? parseToolCalls(fullText) : [];
      const hasToolCalls = toolCalls.length > 0;
      const cleanText = hasToolCalls ? stripToolCalls(fullText) : fullText;

      const openaiMessage = {
        role: 'assistant',
        content: hasToolCalls ? (cleanText || null) : fullText
      };

      if (hasToolCalls) {
        openaiMessage.tool_calls = formatToolCalls(toolCalls);
      }

      const openaiResponse = {
        id: `chatcmpl-${Date.now()}`,
        object: 'chat.completion',
        created: Math.floor(Date.now() / 1000),
        model: model,
        choices: [{
          index: 0,
          message: openaiMessage,
          finish_reason: hasToolCalls ? 'tool_calls' : 'stop'
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

// ============================================================
// Anthropic Messages API 兼容路由
// ============================================================

// 将 Anthropic messages 转换为内部 prompt 格式
function anthropicMessagesToPrompt(messages, tools) {
  let prompt = '';
  let systemInjected = false;
  const toolSystemPrompt = buildToolSystemPrompt(tools);

  for (const msg of messages) {
    if (msg.role === 'user') {
      // Anthropic user content 可以是 string 或 content block 数组
      const text = typeof msg.content === 'string'
        ? msg.content
        : msg.content.map(block => {
            if (block.type === 'text') return block.text;
            if (block.type === 'tool_result') {
              const resultContent = typeof block.content === 'string'
                ? block.content
                : Array.isArray(block.content)
                  ? block.content.map(c => c.text || '').join('\n')
                  : JSON.stringify(block.content);
              return `工具 ${block.tool_use_id} 的执行结果:\n${resultContent}`;
            }
            return '';
          }).join('\n');
      prompt += `用户: ${text}\n`;
    } else if (msg.role === 'assistant') {
      // Anthropic assistant content 可以是 string 或 content block 数组
      if (typeof msg.content === 'string') {
        prompt += `助手: ${msg.content}\n`;
      } else if (Array.isArray(msg.content)) {
        const parts = [];
        for (const block of msg.content) {
          if (block.type === 'text') {
            parts.push(block.text);
          } else if (block.type === 'tool_use') {
            parts.push(`调用工具 ${block.name}，参数: ${JSON.stringify(block.input)}`);
          }
        }
        prompt += `助手: ${parts.join('\n')}\n`;
      }
    }
  }

  prompt += '\n请作为助手继续回复：';

  return { prompt, toolSystemPrompt, systemInjected };
}

// 构建 Anthropic 系统提示词
function buildAnthropicSystem(system, toolSystemPrompt) {
  if (!system && !toolSystemPrompt) return undefined;
  const parts = [];
  if (system) parts.push(typeof system === 'string' ? system : system.map(b => b.text).join('\n'));
  if (toolSystemPrompt) parts.push(toolSystemPrompt.trim());
  return parts.join('\n\n');
}

// POST /v1/messages - Anthropic Messages API
app.post('/v1/messages', async (req, res) => {
  try {
    const {
      model: requestModel = 'deep_seek_v3',
      messages,
      max_tokens = 4096,
      stream = false,
      system,
      tools,
      tool_choice,
      temperature,
      top_p,
      ...rest
    } = req.body;

    if (!messages || !Array.isArray(messages) || messages.length === 0) {
      return res.status(400).json({
        type: 'error',
        error: { type: 'invalid_request_error', message: 'messages is required and must be a non-empty array' }
      });
    }

    // 转换消息格式
    const { prompt: rawPrompt, toolSystemPrompt, systemInjected } = anthropicMessagesToPrompt(messages, tools);

    // 处理系统提示词
    let prompt = '';
    const sysPart = buildAnthropicSystem(system, toolSystemPrompt);
    if (sysPart) {
      prompt = `[系统提示: ${sysPart}]\n\n`;
    }
    prompt += rawPrompt;

    // 获取模型配置
    const modelConfig = getModelConfig(requestModel);
    const conversationId = generateConversationId();
    const agentId = process.env.YUANBAO_AGENT_ID || 'naQivTmsDa';

    // 检测深度思考（Anthropic 扩展参数）
    const useDeepThinking = rest.thinking !== undefined || rest.deep_thinking === true;
    const useInternetSearch = rest.internet_search === true;

    let supportFunctions = [];
    if (useInternetSearch) {
      supportFunctions.push('openInternetSearch');
    } else {
      supportFunctions.push('closeInternetSearch');
    }

    let chatModelId = modelConfig.chatModelId;
    let modelId = modelConfig.chatModelId;
    let subModelId = '';

    if (useDeepThinking) {
      if (requestModel.includes('hunyuan')) {
        chatModelId = 'hunyuan_t1';
        modelId = 'hunyuan_t1';
        subModelId = 'hunyuan_t1';
      } else {
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

    console.log('=== Anthropic API -> 元宝请求 ===');
    console.log('Model:', requestModel, '-> chatModelId:', chatModelId);
    console.log('Tools:', tools ? tools.length : 0);
    console.log('Stream:', stream);

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

    const msgId = `msg_${uuidv4().replace(/-/g, '').slice(0, 24)}`;

    if (stream) {
      // Anthropic 流式格式
      res.setHeader('Content-Type', 'text/event-stream');
      res.setHeader('Cache-Control', 'no-cache');
      res.setHeader('Connection', 'keep-alive');

      let fullText = '';
      let thinkingText = '';
      let buffer = '';
      let textBuffer = '';
      let inToolCall = false;
      const reader = response.body;

      // 发送 message_start 事件
      res.write(`event: message_start\ndata: ${JSON.stringify({
        type: 'message_start',
        message: {
          id: msgId,
          type: 'message',
          role: 'assistant',
          content: [],
          model: requestModel,
          stop_reason: null,
          stop_sequence: null,
          usage: { input_tokens: 0, output_tokens: 0 }
        }
      })}\n\n`);

      // 发送 ping
      res.write('event: ping\ndata: {}\n\n');

      let textBlockStarted = false;
      let thinkingBlockStarted = false;
      let toolBlocksStarted = [];  // 跟踪已开始的 tool_use block 索引

      function flushTextAsContent() {
        if (textBuffer && !inToolCall) {
          if (!textBlockStarted) {
            res.write(`event: content_block_start\ndata: ${JSON.stringify({
              type: 'content_block_start',
              index: thinkingBlockStarted ? 1 : 0,
              content_block: { type: 'text', text: '' }
            })}\n\n`);
            textBlockStarted = true;
          }
          res.write(`event: content_block_delta\ndata: ${JSON.stringify({
            type: 'content_block_delta',
            index: thinkingBlockStarted ? 1 : 0,
            delta: { type: 'text_delta', text: textBuffer }
          })}\n\n`);
          textBuffer = '';
        }
      }

      function processLine(line) {
        const trimmed = line.trim();
        if (!trimmed.startsWith('data:')) return;
        const payload = trimmed.startsWith('data: ') ? trimmed.substring(6) : trimmed.substring(5);
        if (payload === '[DONE]') return;

        let data;
        try { data = JSON.parse(payload); } catch { return; }

        if (data.type === 'think' && data.content) {
          thinkingText += data.content;
          if (!thinkingBlockStarted) {
            res.write(`event: content_block_start\ndata: ${JSON.stringify({
              type: 'content_block_start',
              index: 0,
              content_block: { type: 'thinking', thinking: '' }
            })}\n\n`);
            thinkingBlockStarted = true;
          }
          res.write(`event: content_block_delta\ndata: ${JSON.stringify({
            type: 'content_block_delta',
            index: 0,
            delta: { type: 'thinking_delta', thinking: data.content }
          })}\n\n`);
        }

        if (data.type === 'text' && data.msg) {
          fullText += data.msg;
          textBuffer += data.msg;

          if (tools) {
            const startIdx = textBuffer.indexOf(TOOL_CALL_START);
            if (startIdx !== -1 && !inToolCall) {
              // 先发送标记前的文本
              const beforeTag = textBuffer.substring(0, startIdx);
              textBuffer = textBuffer.substring(startIdx);
              if (beforeTag) {
                if (!textBlockStarted) {
                  const blockIdx = thinkingBlockStarted ? 1 : 0;
                  res.write(`event: content_block_start\ndata: ${JSON.stringify({
                    type: 'content_block_start',
                    index: blockIdx,
                    content_block: { type: 'text', text: '' }
                  })}\n\n`);
                  textBlockStarted = true;
                }
                const blockIdx = thinkingBlockStarted ? 1 : 0;
                res.write(`event: content_block_delta\ndata: ${JSON.stringify({
                  type: 'content_block_delta',
                  index: blockIdx,
                  delta: { type: 'text_delta', text: beforeTag }
                })}\n\n`);
              }
              inToolCall = true;
              textBuffer = '';
            }

            if (inToolCall) {
              const endIdx = fullText.indexOf(TOOL_CALL_END);
              if (endIdx !== -1) {
                inToolCall = false;
              }
            }

            if (!inToolCall && !textBlockStarted) {
              const safeLen = textBuffer.length - TOOL_CALL_START.length;
              if (safeLen > 0) {
                const safeText = textBuffer.substring(0, safeLen);
                textBuffer = textBuffer.substring(safeLen);
                const blockIdx = thinkingBlockStarted ? 1 : 0;
                res.write(`event: content_block_start\ndata: ${JSON.stringify({
                  type: 'content_block_start',
                  index: blockIdx,
                  content_block: { type: 'text', text: '' }
                })}\n\n`);
                textBlockStarted = true;
                res.write(`event: content_block_delta\ndata: ${JSON.stringify({
                  type: 'content_block_delta',
                  index: blockIdx,
                  delta: { type: 'text_delta', text: safeText }
                })}\n\n`);
              }
            }
          } else {
            flushTextAsContent();
          }
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

        // 流式结束：处理工具调用
        const toolCalls = tools ? parseToolCalls(fullText) : [];
        const hasToolCalls = toolCalls.length > 0;

        // 计算下一个可用的 content block index
        let nextIndex = 0;
        if (thinkingBlockStarted) nextIndex++;
        if (textBlockStarted) nextIndex++;

        // 关闭未关闭的文本 block（在工具调用之前）
        if (textBlockStarted) {
          const blockIdx = thinkingBlockStarted ? 1 : 0;
          res.write(`event: content_block_stop\ndata: ${JSON.stringify({
            type: 'content_block_stop',
            index: blockIdx
          })}\n\n`);
          textBlockStarted = false;
        }

        if (hasToolCalls) {
          const formattedCalls = formatToolCalls(toolCalls);
          for (let i = 0; i < formattedCalls.length; i++) {
            const blockIdx = nextIndex + i;
            res.write(`event: content_block_start\ndata: ${JSON.stringify({
              type: 'content_block_start',
              index: blockIdx,
              content_block: {
                type: 'tool_use',
                id: formattedCalls[i].id,
                name: formattedCalls[i].function.name,
                input: {}
              }
            })}\n\n`);

            res.write(`event: content_block_delta\ndata: ${JSON.stringify({
              type: 'content_block_delta',
              index: blockIdx,
              delta: {
                type: 'input_json_delta',
                partial_json: formattedCalls[i].function.arguments
              }
            })}\n\n`);

            res.write(`event: content_block_stop\ndata: ${JSON.stringify({
              type: 'content_block_stop',
              index: blockIdx
            })}\n\n`);
          }
        }

        // 关闭 thinking block
        if (thinkingBlockStarted) {
          res.write(`event: content_block_stop\ndata: ${JSON.stringify({
            type: 'content_block_stop',
            index: 0
          })}\n\n`);
        }

        // 如果没有任何 block 被开始过，发一个文本 block
        if (!textBlockStarted && !thinkingBlockStarted && !hasToolCalls) {
          res.write(`event: content_block_start\ndata: ${JSON.stringify({
            type: 'content_block_start',
            index: 0,
            content_block: { type: 'text', text: '' }
          })}\n\n`);
          if (textBuffer) {
            res.write(`event: content_block_delta\ndata: ${JSON.stringify({
              type: 'content_block_delta',
              index: 0,
              delta: { type: 'text_delta', text: textBuffer }
            })}\n\n`);
          }
          res.write(`event: content_block_stop\ndata: ${JSON.stringify({
            type: 'content_block_stop',
            index: 0
          })}\n\n`);
        }

        // message_delta + message_stop
        const stopReason = hasToolCalls ? 'tool_use' : 'end_turn';
        res.write(`event: message_delta\ndata: ${JSON.stringify({
          type: 'message_delta',
          delta: { stop_reason: stopReason, stop_sequence: null },
          usage: { output_tokens: 0 }
        })}\n\n`);

        res.write(`event: message_stop\ndata: ${JSON.stringify({
          type: 'message_stop'
        })}\n\n`);

        res.end();
      });

    } else {
      // Anthropic 非流式响应
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

      // 解析工具调用
      const toolCalls = tools ? parseToolCalls(fullText) : [];
      const hasToolCalls = toolCalls.length > 0;
      const cleanText = hasToolCalls ? stripToolCalls(fullText) : fullText;

      // 构建 Anthropic 格式的 content blocks
      const content = [];

      if (thinkingText) {
        content.push({ type: 'thinking', thinking: thinkingText });
      }

      if (hasToolCalls) {
        if (cleanText) {
          content.push({ type: 'text', text: cleanText });
        }
        const formattedCalls = formatToolCalls(toolCalls);
        for (const tc of formattedCalls) {
          let input = {};
          try { input = JSON.parse(tc.function.arguments); } catch { input = {}; }
          content.push({
            type: 'tool_use',
            id: tc.id,
            name: tc.function.name,
            input: input
          });
        }
      } else {
        content.push({ type: 'text', text: fullText });
      }

      const anthropicResponse = {
        id: msgId,
        type: 'message',
        role: 'assistant',
        content: content,
        model: requestModel,
        stop_reason: hasToolCalls ? 'tool_use' : 'end_turn',
        stop_sequence: null,
        usage: {
          input_tokens: 0,
          output_tokens: 0
        }
      };

      res.json(anthropicResponse);
    }

  } catch (error) {
    console.error('Anthropic API Error:', error);
    res.status(500).json({
      type: 'error',
      error: { type: 'api_error', message: error.message }
    });
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
