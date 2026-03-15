import axios from 'axios';

// API 基础配置
const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true, // 发送 Cookie
});

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // 未授权，跳转到登录页
      window.location.href = '/';
    }
    return Promise.reject(error);
  }
);

// 认证 API
export const authAPI = {
  login: (username: string, password: string) =>
    api.post('/admin/auth/login', { username, password }),

  logout: () =>
    api.post('/admin/auth/logout'),

  getMe: () =>
    api.get('/admin/auth/me'),
};

// 模型管理 API
export const modelsAPI = {
  list: (activeOnly = true) =>
    api.get(`/admin/models?active=${activeOnly}`),

  create: (data: {
    alias: string;
    model_id: string;
    base_url: string;
    api_key?: string;
    note?: string;
  }) =>
    api.post('/admin/models', data),

  update: (id: string, data: {
    alias?: string;
    model_id?: string;
    base_url?: string;
    api_key?: string;
    note?: string;
  }) =>
    api.put(`/admin/models/${id}`, data),

  delete: (id: string) =>
    api.delete(`/admin/models/${id}`),

  testConnection: (data: {
    base_url: string;
    api_key?: string;
  }) =>
    api.post('/admin/models/test', data),

  toggleActive: (id: string, active: boolean) =>
    api.post(`/admin/models/${id}/toggle?active=${active}`),
};

// API Key 管理 API
export const keysAPI = {
  list: (status = '') =>
    api.get(`/admin/keys?status=${status}`),

  create: (data: {
    label: string;
    model_alias?: string;
    expires_at?: string;
    note?: string;
  }) =>
    api.post('/admin/keys', data),

  revoke: (id: string) =>
    api.delete(`/admin/keys/${id}`),

  getUsage: (id: string) =>
    api.get(`/admin/keys/${id}/usage`),
};

// 日志和统计 API
export const logsAPI = {
  list: (params: {
    page?: number;
    limit?: number;
    key_id?: string;
    model_alias?: string;
    status_code?: number;
    from?: string;
    to?: string;
  }) =>
    api.get('/admin/logs', { params }),

  export: (params: {
    key_id?: string;
    model_alias?: string;
    status_code?: number;
    from?: string;
    to?: string;
  }) =>
    api.get('/admin/logs/export', {
      params,
      responseType: 'blob',
    }),

  getUsage: (params: {
    key_id?: string;
    model_alias?: string;
    from?: string;
    to?: string;
  }) =>
    api.get('/admin/usage', { params }),
};

// 系统 API
export const systemAPI = {
  getHealth: () =>
    api.get('/admin/health'),
};

export default api;
