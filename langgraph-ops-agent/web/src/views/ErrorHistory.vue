<template>
  <div class="error-history">
    <div class="page-header">
      <h2>⚠️ 错误历史</h2>
      <div class="controls">
        <el-select v-model="filterStatus" placeholder="状态筛选" clearable @change="fetchErrors" class="filter-select">
          <el-option label="待处理" value="pending" />
          <el-option label="处理中" value="processing" />
          <el-option label="已解决" value="resolved" />
        </el-select>
        <el-button @click="fetchStats" type="info" plain>统计</el-button>
      </div>
    </div>

    <!-- Stats Panel -->
    <div class="stats-panel" v-if="stats">
      <div class="s-card">
        <div class="s-value">{{ stats.total_errors || 0 }}</div>
        <div class="s-label">错误总数</div>
      </div>
      <div class="s-card">
        <div class="s-value">{{ (stats.avg_resolve_sec || 0).toFixed(0) }}s</div>
        <div class="s-label">平均解决时间</div>
      </div>
      <div class="s-card" v-for="item in stats.by_level" :key="item.level">
        <div class="s-value" :class="'level-' + item.level">{{ item.count }}</div>
        <div class="s-label">{{ item.level }}</div>
      </div>
    </div>

    <!-- Top Hosts -->
    <div class="top-hosts" v-if="stats?.top_hosts?.length">
      <span class="top-label">问题主机 TOP:</span>
      <span v-for="h in stats.top_hosts" :key="h.host_name" class="top-tag">
        {{ h.host_name }} ({{ h.count }})
      </span>
    </div>

    <!-- Errors Table -->
    <div class="panel">
      <div class="panel-body">
        <el-table :data="errors" size="small" empty-text="暂无错误记录" class="tech-table" style="width: 100%">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column label="级别" width="80">
            <template #default="{ row }">
              <span class="level-tag" :class="row.level">{{ row.level }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="source" label="来源" width="80" />
          <el-table-column prop="host_name" label="主机" width="100" />
          <el-table-column prop="message" label="错误信息" min-width="240" show-overflow-tooltip />
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <span class="status-tag" :class="row.status">{{ statusLabel(row.status) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="handled_by" label="处理人" width="90" />
          <el-table-column label="解决耗时" width="90">
            <template #default="{ row }">
              {{ row.resolve_duration ? row.resolve_duration + 's' : '-' }}
            </template>
          </el-table-column>
          <el-table-column label="时间" width="160">
            <template #default="{ row }">{{ fmtTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="140" fixed="right">
            <template #default="{ row }">
              <el-button
                v-if="row.status !== 'resolved'"
                type="success"
                size="small"
                @click="handleResolve(row)"
              >解决</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getErrorHistory, getErrorStats, resolveError } from '../api/index.js'
import { ElMessage, ElMessageBox } from 'element-plus'

const errors = ref([])
const stats = ref(null)
const filterStatus = ref('')

const statusLabel = (s) => ({ pending: '待处理', processing: '处理中', resolved: '已解决' }[s] || s)

const fmtTime = (t) => {
  if (!t) return ''
  return new Date(t).toLocaleString('zh-CN')
}

const fetchErrors = async () => {
  try {
    const res = await getErrorHistory(filterStatus.value || undefined)
    errors.value = res.data || []
  } catch (e) {
    console.error(e)
  }
}

const fetchStats = async () => {
  try {
    const res = await getErrorStats()
    stats.value = res
  } catch (e) {
    console.error(e)
  }
}

const handleResolve = async (row) => {
  try {
    const { value: note } = await ElMessageBox.prompt('请输入处理备注', '解决错误', {
      confirmButtonText: '确认',
      cancelButtonText: '取消',
      inputPlaceholder: '处理说明...'
    })
    const name = localStorage.getItem('username') || 'admin'
    await resolveError(row.id, { handled_by: name, handle_note: note || '' })
    ElMessage.success('已解决')
    fetchErrors()
    fetchStats()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('操作失败')
  }
}

onMounted(() => {
  fetchErrors()
  fetchStats()
})
</script>

<style scoped>
.error-history {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-header h2 { margin: 0; color: #111111; font-size: 20px; }

.controls { display: flex; gap: 12px; }
.filter-select { width: 130px; }

.stats-panel {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 16px;
}

.s-card {
  padding: 16px 20px;
  border-radius: 8px;
  background: #ffffff;
  border: 1px solid #e5e5e5;
  text-align: center;
}

.s-value {
  font-size: 24px;
  font-weight: 700;
  color: #111111;
  line-height: 1.2;
}

.s-value.level-critical { color: #ef4444; }
.s-value.level-warning { color: #f59e0b; }
.s-value.level-info { color: #555555; }

.s-label { font-size: 12px; color: #888888; margin-top: 4px; }

.top-hosts {
  padding: 12px 16px;
  background: #ffffff;
  border: 1px solid #e5e5e5;
  border-radius: 8px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.top-label { color: #888888; font-size: 13px; }

.top-tag {
  padding: 3px 12px;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 12px;
  color: #ef4444;
  font-size: 12px;
}

.panel {
  background: #ffffff;
  border: 1px solid #e5e5e5;
  border-radius: 8px;
  overflow: hidden;
}

.panel-body { padding: 12px; }

.level-tag, .status-tag {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.level-tag.critical { background: #fef2f2; color: #ef4444; }
.level-tag.warning { background: #fffbeb; color: #f59e0b; }
.level-tag.info { background: #f0f0f0; color: #555555; }

.status-tag.pending { background: #fffbeb; color: #f59e0b; }
.status-tag.processing { background: #f0f0f0; color: #555555; }
.status-tag.resolved { background: #f0fdf4; color: #22c55e; }
</style>
