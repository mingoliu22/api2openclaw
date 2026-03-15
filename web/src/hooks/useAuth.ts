import { useState, useEffect } from 'react';
import { authAPI } from '../services/api';
import type { LoginResponse } from '../services/types';

interface UseAuthReturn {
  isAuthenticated: boolean;
  isLoading: boolean;
  user: { username: string; user_id: string } | null;
  login: (username: string, password: string) => Promise<LoginResponse>;
  logout: () => Promise<void>;
  checkAuth: () => Promise<boolean>;
}

export function useAuth(): UseAuthReturn {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [user, setUser] = useState<{ username: string; user_id: string } | null>(null);

  // 检查登录状态
  const checkAuth = async (): Promise<boolean> => {
    try {
      const response = await authAPI.getMe();
      const userData = response.data;
      setUser(userData);
      setIsAuthenticated(true);
      return true;
    } catch {
      setUser(null);
      setIsAuthenticated(false);
      return false;
    } finally {
      setIsLoading(false);
    }
  };

  // 登录
  const login = async (username: string, password: string): Promise<LoginResponse> => {
    const response = await authAPI.login(username, password);
    const data = response.data;

    // 登录成功后获取用户信息
    await checkAuth();

    return data;
  };

  // 登出
  const logout = async () => {
    try {
      await authAPI.logout();
    } finally {
      setUser(null);
      setIsAuthenticated(false);
    }
  };

  // 组件挂载时检查登录状态
  useEffect(() => {
    checkAuth();
  }, []);

  return {
    isAuthenticated,
    isLoading,
    user,
    login,
    logout,
    checkAuth,
  };
}
