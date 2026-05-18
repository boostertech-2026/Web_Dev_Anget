import { reactive } from 'vue'

export const chatStore = reactive({
  messages: [],
  threadId: '',
  thinking: false,
  currentTool: '',

  addMessage(role, content, extra = {}) {
    this.messages.push({ id: Date.now(), role, content, ...extra })
  },

  setThinking(val) {
    this.thinking = val
  },

  setCurrentTool(name) {
    this.currentTool = name
  },

  clear() {
    this.messages = []
    this.threadId = ''
    this.thinking = false
    this.currentTool = ''
  }
})
