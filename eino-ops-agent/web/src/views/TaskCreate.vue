<template>
  <div class="task-create">
    <div class="page-header">
      <div class="header-left">
        <span class="icon">⚡</span>
        <div class="header-text">
          <h2>创建任务</h2>
          <p>Task Creator</p>
        </div>
      </div>
    </div>

    <el-card class="main-card">
      <el-form
        :model="taskForm"
        :rules="rules"
        ref="formRef"
        label-width="120px"
        class="tech-form"
      >
        <el-form-item label="任务名称" prop="name">
          <el-input v-model="taskForm.name" placeholder="请输入任务名称" />
        </el-form-item>
        <el-form-item label="执行方式" prop="exec_type">
          <el-radio-group v-model="taskForm.exec_type">
            <el-radio label="ssh">SSH直连</el-radio>
            <el-radio label="client">客户端</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item
          label="选择主机"
          prop="host_ids"
          v-if="taskForm.exec_type === 'ssh'"
        >
          <el-select
            v-model="taskForm.host_ids"
            multiple
            placeholder="请选择主机"
            style="width: 100%"
          >
            <el-option
              v-for="host in hosts"
              :key="host.id"
              :label="host.name"
              :value="host.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item
          label="选择客户端"
          prop="client_ids"
          v-if="taskForm.exec_type === 'client'"
        >
          <el-select
            v-model="taskForm.client_ids"
            multiple
            placeholder="请选择客户端"
            style="width: 100%"
          >
            <el-option
              v-for="client in clients"
              :key="client.id"
              :label="client.name"
              :value="client.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="快速模板">
          <div class="template-grid">
            <div
              v-for="tpl in templates"
              :key="tpl.name"
              class="template-card"
              :class="{ active: activeTemplate === tpl.name }"
              @click="applyTemplate(tpl)"
            >
              <span class="tpl-icon">{{ tpl.icon }}</span>
              <div class="tpl-info">
                <span class="tpl-name">{{ tpl.name }}</span>
                <span class="tpl-desc">{{ tpl.desc }}</span>
              </div>
            </div>
          </div>
        </el-form-item>
        <el-form-item label="执行命令" prop="command">
          <el-input
            v-model="taskForm.command"
            type="textarea"
            :rows="6"
            placeholder="请输入要执行的命令，或点击上方模板自动填充"
            class="command-textarea"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleCreate" :loading="loading" class="submit-btn">
            创建任务
          </el-button>
          <el-button class="reset-btn" @click="resetForm">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import { getHosts, getClients, createTask } from "../api";

const router = useRouter();
const formRef = ref();
const loading = ref(false);
const hosts = ref([]);
const clients = ref([]);

const taskForm = reactive({
  name: "",
  exec_type: "ssh",
  host_ids: [],
  client_ids: [],
  command: "",
});

const rules = {
  name: [{ required: true, message: "请输入任务名称", trigger: "blur" }],
  exec_type: [{ required: true, message: "请选择执行方式", trigger: "change" }],
  host_ids: [
    {
      validator: (_rule, value, callback) => {
        if (taskForm.exec_type === "ssh" && (!value || value.length === 0)) {
          callback(new Error("请选择至少一台主机"));
        } else {
          callback();
        }
      },
      trigger: "change",
    },
  ],
  client_ids: [
    {
      validator: (_rule, value, callback) => {
        if (taskForm.exec_type === "client" && (!value || value.length === 0)) {
          callback(new Error("请选择至少一个客户端"));
        } else {
          callback();
        }
      },
      trigger: "change",
    },
  ],
  command: [{ required: true, message: "请输入执行命令", trigger: "change" }],
};

const activeTemplate = ref("");

const templates = [
  { icon: "🖥️", name: "系统信息", desc: "uname + 发行版", cmd: "uname -a && cat /etc/os-release" },
  { icon: "💾", name: "磁盘使用", desc: "磁盘空间检查", cmd: "df -h" },
  { icon: "🧠", name: "内存使用", desc: "内存用量统计", cmd: "free -h" },
  { icon: "⚙️", name: "CPU 信息", desc: "CPU + 负载", cmd: "lscpu && uptime" },
  { icon: "📋", name: "进程列表", desc: "CPU 占用 TOP20", cmd: "ps aux --sort=-%cpu | head -20" },
  { icon: "🌐", name: "网络信息", desc: "IP + 端口监听", cmd: "ip addr show && ss -tlnp" },
  { icon: "🔧", name: "服务状态", desc: "运行中的服务", cmd: "systemctl list-units --type=service --state=running" },
  { icon: "🐳", name: "Docker 状态", desc: "容器 + 资源占用", cmd: "docker ps -a && docker stats --no-stream" },
  { icon: "📜", name: "系统日志", desc: "最近 50 条日志", cmd: "journalctl -n 50 --no-pager" },
  { icon: "🔒", name: "安全审计", desc: "登录记录 + 认证日志", cmd: "last -20; grep -E '(Failed|Accepted)' /var/log/auth.log 2>/dev/null | tail -30 || echo '无auth.log'" },
  { icon: "⏰", name: "定时任务", desc: "crontab 列表", cmd: "for u in $(cut -d: -f1 /etc/passwd); do echo \"=== $u ===\"; crontab -u $u -l 2>/dev/null; done" },
  { icon: "🔌", name: "端口扫描", desc: "本机开放端口", cmd: "ss -tlnp 2>/dev/null || netstat -tlnp 2>/dev/null" },
];

const applyTemplate = (tpl) => {
  taskForm.name = tpl.name;
  taskForm.command = tpl.cmd;
  activeTemplate.value = tpl.name;
};

const loadData = async () => {
  try {
    const [hostRes, clientRes] = await Promise.all([getHosts(), getClients()]);
    hosts.value = hostRes.data || [];
    clients.value = clientRes.data || [];
  } catch (err) {
    ElMessage.error("获取数据失败");
  }
};

const handleCreate = async () => {
  const valid = await formRef.value.validate().catch(() => false);
  if (!valid) return;

  loading.value = true;
  try {
    const data = {
      name: taskForm.name,
      exec_type: taskForm.exec_type,
      command: taskForm.command,
    };
    if (taskForm.exec_type === "ssh") {
      data.host_ids = taskForm.host_ids;
    } else {
      data.client_ids = taskForm.client_ids;
    }
    await createTask(data);
    ElMessage.success("任务创建成功");
    router.push("/main/tasks");
  } catch (err) {
    ElMessage.error(err.message || "创建失败");
  } finally {
    loading.value = false;
  }
};

const resetForm = () => {
  formRef.value.resetFields();
  taskForm.host_ids = [];
  taskForm.client_ids = [];
};

onMounted(() => {
  loadData();
});

watch(
  () => taskForm.exec_type,
  () => {
    taskForm.host_ids = [];
    taskForm.client_ids = [];
  },
);
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
  max-width: 800px;
}

.template-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
  width: 100%;
}

.template-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  background: #ffffff;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.template-card:hover {
  border-color: #111111;
}

.template-card.active {
  border-color: #111111;
  background: #f5f5f5;
}

.tpl-icon {
  font-size: 22px;
  flex-shrink: 0;
}

.tpl-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.tpl-name {
  color: #333333;
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
}

.tpl-desc {
  color: #888888;
  font-size: 11px;
  white-space: nowrap;
}

.command-textarea :deep(.el-textarea__inner) {
  font-family: 'Consolas', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
}
</style>
