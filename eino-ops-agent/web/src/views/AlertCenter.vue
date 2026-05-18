<template>
  <div class="alert-center">
    <div class="page-header">
      <h2>🔔 告警中心</h2>
      <div class="controls">
        <el-select v-model="filterStatus" placeholder="状态筛选" clearable @change="fetchData" class="filter-select">
          <el-option label="触发中" value="firing" />
          <el-option label="已确认" value="acked" />
          <el-option label="已解决" value="resolved" />
        </el-select>
        <el-select v-model="filterLevel" placeholder="级别筛选" clearable @change="fetchData" class="filter-select">
          <el-option label="严重" value="critical" />
          <el-option label="警告" value="warning" />
          <el-option label="信息" value="info" />
        </el-select>
        <el-button type="primary" plain @click="showRules = true">告警规则</el-button>
      </div>
    </div>

    <!-- Alerts Table -->
    <div class="panel">
      <div class="panel-body">
        <el-table :data="alerts" size="small" empty-text="暂无告警" class="tech-table" style="width: 100%">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column label="级别" width="80">
            <template #default="{ row }">
              <span class="level-tag" :class="row.level">{{ row.level }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="rule_name" label="规则" width="100" />
          <el-table-column prop="host_name" label="主机" width="120" />
          <el-table-column prop="message" label="告警信息" min-width="200" show-overflow-tooltip />
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <span class="status-tag" :class="row.status">{{ statusLabel(row.status) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="acked_by" label="确认人" width="90" />
          <el-table-column label="时间" width="160">
            <template #default="{ row }">{{ fmtTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="160" fixed="right">
            <template #default="{ row }">
              <el-button v-if="row.status === 'firing'" type="warning" size="small" @click="handleAck(row)">确认</el-button>
              <el-button v-if="row.status !== 'resolved'" type="success" size="small" @click="handleResolve(row)">解决</el-button>
              <span v-if="row.status === 'resolved'" class="resolved-info">
                {{ row.resolved_by }}<br>{{ fmtTime(row.resolved_at) }}
              </span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- Rules Dialog -->
    <el-dialog v-model="showRules" title="告警规则配置" width="700px" class="dark-dialog">
      <el-table :data="rules" size="small" empty-text="暂无规则" class="tech-table" style="width: 100%">
        <el-table-column prop="name" label="规则名称" width="120" />
        <el-table-column prop="metric" label="监控指标" width="100" />
        <el-table-column label="阈值" width="140">
          <template #default="{ row }">
            {{ row.metric === 'host_down' ? '离线' : row.condition + ' ' + row.threshold + '%' }}
          </template>
        </el-table-column>
        <el-table-column label="持续时间" width="100">
          <template #default="{ row }">{{ row.duration }}秒</template>
        </el-table-column>
        <el-table-column prop="level" label="级别" width="80">
          <template #default="{ row }">
            <span class="level-tag" :class="row.level">{{ row.level }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-switch
              v-model="row.enabled"
              @change="(val) => toggleRule(row, val)"
              active-color="#111111"
              inactive-color="#4a5a7a"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button size="small" @click="editRule(row)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- Edit Rule Dialog -->
    <el-dialog v-model="showEditRule" title="编辑告警规则" width="500px" class="dark-dialog">
      <el-form v-if="editingRule" :model="editingRule" label-width="100px">
        <el-form-item label="规则名称">
          <el-input v-model="editingRule.name" disabled />
        </el-form-item>
        <el-form-item label="阈值">
          <el-input-number v-model="editingRule.threshold" :min="0" :max="100" />
        </el-form-item>
        <el-form-item label="持续时间(秒)">
          <el-input-number v-model="editingRule.duration" :min="30" :max="3600" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="editingRule.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditRule = false">取消</el-button>
        <el-button type="primary" @click="saveRule">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getAlerts, ackAlert, resolveAlert, getAlertRules, updateAlertRule } from '../api/index.js'
import { ElMessage, ElMessageBox } from 'element-plus'

const alerts = ref([])
const rules = ref([])
const filterStatus = ref('')
const filterLevel = ref('')
const showRules = ref(false)
const showEditRule = ref(false)
const editingRule = ref(null)

const statusLabel = (s) => ({ firing: '触发中', acked: '已确认', resolved: '已解决' }[s] || s)

const fmtTime = (t) => {
  if (!t) return ''
  return new Date(t).toLocaleString('zh-CN')
}

const fetchData = async () => {
  try {
    const res = await getAlerts(filterStatus.value || undefined, filterLevel.value || undefined)
    alerts.value = res.data || []
  } catch (e) {
    console.error(e)
  }
}

const handleAck = async (row) => {
  try {
    const name = localStorage.getItem('username') || 'admin'
    await ackAlert(row.id, { acked_by: name })
    ElMessage.success('已确认告警')
    fetchData()
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

const handleResolve = async (row) => {
  try {
    const { value: note } = await ElMessageBox.prompt('请输入解决备注', '解决告警', {
      confirmButtonText: '确认',
      cancelButtonText: '取消',
      inputPlaceholder: '解决说明...'
    })
    const name = localStorage.getItem('username') || 'admin'
    await resolveAlert(row.id, { resolved_by: name, resolve_note: note || '' })
    ElMessage.success('已解决告警')
    fetchData()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('操作失败')
  }
}

const toggleRule = async (rule, val) => {
  try {
    await updateAlertRule(rule.id, { enabled: val })
    ElMessage.success(val ? '已启用规则' : '已禁用规则')
  } catch (e) {
    rule.enabled = !val
    ElMessage.error('操作失败')
  }
}

const editRule = (row) => {
  editingRule.value = { ...row }
  showEditRule.value = true
}

const saveRule = async () => {
  try {
    const r = editingRule.value
    await updateAlertRule(r.id, { threshold: r.threshold, duration: r.duration, enabled: r.enabled })
    ElMessage.success('规则已更新')
    showEditRule.value = false
    const res = await getAlertRules()
    rules.value = res.data || []
  } catch (e) {
    ElMessage.error('保存失败')
  }
}

onMounted(async () => {
  await fetchData()
  try {
    const res = await getAlertRules()
    rules.value = res.data || []
  } catch (e) { console.error(e) }
})
</script>

<style scoped>
.alert-center {
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
.filter-select { width: 120px; }

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

.status-tag.firing { background: #fef2f2; color: #ef4444; }
.status-tag.acked { background: #fffbeb; color: #f59e0b; }
.status-tag.resolved { background: #f0fdf4; color: #22c55e; }

.resolved-info { font-size: 11px; color: #888888; line-height: 1.4; }
</style>
