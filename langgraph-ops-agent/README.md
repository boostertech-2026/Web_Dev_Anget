# Eino 智能运维 Agent

基于 CloudWeGo Eino 框架的智能运维 Agent — 纯 Go 实现，能理解、会推理、可自主行动的 AI 运维助手。

## 项目结构

```
langgraph-ops-agent/
├── web/                          # Vue 3 前端
│   ├── src/
│   │   ├── api/index.js          # API 封装
│   │   ├── components/
│   │   │   └── SshTerminal.vue   # SSH 终端组件
│   │   ├── views/
│   │   │   ├── Login.vue         # 登录页
│   │   │   ├── Main.vue          # 主布局 (侧边栏+顶栏)
│   │   │   ├── Dashboard.vue     # 仪表盘概览
│   │   │   ├── AiAssistant.vue   # AI 智能助手 (SSE 流式对话)
│   │   │   ├── HostManage.vue    # 主机管理 (含 IPMI)
│   │   │   ├── ClientManage.vue  # 客户端管理
│   │   │   ├── TaskCreate.vue    # 创建任务 (含自动化模板)
│   │   │   ├── TaskList.vue      # 任务列表
│   │   │   ├── TaskLogDetail.vue # 任务日志详情
│   │   │   ├── LogViewer.vue     # 实时日志 (WebSocket)
│   │   │   ├── SystemMonitor.vue # 系统监控 (CPU/内存/磁盘/负载)
│   │   │   ├── TrafficMonitor.vue# 流量监控 (网络速率)
│   │   │   ├── AlertCenter.vue   # 告警中心 (规则配置/确认/解决)
│   │   │   └── ErrorHistory.vue  # 错误历史 (统计/处理)
│   │   ├── stores/chat.js        # AI 对话状态管理
│   │   ├── styles/global.css     # 全局暗色主题样式
│   │   ├── router.js             # 路由配置
│   │   └── main.js               # 入口
│   ├── index.html
│   ├── vite.config.js
│   ├── package.json
│   ├── Dockerfile
│   └── .dockerignore
│
├── server-go/                    # Go 后端 (Gin + GORM + SQLite + Eino Agent)
│   ├── main.go                   # 入口
│   ├── models/models.go          # 数据模型 (User/Host/Task/Client/Metric/Alert/AlertRule/ErrorHistory/AuditLog)
│   ├── routes/routes.go          # REST + WebSocket 路由
│   ├── handlers/
│   │   ├── handlers.go           # 主机/任务/客户端/SSH/WebSocket 处理
│   │   ├── agent.go              # Agent SSE 对话 + 仪表盘 + IPMI 处理
│   │   └── monitor.go            # 监控/告警/错误历史处理
│   ├── agent/
│   │   ├── state.go              # Agent 状态定义
│   │   ├── graph.go              # Eino StateGraph 组装 (10 节点)
│   │   ├── nodes.go              # 10 个节点实现 (LLM 推理 + 工具调用 + 备份 + 审计)
│   │   ├── agent_tools.go        # 工具注册 (12 个工具)
│   │   ├── store.go              # GORM 数据存储实现
│   │   ├── util.go               # 工具函数
│   │   ├── helpers.go            # SSH 桥接 (供 handler 调用)
│   │   └── tools/                # 工具实现层
│   │       ├── ssh.go            # SSH 执行工具 (golang crypto/ssh)
│   │       ├── ipmi.go           # IPMI 带外管理工具 (ipmitool)
│   │       ├── monitor.go        # 监控查询工具
│   │       └── knowledge.go      # 知识库检索工具
│   ├── go.mod
│   ├── Dockerfile
│   └── .dockerignore
│
├── client-go/                    # Go 目标机客户端 (长连接执行)
│   ├── main.go
│   ├── go.mod
│   └── .dockerignore
│
├── docker-compose.yml            # Docker 一键部署
├── .env                          # 环境变量
├── start.bat                     # Windows 一键启动脚本
├── install.bat                   # Windows 一键安装脚本
├── LICENSE
└── README.md
```

## 技术栈

| 层 | 技术 | 说明 |
|----|------|------|
| 前端 | Vue 3 + Vite + Element Plus + Axios | 暗色主题，13 个页面 |
| 后端 | Go + Gin + GORM + SQLite + Gorilla WebSocket | REST API + WebSocket + SSE 流式输出 |
| **Agent 大脑** | **Go + CloudWeGo Eino + OpenAI/DeepSeek + crypto/ssh** | **ReAct 模式、Tool Calling、人在回路、纯 Go 无外部依赖** |
| 客户端 | Go 原生编译 | 跨平台、无依赖、长连接保活 |

## 快速启动

### 前置要求

- Node.js 18+
- Go 1.21+

### 1. Go 后端 (8080 端口)

```bash
cd server-go
go mod tidy
go run main.go
```

### 2. Vue 前端 (5173 端口)

```bash
cd web
npm install
npm run dev
```

### 3. Go 客户端 (可选)

```bash
cd client-go
go mod tidy
go run main.go
```

## 访问地址

| 服务 | 地址 |
|------|------|
| Web UI | http://localhost:5173 |
| 后端 API | http://localhost:8080 |
| Agent 对话 | http://localhost:8080/api/agent/chat |

## 默认账号

- 用户名：`admin`
- 密码：`admin123`

## 功能特性

### AI 智能助手
- 自然语言交互，LLM 理解意图 → 制定计划 → 自主执行
- SSE 流式输出：实时展示 Agent 思考过程和工具调用
- 12 个 Tool Calling 工具：SSH、IPMI、监控查询、日志检索、知识库
- 高危操作人在回路审批，确认后自动执行版本快照备份（配置/进程/端口/磁盘）
- 故障案例自动沉淀到知识库
- 操作完成后生成不可篡改的结构化审计记录（谁/何时/主机/操作/结果/备份路径）

### 仪表盘
- 主机/告警统计卡片
- 平均 CPU/内存/磁盘使用率
- 最近告警和最近任务面板

### 主机管理
- SSH 终端直连（窗口可缩放）
- IPMI 带外管理（电源控制/启动设备/传感器/SEL 日志）
- 分组标签、在线状态监控

### 监控中心
- **系统监控**：CPU/内存/磁盘/负载，时间范围可选（5m/1h/24h/7d）
- **流量监控**：网络接收/发送速率，历史数据可视化

### 任务管理
- 12 个自动化运维模板（系统检查/网络/服务/Docker/日志/IPMI）
- 批量执行、定时任务
- WebSocket 实时日志推送

### 告警中心
- 告警规则配置（CPU/内存/磁盘/主机离线），阈值自定义，启用/禁用开关
- 告警处理工作流：触发 → 确认 → 解决
- 状态/级别筛选

### 错误历史
- 统计面板（总数、按级别、问题主机 TOP、平均解决时间）
- 错误处理工作流：待处理 → 处理中 → 已解决

## 页面路由

| 路径 | 页面 | 说明 |
|------|------|------|
| `/login` | 登录页 | |
| `/main/dashboard` | 仪表盘 | 默认首页 |
| `/main/assistant` | AI 助手 | 智能对话 |
| `/main/hosts` | 主机管理 | 含 SSH 终端 + IPMI |
| `/main/clients` | 客户端管理 | |
| `/main/task/create` | 创建任务 | 含自动化模板 |
| `/main/tasks` | 任务列表 | |
| `/main/task/detail/:id` | 任务日志详情 | |
| `/main/task-log` | 实时日志 | WebSocket 推送 |
| `/main/monitor/system` | 系统监控 | CPU/内存/磁盘/负载 |
| `/main/monitor/traffic` | 流量监控 | 网络速率 |
| `/main/alerts` | 告警中心 | 规则配置+处理 |
| `/main/errors` | 错误历史 | 统计+处理 |

## API 接口

### Agent
| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/agent/chat` | POST | AI 对话 (SSE 流式) |

### 仪表盘
| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/dashboard/summary` | GET | 仪表盘概览数据 |

### 主机管理
| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/host` | GET/POST | 列表/新增 |
| `/api/host/:id` | PUT/DELETE | 更新/删除 |
| `/api/host/:id/ipmi` | PUT | 更新 IPMI 配置 |
| `/api/host/:id/ipmi/check` | POST | 检测 IPMI 连通性 |

### 监控
| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/metrics/latest` | GET | 最新指标 (支持 host 筛选) |
| `/api/metrics/history` | GET | 历史指标 (host_id + duration) |
| `/api/metrics/traffic` | GET | 流量数据 (host_id) |

### 告警
| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/alerts` | GET | 告警列表 (支持 status/level 筛选) |
| `/api/alerts/:id/ack` | POST | 确认告警 |
| `/api/alerts/:id/resolve` | POST | 解决告警 |
| `/api/alerts/rules` | GET | 告警规则列表 |
| `/api/alerts/rules/:id` | PUT | 更新规则 |

### 错误历史
| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/errors` | GET | 错误列表 (支持 status 筛选) |
| `/api/errors/stats` | GET | 错误统计 |
| `/api/errors/:id/resolve` | POST | 解决错误 |

### 审计
| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/audit/list` | GET | 审计记录列表 (支持 thread_id/status 筛选) |
| `/api/audit/:id` | GET | 审计记录详情 |

### 其他
| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/login` | POST | 登录 |
| `/api/task/list` | GET | 任务列表 |
| `/api/task/create` | POST | 创建任务 |
| `/api/client/*` | CRUD | 客户端管理 |
| `/api/ssh/connect` | POST | SSH 连接 |
| `/api/ssh/execute` | POST | SSH 执行 |
| `/ws/log` | WS | 实时日志推送 |
| `/ws/client` | WS | 客户端 WebSocket |

## 架构核心：Eino ReAct Agent 循环

基于 CloudWeGo Eino 框架构建的 10 节点 StateGraph，LLM 推理 + Tool Calling + 备份 + 审计全部在 Go 进程内完成，无需跨语言 HTTP 桥接。

```
用户输入 → [LLM 理解意图] → [LLM 制定计划]
        → [选择工具] → [执行] → [观察结果]
        → [LLM 判断: 继续还是完成?]
        → ... (动态循环直到目标达成)
        → [高危操作? → 人工确认 → 版本快照备份 → 执行]
        → [LLM 生成报告] → [保存知识库] → [审计记录] → END
```

10 个 Agent 节点：understand → plan → agent ⇄ tool_call → observe → human_approve → backup → agent → summarize → save_knowledge → audit → END

## 默认告警规则

| 规则 | 条件 | 级别 |
|------|------|------|
| CPU 过高 | CPU > 90% 持续 300s | critical |
| 内存不足 | 内存 > 95% 持续 300s | critical |
| 磁盘空间低 | 磁盘 > 85% 持续 300s | warning |
| 主机离线 | 心跳超时 120s | critical |

## Docker 部署

```bash
docker-compose up -d --build
```

## 技术参数

| 参数 | 值 |
|------|-----|
| 数据库 | ops.db (SQLite) |
| 后端端口 | 8080 |
| 前端端口 | 5173 (dev) |
| 客户端心跳 | 30s |
| 监控采集间隔 | 60s |
| WebSocket 日志缓存 | 1000 条 |

---

**默认管理员账号：admin / admin123**
