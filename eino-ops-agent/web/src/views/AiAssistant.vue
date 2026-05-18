<template>
  <div class="ai-assistant">
    <div class="chat-container">
      <div class="chat-header">
        <div class="agent-status">
          <span class="status-indicator" :class="{ online: !store.thinking, thinking: store.thinking }"></span>
          <span>Eino Agent {{ store.thinking ? '思考中...' : '就绪' }}</span>
        </div>
        <div class="header-actions">
          <el-button text size="small" @click="showSettings = true">设置</el-button>
          <el-button text size="small" @click="store.clear()">清空对话</el-button>
        </div>
      </div>

      <div class="chat-messages" ref="msgContainer">
        <div v-if="store.messages.length === 0" class="welcome">
          <h3>Eino 智能运维 Agent</h3>
          <p>我可以帮你管理服务器、诊断故障、监控系统状态</p>
          <div class="quick-prompts">
            <span class="prompt-tag" @click="sendQuick('检查所有主机状态')">检查所有主机状态</span>
            <span class="prompt-tag" @click="sendQuick('查看最近的告警信息')">查看最近的告警信息</span>
            <span class="prompt-tag" @click="sendQuick('服务器CPU高了怎么办')">服务器CPU高了怎么办</span>
            <span class="prompt-tag" @click="sendQuick('运行 yum update -y')">运行 yum update -y</span>
          </div>
        </div>

        <div v-for="msg in store.messages" :key="msg.id" class="msg-row" :class="msg.role">
          <div class="msg-avatar">{{ msg.role === 'user' ? '👤' : '🤖' }}</div>
          <div class="msg-body">
            <div class="msg-meta">
              <span class="msg-role">{{ msg.role === 'user' ? '你' : 'Agent' }}</span>
              <span class="msg-time">{{ formatTime(msg.id) }}</span>
            </div>
            <div class="msg-content" v-html="renderContent(msg.content)"></div>
            <div v-if="msg.tool" class="msg-tool">
              <span class="tool-badge">工具: {{ msg.tool }}</span>
              <pre v-if="msg.toolOutput" class="tool-output">{{ msg.toolOutput }}</pre>
            </div>
          </div>
        </div>

        <div v-if="store.thinking" class="msg-row agent">
          <div class="msg-avatar">🤖</div>
          <div class="msg-body">
            <div class="thinking-dots">
              <span></span><span></span><span></span>
            </div>
            <span v-if="store.currentTool" class="tool-active">{{ store.currentTool }}</span>
          </div>
        </div>
      </div>

      <!-- Approval bar -->
      <div v-if="pendingApproval" class="approval-bar">
        <span class="approval-icon">⚠️</span>
        <span class="approval-text">检测到高危操作，需要人工确认后才能执行</span>
        <el-button type="danger" size="small" @click="approve" :loading="store.thinking">
          确认执行
        </el-button>
        <el-button size="small" @click="cancelApproval" :disabled="store.thinking">
          取消
        </el-button>
      </div>

      <div class="chat-input-area">
        <el-input
          v-model="inputMsg"
          type="textarea"
          :rows="2"
          placeholder="输入运维指令，Agent 将自动规划并执行..."
          @keydown.enter.exact="send"
          :disabled="store.thinking"
          resize="none"
        />
        <el-button
          type="primary"
          @click="send"
          :loading="store.thinking"
          :disabled="!inputMsg.trim()"
          class="send-btn"
        >
          {{ store.thinking ? '处理中' : '发送' }}
        </el-button>
      </div>
    </div>

    <!-- LLM Settings Dialog -->
    <el-dialog v-model="showSettings" title="LLM 模型设置" width="480px">
      <el-form :model="llmSettings" label-width="100px">
        <el-alert
          title="如不填写则使用服务端默认配置"
          type="info"
          :closable="false"
          show-icon
          style="margin-bottom: 16px"
        />
        <el-form-item label="API Key">
          <el-input v-model="llmSettings.api_key" type="password" show-password placeholder="sk-..." />
          <div class="form-hint">支持 OpenAI / DeepSeek 等兼容 API 的 Key</div>
        </el-form-item>
        <el-form-item label="Base URL">
          <el-input v-model="llmSettings.base_url" placeholder="https://api.deepseek.com/v1" />
          <div class="form-hint">API 地址，留空则使用默认</div>
        </el-form-item>
        <el-form-item label="模型名称">
          <el-input v-model="llmSettings.model" placeholder="deepseek-chat" />
          <div class="form-hint">如 deepseek-chat / gpt-4o / qwen-plus 等</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showSettings = false">取消</el-button>
        <el-button type="primary" @click="saveSettings">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, nextTick, onUnmounted } from 'vue'
import { chatStore as store } from '../stores/chat.js'

const inputMsg = ref('')
const msgContainer = ref(null)
let reader = null
let abortCtrl = null
const pendingApproval = ref(false)
let lastMessage = ''

const showSettings = ref(false)
const llmSettings = ref({
  api_key: localStorage.getItem('llm_api_key') || '',
  base_url: localStorage.getItem('llm_base_url') || '',
  model: localStorage.getItem('llm_model') || '',
})

const saveSettings = () => {
  localStorage.setItem('llm_api_key', llmSettings.value.api_key)
  localStorage.setItem('llm_base_url', llmSettings.value.base_url)
  localStorage.setItem('llm_model', llmSettings.value.model)
  showSettings.value = false
  store.addMessage('system', 'LLM 设置已保存')
}

const formatTime = (ts) => {
  const d = new Date(ts)
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

const renderContent = (text) => {
  if (!text) return ''
  // Process code blocks first (protect from other rules)
  const codeBlocks = []
  let html = text.replace(/```(\w*)\n([\s\S]*?)```/g, (_, lang, code) => {
    codeBlocks.push(`<pre class="code-block"><code>${escHtml(code)}</code></pre>`)
    return `%%CODEBLOCK_${codeBlocks.length - 1}%%`
  })

  // Inline code
  html = html.replace(/`([^`]+)`/g, '<code class="inline-code">$1</code>')

  // Bold and italic
  html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
  html = html.replace(/\*(.+?)\*/g, '<em>$1</em>')

  // Headings
  html = html.replace(/^### (.+)$/gm, '<h4 class="md-heading">$1</h4>')
  html = html.replace(/^## (.+)$/gm, '<h3 class="md-heading">$1</h3>')
  html = html.replace(/^# (.+)$/gm, '<h3 class="md-heading">$1</h3>')

  // Horizontal rules
  html = html.replace(/^---$/gm, '<hr class="md-hr">')

  // Tables — convert markdown table to HTML
  html = html.replace(/(\|[^\n]+\|\n\|[-:\|\s]+\|\n((?:\|[^\n]+\|\n?)*))/gm, (match) => {
    const lines = match.trim().split('\n')
    if (lines.length < 2) return match
    const headers = lines[0].split('|').filter(c => c.trim()).map(c => `<th>${c.trim()}</th>`).join('')
    const bodyLines = lines.slice(2).map(line => {
      const cells = line.split('|').filter(c => c.trim()).map(c => `<td>${c.trim()}</td>`).join('')
      return `<tr>${cells}</tr>`
    }).join('')
    return `<table class="md-table"><thead><tr>${headers}</tr></thead><tbody>${bodyLines}</tbody></table>`
  })

  // Numbered lists — group consecutive numbered lines
  html = html.replace(/((?:^\d+\.\s+.+\n?)+)/gm, (match) => {
    const items = match.trim().split('\n').map(line =>
      line.replace(/^\d+\.\s+(.+)$/, '<li>$1</li>')
    ).join('')
    return `<ol class="md-list">${items}</ol>`
  })

  // Unordered lists — group consecutive - or * lines
  html = html.replace(/((?:^[-*]\s+.+\n?)+)/gm, (match) => {
    const items = match.trim().split('\n').map(line =>
      line.replace(/^[-*]\s+(.+)$/, '<li>$1</li>')
    ).join('')
    return `<ul class="md-list">${items}</ul>`
  })

  // Blockquotes
  html = html.replace(/^>\s?(.+)$/gm, '<blockquote class="md-quote">$1</blockquote>')

  // Merge consecutive blockquotes
  html = html.replace(/<\/blockquote>\n<blockquote class="md-quote">/g, '\n')

  // Double newlines → paragraph breaks
  html = html.replace(/\n\n/g, '</p><p>')
  // Single newlines → <br>
  html = html.replace(/\n/g, '<br>')

  // Wrap in paragraph if not already
  if (!html.startsWith('<')) {
    html = '<p>' + html
  }
  if (!html.endsWith('>') || html.endsWith('</br>')) {
    html = html + '</p>'
  }

  // Restore code blocks
  html = html.replace(/%%CODEBLOCK_(\d+)%%/g, (_, i) => codeBlocks[parseInt(i)] || '')

  // Clean up empty paragraphs
  html = html.replace(/<p><\/p>/g, '')
  html = html.replace(/<p><br><\/p>/g, '<br>')

  return html
}

const escHtml = (s) => s
  .replace(/&/g, '&amp;')
  .replace(/</g, '&lt;')
  .replace(/>/g, '&gt;')
  .replace(/"/g, '&quot;')

const scrollBottom = async () => {
  await nextTick()
  if (msgContainer.value) {
    msgContainer.value.scrollTop = msgContainer.value.scrollHeight
  }
}

const sendQuick = (text) => {
  inputMsg.value = text
  send()
}

const send = async (approved = false) => {
  const text = inputMsg.value.trim() || lastMessage
  if (!text || store.thinking) return

  if (!approved) {
    store.addMessage('user', text)
    lastMessage = text
    inputMsg.value = ''
  }
  store.setThinking(true)
  pendingApproval.value = false
  await scrollBottom()

  store.addMessage('agent', '')
  const agentMsg = store.messages[store.messages.length - 1]

  abortCtrl = new AbortController()

  try {
    const headers = { 'Content-Type': 'application/json' }
    const token = localStorage.getItem('token')
    if (token) headers['Authorization'] = `Bearer ${token}`

    const resp = await fetch('/api/agent/chat', {
      method: 'POST',
      headers,
      body: JSON.stringify({
        message: text,
        thread_id: store.threadId,
        approved: approved,
        llm_config: llmSettings.value.api_key ? {
          api_key: llmSettings.value.api_key,
          base_url: llmSettings.value.base_url || undefined,
          model: llmSettings.value.model || undefined
        } : undefined
      }),
      signal: abortCtrl.signal
    })

    reader = resp.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (line.startsWith('event: ')) {
          pendingEvent = line.slice(7)
          continue
        }
        if (!line.startsWith('data: ')) continue
        const dataStr = line.slice(6)
        try {
          const parsed = JSON.parse(dataStr)
          handleSSE(parsed, agentMsg, pendingEvent)
          pendingEvent = ''
        } catch {
          // skip
        }
      }
    }
  } catch (err) {
    if (err.name !== 'AbortError') {
      agentMsg.content = '请求失败: ' + err.message
    }
  } finally {
    store.setThinking(false)
    store.setCurrentTool('')
    reader = null
    abortCtrl = null
    await scrollBottom()
  }
}

let pendingEvent = ''

const handleSSE = (parsed, agentMsg, evType) => {
  switch (evType) {
    case 'tool_start':
      store.setCurrentTool(parsed.tool)
      break
    case 'tool_end':
      agentMsg.tool = parsed.tool
      agentMsg.toolOutput = parsed.output?.substring(0, 500)
      store.setCurrentTool('')
      break
    case 'intent':
      agentMsg.content += '\n[意图] ' + (parsed.intent || '')
      break
    case 'plan':
      agentMsg.content = parsed.content || ''
      break
    case 'require_approval':
      pendingApproval.value = true
      agentMsg.content += '\n\n⚠️ 此操作涉及高危动作，需要人工确认。请点击下方的 [确认执行] 或 [取消]。'
      store.threadId = parsed.thread_id || store.threadId
      break
    case 'backup_info':
      agentMsg.content += `\n\n📦 版本快照已创建\n路径: ${parsed.path}\n${parsed.summary || ''}`
      break
    case 'done':
      agentMsg.content = parsed.report || parsed.content || ''
      store.threadId = parsed.thread_id || store.threadId
      break
    case 'error':
      agentMsg.content = '错误: ' + (parsed.message || '')
      break
    case 'stream_end':
      store.threadId = parsed.thread_id || store.threadId
      break
    default:
      if (parsed.content) {
        agentMsg.content += parsed.content
      }
  }
  if (parsed.report) {
    agentMsg.content = parsed.report
  }
  if (parsed.thread_id) {
    store.threadId = parsed.thread_id
  }
  scrollBottom()
}

const approve = () => {
  send(true)
}

const cancelApproval = () => {
  pendingApproval.value = false
  store.addMessage('system', '已取消高危操作，如需继续请重新输入指令。')
}

onUnmounted(() => {
  if (abortCtrl) abortCtrl.abort()
})
</script>

<style scoped>
.ai-assistant {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.chat-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #ffffff;
  border: 1px solid #e5e5e5;
  border-radius: 8px;
  overflow: hidden;
  height: calc(100vh - 140px);
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 20px;
  border-bottom: 1px solid #f0f0f0;
  flex-shrink: 0;
}

.agent-status {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #333333;
  font-size: 14px;
}

.status-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-indicator.online {
  background: #22c55e;
}

.status-indicator.thinking {
  background: #f59e0b;
  animation: pulse 1.2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  background: #fafafa;
}

.welcome {
  text-align: center;
  padding: 80px 20px 60px;
}

.welcome h3 {
  color: #111111;
  font-size: 22px;
  margin: 0 0 8px 0;
}

.welcome p {
  color: #888888;
  margin: 0 0 28px 0;
}

.quick-prompts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
}

.prompt-tag {
  padding: 6px 16px;
  background: #ffffff;
  border: 1px solid #d0d0d0;
  border-radius: 20px;
  color: #333333;
  cursor: pointer;
  font-size: 13px;
  transition: all 0.2s;
}

.prompt-tag:hover {
  background: #111111;
  border-color: #111111;
  color: #ffffff;
}

.msg-row {
  display: flex;
  gap: 10px;
  margin-bottom: 18px;
}

.msg-row.user {
  flex-direction: row-reverse;
}

.msg-avatar {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: #f0f0f0;
  font-size: 16px;
  flex-shrink: 0;
}

.msg-row.user .msg-avatar {
  background: #111111;
}

.msg-body {
  max-width: 72%;
}

.msg-row.user .msg-body {
  text-align: right;
}

.msg-meta {
  display: flex;
  gap: 8px;
  margin-bottom: 4px;
  font-size: 12px;
}

.msg-row.user .msg-meta {
  justify-content: flex-end;
}

.msg-role {
  color: #555555;
  font-weight: 600;
}

.msg-time {
  color: #bbbbbb;
}

.msg-content {
  color: #333333;
  line-height: 1.7;
  word-break: break-word;
}

.msg-row.user .msg-content {
  background: #111111;
  color: #ffffff;
  border-radius: 12px 4px 12px 12px;
  padding: 10px 16px;
}

.msg-row.agent .msg-content {
  background: #ffffff;
  border: 1px solid #e5e5e5;
  border-radius: 4px 12px 12px 12px;
  padding: 10px 16px;
}

.msg-content :deep(.code-block) {
  background: #f5f5f5;
  border: 1px solid #e5e5e5;
  border-radius: 6px;
  padding: 12px;
  overflow-x: auto;
  font-family: 'SF Mono', 'Cascadia Code', 'Consolas', monospace;
  font-size: 13px;
  color: #333333;
  margin: 8px 0;
}

.msg-content :deep(.inline-code) {
  background: #f5f5f5;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 13px;
  color: #111111;
}

.msg-content :deep(.md-heading) {
  margin: 16px 0 8px;
  color: #222;
}

.msg-content :deep(.md-hr) {
  border: none;
  border-top: 1px solid #e5e5e5;
  margin: 12px 0;
}

.msg-content :deep(.md-list) {
  margin: 8px 0;
  padding-left: 24px;
}

.msg-content :deep(.md-list li) {
  margin-bottom: 4px;
  line-height: 1.7;
}

.msg-content :deep(.md-quote) {
  border-left: 3px solid #d0d0d0;
  padding-left: 12px;
  color: #777;
  margin: 8px 0;
}

.msg-content :deep(.md-table) {
  border-collapse: collapse;
  width: 100%;
  margin: 8px 0;
  font-size: 13px;
}

.msg-content :deep(.md-table th) {
  background: #f5f5f5;
  border: 1px solid #e5e5e5;
  padding: 6px 10px;
  text-align: left;
  font-weight: 600;
}

.msg-content :deep(.md-table td) {
  border: 1px solid #e5e5e5;
  padding: 6px 10px;
}

.msg-tool {
  margin-top: 8px;
  padding: 8px 12px;
  background: #f8f8f8;
  border: 1px solid #e5e5e5;
  border-radius: 6px;
}

.tool-badge {
  color: #555555;
  font-size: 12px;
  font-weight: 600;
}

.tool-output {
  margin: 6px 0 0;
  padding: 8px;
  background: #f0f0f0;
  border-radius: 4px;
  color: #555555;
  font-size: 12px;
  max-height: 200px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
}

.thinking-dots {
  display: inline-flex;
  gap: 4px;
  margin-right: 8px;
}

.thinking-dots span {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #999999;
  animation: dotBounce 1.4s infinite ease-in-out both;
}

.thinking-dots span:nth-child(1) { animation-delay: -0.32s; }
.thinking-dots span:nth-child(2) { animation-delay: -0.16s; }
.thinking-dots span:nth-child(3) { animation-delay: 0s; }

@keyframes dotBounce {
  0%, 80%, 100% { transform: scale(0.6); }
  40% { transform: scale(1); }
}

.tool-active {
  color: #888888;
  font-size: 13px;
}

.approval-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 20px;
  background: #fff3cd;
  border-top: 1px solid #ffc107;
  flex-shrink: 0;
}

.approval-icon {
  font-size: 18px;
}

.approval-text {
  flex: 1;
  color: #856404;
  font-size: 13px;
}

.chat-input-area {
  display: flex;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid #e5e5e5;
  flex-shrink: 0;
}

.chat-input-area :deep(.el-textarea__inner) {
  background: #ffffff;
  border-color: #d0d0d0;
  color: #333333;
  border-radius: 6px;
}

.chat-input-area :deep(.el-textarea__inner:focus) {
  border-color: #111111;
}

.send-btn {
  align-self: flex-end;
  min-width: 80px;
}

.form-hint {
  font-size: 11px;
  color: #aaaaaa;
  margin-top: 4px;
}
</style>
