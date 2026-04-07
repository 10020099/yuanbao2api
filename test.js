#!/usr/bin/env node

/**
 * 元宝2API 测试脚本
 * 用于测试 API 服务是否正常工作
 */

const http = require('http');

const API_BASE = process.env.API_BASE || 'http://localhost:3000';

// 测试配置
const tests = [
  {
    name: '健康检查',
    path: '/health',
    method: 'GET'
  },
  {
    name: '模型列表',
    path: '/v1/models',
    method: 'GET',
    validate: (result) => {
      if (!result.data.data || !Array.isArray(result.data.data)) {
        return '响应格式错误';
      }
      if (result.data.data.length === 0) {
        return '模型列表为空';
      }
      const models = result.data.data.map(m => m.id).join(', ');
      console.log(`  可用模型: ${models}`);
      return null;
    }
  },
  {
    name: '非流式聊天',
    path: '/v1/chat/completions',
    method: 'POST',
    body: {
      model: 'deep_seek_v3',
      messages: [
        { role: 'user', content: '你好，请用一句话介绍你自己' }
      ],
      stream: false
    },
    validate: (result) => {
      if (!result.data.choices || !result.data.choices[0]) {
        return '响应格式错误';
      }
      const content = result.data.choices[0].message.content;
      console.log(`  响应: ${content.substring(0, 50)}${content.length > 50 ? '...' : ''}`);
      return null;
    }
  },
  {
    name: 'Hunyuan 模型测试',
    path: '/v1/chat/completions',
    method: 'POST',
    body: {
      model: 'hunyuan',
      messages: [
        { role: 'user', content: '你好' }
      ],
      stream: false
    },
    validate: (result) => {
      if (!result.data.choices || !result.data.choices[0]) {
        return '响应格式错误';
      }
      const content = result.data.choices[0].message.content;
      console.log(`  响应: ${content.substring(0, 50)}${content.length > 50 ? '...' : ''}`);
      return null;
    }
  },
  {
    name: '多轮对话测试',
    path: '/v1/chat/completions',
    method: 'POST',
    body: {
      model: 'deep_seek_v3',
      messages: [
        { role: 'user', content: '我叫小明' },
        { role: 'assistant', content: '你好小明！很高兴认识你。' },
        { role: 'user', content: '我叫什么名字？' }
      ],
      stream: false
    },
    validate: (result) => {
      if (!result.data.choices || !result.data.choices[0]) {
        return '响应格式错误';
      }
      const content = result.data.choices[0].message.content;
      console.log(`  响应: ${content.substring(0, 80)}${content.length > 80 ? '...' : ''}`);

      // 检查是否包含"小明"，验证上下文是否生效
      if (content.includes('小明')) {
        console.log(`  ✓ 上下文保持正确`);
      } else {
        console.log(`  ⚠ 可能未正确理解上下文`);
      }
      return null;
    }
  }
];

// 发送 HTTP 请求
function request(options, body) {
  return new Promise((resolve, reject) => {
    const url = new URL(options.path, API_BASE);
    const req = http.request({
      hostname: url.hostname,
      port: url.port,
      path: url.pathname,
      method: options.method,
      headers: {
        'Content-Type': 'application/json'
      }
    }, (res) => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => {
        try {
          resolve({
            status: res.statusCode,
            data: JSON.parse(data)
          });
        } catch (e) {
          resolve({
            status: res.statusCode,
            data: data
          });
        }
      });
    });

    req.on('error', reject);

    if (body) {
      req.write(JSON.stringify(body));
    }

    req.end();
  });
}

// 运行测试
async function runTests() {
  console.log('🧪 元宝2API 测试开始\n');
  console.log(`API 地址: ${API_BASE}\n`);

  let passed = 0;
  let failed = 0;

  for (const test of tests) {
    process.stdout.write(`测试: ${test.name} ... `);

    try {
      const result = await request(test, test.body);

      if (result.status === 200) {
        // 运行自定义验证
        if (test.validate) {
          const error = test.validate(result);
          if (error) {
            console.log(`❌ 失败: ${error}`);
            failed++;
            console.log('');
            continue;
          }
        }

        console.log('✅ 通过');
        passed++;
      } else {
        console.log(`❌ 失败 (状态码: ${result.status})`);
        console.log(`  错误: ${JSON.stringify(result.data)}`);
        failed++;
      }
    } catch (error) {
      console.log(`❌ 失败`);
      console.log(`  错误: ${error.message}`);
      failed++;
    }

    console.log('');
  }

  console.log('='.repeat(50));
  console.log(`测试完成: ${passed} 通过, ${failed} 失败`);

  if (failed > 0) {
    console.log('\n💡 提示：');
    console.log('1. 确保服务已启动 (npm start)');
    console.log('2. 检查 .env 文件中的 YUANBAO_COOKIE 是否有效');
    console.log('3. 查看服务日志了解详细错误信息');
    process.exit(1);
  } else {
    console.log('\n🎉 所有测试通过！API 服务运行正常');
  }
}

// 运行
runTests().catch(error => {
  console.error('测试运行失败:', error);
  process.exit(1);
});
