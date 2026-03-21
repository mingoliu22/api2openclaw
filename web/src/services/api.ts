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
    // 只在非登录页面时才重定向
    if (error.response?.status === 401 && window.location.pathname !== '/') {
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

  get: (id: string) =>
    api.get(`/admin/keys/${id}`),

  create: (data: {
    label: string;
    model_alias?: string;
    expires_at?: string;
    note?: string;
    daily_token_soft_limit?: number;
    daily_token_hard_limit?: number;
    priority?: 'high' | 'normal' | 'low';
  }) =>
    api.post('/admin/keys', data),

  update: (id: string, data: {
    label?: string;
    note?: string;
    daily_token_soft_limit?: number;
    daily_token_hard_limit?: number;
    priority?: 'high' | 'normal' | 'low';
  }) =>
    api.put(`/admin/keys/${id}`, data),

  revoke: (id: string) =>
    api.delete(`/admin/keys/${id}`),

  getUsage: (id: string) =>
    api.get(`/admin/keys/${id}/usage`),

  getQuota: (id: string) =>
    api.get(`/admin/keys/${id}/quota`),
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

// 用户管理 API
export const usersAPI = {
  list: (params: { page?: number; limit?: number }) =>
    api.get('/admin/users', { params }),

  get: (id: string) =>
    api.get(`/admin/users/${id}`),

  create: (data: {
    username: string;
    password: string;
    email?: string;
    role: 'super_admin' | 'admin' | 'operator' | 'viewer';
  }) =>
    api.post('/admin/users', data),

  update: (id: string, data: {
    email?: string;
    role?: 'super_admin' | 'admin' | 'operator' | 'viewer';
    password?: string;
    is_active?: boolean;
  }) =>
    api.put(`/admin/users/${id}`, data),

  delete: (id: string) =>
    api.delete(`/admin/users/${id}`),
};

// 插件管理 API
export const pluginsAPI = {
  list: () =>
    api.get('/admin/plugins'),

  getBuiltin: () =>
    api.get('/admin/plugins/builtin'),

  get: (name: string) =>
    api.get(`/admin/plugins/${name}`),

  upload: (file: File, config: { name?: string; symbol?: string; config?: string }) => {
    const formData = new FormData();
    formData.append('plugin', file);
    if (config.name) formData.append('name', config.name);
    if (config.symbol) formData.append('symbol', config.symbol);
    if (config.config) formData.append('config', config.config);
    return api.post('/admin/plugins', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },

  enable: (name: string, config?: Record<string, unknown>) =>
    api.put(`/admin/plugins/${name}/enable`, { config }),

  disable: (name: string) =>
    api.put(`/admin/plugins/${name}/disable`),

  updateConfig: (name: string, config: Record<string, unknown>) =>
    api.put(`/admin/plugins/${name}/config`, { config }),

  download: (name: string) =>
    api.get(`/admin/plugins/${name}/download`, { responseType: 'blob' }),

  getLogs: (name: string) =>
    api.get(`/admin/plugins/${name}/logs`),

  test: (name: string, data: {
    input_format: string;
    output_format: string;
    test_data: string;
    config?: Record<string, unknown>;
  }) =>
    api.post(`/admin/plugins/${name}/test`, data),
};

// 计费管理 API
export const billingAPI = {
  // 用量统计
  getUsage: (params: {
    key_id: string;
    start_date?: string;
    end_date?: string;
  }) =>
    api.get('/admin/billing/usage', { params }),

  // 计费规则
  listRules: (activeOnly = false) =>
    api.get(`/admin/billing/rules?active_only=${activeOnly}`),

  getRule: (id: number) =>
    api.get(`/admin/billing/rules/${id}`),

  createRule: (data: {
    name: string;
    description?: string;
    rule_type: 'token_based' | 'request_based' | 'tier';
    model_alias?: string;
    key_id?: string;
    unit_price: number;
    currency: string;
    free_quota?: number;
    tier_threshold?: number;
    tier_price?: number;
    is_active?: boolean;
    valid_from: string;
    valid_until?: string;
  }) =>
    api.post('/admin/billing/rules', data),

  updateRule: (id: number, data: {
    name?: string;
    description?: string;
    rule_type?: 'token_based' | 'request_based' | 'tier';
    model_alias?: string;
    key_id?: string;
    unit_price?: number;
    currency?: string;
    free_quota?: number;
    tier_threshold?: number;
    tier_price?: number;
    is_active?: boolean;
    valid_from?: string;
    valid_until?: string;
  }) =>
    api.put(`/admin/billing/rules/${id}`, data),

  deleteRule: (id: number) =>
    api.delete(`/admin/billing/rules/${id}`),

  // 账单管理
  listInvoices: (params: {
    page?: number;
    limit?: number;
    key_id?: string;
    status?: string;
  }) =>
    api.get('/admin/billing/invoices', { params }),

  getInvoice: (id: number) =>
    api.get(`/admin/billing/invoices/${id}`),

  generateInvoice: (data: {
    key_id: string;
    start_date: string;
    end_date: string;
  }) =>
    api.post('/admin/billing/invoices/generate', data),

  updateInvoiceStatus: (id: number, status: 'pending' | 'paid' | 'overdue' | 'cancelled') =>
    api.put(`/admin/billing/invoices/${id}/status`, { status }),

  exportInvoice: (id: number) =>
    api.get(`/admin/billing/invoices/${id}/export`, { responseType: 'blob' }),

  exportInvoicesCSV: (params?: {
    key_id?: string;
    status?: string;
  }) =>
    api.get('/admin/billing/invoices/export', {
      params,
      responseType: 'blob',
    }),

  // 付款管理
  createPayment: (invoiceId: number, data: {
    amount: number;
    payment_method: string;
    payment_reference?: string;
    notes?: string;
  }) =>
    api.post(`/admin/billing/invoices/${invoiceId}/payments`, data),
};

// Token Factory - 统计 API（公开）
export const statsAPI = {
  // 获取实时统计概览
  getOverview: () =>
    api.get('/api/stats/overview'),

  // 获取每日趋势图数据
  getDailyChart: (days = 7) =>
    api.get(`/api/stats/daily-chart?days=${days}`),
};

// Token Factory - 成本 API
export const costAPI = {
  // 获取成本统计（公开）
  getStats: (days = 7) =>
    api.get(`/api/cost/stats?days=${days}`),

  // 管理接口：成本配置列表
  listConfigs: () =>
    api.get('/admin/cost/configs'),

  // 获取模型的成本配置
  getModelConfigs: (modelId: string) =>
    api.get(`/admin/cost/configs/model/${modelId}`),

  // 获取当前生效的成本配置
  getActiveConfig: (modelId: string) =>
    api.get(`/admin/cost/configs/model/${modelId}/active`),

  // 创建成本配置
  createConfig: (data: {
    model_id: string;
    gpu_count: number;
    power_per_gpu_w: number;
    electricity_price_per_kwh: number;
    depreciation_per_gpu_month: number;
    pue: number;
    effective_from?: string;
  }) =>
    api.post('/admin/cost/configs', data),

  // 更新成本配置
  updateConfig: (id: string, data: {
    gpu_count?: number;
    power_per_gpu_w?: number;
    electricity_price_per_kwh?: number;
    depreciation_per_gpu_month?: number;
    pue?: number;
  }) =>
    api.put(`/admin/cost/configs/${id}`, data),

  // 删除成本配置
  deleteConfig: (id: string) =>
    api.delete(`/admin/cost/configs/${id}`),

  // 获取每日成本统计
  getDailyStats: (days = 30) =>
    api.get(`/admin/cost/stats/daily?days=${days}`),

  // 获取模型的每日成本统计
  getModelDailyStats: (modelAlias: string, days = 30) =>
    api.get(`/admin/cost/stats/daily/model/${modelAlias}?days=${days}`),

  // 获取成本汇总
  getSummary: (days = 30) =>
    api.get(`/admin/cost/stats/summary?days=${days}`),

  // 刷新成本统计
  refresh: () =>
    api.post('/admin/cost/stats/refresh'),

  // 计算指定日期的成本
  calculate: (statDate: string) =>
    api.post('/admin/cost/stats/calculate', { stat_date: statDate }),
};

// Token Factory - 配额 API
export const quotaAPI = {
  // 获取 API Key 配额状态
  getStatus: (keyId: string) =>
    api.get(`/admin/keys/${keyId}/quota`),

  // 更新 API Key（配额字段）
  updateKey: (id: string, data: {
    label?: string;
    note?: string;
    daily_token_soft_limit?: number;
    daily_token_hard_limit?: number;
    priority?: 'high' | 'normal' | 'low';
  }) =>
    api.put(`/admin/keys/${id}`, data),
};

export default api;
