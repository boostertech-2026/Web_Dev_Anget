<template>
  <div class="client-manage">
    <div class="page-header">
      <div class="header-left">
        <span class="icon">🔗</span>
        <div class="header-text">
          <h2>客户端管理</h2>
          <p>Client Management</p>
        </div>
      </div>
      <el-button type="primary" class="add-btn" @click="showAddDialog">
        <span>+ 添加客户端</span>
      </el-button>
    </div>

    <el-card class="main-card">
      <el-table :data="clients" class="tech-table">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="host" label="主机" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'online' ? 'success' : 'danger'">
              {{ row.status === 'online' ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_heart" label="最后心跳" width="180">
          <template #default="{ row }">
            {{ formatTime(row.last_heart) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button size="small" @click="showEditDialog(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="deleteClientHandler(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 添加/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="500px" class="tech-dialog">
      <el-form :model="clientForm" :rules="rules" ref="formRef" class="tech-form">
        <el-form-item label="名称" prop="name">
          <el-input v-model="clientForm.name" placeholder="请输入客户端名称" />
        </el-form-item>
        <el-form-item label="主机" prop="host">
          <el-input v-model="clientForm.host" placeholder="请输入主机地址" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button class="cancel-btn" @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" class="confirm-btn" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getClients, addClient, updateClient, deleteClient } from '../api'

const clients = ref([])
const dialogVisible = ref(false)
const dialogTitle = ref('添加客户端')
const formRef = ref()
const editingId = ref(null)

const clientForm = reactive({
  name: '',
  host: ''
})

const rules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  host: [{ required: true, message: '请输入主机', trigger: 'blur' }]
}

const loadClients = async () => {
  try {
    const res = await getClients()
    clients.value = res.data
  } catch {
    ElMessage.error('加载客户端列表失败')
  }
}

const showAddDialog = () => {
  dialogTitle.value = '添加客户端'
  clientForm.name = ''
  clientForm.host = ''
  editingId.value = null
  dialogVisible.value = true
}

const showEditDialog = (client) => {
  dialogTitle.value = '编辑客户端'
  clientForm.name = client.name
  clientForm.host = client.host
  editingId.value = client.id
  dialogVisible.value = true
}

const submitForm = async () => {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  try {
    if (editingId.value) {
      await updateClient(editingId.value, { name: clientForm.name, host: clientForm.host })
      ElMessage.success('更新成功')
    } else {
      await addClient({ name: clientForm.name, host: clientForm.host })
      ElMessage.success('添加成功')
    }
    dialogVisible.value = false
    loadClients()
  } catch {
    ElMessage.error('操作失败')
  }
}

const deleteClientHandler = async (id) => {
  try {
    await ElMessageBox.confirm('确定删除此客户端吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteClient(id)
    ElMessage.success('删除成功')
    loadClients()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

const formatTime = (timeStr) => {
  if (!timeStr) return '-'
  return new Date(timeStr).toLocaleString()
}

onMounted(() => {
  loadClients()
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

.main-card {
  background: #ffffff;
  border: 1px solid #e5e5e5;
  border-radius: 8px;
}

.main-card :deep(.el-card__body) {
  padding: 0;
}
</style>
