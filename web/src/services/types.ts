// 认证相关类型
export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  username: string;
  last_login_at?: string;
  expires_at: string;
}

export interface MeResponse {
  username: string;
  user_id: string;
}

// 模型相关类型
export interface Model {
  id: string;
  alias: string;
  model_id: string;
  base_url: string;
  note?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  // v0.3.0: 模型能力字段
  supports_streaming?: boolean;
  supports_tool_use?: boolean;
  supports_json_mode?: boolean;
  context_window?: number;
  model_family?: string;
  health_status?: {
    status: 'healthy' | 'unhealthy' | 'unknown';
    latency_ms?: number;
    error?: string;
    checked_at: string;
  };
}

export interface CreateModelRequest {
  alias: string;
  model_id: string;
  base_url: string;
  api_key?: string;
  note?: string;
  // v0.3.0: 模型能力字段
  supports_streaming?: boolean;
  supports_tool_use?: boolean;
  supports_json_mode?: boolean;
  context_window?: number;
  model_family?: string;
}

export interface UpdateModelRequest {
  alias?: string;
  model_id?: string;
  base_url?: string;
  api_key?: string;
  note?: string;
  // v0.3.0: 模型能力字段
  supports_streaming?: boolean;
  supports_tool_use?: boolean;
  supports_json_mode?: boolean;
  context_window?: number;
  model_family?: string;
}

export interface TestConnectionRequest {
  base_url: string;
  api_key?: string;
}

export interface TestConnectionResponse {
  ok: boolean;
  latency_ms?: number;
  error?: string;
}

// API Key 相关类型
export interface APIKey {
  id: string;
  label: string;
  key_prefix: string;
  key?: string; // 仅在创建时返回
  model_alias?: string;
  expires_at?: string;
  status: 'active' | 'revoked' | 'expired';
  note?: string;
  created_at: string;
  revoked_at?: string;
}

export interface CreateKeyRequest {
  label: string;
  model_alias?: string;
  expires_at?: string;
  note?: string;
}

// 日志相关类型
export interface RequestLog {
  id: string;
  key_id?: string;
  model_alias: string;
  model_actual?: string;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  latency_ms: number;
  status_code: number;
  error_code?: string;
  error_message?: string;
  request_id?: string;
  ip_address?: string;
  created_at: string;
}

export interface LogsResponse {
  data: RequestLog[];
  page: number;
  limit: number;
  total: number;
}

// 统计相关类型
export interface UsageStats {
  total_requests: number;
  total_tokens: number;
  prompt_tokens: number;
  completion_tokens: number;
  active_keys: number;
  active_models: number;
}

// 健康状态类型
export interface HealthStatus {
  status: 'ok' | 'error';
  timestamp: number;
  services: {
    database: 'ok' | 'error';
    router: 'ok' | 'error';
  };
  models?: Array<{
    alias: string;
    status: 'healthy' | 'unhealthy';
  }>;
}

// 错误响应类型
export interface ErrorResponse {
  error: {
    code: string;
    message: string;
  };
}

// 用户相关类型
export interface AdminUser {
  id: string;
  username: string;
  email?: string;
  role: UserRole;
  is_active: boolean;
  created_by?: string;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

export type UserRole = 'super_admin' | 'admin' | 'operator' | 'viewer';

export interface CreateUserRequest {
  username: string;
  password: string;
  email?: string;
  role: UserRole;
}

export interface UpdateUserRequest {
  email?: string;
  role?: UserRole;
  password?: string;
  is_active?: boolean;
}

export interface Claims {
  user_id: string;
  username: string;
  role: UserRole;
}

// 插件相关类型
export interface Plugin {
  name: string;
  type: 'builtin' | 'so';
  path?: string;
  enabled: boolean;
  config: Record<string, unknown>;
  version?: string;
}

export interface BuiltinPlugin {
  name: string;
  type: 'builtin';
  description: string;
  version: string;
  author: string;
  input_formats: string[];
  output_formats: string[];
}

export interface PluginTestResult {
  success: boolean;
  output?: string;
  error?: string;
}

// 计费相关类型
export interface BillingRule {
  id: number;
  name: string;
  description: string;
  rule_type: 'token_based' | 'request_based' | 'tier';
  model_alias?: string;
  key_id?: string;
  unit_price: number;
  currency: string;
  free_quota: number;
  tier_threshold?: number;
  tier_price?: number;
  is_active: boolean;
  valid_from: string;
  valid_until?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateBillingRuleRequest {
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
}

export interface Invoice {
  id: number;
  invoice_number: string;
  key_id?: string;
  billing_period_start: string;
  billing_period_end: string;
  currency: string;
  subtotal: number;
  tax: number;
  discount: number;
  total: number;
  status: 'pending' | 'paid' | 'overdue' | 'cancelled';
  due_date?: string;
  paid_date?: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface InvoiceItem {
  id: number;
  invoice_id: number;
  model_alias: string;
  request_count: number;
  token_count: number;
  unit_price: number;
  tier_applied: boolean;
  line_total: number;
  created_at: string;
}

export interface Payment {
  id: number;
  invoice_id: number;
  amount: number;
  payment_method: string;
  payment_reference?: string;
  status: 'pending' | 'completed' | 'failed' | 'refunded';
  paid_at?: string;
  notes?: string;
  created_at: string;
}

export interface InvoiceDetail {
  invoice: Invoice;
  items: InvoiceItem[];
  payments: Payment[];
}

export interface GenerateInvoiceRequest {
  key_id: string;
  start_date: string;
  end_date: string;
}

export interface UsageStats {
  key_id: string;
  start_date: string;
  end_date: string;
  total_requests: number;
  total_tokens: number;
  total_cost: number;
  cost_by_model: Record<string, {
    request_count: number;
    token_count: number;
    cost: number;
  }>;
}

export interface InvoicesResponse {
  invoices: Invoice[];
  total: number;
  page: number;
  limit: number;
}
