import { Outlet } from 'react-router-dom';
import { useApiErrorHandler } from '../hooks/useApiErrorHandler';
import { useAuth } from '../hooks/useAuth';
import Sidebar from '../components/Sidebar';

export default function DashboardLayout() {
  useApiErrorHandler(); // 设置全局错误处理
  const { isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 flex">
      <Sidebar />
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  );
}
