<template>
  <div class="log-viewer">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>实时日志 - {{ taskId || "全部任务" }}</span>
          <div>
            <el-button type="success" @click="connectWS" :disabled="wsConnected"
              >重连</el-button
            >
            <el-button
              type="danger"
              @click="disconnectWS"
              :disabled="!wsConnected"
              >断开</el-button
            >
            <el-button @click="clearLog">清空</el-button>
          </div>
        </div>
      </template>
      <div class="log-container" ref="logContainer">
        <div v-for="(log, index) in logs" :key="index" class="log-line">
          <span class="log-time">{{ log.time }}</span>
          <span :class="['log-level', log.level]">{{
            log.level.toUpperCase()
          }}</span>
          <span class="log-content">{{ log.message }}</span>
        </div>
        <div v-if="logs.length === 0" class="no-log">暂无日志</div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from "vue";
import { useRoute } from "vue-router";
import { ElMessage } from "element-plus";

const route = useRoute();
const taskId = ref(route.query.taskId || "");
const logs = ref([]);
const wsConnected = ref(false);
const logContainer = ref(null);
const maxLogs = 1000;
let ws = null;
let reconnectAttempts = 0;
const maxReconnectAttempts = 5;
let reconnectTimer = null;

const connectWS = () => {
  const protocol = globalThis.location.protocol === "https:" ? "wss:" : "ws:";
  const host = globalThis.location.host;
  const url = `${protocol}//${host}/ws/log${taskId.value ? "?task_id=" + taskId.value : ""}`;

  ws = new WebSocket(url);

  ws.onopen = () => {
    wsConnected.value = true;
    reconnectAttempts = 0; // 重置重连计数
    ElMessage.success("WebSocket已连接");
  };

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      if (!taskId.value || !data.task_id || Number(data.task_id) === Number(taskId.value)) {
        logs.value.push(data);
      }
      // 限制日志数量，超出移除旧日志
      if (logs.value.length > maxLogs) {
        logs.value = logs.value.slice(-maxLogs);
      }
      nextTick(() => {
        if (logContainer.value) {
          logContainer.value.scrollTop = logContainer.value.scrollHeight;
        }
      });
    } catch {
      logs.value.push({
        time: new Date().toLocaleTimeString(),
        level: "info",
        message: event.data,
      });
    }
  };

  ws.onerror = () => {
    ElMessage.error("WebSocket连接错误");
  };

  ws.onclose = () => {
    wsConnected.value = false;
    ElMessage.warning("WebSocket已断开");
    // 自动重连（指数退避）
    if (reconnectAttempts < maxReconnectAttempts) {
      const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
      reconnectAttempts++;
      reconnectTimer = setTimeout(() => {
        connectWS();
      }, delay);
    }
  };
};

const disconnectWS = () => {
  if (ws) {
    ws.close();
    ws = null;
  }
};

const clearLog = () => {
  logs.value = [];
};

onMounted(() => {
  connectWS();
});

onUnmounted(() => {
  disconnectWS();
});
</script>

<style scoped>
.log-container {
  height: 500px;
  overflow-y: auto;
  background-color: #1e1e1e;
  color: #d4d4d4;
  padding: 10px;
  font-family: "Consolas", monospace;
  font-size: 13px;
  border-radius: 4px;
}

.log-line {
  display: flex;
  margin-bottom: 4px;
  line-height: 1.6;
}

.log-time {
  color: #888;
  margin-right: 10px;
  white-space: nowrap;
}

.log-level {
  width: 60px;
  margin-right: 10px;
  text-align: center;
  padding: 0 4px;
  border-radius: 3px;
  font-weight: bold;
}

.log-level.info {
  color: #569cd6;
}

.log-level.warn {
  color: #f59e0b;
}

.log-level.error {
  color: #ef4444;
}

.log-level.success {
  color: #22c55e;
}

.log-content {
  flex: 1;
  word-break: break-all;
}

.no-log {
  text-align: center;
  color: #666;
  padding-top: 200px;
}
</style>
