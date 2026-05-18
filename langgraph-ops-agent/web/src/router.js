import { createRouter, createWebHistory } from 'vue-router'
import Login from './views/Login.vue'
import Main from './views/Main.vue'
import HostManage from './views/HostManage.vue'
import TaskCreate from './views/TaskCreate.vue'
import TaskList from './views/TaskList.vue'
import LogViewer from './views/LogViewer.vue'
import ClientManage from './views/ClientManage.vue'
import TaskLogDetail from './views/TaskLogDetail.vue'
import AiAssistant from './views/AiAssistant.vue'
import Dashboard from './views/Dashboard.vue'
import SystemMonitor from './views/SystemMonitor.vue'
import TrafficMonitor from './views/TrafficMonitor.vue'
import AlertCenter from './views/AlertCenter.vue'
import ErrorHistory from './views/ErrorHistory.vue'

const routes = [
  {
    path: '/',
    redirect: '/login'
  },
  {
    path: '/login',
    name: 'Login',
    component: Login
  },
  {
    path: '/main',
    name: 'Main',
    component: Main,
    children: [
      {
        path: '',
        redirect: '/main/dashboard'
      },
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: Dashboard
      },
      {
        path: 'assistant',
        name: 'AiAssistant',
        component: AiAssistant
      },
      {
        path: 'hosts',
        name: 'HostManage',
        component: HostManage
      },
      {
        path: 'clients',
        name: 'ClientManage',
        component: ClientManage
      },
      {
        path: 'task/create',
        name: 'TaskCreate',
        component: TaskCreate
      },
      {
        path: 'tasks',
        name: 'TaskList',
        component: TaskList
      },
      {
        path: 'task/detail/:id',
        name: 'TaskLogDetail',
        component: TaskLogDetail
      },
      {
        path: 'task-log',
        name: 'LogViewer',
        component: LogViewer
      },
      {
        path: 'monitor/system',
        name: 'SystemMonitor',
        component: SystemMonitor
      },
      {
        path: 'monitor/traffic',
        name: 'TrafficMonitor',
        component: TrafficMonitor
      },
      {
        path: 'alerts',
        name: 'AlertCenter',
        component: AlertCenter
      },
      {
        path: 'errors',
        name: 'ErrorHistory',
        component: ErrorHistory
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  if (to.path !== '/login' && !token) {
    next('/login')
  } else if (to.path === '/login' && token) {
    next('/main')
  } else {
    next()
  }
})

export default router
