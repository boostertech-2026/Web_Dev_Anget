<template>
  <div class="dashboard">
    <!-- Stat Cards -->
    <div class="stat-cards">
      <div class="stat-card hosts">
        <div class="stat-icon">💻</div>
        <div class="stat-info">
          <div class="stat-value">{{ summary.hosts?.total || 0 }}</div>
          <div class="stat-label">主机总数</div>
        </div>
        <div class="stat-detail">
          <span class="online">{{ summary.hosts?.online || 0 }} 在线</span>
          <span class="offline">{{ summary.hosts?.offline || 0 }} 离线</span>
        </div>
      </div>
      <div class="stat-card alerts">
        <div class="stat-icon">🔔</div>
        <div class="stat-info">
          <div class="stat-value">{{ summary.alerts?.count || 0 }}</div>
          <div class="stat-label">活跃告警</div>
        </div>
      </div>
      <div class="stat-card cpu">
        <div class="stat-icon">⚙️</div>
        <div class="stat-info">
          <div class="stat-value">{{ (summary.metrics?.avg_cpu || 0).toFixed(1) }}%</div>
          <div class="stat-label">平均 CPU</div>
        </div>
      </div>
      <div class="stat-card mem">
        <div class="stat-icon">🧠</div>
        <div class="stat-info">
          <div class="stat-value">{{ (summary.metrics?.avg_mem || 0).toFixed(1) }}%</div>
          <div class="stat-label">平均内存</div>
        </div>
      </div>
      <div class="stat-card disk">
        <div class="stat-icon">💾</div>
        <div class="stat-info">
          <div class="stat-value">{{ (summary.metrics?.avg_disk || 0).toFixed(1) }}%</div>
          <div class="stat-label">平均磁盘</div>
        </div>
      </div>
    </div>

    <div class="dashboard-grid">
      <!-- Recent Alerts -->
      <div class="panel">
        <div class="panel-header">
          <h3>🔔 最近告警</h3>
          <el-button text type="primary" @click="$router.push('/main/alerts')">查看全部</el-button>
        </div>
        <div class="panel-body">
          <div v-if="!summary.alerts?.recent?.length" class="empty">暂无告警</div>
          <div v-for="alert in summary.alerts?.recent" :key="alert.id" class="list-item alert-item">
            <div class="item-left">
              <span class="level-tag" :class="alert.level">{{ alert.level }}</span>
              <span class="item-name">{{ alert.host_name }}</span>
            </div>
            <div class="item-right">
              <span class="item-msg">{{ alert.message }}</span>
              <span class="item-time">{{ fmtTime(alert.created_at) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Recent Tasks -->
      <div class="panel">
        <div class="panel-header">
          <h3>📋 最近任务</h3>
          <el-button text type="primary" @click="$router.push('/main/tasks')">查看全部</el-button>
        </div>
        <div class="panel-body">
          <div v-if="!summary.tasks?.recent?.length" class="empty">暂无任务</div>
          <div v-for="task in summary.tasks?.recent" :key="task.id" class="list-item">
            <div class="item-left">
              <span class="status-tag" :class="task.status">{{ task.status }}</span>
              <span class="item-name">{{ task.name }}</span>
            </div>
            <div class="item-right">
              <span class="item-type">{{ task.exec_type }}</span>
              <span class="item-time">{{ fmtTime(task.created_at) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getDashboardSummary } from '../api/index.js'

const summary = ref({})

const fmtTime = (t) => {
  if (!t) return ''
  const d = new Date(t)
  return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

const fetchData = async () => {
  try {
    const res = await getDashboardSummary()
    summary.value = res
  } catch (e) {
    console.error('Dashboard fetch error:', e)
  }
}

onMounted(fetchData)
</script>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.stat-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.stat-card {
  padding: 20px 24px;
  border-radius: 8px;
  background: #ffffff;
  border: 1px solid #e5e5e5;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}

.stat-icon {
  font-size: 32px;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #111111;
  line-height: 1.2;
}

.stat-label {
  font-size: 13px;
  color: #888888;
  margin-top: 2px;
}

.stat-detail {
  width: 100%;
  display: flex;
  gap: 16px;
  font-size: 12px;
  margin-top: 8px;
}

.stat-detail .online { color: #22c55e; }
.stat-detail .offline { color: #999999; }

.dashboard-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 20px;
}

.panel {
  background: #ffffff;
  border: 1px solid #e5e5e5;
  border-radius: 8px;
  overflow: hidden;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  border-bottom: 1px solid #f0f0f0;
}

.panel-header h3 {
  margin: 0;
  color: #111111;
  font-size: 15px;
  font-weight: 600;
}

.panel-body {
  padding: 12px 24px;
  max-height: 360px;
  overflow-y: auto;
}

.empty {
  text-align: center;
  padding: 30px;
  color: #aaaaaa;
}

.list-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 0;
  border-bottom: 1px solid #f5f5f5;
}

.list-item:last-child { border-bottom: none; }

.item-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.level-tag, .status-tag {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.level-tag.critical { background: #fef2f2; color: #ef4444; }
.level-tag.warning { background: #fffbeb; color: #f59e0b; }
.level-tag.info { background: #f0f0f0; color: #555555; }

.status-tag.completed, .status-tag.success { background: #f0fdf4; color: #22c55e; }
.status-tag.running, .status-tag.pending { background: #f0f0f0; color: #555555; }
.status-tag.failed { background: #fef2f2; color: #ef4444; }

.item-name { color: #333333; font-size: 14px; }

.item-right {
  display: flex;
  gap: 16px;
  align-items: center;
}

.item-msg {
  color: #888888;
  font-size: 13px;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-type {
  color: #555555;
  font-size: 12px;
  padding: 2px 8px;
  background: #f5f5f5;
  border-radius: 4px;
}

.item-time {
  color: #bbbbbb;
  font-size: 12px;
}
</style>
