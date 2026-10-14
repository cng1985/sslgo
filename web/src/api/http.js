import axios from 'axios'
import { ElMessage } from 'element-plus'

const http = axios.create({
  baseURL: '/api/v1',
  timeout: 30000
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('ssl_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  config.headers['Content-Type'] = 'application/json'
  return config
})

http.interceptors.response.use((res) => {
  const body = res.data
  if (!body || body.code !== 200) {
    const msg = body?.msg || '请求失败'
    ElMessage.error(msg)
    return Promise.reject(new Error(msg))
  }
  return body
}, (err) => {
  const msg = err.response?.data?.msg || err.message || '网络错误'
  if (err.response?.status === 401 && !location.pathname.startsWith('/login')) {
    localStorage.removeItem('ssl_token')
    location.href = '/login'
  }
  ElMessage.error(msg)
  return Promise.reject(new Error(msg))
})

export default http
