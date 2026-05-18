<template>
  <div class="host-manage">
    <div class="page-header">
      <div class="header-left">
        <span class="icon">💻</span>
        <div class="header-text">
          <h2>主机管理</h2>
          <p>Host Management</p>
        </div>
      </div>
      <el-button type="primary" class="add-btn" @click="openAddDialog">
        <span class="btn-icon">+</span>
        添加主机
      </el-button>
    </div>
    
    <el-card class="main-card">
      <el-table 
        :data="hosts" 
        class="tech-table"
        :max-height="400"
      >
        <el-table-column prop="id" label="ID" width="60">
          <template #default="{ row }">
            <span class="id-text">#{{ row.id }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="主机名" width="120">
          <template #default="{ row }">
            <div class="name-cell">
              <span class="name-icon">🖥️</span>
              <span>{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="host" label="IP地址" width="140">
          <template #default="{ row }">
            <span class="ip-text">{{ row.host }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="port" label="端口" width="70">
          <template #default="{ row }">
            <span class="port-tag">{{ row.port }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="username" label="用户名" width="90">
          <template #default="{ row }">
            <span class="user-text">{{ row.username }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="auth_type" label="认证方式" width="90">
          <template #default="{ row }">
            <el-tag :type="row.auth_type === 'password' ? 'info' : 'success'" size="small">
              {{ row.auth_type === "password" ? "密码" : "密钥" }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <div class="status-cell">
              <span class="status-dot" :class="row.status === 'online' ? 'online' : 'offline'"></span>
              <span>{{ row.status === "online" ? "在线" : "离线" }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="260">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-button size="small" type="success" @click="openSsh(row)">
                SSH
              </el-button>
              <el-button size="small" type="primary" @click="openEditDialog(row)">
                编辑
              </el-button>
              <el-button size="small" type="danger" @click="handleDelete(row.id)">
                删除
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <SshTerminal
      v-if="sshHost"
      :visible="sshVisible"
      :host="sshHost"
      @close="sshVisible = false"
    />

    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑主机' : '添加主机'"
      width="500px"
      class="tech-dialog"
    >
      <el-form
        :model="hostForm"
        :rules="rules"
        ref="formRef"
        label-width="100px"
        class="tech-form"
      >
        <el-form-item label="主机名" prop="name">
          <el-input v-model="hostForm.name" placeholder="请输入主机名" class="custom-input" />
        </el-form-item>
        <el-form-item label="IP地址" prop="host">
          <el-input v-model="hostForm.host" placeholder="请输入IP地址" class="custom-input" />
        </el-form-item>
        <el-form-item label="端口" prop="port">
          <el-input-number v-model="hostForm.port" :min="1" :max="65535" class="custom-input" />
        </el-form-item>
        <el-form-item label="用户名" prop="username">
          <el-input v-model="hostForm.username" placeholder="请输入用户名" class="custom-input" />
        </el-form-item>
        <el-form-item label="认证方式" prop="auth_type">
          <el-radio-group v-model="hostForm.auth_type" class="custom-radio">
            <el-radio label="password">密码</el-radio>
            <el-radio label="key">密钥</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="密码/密钥" prop="credential">
          <el-input
            v-model="hostForm.credential"
            type="password"
            placeholder="请输入密码或密钥"
            class="custom-input"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false" class="cancel-btn">取消</el-button>
        <el-button type="primary" @click="handleSubmit" class="confirm-btn">
          <span class="btn-icon">✅</span>
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { getHosts, addHost, updateHost, deleteHost } from "../api";
import SshTerminal from "../components/SshTerminal.vue";

const hosts = ref([]);
const dialogVisible = ref(false);
const isEdit = ref(false);
const formRef = ref();
const sshVisible = ref(false);
const sshHost = ref(null);

const hostForm = reactive({
  id: null,
  name: "",
  host: "",
  port: 22,
  username: "",
  auth_type: "password",
  credential: "",
});

const rules = {
  name: [{ required: true, message: "请输入主机名", trigger: "blur" }],
  host: [{ required: true, message: "请输入IP地址", trigger: "blur" }],
  port: [{ required: true, message: "请输入端口", trigger: "blur" }],
  username: [{ required: true, message: "请输入用户名", trigger: "blur" }],
  auth_type: [{ required: true, message: "请选择认证方式", trigger: "change" }],
  credential: [
    { required: true, message: "请输入密码或密钥", trigger: "blur" },
  ],
};

const loadHosts = async () => {
  try {
    const res = await getHosts();
    hosts.value = res.data || [];
  } catch (err) {
    ElMessage.error("获取主机列表失败");
  }
};

const openAddDialog = () => {
  isEdit.value = false;
  Object.assign(hostForm, {
    id: null,
    name: "",
    host: "",
    port: 22,
    username: "",
    auth_type: "password",
    credential: "",
  });
  dialogVisible.value = true;
};

const openEditDialog = (row) => {
  isEdit.value = true;
  Object.assign(hostForm, { ...row, credential: "" });
  dialogVisible.value = true;
};

const openSsh = (row) => {
  sshHost.value = row;
  sshVisible.value = true;
};

const handleSubmit = async () => {
  const valid = await formRef.value.validate().catch(() => false);
  if (!valid) return;

  try {
    const data = { ...hostForm };
    delete data.id;
    if (isEdit.value) {
      await updateHost(hostForm.id, data);
      ElMessage.success("更新成功");
    } else {
      await addHost(data);
      ElMessage.success("添加成功");
    }
    dialogVisible.value = false;
    loadHosts();
  } catch (err) {
    ElMessage.error(err.message || "操作失败");
  }
};

const handleDelete = async (id) => {
  try {
    await ElMessageBox.confirm("确定要删除该主机吗？", "提示", {
      type: "warning",
    });
    await deleteHost(id);
    ElMessage.success("删除成功");
    loadHosts();
  } catch (err) {
    if (err !== "cancel") {
      ElMessage.error("删除失败");
    }
  }
};

onMounted(() => {
  loadHosts();
});
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
  overflow: hidden;
}

.main-card :deep(.el-card__body) {
  padding: 0;
}

.status-cell {
  display: flex;
  align-items: center;
  gap: 6px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-dot.online {
  background: #22c55e;
}

.status-dot.offline {
  background: #999999;
}

.action-buttons {
  display: flex;
  gap: 4px;
  flex-wrap: nowrap;
}

.action-buttons .el-button {
  padding: 5px 10px;
  font-size: 12px;
}
</style>