// 在浏览器控制台运行此脚本来提取配置信息

(function() {
  console.log('=== 元宝2API 配置提取工具 ===\n');

  // 提取 Cookie
  const cookie = document.cookie;
  console.log('1. Cookie:');
  console.log(cookie);
  console.log('\n');

  // 提取 Agent ID
  const url = window.location.pathname;
  const agentId = url.split('/')[2] || 'naQivTmsDa';
  console.log('2. Agent ID:');
  console.log(agentId);
  console.log('\n');

  // 生成 .env 文件内容
  console.log('=== 复制以下内容到 .env 文件 ===\n');
  console.log(`YUANBAO_COOKIE="${cookie}"`);
  console.log(`YUANBAO_AGENT_ID="${agentId}"`);
  console.log(`PORT=3000`);
  console.log('\n=== 配置提取完成 ===');
  console.log('\n提示：使用临时对话模式，无需配置会话 ID！');
})();
