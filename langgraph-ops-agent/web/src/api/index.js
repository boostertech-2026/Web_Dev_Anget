import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000
})

api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  response => response.data,
  error => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      globalThis.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export const login = (data) => api.post('/login', data)
export const getHosts = () => api.get('/host')
export const addHost = (data) => api.post('/host', data)
export const updateHost = (id, data) => api.put(`/host/${id}`, data)
export const deleteHost = (id) => api.delete(`/host/${id}`)
export const getTasks = () => api.get('/task/list')
export const createTask = (data) => api.post('/task/create', data)
export const getClients = () => api.get('/client/list')
export const addClient = (data) => api.post('/client', data)
export const updateClient = (id, data) => api.put(`/client/${id}`, data)
export const deleteClient = (id) => api.delete(`/client/${id}`)
export const sendClientCmd = (clientId, command, taskId) => api.post('/client/send', {
  client_id: clientId,
  command,
  task_id: taskId,
})

// Agent
export const agentChat = (data) => api.post('/agent/chat', data)

// Dashboard
export const getDashboardSummary = () => api.get('/dashboard/summary')

// Metrics
export const getLatestMetrics = (host) => api.get('/metrics/latest', { params: { host } })
export const getMetricsHistory = (hostId, duration) => api.get('/metrics/history', { params: { host_id: hostId, duration } })
export const getTrafficMetrics = (hostId) => api.get('/metrics/traffic', { params: { host_id: hostId } })

// Alerts
export const getAlerts = (status, level) => api.get('/alerts', { params: { status, level } })
export const ackAlert = (id, data) => api.post(`/alerts/${id}/ack`, data)
export const resolveAlert = (id, data) => api.post(`/alerts/${id}/resolve`, data)
export const getAlertRules = () => api.get('/alerts/rules')
export const updateAlertRule = (id, data) => api.put(`/alerts/rules/${id}`, data)

// Errors
export const getErrorHistory = (status) => api.get('/errors', { params: { status } })
export const getErrorStats = () => api.get('/errors/stats')
export const resolveError = (id, data) => api.post(`/errors/${id}/resolve`, data)

// IPMI
export const updateHostIpmi = (id, data) => api.put(`/host/${id}/ipmi`, data)
export const checkIpmiConnectivity = (id) => api.post(`/host/${id}/ipmi/check`)

// SSH
export const sshConnect = (hostId) => api.post('/ssh/connect', { host_id: hostId })
export const sshExecute = (hostId, command) => api.post('/ssh/execute', { host_id: hostId, command })

export default api
