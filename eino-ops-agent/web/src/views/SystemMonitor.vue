<template>
  <div class="system-monitor">
    <div class="page-header">
      <h2>🖥️ 系统监控</h2>
      <div class="controls">
        <el-select v-model="selectedHost" placeholder="选择主机" filterable clearable @change="fetchData" class="host-select">
          <el-option v-for="h in hosts" :key="h.id" :label="h.name" :value="h.id" />
        </el-select>
        <el-select v-model="duration" @change="fetchData" class="dur-select">
          <el-option label="最近5分钟" value="5m" />
          <el-option label="最近1小时" value="1h" />
          <el-option label="最近24小时" value="24h" />
          <el-option label="最近7天" value="7d" />
        </el-select>
      </div>
    </div>

    <!-- Latest Metric Cards -->
    <div class="metric-cards" v-if="latest">
      <div class="m-card cpu" :class="level(latest.cpu_percent, 70, 90)">
        <div class="m-icon">⚙️</div>
        <div class="m-body">
          <div class="m-value">{{ (latest.cpu_percent || 0).toFixed(1) }}%</div>
          <div class="m-label">CPU 使用率</div>
        </div>
        <div class="m-bar"><div class="m-fill" :style="{ width: (latest.cpu_percent || 0) + '%' }"></div></div>
      </div>
      <div class="m-card mem" :class="level(latest.mem_percent, 80, 95)">
        <div class="m-icon">🧠</div>
        <div class="m-body">
          <div class="m-value">{{ (latest.mem_percent || 0).toFixed(1) }}%</div>
          <div class="m-label">内存使用率</div>
        </div>
        <div class="m-detail">{{ fmtBytes(latest.mem_used) }} / {{ fmtBytes(latest.mem_total) }}</div>
        <div class="m-bar"><div class="m-fill" :style="{ width: (latest.mem_percent || 0) + '%' }"></div></div>
      </div>
      <div class="m-card disk" :class="level(latest.disk_percent, 75, 90)">
        <div class="m-icon">💾</div>
        <div class="m-body">
          <div class="m-value">{{ (latest.disk_percent || 0).toFixed(1) }}%</div>
          <div class="m-label">磁盘使用率</div>
        </div>
        <div class="m-detail">{{ fmtBytes(latest.disk_used) }} / {{ fmtBytes(latest.disk_total) }}</div>
        <div class="m-bar"><div class="m-fill" :style="{ width: (latest.disk_percent || 0) + '%' }"></div></div>
      </div>
      <div class="m-card load">
        <div class="m-icon">📈</div>
        <div class="m-body">
          <div class="m-value">{{ (latest.load_1m || 0).toFixed(1) }} / {{ (latest.load_5m || 0).toFixed(1) }} / {{ (latest.load_15m || 0).toFixed(1) }}</div>
          <div class="m-label">系统负载 (1m / 5m / 15m)</div>
        </div>
      </div>
    </div>

    <!-- History Table -->
    <div class="panel">
      <div class="panel-header"><h3>📊 历史数据 ({{ history.length }} 条)</h3></div>
      <div class="panel-body">
        <el-table :data="history" size="small" empty-text="暂无数据" class="tech-table" style="width: 100%">
          <el-table-column prop="created_at" label="时间" width="160">
            <template #default="{ row }">{{ fmtTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column prop="cpu_percent" label="CPU" width="80">
            <template #default="{ row }">{{ (row.cpu_percent || 0).toFixed(1) }}%</template>
          </el-table-column>
          <el-table-column prop="mem_percent" label="内存" width="80">
            <template #default="{ row }">{{ (row.mem_percent || 0).toFixed(1) }}%</template>
          </el-table-column>
          <el-table-column prop="disk_percent" label="磁盘" width="80">
            <template #default="{ row }">{{ (row.disk_percent || 0).toFixed(1) }}%</template>
          </el-table-column>
          <el-table-column label="负载 (1m/5m/15m)" width="150">
            <template #default="{ row }">{{ (row.load_1m || 0).toFixed(1) }} / {{ (row.load_5m || 0).toFixed(1) }} / {{ (row.load_15m || 0).toFixed(1) }}</template>
          </el-table-column>
          <el-table-column prop="net_rx_rate" label="网络入" width="90">
            <template #default="{ row }">{{ fmtBytes(row.net_rx_rate) }}/s</template>
          </el-table-column>
          <el-table-column prop="net_tx_rate" label="网络出" width="90">
            <template #default="{ row }">{{ fmtBytes(row.net_tx_rate) }}/s</template>
          </el-table-column>
          <el-table-column prop="process_top" label="Top进程" min-width="200">
            <template #default="{ row }">{{ row.process_top?.substring(0, 120) }}</template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getHosts, getLatestMetrics, getMetricsHistory } from '../api/index.js'

const hosts = ref([])
const selectedHost = ref(null)
const duration = ref('1h')
const history = ref([])
const metrics = ref([])

const latest = computed(() => metrics.value[0] || null)

const level = (val, warn, crit) => {
  if (val >= crit) return 'critical'
  if (val >= warn) return 'warning'
  return ''
}

const fmtBytes = (b) => {
  if (!b || b === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(Math.abs(b)) / Math.log(k))
  return parseFloat((b / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

const fmtTime = (t) => {
  if (!t) return ''
  return new Date(t).toLocaleString('zh-CN')
}

const fetchData = async () => {
  try {
    if (selectedHost.value) {
      const [histRes, latestRes] = await Promise.all([
        getMetricsHistory(selectedHost.value, duration.value),
        getLatestMetrics(String(selectedHost.value))
      ])
      history.value = histRes.data || []
      metrics.value = latestRes.data || []
    } else {
      const res = await getLatestMetrics()
      metrics.value = res.data || []
      history.value = []
    }
  } catch (e) {
    console.error('Monitor fetch error:', e)
  }
}

onMounted(async () => {
  try {
    const res = await getHosts()
    hosts.value = res.data || []
    if (hosts.value.length > 0) {
      selectedHost.value = hosts.value[0].id
    }
    await fetchData()
  } catch (e) {
    console.error(e)
  }
})
</script>

<style scoped>
.system-monitor {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-header h2 {
  margin: 0;
  color: #111111;
  font-size: 20px;
}

.controls {
  display: flex;
  gap: 12px;
}

.host-select { width: 200px; }
.dur-select { width: 140px; }

.metric-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;
}

.m-card {
  padding: 20px;
  border-radius: 8px;
  background: #ffffff;
  border: 1px solid #e5e5e5;
}

.m-card.warning { border-color: #f59e0b; }
.m-card.critical { border-color: #ef4444; }

.m-icon { font-size: 28px; margin-bottom: 8px; }

.m-body { margin-bottom: 8px; }

.m-value {
  font-size: 28px;
  font-weight: 700;
  color: #111111;
  line-height: 1.2;
}

.warning .m-value { color: #f59e0b; }
.critical .m-value { color: #ef4444; }

.m-label { font-size: 13px; color: #888888; }

.m-detail {
  font-size: 11px;
  color: #888888;
  margin-bottom: 6px;
}

.m-bar {
  height: 4px;
  background: #f0f0f0;
  border-radius: 2px;
  overflow: hidden;
}

.m-fill {
  height: 100%;
  background: #555555;
  border-radius: 2px;
  transition: width 0.6s;
}

.warning .m-fill { background: #f59e0b; }
.critical .m-fill { background: #ef4444; }

.panel {
  background: #ffffff;
  border: 1px solid #e5e5e5;
  border-radius: 8px;
  overflow: hidden;
}

.panel-header {
  padding: 14px 20px;
  border-bottom: 1px solid #f0f0f0;
}

.panel-header h3 { margin: 0; color: #111111; font-size: 15px; }

.panel-body { padding: 12px; }
</style>
