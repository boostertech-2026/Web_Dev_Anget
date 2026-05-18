# Eino 智能运维 Agent 项目文档（v1.0 — 纯 Go 架构）

## 一、项目概述

### 1.1 项目名称

**Eino 智能运维 Agent** —— 能理解、会推理、可自主行动的 AI 运维助手（纯 Go 实现）

### 1.2 核心命题

传统运维平台是"工具"——人告诉它做什么，它照做。
**本项目要做的是"助手"**——人告诉它目标，它自己思考怎么达成。

```
传统运维：  人 → 分析问题 → 找命令 → 选机器 → 执行 → 检查结果
本Agent：  人 → "帮我把问题修好" → Agent → 诊断 → 规划 → 执行 → 验证 → 报告
```

### 1.3 Eino 智能 Agent 的核心能力

Eino 是字节跳动 CloudWeGo 开源的 Go 语言 LLM 应用框架，对标 LangGraph。本项目选择 Eino 的核心理由：

相比于 Python LangGraph 方案，纯 Go 实现的优势：

| 能力             | 说明                                         | 本项目应用                                 |
| ---------------- | -------------------------------------------- | ------------------------------------------ |
| **LLM 推理驱动** | Agent 调用大模型理解意图、分析问题、制定计划 | 用户自然语言下指令，Agent 自主拆解         |
| **Tool Calling** | Agent 自主选择调用哪些工具、按什么顺序       | 根据问题类型自动选择 SSH/IPMI/API/监控查询 |
| **动态路由**     | 不是固定分支，根据中间结果动态调整下一步     | 执行失败 → 自动换方案，而非直接报错        |
| **持久化记忆**   | Checkpointer 保存状态，跨会话记住上下文      | 知道某台机器上次出过什么问题               |
| **人在回路**     | 危险操作前暂停等待人工确认                   | 重启生产服务前弹出审批                     |
| **流式输出**     | 每步执行实时推送到前端                       | 用户看到 Agent 思考过程和工具调用的每一步  |
| **故障案例沉淀** | 解决后自动提取知识存入知识库                 | 下次相似问题直接参考历史方案               |
| **子图嵌套**     | 复杂流程封装为可复用子图                     | 故障诊断子图、系统重装子图等               |

### 1.4 核心体验

用户在聊天框输入自然语言，Agent 自主完成：

```
用户： "web-05 响应很慢，帮我看看怎么回事"

Agent 自主执行：
  思考: "响应慢可能的原因有 CPU高/内存不足/磁盘IO/网络/应用层问题"
  步骤1: 查监控数据 → CPU 94%, 内存正常, 磁盘正常
  步骤2: 查进程TOP → java 进程占用 89% CPU
  步骤3: 查 java 应用日志 → 大量 Full GC 日志
  步骤4: 查近24h部署记录 → 2h前有新版本上线
  结论: 新版本可能存在性能问题
  建议: 回滚到上一版本，或增加 JVM 堆内存
  确认: "是否执行回滚操作？[是] [否] [先加内存]"
```

```
用户： "所有生产服务器检查一下磁盘，快满的发告警"

Agent 自主执行：
  思考: "需要查询所有标签为 production 的主机的磁盘使用率"
  步骤1: 查询主机列表 → 找到 8 台 production 主机
  步骤2: 并行 SSH 执行 df -h → 7台正常, 1台 /data 分区 92%
  步骤3: 分析大文件 → /data/logs/ 占 320GB
  步骤4: 检查日志轮转配置 → logrotate 未配置该目录
  步骤5: 自动配置 logrotate + 压缩旧日志
  步骤6: 推送告警通知
  报告: "db-03 /data 已从 92% 降至 58%，logrotate 已配置。共释放 180GB"
```

### 1.5 技术选型

| 层             | 技术栈                                                              | 说明                                 |
| -------------- | ------------------------------------------------------------------- | ------------------------------------ |
| 前端           | Vue 3 + Vite + Element Plus + Axios + CSS 图表                      | 暗色主题，13 个页面，SSE 流式对话    |
| 后端           | Go + Gin + GORM + SQLite + Gorilla WebSocket                        | REST API + WebSocket + SSE           |
| **Agent 大脑** | **Go + CloudWeGo Eino + OpenAI/DeepSeek + crypto/ssh** | **LLM 驱动、Tool Calling、人在回路、纯 Go 无外部依赖** |
| 客户端         | Go 原生编译（跨平台、无依赖）                                       | 内网机器长连接执行                   |
| 部署           | Docker Compose 一键部署                                             | 三服务容器化（Agent 已整合进后端）   |

### 1.6 功能矩阵

| 模块         | 传统平台能做到 | 本 Agent 额外能做到                                                                |
| ------------ | -------------- | ---------------------------------------------------------------------------------- |
| **智能对话** | -              | 自然语言交互，多轮对话，理解上下文                                                 |
| **故障诊断** | 阈值告警       | 自动关联分析（监控+日志+变更记录），给出根因                                       |
| **自主修复** | 手动执行脚本   | Agent 诊断后自动规划修复方案，人工确认后执行                                       |
| **知识积累** | -              | 每次故障的诊断过程和处理结果存入知识库，下次更快                                   |
| **任务编排** | 固定模板       | Agent 根据目标动态编排步骤，失败自动换方案                                         |
| **批量操作** | 选机器→跑命令  | "把上周五之后部署过的机器上的 nginx 都重启" → Agent 自己查部署记录、定位机器、执行 |
| **安全审批** | -              | 高危操作（重启/关机/重装）强制人工确认，确认后自动备份                             |
| **版本快照** | 人工手动备份   | 高危操作前自动备份目标主机状态（配置/进程/端口/磁盘），可用于回滚验证              |
| **审计追溯** | 操作日志散落   | 每次 Agent 操作自动生成结构化审计记录（谁/何时/对哪些主机/做了什么/结果/备份路径） |
| **报告生成** | -              | 操作完成后自动生成结构化报告（做了什么、为什么、结果如何）                         |

---

## 二、系统架构

### 2.1 整体架构图

```
                      ┌──────────────────────────┐
                      │      LLM 大模型           │
                      │  (OpenAI / DeepSeek)     │
                      │  推理 · 规划 · 决策       │
                      └───────────┬──────────────┘
                                  │ API 调用
                                  ▼
┌──────────────────────────────────────────────────────────────────┐
│                   Web 前端 (Vue 3)                                 │
│  AI对话 │ 仪表盘 │ 主机管理 │ 监控 │ 流量 │ 任务 │ 日志 │ 告警    │
└────┬─────────────┬────────────────┬──────────────┬───────────────┘
     │ HTTP/SSE    │ WebSocket      │ REST API     │ ECharts 轮询
     ▼             ▼                ▼              ▼
┌──────────────────────────────────────────────────────────────────┐
│                    Go 后端服务 (Gin + GORM)                        │
│  API路由 │ JWT鉴权 │ 任务调度 │ WebSocket Hub │ SSE推送 │ 数据持久化 │
└──────────┬────────────────────────────────────────┬──────────────┘
           │ 内部调用                                  │ TCP 长连接
           ▼                                           ▼
┌────────────────────────────────┐      ┌──────────────────────────┐
│   Eino 智能 Agent (大脑)         │      │   Go 本地客户端            │
│   已整合进 Go 后端进程            │      │   (目标机器部署)            │
│                                │      │                          │
│  ┌──────────────────────────┐  │      │  · 心跳保活 (30s)         │
│  │ LLM 推理引擎              │  │      │  · 命令本地执行            │
│  │ · 意图理解               │  │      │  · 日志实时回传            │
│  │ · 任务规划               │  │      │  · 监控指标上报            │
│  │ · 动态决策               │  │      └──────────────────────────┘
│  └──────────────────────────┘  │
│  ┌──────────────────────────┐  │
│  │ Tool Calling 工具层       │  │
│  │ · SSH 工具 (crypto/ssh)  │  │
│  │ · IPMI 工具 (ipmitool)   │  │
│  │ · 监控查询工具            │  │
│  │ · 日志查询工具            │  │
│  │ · 知识库工具            │  │
│  └──────────────────────────┘  │
│  ┌──────────────────────────┐  │
│  │ 持久化层                  │  │
│  │ · GORM 直接读写 SQLite    │  │
│  │ · 知识库 (故障案例积累)    │  │
│  │ · 审计日志 (audit_logs)   │  │
│  │ · 短期 + 长期记忆         │  │
│  └──────────────────────────┘  │
│  ┌──────────────────────────┐  │
│  │ 安全防护层 (NEW)          │  │
│  │ · 版本快照备份 (Backup)   │  │
│  │ · 操作审计追溯 (Audit)    │  │
│  └──────────────────────────┘  │
└────────────┬───────────────────┘
             │ SSH / IPMI / API
             ▼
┌──────────────────────────────────────────────────────────────────┐
│                        目标服务器集群                              │
│   Linux / Windows  │  IPMI/BMC  │  交换机/网络设备                │
└──────────────────────────────────────────────────────────────────┘
```

### 2.2 核心数据流

```
用户输入自然语言
       │
       ▼
  ┌─────────────────────────────────────────────┐
  │ Agent 推理循环 (Eino StateGraph)             │
  │                                             │
  │  ① 理解意图 (LLM)                            │
  │     "web-05很慢" → {意图:故障诊断, 目标:web-05, 症状:响应慢}
  │                                             │
  │  ② 制定计划 (LLM)                            │
  │     步骤1: 查监控 → 步骤2: 查进程 → 步骤3: 查日志 → ...
  │                                             │
  │  ③ 执行步骤 (Tool Calling)                   │
  │     调用 check_monitor("web-05")             │
  │     → CPU 94%, 内存正常                      │
  │     调用 exec_ssh("web-05", "ps aux")        │
  │     → java 进程占 89%                        │
  │     调用 query_logs("web-05", "java")        │
  │     → Full GC 频繁                           │
  │                                             │
  │  ④ 观察结果 → 调整计划 (LLM 反思)             │
  │     发现: CPU 高 + java 异常 + GC日志         │
  │     新步骤: 查部署历史 → 发现2h前上线          │
  │                                             │
  │  ⑤ 生成结论与建议 (LLM)                       │
  │     "新版本可能存在性能问题，建议回滚"          │
  │                                             │
  │  ⑥ 人在回路确认 (高危操作)                    │
  │     等待用户点击 [确认] / [取消]               │
  │                                             │
  │  ⑦ 版本快照备份 (Backup)                     │
  │     确认后自动备份目标主机：配置(/etc)、进程、  │
  │     端口、磁盘使用率 → /var/backup/agent/     │
  │                                             │
  │  ⑧ 执行修复 + 验证                           │
  │     回滚 → 检查监控 → CPU 降至 23%            │
  │                                             │
  │  ⑨ 保存到知识库                              │
  │     {症状:响应慢, 根因:新版本FullGC, 修复:回滚} │
  │                                             │
  │  ⑩ 审计记录 (Audit)                         │
  │     记录: 谁/何时/对哪些主机/做了什么/结果/    │
  │     备份路径/是否人工确认 → audit_logs 表      │
  │                                             │
  │ 每一步都通过 SSE 实时推送到前端                 │
  └─────────────────────────────────────────────┘
```

### 2.3 智能体 vs 传统管道 对比

```
传统管道 (v2.0):
  parse → route → execute → validate → summarize → END
  (固定5步，无LLM参与，等价于高级if-else)

Eino 智能 Agent (v3.0):
  输入 → [LLM理解意图] → [LLM制定计划]
       → [选择工具A] → [执行] → [观察结果]
       → [LLM判断: 成功? 失败? 需要更多信息?]
       → [选择工具B] → [执行] → [观察结果]
       → ... (动态循环, 直到目标达成)
       → [高危操作? → 人工确认 → 版本快照备份]
       → [LLM生成报告] → [保存知识] → [审计记录] → END

  差异: 步数不固定, 路径动态决定, LLM参与每一步推理。Agent 已整合进 Go 后端，无需单独部署。
  高危操作前自动备份目标主机状态（配置/进程/端口/磁盘），事后生成不可篡改的审计记录。
```

---

## 三、项目目录结构

```
langgraph-ops-agent/
├── web/                    # Vue 3 前端
│   ├── src/
│   │   ├── api/index.js    # API 封装 (axios 拦截器 + 全部接口)
│   │   ├── components/     # 公共组件
│   │   │   └── SshTerminal.vue       # SSH 终端 (可缩放窗口)
│   │   ├── views/          # 页面组件 (13 个)
│   │   │   ├── Login.vue            # 登录页
│   │   │   ├── Dashboard.vue        # 仪表盘概览 (统计卡片+最近告警/任务)
│   │   │   ├── AiAssistant.vue      # AI 助手对话页 (SSE 流式, Markdown 渲染)
│   │   │   ├── Main.vue             # 主布局 (侧边栏+顶栏, 11 项导航)
│   │   │   ├── HostManage.vue       # 主机管理 (含 SSH 终端 + IPMI 配置)
│   │   │   ├── ClientManage.vue     # 客户端管理
│   │   │   ├── TaskCreate.vue       # 创建任务 (12 个自动化模板)
│   │   │   ├── TaskList.vue         # 任务列表
│   │   │   ├── TaskLogDetail.vue    # 任务日志详情
│   │   │   ├── LogViewer.vue        # 实时日志 (WebSocket)
│   │   │   ├── SystemMonitor.vue    # 系统监控 (CPU/内存/磁盘/负载, 颜色阈值)
│   │   │   ├── TrafficMonitor.vue   # 流量监控 (网络速率, 可视化占比条)
│   │   │   ├── AlertCenter.vue      # 告警中心 (规则配置/确认/解决)
│   │   │   └── ErrorHistory.vue     # 错误历史 (统计面板+处理工作流)
│   │   ├── stores/chat.js  # AI 对话状态管理 (Vue reactive 组合式)
│   │   ├── styles/global.css # 全局暗色主题 (Element Plus CSS 变量覆盖)
│   │   ├── router.js       # 路由配置 (13 条路由)
│   │   └── main.js         # 入口
│   ├── index.html
│   ├── vite.config.js
│   ├── package.json
│   ├── Dockerfile
│   └── .dockerignore
│
├── server-go/              # Go 后端 (Gin + GORM + Eino)
│   ├── main.go
│   ├── models/models.go    # 8 个数据模型 (GORM AutoMigrate)
│   ├── routes/routes.go    # REST + WebSocket 路由 (30+ 端点)
│   ├── handlers/
│   │   ├── handlers.go     # 主机/任务/客户端/SSH/WebSocket/认证
│   │   ├── agent.go        # Agent SSE 流式对话/仪表盘/IPMI
│   │   └── monitor.go      # 监控指标/告警/错误历史
│   ├── agent/              # Eino 智能 Agent (纯 Go，12 个工具)
│   │   ├── state.go        # Agent 状态定义
│   │   ├── graph.go        # Eino StateGraph 组装 (10 节点)
│   │   ├── nodes.go        # 10 个节点实现 (LLM 推理 + Tool Calling + Backup + Audit)
│   │   ├── agent_tools.go  # 工具注册 (12 个工具)
│   │   ├── store.go        # GORM 数据存储实现 (SSH/IPMI/KB/Monitor)
│   │   ├── util.go         # 工具函数
│   │   ├── helpers.go      # SSH 桥接 (供 handler 调用)
│   │   └── tools/          # 工具实现层
│   │       ├── ssh.go      # SSH 执行 (golang crypto/ssh)
│   │       ├── ipmi.go     # IPMI 带外管理 (ipmitool)
│   │       ├── monitor.go  # 监控查询 + 日志检索
│   │       └── knowledge.go# 知识库 (SQLite 故障案例)
│   ├── go.mod
│   ├── Dockerfile
│   └── .dockerignore
│
├── client-go/              # 目标机客户端
│   ├── main.go
│   ├── go.mod
│   └── .dockerignore
│
├── docker-compose.yml
├── .env
├── start.bat               # Windows 一键启动
├── install.bat             # Windows 一键安装
├── LICENSE
└── README.md
```

---

## 四、页面设计详情

### 4.1 Dashboard 仪表盘（新增）

登录后首页，全局运维状态一目了然。

```
┌──────────────────────────────────────────────────────────────┐
│  📊 总主机 12    ●在线 8    ○离线 3    ⚠告警 1               │
│                                                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐│
│  │ CPU 平均  │ │ 内存平均  │ │ 磁盘平均  │ │ 今日告警          ││
│  │  34.2%   │ │  62.7%   │ │  51.3%   │ │  🔴 3  🟡 5  🟢 2 ││
│  │  ↑ 2.1%  │ │  ↓ 1.5%  │ │  → 0.3%  │ │  较昨日 +2         ││
│  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘│
│                                                              │
│  ┌──────────────────────┐ ┌────────────────────────────────┐ │
│  │ 最近告警 (5)          │ │ 最近任务 (5)                    │ │
│  │ 🔴 prod-01 CPU 94%    │ │ ✅ 批量巡检→3台      10:30    │ │
│  │    2 分钟前            │ │ 🔄 安装Nginx→web-05  10:28    │ │
│  │ 🟡 db-02 磁盘 82%     │ │ ❌ 重启服务→db-01    10:15    │ │
│  │    15 分钟前           │ │ ✅ 日志清理→5台      09:00    │ │
│  │ 🟡 web-03 内存 88%    │ │ 🔄 IPMI重置→storage  08:45    │ │
│  │    32 分钟前           │ │                              │ │
│  └──────────────────────┘ └────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

**核心指标卡片**：每张卡片显示当前值 + 环比变化箭头 + 迷你趋势线。

### 4.2 主机管理（重构）

**功能要点**：

| 功能      | 说明                                                 |
| --------- | ---------------------------------------------------- |
| 主机列表  | ID、名称、IP、SSH 端口、分组标签、在线状态、SSH 状态 |
| IPMI 信息 | IPMI 地址、账号、密码、BMC 版本、IPMI 可达性         |
| 分组筛选  | 按环境（生产/测试）、地域、角色标签筛选              |
| 批量操作  | 多选执行命令、批量 IPMI 操作                         |
| SSH 终端  | 弹窗直连终端，支持窗口缩放                           |
| 操作记录  | 每台主机最近操作时间与结果                           |

**IPMI 信息区域**：

```
┌─────────────────────────────────────────────────────┐
│  IPMI 配置                                          │
│  IPMI 地址: 192.168.1.100   账号: admin             │
│  密码: ********            BMC 版本: 3.42.00        │
│  状态: ● 可达             最后检测: 2026-05-10 10:30│
│  ┌─────────────────────────────────────────────┐    │
│  │ ⚠ 该主机尚未配置 IPMI 信息，点击此处添加       │    │
│  │   配置 IPMI 后可进行带外管理操作                │    │
│  └─────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────┘
```

**数据模型扩展**：

```
Host 表新增字段：
  - ipmi_host       VARCHAR   IPMI 地址
  - ipmi_user       VARCHAR   IPMI 账号
  - ipmi_password   VARCHAR   IPMI 密码
  - ipmi_status     VARCHAR   IPMI 状态 (online/offline/not_configured)
  - bmc_version     VARCHAR   BMC 固件版本
  - group_tag       VARCHAR   分组标签 (production/testing/development)
  - region_tag      VARCHAR   地域标签
  - last_operated   DATETIME  最近操作时间
```

### 4.3 系统监控（新增）

**功能要点**：

- **概览卡片**：CPU/内存/磁盘使用率 + 系统负载，颜色阈值告警（绿/黄/红）
- **历史表格**：CPU/内存/磁盘/负载/网络/进程 TOP 完整时间序列
- **时间范围切换**：5 分钟 / 1 小时 / 24 小时 / 7 天
- **主机切换**：下拉选择不同主机查看各自监控数据
- **CSS 进度条**：纯 CSS 实现使用率可视化，绿色/黄色/红色渐变
- **数据持久化**：通过 Agent 工具采集，存入 metrics 表

**页面布局**：

```
┌──────────────────────────────────────────────────────────┐
│  主机选择: [prod-01 ▼]  时间范围: [1小时]  [自动刷新: 10s]│
├──────────┬──────────┬──────────┬──────────┬─────────────┤
│ CPU 使用率│ 内存使用率 │ 磁盘使用率 │ 系统负载  │ 网络流量    │
│  42.5%   │  67.8%   │  53.2%   │  1.24    │  ↑12 ↓8    │
│  ▓▓▓▓░░  │  ▓▓▓▓▓░  │  ▓▓▓░░░  │  ▓▓▓░░░  │  ▓▓▓░░░    │
└──────────┴──────────┴──────────┴──────────┴─────────────┘
┌──────────────────────────┐ ┌────────────────────────────┐
│     CPU 使用率趋势        │ │      磁盘各分区占用         │
│  60│    ╱╲               │ │     ┌──────────┐           │
│  40│╱──╱──╲──╱──         │ │    ╱  /      50%          │
│  20│╱          ╲         │ │   ▓  /home   30%          │
│    └────────────────      │ │   ░  /boot   12%          │
│     08  09  10  11  12    │ │   ░  /var     8%          │
│                           │ │     └──────────┘           │
├───────────────────────────┤ ├────────────────────────────┤
│    内存使用趋势            │ │    进程 TOP 5 (按CPU)       │
│  80│    ╱╲    ╱╲         │ │  1. java    34.2%  2.1GB  │
│  60│╱──╱──╲──╱──╲──      │ │  2. mysqld  18.7%  1.2GB  │
│  40│╱          ╲         │ │  3. nginx    8.3%  256MB  │
│    └────────────────      │ │  4. python   5.1%  512MB  │
│     08  09  10  11  12    │ │  5. redis    3.2%  128MB  │
└───────────────────────────┘ └────────────────────────────┘
```

### 4.4 流量监控（新增）

**功能要点**：

- **速率卡片**：接收/发送速率实时显示，累计流量统计
- **历史表格**：时间/接收速率/发送速率/累计流量/速率占比可视化条
- **主机切换**：下拉选择不同主机查看各自流量数据
- **CSS 占比条**：绿色(接收)/橙色(发送) 双色占比条，直观对比

**技术方案**：

- 监控数据通过 Agent 工具查询或客户端上报，存入 metrics 表
- 前端通过 API 拉取最近 1 小时数据（最多 200 条）
- CSS 实现速率占比可视化，无第三方图表库依赖

### 4.5 自动化任务管理（重构）

**4.5.1 运维模板**

| 分类          | 模板          | 命令/操作                                             | 执行方式 |
| ------------- | ------------- | ----------------------------------------------------- | -------- |
| **系统检查**  | 系统信息      | `uname -a && cat /etc/os-release`                     | SSH      |
|               | 磁盘使用      | `df -h`                                               | SSH      |
|               | 内存使用      | `free -h`                                             | SSH      |
|               | CPU 信息      | `lscpu && uptime`                                     | SSH      |
|               | 进程列表      | `ps aux --sort=-%cpu \| head -20`                     | SSH      |
| **网络检查**  | 网络信息      | `ip addr show && ss -tlnp`                            | SSH      |
|               | 端口扫描      | `ss -tlnp`                                            | SSH      |
| **服务管理**  | 服务状态      | `systemctl list-units --type=service --state=running` | SSH      |
|               | 重启服务      | `systemctl restart <service>`                         | SSH      |
| **容器管理**  | Docker 状态   | `docker ps -a && docker stats --no-stream`            | SSH      |
| **日志检查**  | 系统日志      | `journalctl -n 50 --no-pager`                         | SSH      |
|               | 安全审计      | 登录失败记录 + 认证日志                               | SSH      |
|               | 定时任务      | 遍历 crontab                                          | SSH      |
| **IPMI 操作** | 重装系统      | PXE 启动 + 远程挂载 ISO                               | IPMI     |
|               | 重置 BMC 密码 | `ipmitool user set password`                          | IPMI     |
|               | 电源控制      | 开机/关机/强制重启/状态查询                           | IPMI     |
|               | 设置启动顺序  | BIOS/PXE/CDROM/硬盘                                   | IPMI     |
|               | 查询硬件信息  | FRU、传感器、SEL 日志                                 | IPMI     |
|               | BIOS 配置导出 | 导出当前 BIOS 配置                                    | IPMI     |

**4.5.2 任务编排**

支持将多个原子操作串联为执行流程：

```
示例：紧急故障恢复流程
  1. 查询主机电源状态 (IPMI)
  2. 强制关机 (IPMI)
  3. 等待 10 秒
  4. 开机 (IPMI)
  5. 等待系统启动 (ping 检测, 最多 120 秒)
  6. SSH 验证服务状态
  7. 发送完成通知
```

### 4.6 实时日志（增强）

**功能要点**：

- WebSocket 实时推送，支持自动重连
- **筛选维度**：主机、任务 ID、日志级别（INFO/WARN/ERROR）
- **关键字高亮**：error/failed/exception 红色标注，warning 黄色标注
- **时间戳**：精确到毫秒
- **自动滚动**：可暂停/恢复
- **日志下载**：导出为 .log 或 .csv 文件
- **多主机聚合**：同一任务的多个主机日志合并在同一视图

### 4.7 告警中心（新增）

**告警规则配置**：

| 规则       | 条件                  | 级别    | 通知渠道    |
| ---------- | --------------------- | ------- | ----------- |
| CPU 过高   | CPU > 90% 持续 5 分钟 | 严重 🔴 | 钉钉 + 企微 |
| 内存不足   | 内存 > 95%            | 严重 🔴 | 钉钉        |
| 磁盘空间低 | 磁盘 > 85%            | 警告 🟡 | 企微        |
| 主机离线   | 心跳超时 120 秒       | 严重 🔴 | 钉钉 + 企微 |
| 流量异常   | 突发流量 > 基线 3x    | 警告 🟡 | 企微        |
| 服务停止   | systemd 服务状态变化  | 严重 🔴 | 钉钉        |
| 任务失败   | 任务执行返回 failed   | 信息 🔵 | 站内通知    |

**告警处理工作流**：

```
触发告警 → 推送通知 → 值班人员认领 → 处理中 →
解决确认 → 填写原因 → 关闭 → 计入历史统计
```

### 4.8 历史报错记录（新增）

**统计面板**：

- 错误趋势折线图（按天/周/月统计）
- TOP 10 错误类型
- TOP 10 故障主机
- 平均修复时间 (MTTR) 统计

**记录字段**：时间、主机、错误内容、严重级别、状态（待处理/处理中/已解决）、处理人、处理备注、修复用时

---

## 五、路由设计

| 路径                 | 页面           | 权限   |
| -------------------- | -------------- | ------ |
| `/login`             | 登录页         | 公开   |
| `/dashboard`         | 仪表盘概览     | 登录   |
| `/hosts`             | 主机管理       | 登录   |
| `/hosts/:id/monitor` | 单主机监控详情 | 登录   |
| `/monitor/system`    | 系统监控       | 登录   |
| `/monitor/traffic`   | 流量监控       | 登录   |
| `/tasks/create`      | 创建任务       | 登录   |
| `/tasks`             | 任务列表       | 登录   |
| `/tasks/:id/log`     | 任务日志详情   | 登录   |
| `/logs`              | 实时日志       | 登录   |
| `/alerts`            | 告警中心       | 登录   |
| `/alerts/rules`      | 告警规则配置   | 管理员 |
| `/errors`            | 历史报错记录   | 登录   |

---

## 六、后端 API 设计

### 6.1 新增接口

| 接口                       | 方法 | 说明                              | 状态   |
| -------------------------- | ---- | --------------------------------- | ------ |
| `/api/agent/chat`          | POST | AI 对话 (SSE 流式)                | 已实现 |
| `/api/dashboard/summary`   | GET  | 仪表盘概览数据                    | 已实现 |
| `/api/host/:id/ipmi`       | PUT  | 更新主机 IPMI 配置                | 已实现 |
| `/api/host/:id/ipmi/check` | POST | 检测 IPMI 连通性                  | 已实现 |
| `/api/metrics/latest`      | GET  | 最新监控数据 (支持 host 筛选)     | 已实现 |
| `/api/metrics/history`     | GET  | 历史监控数据 (host_id + duration) | 已实现 |
| `/api/metrics/traffic`     | GET  | 流量监控数据 (host_id)            | 已实现 |
| `/api/alerts`              | GET  | 告警列表 (status/level 筛选)      | 已实现 |
| `/api/alerts/:id/ack`      | POST | 确认告警                          | 已实现 |
| `/api/alerts/:id/resolve`  | POST | 解决告警                          | 已实现 |
| `/api/alerts/rules`        | GET  | 告警规则列表                      | 已实现 |
| `/api/alerts/rules/:id`    | PUT  | 更新告警规则                      | 已实现 |
| `/api/errors`              | GET  | 历史报错列表 (status 筛选)        | 已实现 |
| `/api/errors/stats`        | GET  | 报错统计数据                      | 已实现 |
| `/api/errors/:id/resolve`  | POST | 解决错误                          | 已实现 |
| `/api/audit/list`          | GET  | 审计记录列表 (thread_id/status)   | 已实现 |
| `/api/audit/:id`           | GET  | 审计记录详情                      | 已实现 |

### 6.2 数据库表

所有表通过 GORM AutoMigrate 自动创建/迁移，无需手动执行 SQL。核心表：

- **hosts** — 主机信息 (含 IPMI 字段、分组/地域标签)
- **metrics** — 监控指标 (CPU/内存/磁盘/负载/网络/进程TOP)
- **alerts** — 告警记录 (firing/acked/resolved 工作流)
- **alert_rules** — 告警规则 (metric + condition + threshold + duration)
- **error_histories** — 错误历史 (pending/processing/resolved 工作流)
- **audit_logs** — 审计日志 (thread_id/intent/plan/tools_called/high_risk_ops/approved/backup_path/hosts_affected/final_result)
- **knowledge_cases** — 知识库案例 (symptoms/diagnosis/root_cause/solution)
- **tasks** — 任务记录
- **clients** — 客户端
- **users** — 用户

---

## 七、Eino 智能 Agent 设计（核心）

这是整个项目区别于传统运维平台的**关键差异化设计**。Agent 全部用 Go 实现，与后端共享同一进程和数据库连接，无需跨语言 HTTP 桥接。

### 7.1 Agent 架构：ReAct 模式 + Tool Calling

```
┌─────────────────────────────────────────────────────────┐
│                   Eino StateGraph                        │
│                                                         │
│  State = {                                              │
│    messages: []          # 完整对话历史                   │
│    user_intent: str      # LLM 解析的用户意图             │
│    plan: []              # 当前执行计划                   │
│    current_step: int     # 当前步骤序号                   │
│    tool_results: []      # 工具执行结果                   │
│    observations: []      # 中间观察与发现                 │
│    final_answer: str     # 最终回复                      │
│    require_approval: bool # 是否需要人工审批              │
│    approved: bool        # 人工确认已通过                 │
│    backup_path: str      # 版本快照备份路径               │
│    backup_results: str   # 备份结果摘要                   │
│    knowledge_context: [] # 从知识库检索的相关历史案例      │
│  }                                                      │
│                                                         │
│  节点流程:                                               │
│                                                         │
│  ┌──────────┐     ┌──────────────┐     ┌─────────────┐  │
│  │  入口     │────▶│ understand   │────▶│  retrieve   │  │
│  │  entry   │     │  (LLM理解)    │     │ (检索知识库) │  │
│  └──────────┘     └──────────────┘     └─────────────┘  │
│                                                │         │
│                          ┌─────────────────────┘         │
│                          ▼                               │
│                   ┌──────────────┐                       │
│                   │    plan      │  LLM 制定执行计划       │
│                   │  (制定计划)   │                       │
│                   └──────┬───────┘                       │
│                          │                               │
│                          ▼                               │
│                   ┌──────────────┐                       │
│              ┌───▶│    agent     │◀──┐                   │
│              │    │ (LLM决策+行动)│   │  核心循环          │
│              │    └──────┬───────┘   │  (Agent Loop)     │
│              │           │           │                   │
│              │    ┌──────▼───────┐   │                   │
│              │    │  tool_call   │   │                   │
│              │    │  (执行工具)   │───┘                   │
│              │    └──────────────┘  如果还需要更多信息     │
│              │           │                               │
│              │           │ 检测到高危操作                 │
│              │    ┌──────▼───────┐                       │
│              │    │ human_approve│  等待人工确认          │
│              │    │  (人在回路)   │                       │
│              │    └──────┬───────┘                       │
│              │           │                               │
│              │           ├── 已确认 ──┐                  │
│              │           │           ▼                  │
│              │    ┌──────▼───────┐                       │
│              │    │   backup     │  ◀── NEW 版本快照      │
│              │    │  (备份状态)   │  备份配置/进程/端口    │
│              │    └──────┬───────┘                       │
│              │           │                               │
│              │           └──→ 返回 agent 执行已审批工具 ──┘│
│              │               (或未确认 → summarize)       │
│              │                                           │
│              │ 目标已达成                                 │
│              │    ┌──────▼───────┐                       │
│              └────│  observe     │                       │
│                   │  (观察反思)   │                       │
│                   └──────┬───────┘                       │
│                          │                               │
│                   ┌──────▼───────┐                       │
│                   │  summarize   │  生成报告              │
│                   │  (总结汇报)   │                       │
│                   └──────┬───────┘                       │
│                          │                               │
│                   ┌──────▼───────┐                       │
│                   │save_knowledge│  存入知识库            │
│                   │  (知识沉淀)   │                       │
│                   └──────┬───────┘                       │
│                          │                               │
│                   ┌──────▼───────┐                       │
│                   │    audit     │  ◀── NEW 审计记录      │
│                   │  (操作审计)   │  不可篡改操作追溯      │
│                   └──────┬───────┘                       │
│                          │                               │
│                         END                              │
└─────────────────────────────────────────────────────────┘
```

### 7.2 核心节点详解

**① understand — 意图理解节点**

```go
// nodes.go - understandNode
func understandNode(ctx context.Context, state *AgentState) (*AgentState, error) {
    input := state.UserInput
    llm := makeBaseLLM(state)
    prompt := fmt.Sprintf(`Analyze the following user request and extract intent as JSON.
Return ONLY valid JSON with these fields:
  intent: one of [diagnose, execute, query, deploy, repair, batch_check, general]
  target: specific host(s) or system(s) mentioned
  description: one-line summary of what the user wants
  urgency: high/medium/low
User request: "%s"`, input)

    resp, err := llm.Generate(ctx, msgs(userMsg(prompt)))
    // ... parse JSON into state.Intent

    // 输入: "web-05 响应很慢，帮我看看怎么回事"
    // 输出: {"intent":"diagnose","target":"web-05","description":"响应延迟诊断","urgency":"high"}

    // 输入: "所有生产机器检查磁盘，超过80%的清理一下日志"
    // 输出: {"intent":"batch_check","target":"production*","description":"批量磁盘检查与日志清理","urgency":"medium"}
    return state, nil
}
```

**② plan — 计划制定节点**

LLM 根据意图 + 知识库检索结果，制定**可动态调整的分步计划**：

```json
{
  "goal": "诊断 web-05 响应慢的根因",
  "steps": [
    {
      "tool": "check_monitor",
      "args": { "host": "web-05" },
      "purpose": "获取CPU/内存/磁盘基线"
    },
    {
      "tool": "ssh_exec",
      "args": { "host": "web-05", "cmd": "ps aux --sort=-%cpu | head -10" },
      "purpose": "查找高CPU进程"
    },
    {
      "tool": "query_logs",
      "args": { "host": "web-05", "service": "java", "lines": 100 },
      "purpose": "检查应用日志"
    },
    {
      "tool": "query_deploy_history",
      "args": { "host": "web-05", "hours": 24 },
      "purpose": "检查近期变更"
    }
  ],
  "success_criteria": "找到根因并给出修复建议"
}
```

**③ agent + tool_call — 核心决策循环**

Eino 的 Tool Calling 模式，每个工具都是实现了 `tool.BaseTool` 接口的 Go 结构体：

```go
// agent_tools.go - 工具注册示例
type sshExecTool struct { resolver toolsHostResolver }

func (t *sshExecTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "ssh_exec",
        Desc: "Execute a command on a single host via SSH.",
        ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
            "host": {Type: "string", Desc: "Host name or ID", Required: true},
            "cmd":  {Type: "string", Desc: "Command to execute", Required: true},
        }),
    }, nil
}

func (t *sshExecTool) InvokableRun(ctx context.Context, args string, _ ...tool.Option) (string, error) {
    m := mustParse(args)
    return tools.SshExec(ctx, t.resolver, m["host"], m["cmd"])
}
```

12 个工具全部实现 `tool.BaseTool` 接口：

LLM 在决策时**自主选择**调用哪个工具、传什么参数、按什么顺序，Agent 框架负责执行并将结果喂回 LLM，LLM 根据结果决定下一步。

**④ observe — 观察反思节点**

LLM 分析上一步工具结果，更新自己的理解：

```
工具返回: CPU 94%, java 进程占 89%
LLM反思: "CPU 高是 java 导致的，需要深入看 java 在做什么"
→ 更新计划: 下一步查 java 日志而非泛查系统日志

工具返回: 磁盘正常, 内存正常, 网络正常
LLM反思: "排除了资源瓶颈，问题在应用层"
→ 缩小排查范围

工具返回: 日志显示大量 Full GC, 2h 前有新版本部署
LLM反思: "根因很可能是新版本代码导致的内存问题"
→ 可以生成结论了
```

**⑤ human_approve — 人在回路（增强）**

```go
// nodes.go - humanApproveNode
var highRiskKeywords = []string{
    "reboot", "shutdown", "reinstall", "reset_password",
    "rm -rf", "fdisk", "mkfs", "dd if",
    "systemctl stop", "docker rm", "kubectl delete",
    "iptables -F", "power_off", "power_reset",
}

func humanApproveNode(ctx context.Context, state *AgentState) (*AgentState, error) {
    // ... 扫描消息中的高危关键词 ...
    state.RequireApproval = riskFound
    if riskFound {
        if state.Approved {
            // 用户在前端已点击 [确认]，路由到 backup
            return state, nil
        }
        // 等待用户确认，SSE 推送审批请求到前端
        warning := "⚠ 此操作涉及高危动作，已暂停等待人工确认..."
        state.Messages = append(state.Messages, schema.AssistantMessage(warning, nil))
    }
    return state, nil
}
```

**审批后路由**：`human_approve` → 分支判断 `state.Approved`：
- `true` → `backup`（版本快照）→ `agent`（重新执行已审批工具）
- `false` → `summarize`（放弃操作，生成报告）

**⑥ backup — 版本快照备份（NEW）**

高危操作确认后、执行前，自动备份目标主机当前状态：

```go
// nodes.go - backupNode
func backupNode(ctx context.Context, state *AgentState) (*AgentState, error) {
    hosts := extractTargetHosts(state)  // 从 pending tool calls 提取目标主机
    backupBase := fmt.Sprintf("/var/backup/agent/%s/%s", state.ThreadID, time.Now().Format("20060102_150405"))

    for _, host := range hosts {
        // SSH 连接目标主机，执行备份命令
        cmds := []string{
            fmt.Sprintf("mkdir -p %s/%s", backupBase, host),
            fmt.Sprintf("cp -a /etc %s/%s/etc"),           // 配置文件快照
            fmt.Sprintf("ps aux > %s/%s/processes.txt"),   // 进程列表
            fmt.Sprintf("ss -tlnp > %s/%s/ports.txt"),     // 端口占用
            fmt.Sprintf("df -h > %s/%s/disk_usage.txt"),   // 磁盘使用率
        }
        for _, cmd := range cmds {
            tools.SshExec(ctx, knowledgeStore, host, cmd)
        }
    }
    state.BackupPath = backupBase
    state.BackupResults = fmt.Sprintf("%d hosts backed up to %s", len(hosts), backupBase)
    return state, nil
}
```

**备份内容**：
| 备份项 | 命令 | 用途 |
|--------|------|------|
| 系统配置 | `cp -a /etc` | 回滚时恢复配置 |
| 进程快照 | `ps aux` | 事后对比进程变化 |
| 端口占用 | `ss -tlnp` | 验证服务状态 |
| 磁盘使用 | `df -h` | 确认磁盘空间充足 |

**备份路径规范**：`/var/backup/agent/{thread_id}/{timestamp}/{host}/`

**⑦ audit — 操作审计（NEW）**

每次 Agent 操作完成后，自动生成不可篡改的结构化审计记录：

```go
// nodes.go - auditNode
func auditNode(ctx context.Context, state *AgentState) (*AgentState, error) {
    entry := AuditEntry{
        ThreadID:       state.ThreadID,
        Intent:         mustJSON(state.Intent),
        Plan:           state.Plan,
        ToolsCalled:    collectToolNames(state),
        HighRiskOps:    collectHighRiskOps(state),
        Approved:       state.Approved,
        BackupPath:     state.BackupPath,
        HostsAffected:  strings.Join(extractTargetHosts(state), ","),
        FinalResult:    state.FinalAnswer,
        Observations:   strings.Join(state.Observations, " | "),
        KnowledgeSaved: true,
        Status:         "completed",
    }
    knowledgeStore.InsertAuditLog(entry)
    return state, nil
}
```

**审计记录字段**（`audit_logs` 表）：

| 字段 | 说明 | 示例 |
|------|------|------|
| `thread_id` | 会话追踪 ID | `uuid-xxxx` |
| `intent` | 意图 JSON | `{"intent":"diagnose","target":"web-05"}` |
| `plan` | 执行计划 | "步骤1: 查监控 → 步骤2: 查进程 → ..." |
| `tools_called` | 调用的工具列表 | `check_monitor,ssh_exec,query_logs` |
| `high_risk_ops` | 高危操作列表 | `ipmi_power` |
| `approved` | 是否人工确认 | `true` |
| `backup_path` | 备份路径 | `/var/backup/agent/uuid/20260519_103000/` |
| `hosts_affected` | 受影响主机 | `web-05,db-03` |
| `final_result` | 最终结果报告 | Markdown 格式完整报告 |
| `observations` | 中间观察记录 | "CPU 94% | Full GC频繁 | ..." |
| `knowledge_saved` | 是否沉淀知识 | `true` |
| `status` | 执行状态 | `completed` / `aborted` |

审计记录可回答：**谁在什么时间对哪些机器做了什么操作、结果如何、是否经过人工确认、备份在哪里。**

### 7.3 Tool Calling 实现

使用 Eino 的 `tool.BaseTool` 接口 + GORM 直连数据库，无需跨语言调用：

```go
// tools/ssh.go - SSH 执行工具
type SSHConfig struct {
    Name string; Host string; Port int
    Username string; AuthType string; Password string
}

func SshExec(ctx context.Context, resolver HostResolver, hostID, cmd string) (string, error) {
    cfg, _ := resolver.Resolve(hostID)  // GORM 直接查数据库
    result := execSSH(cfg, cmd)         // crypto/ssh 连接
    return jsonString(result), nil
}

// tools/ipmi.go - IPMI 工具
func IpmiPower(ctx context.Context, resolver IPMIResolver, host, action string) (string, error) {
    cfg, _ := resolver.ResolveIPMI(host)  // GORM 直查 IPMI 配置
    r := runIPMI(cfg, "chassis", "power", action)  // 调用 ipmitool
    return ipmiResult(cfg, "power_"+action, r), nil
}

// tools/knowledge.go - 知识库工具
func SaveToKnowledge(ctx context.Context, store KnowledgeStore,
    symptoms, diagnosis, rootCause, solution, hosts, tags string) (string, error) {
    id, _ := store.Insert(symptoms, diagnosis, rootCause, solution, hosts, tags)
    return fmt.Sprintf(`{"success":true,"case_id":%d}`, id), nil
}

// 工具注册在 graph.go InitGraph() 中一次性完成:
AllTools = []tool.BaseTool{
    &sshExecTool{resolver: knowledgeStore},
    &sshBatchTool{resolver: knowledgeStore},
    &ipmiPowerTool{resolver: knowledgeStore},
    // ... 共 12 个工具
}
```

### 7.4 持久化记忆与知识库

**三层记忆架构：**

```
┌─────────────────────────────────────────────┐
│ 第一层: 短期记忆 (当前会话)                    │
│ · Eino CheckpointStore                      │
│ · 保存 StateGraph 每一步的状态快照             │
│ · 支持暂停/恢复/回溯                         │
│ · 存储: SQLite (GORM)                        │
├─────────────────────────────────────────────┤
│ 第二层: 工作记忆 (跨会话)                     │
│ · 最近 N 次对话摘要                          │
│ · 每台主机的"当前状态认知"                    │
│ · 例: Agent 知道 web-05 昨天刚修过内存问题     │
│ · 存储: SQLite (agent_memory 表)             │
├─────────────────────────────────────────────┤
│ 第三层: 长期知识库 (案例积累)                  │
│ · 每次故障处理的完整记录                      │
│ · 症状 → 诊断过程 → 根因 → 修复方案            │
│ · 新问题先查知识库找相似案例                   │
│ · 存储: SQLite + 向量嵌入(可选, 语义检索)      │
└─────────────────────────────────────────────┘
```

**知识沉淀流程：**

```
故障处理完成
     │
     ▼
LLM 自动生成案例摘要:
  {
    "symptoms": ["响应延迟", "CPU 高"],
    "diagnosis": "新版本代码 Full GC 频繁导致 CPU 飙升",
    "root_cause": "JVM 堆配置不足 + 代码内存泄漏",
    "solution": "回滚版本 + 增大 -Xmx 至 4G",
    "hosts": ["web-05"],
    "tags": ["java", "gc", "performance", "deployment"],
    "resolved_at": "2026-05-10 11:30"
  }
     │
     ▼
存入 knowledge_cases 表
     │
     ▼
下次类似问题 → retrieve 节点检索 → 直接参考历史方案
```

### 7.5 流式输出 (SSE)

用户看到的不是"等待 → 结果"，而是实时思考过程：

```
前端 AI 对话界面:

┌─────────────────────────────────────────────┐
│  🤖 Agent 正在思考...                        │
│                                             │
│  💭 分析意图: 用户想排查 web-05 响应慢问题     │
│  📋 制定计划: 4个步骤                         │
│                                             │
│  ✅ [1/4] 查询监控 → CPU 94%, 内存62%, 磁盘51%│
│  ✅ [2/4] 查进程TOP → java 占89% CPU          │
│  🔄 [3/4] 查询 java 日志...                   │
│  ⏳ [4/4] 查询部署历史...                      │
│                                             │
│  💡 初步判断: java 应用异常, Full GC 频繁      │
│  📊 根因分析: 2小时前的新版本可能存在性能问题    │
│  🛡️ 建议: 回滚到上一版本                      │
│                                             │
│  ⚠️ 回滚操作将影响生产流量, 是否继续?           │
│  [确认回滚] [先看详细日志] [联系开发]           │
└─────────────────────────────────────────────┘
```

**实现方式**：Go 后端通过 Eino 的 `StreamReader` + SSE 将每一步结果推送到前端。

```go
// handlers/agent.go - SSE 流式输出
func AgentChat(c *gin.Context) {
    state := &agent.AgentState{
        UserInput: req.Message,
        LlmConfig: llmCfg,
        ThreadID:  threadID,
    }

    c.Writer.Header().Set("Content-Type", "text/event-stream")
    finalState, err := agent.Invoke(c.Request.Context(), state)

    // 发送事件流
    sendSSE(w, "intent", map[string]string{"intent": intentJSON})
    sendSSE(w, "plan", map[string]string{"content": planText})
    sendSSE(w, "done", map[string]string{"report": finalState.FinalAnswer, "thread_id": threadID})
    sendSSE(w, "stream_end", map[string]string{"thread_id": threadID})
}

func sendSSE(w io.Writer, eventType string, data interface{}) {
    b, _ := json.Marshal(data)
    fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(b))
    if flusher, ok := w.(http.Flusher); ok {
        flusher.Flush()
    }
}
```

---

## 八、部署方案

### 8.1 Docker Compose

```yaml
version: "3"
services:
  web:
    build: ./web
    ports:
      - "5173:5173"
    depends_on:
      - server-go

  server-go:
    build: ./server-go
    ports:
      - "8080:8080"
    volumes:
      - ./server-go:/app
```

```bash
docker-compose up -d --build
```

---

## 九、监控采集脚本设计

### 9.1 采集间隔

| 指标          | 间隔 | 说明                         |
| ------------- | ---- | ---------------------------- |
| CPU/内存/磁盘 | 60s  | 常规资源 + 存入数据库        |
| 网络流量      | 5s   | 实时性要求高，环形缓冲区缓存 |
| 进程 TOP      | 120s | 开销较大，降低频率           |
| IPMI 传感器   | 300s | IPMI 查询较慢                |

### 9.2 采集方式

- **SSH 模式**：Agent 通过 crypto/ssh 连接目标主机执行采集脚本
- **客户端模式**：目标机 Go 客户端本地采集，通过 WebSocket 上报
- **脚本**：Go 单文件 `monitor_collect.go`，输出 JSON

---

## 十、实施计划

| 阶段                     | 内容                                                                                           | 状态      |
| ------------------------ | ---------------------------------------------------------------------------------------------- | --------- |
| **Phase 1: Agent 大脑**  | Eino StateGraph 搭建、LLM 接入、Tool Calling 框架、12 个核心工具实现、SSE 流式输出          | ✅ 已完成 |
| **Phase 2: AI 对话界面** | AiAssistant 页面含 Markdown 渲染、流式消息接收、chat.js 状态管理                               | ✅ 已完成 |
| Phase 3: 知识库          | 知识库表结构、故障案例自动沉淀 (save_knowledge 节点)、相似案例检索 (query_knowledge_base 工具) | ✅ 已完成 |
| Phase 4: 仪表盘+监控     | Dashboard 页面、系统监控 (CSS 进度条)、流量监控 (CSS 占比条)、监控 CRUD API                    | ✅ 已完成 |
| Phase 5: 主机管理        | IPMI 字段扩展、分组/地域标签、IPMI 配置 UI、IPMI 工具集实现                                    | ✅ 已完成 |
| Phase 6: 告警中心        | 告警规则 CRUD、告警认领与关闭工作流、错误历史统计面板                                          | ✅ 已完成 |
| Phase 7: 人在回路        | 高危操作审批节点 (human_approve)、前端 SSE 事件接收                                            | ✅ 已完成 |
| **Phase 8: 安全防护**    | **版本快照备份 (backup 节点) + 操作审计追溯 (audit 节点)、审计记录 API、audit_logs 表**      | **✅ 已完成** |
| Phase 9: 扩展增强        | 通知渠道(钉钉/企微)、ECharts 图表集成、批量操作增强、Ansible 集成                              | 🔜 规划中 |

> **当前状态**：Phase 1-8 全部完成，用户可通过自然语言与 Agent 交互，Agent 能自主调用工具完成运维任务。高危操作前自动备份目标主机状态，操作后生成不可篡改的审计记录。

---

## 十一、技术参数

| 参数               | 值                     | 说明                 |
| ------------------ | ---------------------- | -------------------- |
| 默认管理员         | admin / admin123       |                      |
| 数据库             | ops.db (SQLite)        | 单文件，免安装       |
| 后端端口           | 8080                   | Go Gin               |
| Agent 架构         | Go + Eino              | 整合进后端进程        |
| 前端端口           | 5173 (dev) / 80 (prod) | Vite / Nginx         |
| JWT 过期时间       | 24 小时                |                      |
| 客户端心跳         | 30s                    | WebSocket            |
| 监控采集间隔       | 60s                    | CPU/内存/磁盘        |
| 流量采集间隔       | 5s                     | 实时流量             |
| 告警规则检查       | 60s                    |                      |
| WebSocket 日志缓存 | 1000 条                | 环形队列             |
| 历史指标数据点     | 最多 500 条            | 前端分页 + API Limit |

---

## 十二、扩展方向

1. **LLM 集成**：自然语言下运维指令，Agent 自动翻译为执行计划
2. **RBAC 权限**：多角色（管理员/运维/只读），操作审计日志
3. **多集群管理**：跨数据中心统一管控
4. **Ansible 集成**：对接现有 Ansible Playbook
5. **移动端适配**：响应式布局或独立移动 App
6. **分布式监控**：Prometheus + Grafana 数据源对接
7. **定时备份调度**：配置文件、数据库定期自动备份与过期清理

---

_本文档为 v3.0 智能体版本设计，当前 Phase 1-8 已全部实现，覆盖 Agent 推理 → 工具调用 → 人工审批 → 版本快照备份 → 审计追溯 → SSE 流式 → 监控告警 → IPMI 管控 → 知识沉淀的完整智能运维闭环。_
