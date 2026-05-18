<template>
  <div class="task-log-detail">
    <div class="page-header">
      <div class="header-left">
        <span class="icon">📊</span>
        <div class="header-text">
          <h2>任务日志详情</h2>
          <p>Task Log Detail</p>
        </div>
      </div>
      <el-button class="back-btn" @click="$router.back()">
        <span>← 返回</span>
      </el-button>
    </div>

    <el-card v-if="task" class="info-card">
      <el-descriptions :column="2" border class="tech-descriptions">
        <el-descriptions-item label="任务ID">{{ task.id }}</el-descriptions-item>
        <el-descriptions-item label="任务名称">{{ task.name }}</el-descriptions-item>
        <el-descriptions-item label="执行方式">{{ task.exec_type === 'ssh' ? 'SSH' : '客户端' }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="getStatusType(task.status)">{{ getStatusText(task.status) }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="创建时间">{{ task.created_at }}</el-descriptions-item>
        <el-descriptions-item label="结果">{{ task.result || '-' }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card class="log-card">
      <template #header>
        <span class="log-card-title">执行日志</span>
      </template>
      <pre class="log-content">{{ task ? task.logs || '暂无日志' : '加载中...' }}</pre>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getTasks } from '../api'

const route = useRoute()
const task = ref(null)

const getStatusType = (status) => {
  const map = { pending: 'info', running: 'primary', success: 'success', failed: 'danger' }
  return map[status] || 'info'
}

const getStatusText = (status) => {
  const map = { pending: '待执行', running: '执行中', success: '成功', failed: '失败' }
  return map[status] || status
}

const loadTaskDetail = async () => {
  try {
    const res = await getTasks()
    const tasks = res.data || []
    const taskId = parseInt(route.params.id)
    task.value = tasks.find(t => t.id === taskId)
    if (!task.value) {
      ElMessage.error('任务不存在')
    }
  } catch {
    ElMessage.error('加载任务详情失败')
  }
}

onMounted(() => {
  loadTaskDetail()
})
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 14px;
}

.header-left .icon {
  font-size: 32px;
}

.header-text h2 {
  margin: 0;
  color: #111111;
  font-size: 20px;
  font-weight: 600;
}

.header-text p {
  margin: 0;
  color: #888888;
  font-size: 12px;
}

.info-card {
  background: #ffffff;
  border: 1px solid #e5e5e5;
  border-radius: 8px;
  margin-bottom: 20px;
}

.log-card {
  background: #ffffff;
  border: 1px solid #e5e5e5;
  border-radius: 8px;
}

.log-card-title {
  color: #111111;
  font-weight: 600;
}

.log-content {
  font-family: 'Consolas', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-wrap: break-word;
  margin: 0;
  padding: 16px;
  background: #f5f5f5;
  border: 1px solid #e5e5e5;
  border-radius: 8px;
  color: #333333;
  max-height: 500px;
  overflow-y: auto;
}
</style>
