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
}

export interface UpdateModelRequest {
  alias?: string;
  model_id?: string;
  base_url?: string;
  api_key?: string;
  note?: string;
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
