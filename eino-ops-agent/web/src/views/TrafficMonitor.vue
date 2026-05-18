<template>
  <div class="traffic-monitor">
    <div class="page-header">
      <h2>🌐 流量监控</h2>
      <div class="controls">
        <el-select v-model="selectedHost" placeholder="选择主机" filterable @change="fetchData" class="host-select">
          <el-option v-for="h in hosts" :key="h.id" :label="h.name" :value="h.id" />
        </el-select>
        <el-button @click="fetchData" type="primary" plain :loading="loading">刷新</el-button>
      </div>
    </div>

    <!-- Traffic Summary -->
    <div class="traffic-cards" v-if="latest">
      <div class="t-card rx">
        <div class="t-icon">📥</div>
        <div class="t-body">
          <div class="t-value">{{ fmtBytes(latest.net_rx_rate || 0) }}/s</div>
          <div class="t-label">接收速率</div>
        </div>
        <div class="t-total">累计: {{ fmtBytes(latest.net_rx_bytes || 0) }}</div>
      </div>
      <div class="t-card tx">
        <div class="t-icon">📤</div>
        <div class="t-body">
          <div class="t-value">{{ fmtBytes(latest.net_tx_rate || 0) }}/s</div>
          <div class="t-label">发送速率</div>
        </div>
        <div class="t-total">累计: {{ fmtBytes(latest.net_tx_bytes || 0) }}</div>
      </div>
    </div>

    <!-- Traffic Table -->
    <div class="panel">
      <div class="panel-header"><h3>📊 网络流量历史 ({{ traffic.length }} 条)</h3></div>
      <div class="panel-body">
        <el-table :data="traffic" size="small" empty-text="请选择主机查看数据" class="tech-table" style="width: 100%">
          <el-table-column prop="created_at" label="时间" width="170">
            <template #default="{ row }">{{ fmtTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="接收速率" width="130">
            <template #default="{ row }">
              <span class="rate rx-rate">{{ fmtBytes(row.net_rx_rate || 0) }}/s</span>
            </template>
          </el-table-column>
          <el-table-column label="发送速率" width="130">
            <template #default="{ row }">
              <span class="rate tx-rate">{{ fmtBytes(row.net_tx_rate || 0) }}/s</span>
            </template>
          </el-table-column>
          <el-table-column label="累计接收" width="130">
            <template #default="{ row }">{{ fmtBytes(row.net_rx_bytes || 0) }}</template>
          </el-table-column>
          <el-table-column label="累计发送" width="130">
            <template #default="{ row }">{{ fmtBytes(row.net_tx_bytes || 0) }}</template>
          </el-table-column>
          <el-table-column label="速率占比" min-width="200">
            <template #default="{ row }">
              <div class="rate-bar-wrap">
                <div class="rate-bar">
                  <div class="bar-rx" :style="{ width: rxPct(row) + '%' }"></div>
                  <div class="bar-tx" :style="{ width: txPct(row) + '%' }"></div>
                </div>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getHosts, getTrafficMetrics, getLatestMetrics } from '../api/index.js'

const hosts = ref([])
const selectedHost = ref(null)
const traffic = ref([])
const loading = ref(false)

const allMetrics = computed(() => traffic.value)
const latest = computed(() => traffic.value[traffic.value.length - 1] || null)

const maxRate = computed(() => {
  let max = 1
  traffic.value.forEach(t => {
    max = Math.max(max, (t.net_rx_rate || 0) + (t.net_tx_rate || 0))
  })
  return max
})

const rxPct = (row) => maxRate.value > 0 ? ((row.net_rx_rate || 0) / maxRate.value * 100) : 0
const txPct = (row) => maxRate.value > 0 ? ((row.net_tx_rate || 0) / maxRate.value * 100) : 0

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
  if (!selectedHost.value) return
  loading.value = true
  try {
    const res = await getTrafficMetrics(selectedHost.value)
    traffic.value = res.data || []
  } catch (e) {
    console.error('Traffic fetch error:', e)
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    const [hostRes] = await Promise.all([getHosts()])
    hosts.value = hostRes.data || []
    if (hosts.value.length > 0) {
      selectedHost.value = hosts.value[0].id
      await fetchData()
    }
  } catch (e) {
    console.error(e)
  }
})
</script>

<style scoped>
.traffic-monitor {
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

.traffic-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 16px;
}

.t-card {
  padding: 24px;
  border-radius: 8px;
  background: #ffffff;
  border: 1px solid #e5e5e5;
}

.t-card.rx { border-left: 3px solid #22c55e; }
.t-card.tx { border-left: 3px solid #f59e0b; }

.t-icon { font-size: 32px; margin-bottom: 8px; }

.t-body { margin-bottom: 8px; }

.t-value {
  font-size: 28px;
  font-weight: 700;
  line-height: 1.2;
  color: #111111;
}

.rx .t-value { color: #22c55e; }
.tx .t-value { color: #f59e0b; }

.t-label { font-size: 13px; color: #888888; }
.t-total { font-size: 12px; color: #888888; }

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

.rate { font-weight: 600; }
.rx-rate { color: #22c55e; }
.tx-rate { color: #f59e0b; }

.rate-bar-wrap { width: 100%; }
.rate-bar {
  height: 8px;
  border-radius: 4px;
  background: #f0f0f0;
  display: flex;
  overflow: hidden;
}

.bar-rx {
  height: 100%;
  background: #22c55e;
  transition: width 0.4s;
}

.bar-tx {
  height: 100%;
  background: #f59e0b;
  transition: width 0.4s;
}
</style>
