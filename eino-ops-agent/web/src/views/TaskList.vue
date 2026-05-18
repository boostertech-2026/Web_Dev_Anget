<template>
  <div class="task-list">
    <div class="page-header">
      <div class="header-left">
        <span class="icon">📋</span>
        <div class="header-text">
          <h2>任务列表</h2>
          <p>Task List</p>
        </div>
      </div>
      <el-button type="primary" class="refresh-btn" @click="loadTasks">
        <span>🔄 刷新</span>
      </el-button>
    </div>

    <el-card class="main-card">
      <el-table :data="tasks" border stripe v-loading="loading" class="tech-table">
        <template #empty>
          <el-empty description="暂无任务" />
        </template>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="任务名称" />
        <el-table-column prop="exec_type" label="执行方式" width="100">
          <template #default="{ row }">
            {{ row.exec_type === "ssh" ? "SSH" : "客户端" }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">{{
              getStatusText(row.status)
            }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column prop="result" label="结果" show-overflow-tooltip />
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="viewLog(row)"
              >查看日志</el-button
            >
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import { getTasks } from "../api";

const router = useRouter();
const tasks = ref([]);
const loading = ref(false);
let refreshTimer = null;

const getStatusType = (status) => {
  const map = {
    pending: "info",
    running: "primary",
    success: "success",
    failed: "danger",
  };
  return map[status] || "info";
};

const getStatusText = (status) => {
  const map = {
    pending: "待执行",
    running: "执行中",
    success: "成功",
    failed: "失败",
  };
  return map[status] || status;
};

const loadTasks = async () => {
  loading.value = true;
  try {
    const res = await getTasks();
    tasks.value = res.data || [];
  } catch (err) {
    ElMessage.error("获取任务列表失败");
  } finally {
    loading.value = false;
  }
};

const viewLog = (row) => {
  router.push({ name: 'TaskLogDetail', params: { id: row.id } });
};

onMounted(() => {
  loadTasks();
  refreshTimer = setInterval(loadTasks, 30000);
});

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer);
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
}

.main-card :deep(.el-card__body) {
  padding: 0;
}
</style>
