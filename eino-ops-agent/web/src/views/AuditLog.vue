<template>
  <div class="audit-log">
    <div class="page-header">
      <h2>📝 操作审计</h2>
      <div class="controls">
        <el-select v-model="filterStatus" placeholder="状态筛选" clearable @change="fetchLogs" class="filter-select">
          <el-option label="已完成" value="completed" />
          <el-option label="已中止" value="aborted" />
        </el-select>
        <el-input v-model="filterThread" placeholder="Thread ID 搜索" clearable @change="fetchLogs" class="filter-input" />
        <el-button @click="fetchLogs" type="primary" plain>查询</el-button>
      </div>
    </div>

    <!-- Logs Table -->
    <div class="panel">
      <div class="panel-body">
        <el-table :data="logs" size="small" empty-text="暂无审计记录" class="tech-table" style="width: 100%">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="thread_id" label="Thread ID" width="140" show-overflow-tooltip />
          <el-table-column label="意图" width="120" show-overflow-tooltip>
            <template #default="{ row }">
              {{ parseIntent(row.intent) }}
            </template>
          </el-table-column>
          <el-table-column prop="tools_called" label="工具调用" min-width="160" show-overflow-tooltip />
          <el-table-column label="高危操作" width="140" show-overflow-tooltip>
            <template #default="{ row }">
              <span v-if="row.high_risk_ops" class="risk-tag">{{ row.high_risk_ops }}</span>
              <span v-else class="safe-tag">无</span>
            </template>
          </el-table-column>
          <el-table-column label="人工确认" width="90">
            <template #default="{ row }">
              <span :class="row.approved ? 'approved-tag' : 'na-tag'">{{ row.approved ? '是' : '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="备份路径" width="200" show-overflow-tooltip>
            <template #default="{ row }">
              <span v-if="row.backup_path" class="backup-path">{{ row.backup_path }}</span>
              <span v-else class="na-tag">-</span>
            </template>
          </el-table-column>
          <el-table-column prop="hosts_affected" label="受影响主机" width="130" show-overflow-tooltip />
          <el-table-column label="知识沉淀" width="90">
            <template #default="{ row }">
              <span :class="row.knowledge_saved ? 'approved-tag' : 'na-tag'">{{ row.knowledge_saved ? '是' : '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="时间" width="160">
            <template #default="{ row }">{{ fmtTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="80" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" size="small" @click="showDetail(row)">详情</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- Pagination -->
    <div class="pagination-wrap" v-if="total > pageSize">
      <el-pagination
        v-model:current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="fetchLogs"
      />
    </div>

    <!-- Detail Dialog -->
    <el-dialog v-model="detailVisible" title="审计记录详情" width="700px">
      <div v-if="currentLog" class="detail-content">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="ID">{{ currentLog.id }}</el-descriptions-item>
          <el-descriptions-item label="Thread ID">{{ currentLog.thread_id }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ currentLog.status }}</el-descriptions-item>
          <el-descriptions-item label="人工确认">{{ currentLog.approved ? '是' : '否' }}</el-descriptions-item>
          <el-descriptions-item label="备份路径" :span="2">{{ currentLog.backup_path || '-' }}</el-descriptions-item>
          <el-descriptions-item label="受影响主机" :span="2">{{ currentLog.hosts_affected || '-' }}</el-descriptions-item>
          <el-descriptions-item label="时间">{{ fmtTime(currentLog.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="知识沉淀">{{ currentLog.knowledge_saved ? '是' : '否' }}</el-descriptions-item>
        </el-descriptions>

        <h4 style="margin: 16px 0 8px">意图</h4>
        <pre class="json-block">{{ formatJSON(currentLog.intent) }}</pre>

        <h4 style="margin: 16px 0 8px">执行计划</h4>
        <pre class="text-block">{{ currentLog.plan || '-' }}</pre>

        <h4 style="margin: 16px 0 8px">工具调用</h4>
        <pre class="text-block">{{ currentLog.tools_called || '-' }}</pre>

        <h4 v-if="currentLog.high_risk_ops" style="margin: 16px 0 8px">高危操作</h4>
        <pre v-if="currentLog.high_risk_ops" class="text-block risk">{{ currentLog.high_risk_ops }}</pre>

        <h4 style="margin: 16px 0 8px">观察记录</h4>
        <pre class="text-block">{{ currentLog.observations || '-' }}</pre>

        <h4 style="margin: 16px 0 8px">最终结果</h4>
        <div class="final-result" v-html="renderMarkdown(currentLog.final_result || '-')"></div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getAuditLogs, getAuditLog } from '../api/index.js'

const logs = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const filterStatus = ref('')
const filterThread = ref('')
const detailVisible = ref(false)
const currentLog = ref(null)

const fetchLogs = async () => {
  const params = { page: page.value, page_size: pageSize }
  if (filterStatus.value) params.status = filterStatus.value
  if (filterThread.value) params.thread_id = filterThread.value

  try {
    const res = await getAuditLogs(params)
    logs.value = res.data || []
    total.value = res.total || 0
  } catch {
    // ignore
  }
}

const showDetail = async (row) => {
  try {
    const res = await getAuditLog(row.id)
    currentLog.value = res.data
    detailVisible.value = true
  } catch {
    // ignore
  }
}

const parseIntent = (raw) => {
  try {
    const j = JSON.parse(raw)
    return j.intent || j.description || raw
  } catch {
    return raw
  }
}

const formatJSON = (raw) => {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

const renderMarkdown = (text) => {
  if (!text) return '-'
  return text
    .replace(/```(\w*)\n([\s\S]*?)```/g, '<pre class="code-block"><code>$2</code></pre>')
    .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
    .replace(/\n/g, '<br>')
}

const fmtTime = (ts) => {
  if (!ts) return '-'
  const d = new Date(ts)
  return d.toLocaleString('zh-CN', { hour12: false })
}

onMounted(() => {
  fetchLogs()
})
</script>

<style scoped>
.audit-log {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
  font-size: 20px;
  color: #333;
}

.controls {
  display: flex;
  gap: 10px;
  align-items: center;
}

.filter-select {
  width: 130px;
}

.filter-input {
  width: 200px;
}

.panel {
  background: #fff;
  border: 1px solid #e5e5e5;
  border-radius: 8px;
  overflow: hidden;
}

.panel-body {
  padding: 0;
}

.tech-table {
  font-size: 13px;
}

.risk-tag {
  color: #dc2626;
  font-weight: 600;
  font-size: 12px;
}

.safe-tag {
  color: #aaa;
  font-size: 12px;
}

.approved-tag {
  color: #22c55e;
  font-weight: 600;
  font-size: 12px;
}

.na-tag {
  color: #ccc;
  font-size: 12px;
}

.backup-path {
  font-family: monospace;
  font-size: 11px;
  color: #666;
}

.pagination-wrap {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.detail-content {
  max-height: 60vh;
  overflow-y: auto;
}

.json-block {
  background: #f8f8f8;
  border: 1px solid #e5e5e5;
  border-radius: 6px;
  padding: 12px;
  font-family: 'SF Mono', 'Consolas', monospace;
  font-size: 12px;
  color: #555;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 200px;
  overflow-y: auto;
}

.text-block {
  background: #f8f8f8;
  border: 1px solid #e5e5e5;
  border-radius: 6px;
  padding: 12px;
  font-size: 13px;
  color: #555;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 200px;
  overflow-y: auto;
}

.text-block.risk {
  border-color: #fecaca;
  background: #fef2f2;
  color: #dc2626;
}

.final-result {
  background: #f8f8f8;
  border: 1px solid #e5e5e5;
  border-radius: 6px;
  padding: 12px;
  font-size: 13px;
  color: #333;
  line-height: 1.7;
  max-height: 300px;
  overflow-y: auto;
}

.final-result :deep(.code-block) {
  background: #f0f0f0;
  padding: 8px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
}
</style>
