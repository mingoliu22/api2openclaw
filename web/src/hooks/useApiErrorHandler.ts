import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import axios, { AxiosError } from 'axios';
import { useToast } from '../components/Toast';

interface ApiError {
  code?: string;
  message: string;
}

interface ErrorResponse {
  error: ApiError;
}

export function useApiErrorHandler() {
  const navigate = useNavigate();
  const toast = useToast();

  useEffect(() => {
    // 设置 axios 响应拦截器处理错误
    const interceptor = axios.interceptors.response.use(
      (response) => response,
      (error: AxiosError<ErrorResponse>) => {
        const status = error.response?.status;
        const data = error.response?.data;

        // 处理 401 未授权错误
        if (status === 401) {
          // 不显示 toast，直接跳转到登录页
          navigate('/');
          return Promise.reject(error);
        }

        // 处理其他错误
        if (data?.error) {
          const errorMessage = data.error.message || '请求失败';
          toast.error(errorMessage);
        } else if (error.message) {
          // 网络错误或其他错误
          toast.error('网络错误，请稍后重试');
        }

        return Promise.reject(error);
      }
    );

    return () => {
      axios.interceptors.response.eject(interceptor);
    };
  }, [navigate, toast]);
}
