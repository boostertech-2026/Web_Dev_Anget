<template>
  <div class="main-layout">
    <el-container class="main-container">
      <!-- Sidebar -->
      <el-aside width="220px" class="sidebar">
        <div class="logo">
          <h3>Eino Agent</h3>
        </div>
        <el-menu
          :default-active="activeMenu"
          router
          class="sidebar-menu"
          background-color="transparent"
          text-color="#555555"
          active-text-color="#111111"
        >
          <el-menu-item index="/main/dashboard">
            <span>仪表盘</span>
          </el-menu-item>
          <el-menu-item index="/main/assistant">
            <span>AI 助手</span>
          </el-menu-item>
          <el-menu-item index="/main/hosts">
            <span>主机管理</span>
          </el-menu-item>
          <el-menu-item index="/main/clients">
            <span>客户端管理</span>
          </el-menu-item>
          <el-menu-item index="/main/task/create">
            <span>创建任务</span>
          </el-menu-item>
          <el-menu-item index="/main/tasks">
            <span>任务列表</span>
          </el-menu-item>
          <el-menu-item index="/main/task-log">
            <span>实时日志</span>
          </el-menu-item>
          <el-menu-item index="/main/monitor/system">
            <span>系统监控</span>
          </el-menu-item>
          <el-menu-item index="/main/monitor/traffic">
            <span>流量监控</span>
          </el-menu-item>
          <el-menu-item index="/main/alerts">
            <span>告警中心</span>
          </el-menu-item>
          <el-menu-item index="/main/errors">
            <span>错误历史</span>
          </el-menu-item>
          <el-menu-item index="/main/audit">
            <span>操作审计</span>
          </el-menu-item>
        </el-menu>
        <div class="sidebar-footer">
          <div class="user-info">
            <span class="user-name">{{ username }}</span>
          </div>
          <el-button text size="small" class="logout-btn" @click="handleLogout">
            退出
          </el-button>
        </div>
      </el-aside>

      <!-- Right area -->
      <el-container class="right-container">
        <el-header class="top-header">
          <div class="header-left">
            <h2>Eino 智能运维 Agent</h2>
          </div>
          <div class="header-right">
            <span class="time-display">{{ currentTime }}</span>
          </div>
        </el-header>
        <el-main class="main-content">
          <router-view />
        </el-main>
      </el-container>
    </el-container>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from "vue";
import { useRouter, useRoute } from "vue-router";

const router = useRouter();
const route = useRoute();
const username = ref(localStorage.getItem("username") || "admin");
const currentTime = ref("");
let timer = null;

const activeMenu = computed(() => route.path);

const updateTime = () => {
  const now = new Date();
  currentTime.value = now.toLocaleTimeString("zh-CN", {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
};

const handleLogout = () => {
  localStorage.removeItem("token");
  localStorage.removeItem("username");
  router.push("/login");
};

onMounted(() => {
  updateTime();
  timer = setInterval(updateTime, 1000);
});

onUnmounted(() => {
  if (timer) clearInterval(timer);
});
</script>

<style scoped>
.main-layout {
  height: 100vh;
  background: #f8f9fa;
}

.main-container {
  height: 100%;
}

/* ---- Sidebar ---- */
.sidebar {
  background: #ffffff;
  border-right: 1px solid #e5e5e5;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.logo {
  padding: 24px 20px;
  border-bottom: 1px solid #f0f0f0;
  flex-shrink: 0;
}

.logo h3 {
  color: #111111;
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  letter-spacing: 1px;
}

.sidebar-menu {
  border-right: none;
  background: transparent;
  padding: 8px 0;
  flex: 1;
  overflow-y: auto;
}

.sidebar-menu :deep(.el-menu-item) {
  margin: 2px 8px;
  border-radius: 6px;
  height: 40px;
  line-height: 40px;
  font-size: 14px;
  padding-left: 20px !important;
}

.sidebar-menu :deep(.el-menu-item:hover) {
  background: #f5f5f5 !important;
}

.sidebar-menu :deep(.el-menu-item.is-active) {
  background: #111111 !important;
  color: #ffffff !important;
}

.sidebar-footer {
  padding: 16px;
  border-top: 1px solid #f0f0f0;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.user-name {
  font-size: 13px;
  color: #555555;
}

.logout-btn {
  color: #999999;
  font-size: 12px;
}

.logout-btn:hover {
  color: #111111;
}

/* ---- Right column ---- */
.right-container {
  flex-direction: column;
}
</style>

<style>
/* unscoped — header and main inside right-container */
.right-container > .el-header {
  background: #ffffff;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 32px;
  border-bottom: 1px solid #e5e5e5;
  height: 56px;
  flex-shrink: 0;
}

.right-container > .el-main {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  background: #f8f9fa;
}

.header-left h2 {
  margin: 0;
  font-size: 18px;
  color: #111111;
  font-weight: 600;
  letter-spacing: 1px;
  line-height: 56px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 24px;
}

.time-display {
  color: #888888;
  font-family: 'SF Mono', 'Cascadia Code', 'Consolas', monospace;
  font-size: 14px;
}
</style>
