import { lazy } from 'react';
import { createBrowserRouter, Navigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';

// 懒加载页面组件
const LoginPage = lazy(() => import('../pages/LoginPage'));
const DashboardLayout = lazy(() => import('../pages/DashboardLayout'));
const DashboardOverview = lazy(() => import('../pages/DashboardOverview'));
const ModelsPage = lazy(() => import('../pages/ModelsPage'));
const KeysPage = lazy(() => import('../pages/KeysPage'));
const LogsPage = lazy(() => import('../pages/LogsPage'));
const UsersPage = lazy(() => import('../pages/UsersPage'));
const PluginsPage = lazy(() => import('../pages/PluginsPage'));

// 路由守卫组件
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/" replace />;
  }

  return <>{children}</>;
}

// 路由配置
export const router = createBrowserRouter([
  {
    path: '/',
    element: <LoginPage />,
  },
  {
    path: '/dashboard',
    element: (
      <ProtectedRoute>
        <DashboardLayout />
      </ProtectedRoute>
    ),
    children: [
      {
        index: true,
        element: <Navigate to="/dashboard/overview" replace />,
      },
      {
        path: 'overview',
        element: <DashboardOverview />,
      },
      {
        path: 'models',
        element: <ModelsPage />,
      },
      {
        path: 'keys',
        element: <KeysPage />,
      },
      {
        path: 'logs',
        element: <LogsPage />,
      },
      {
        path: 'users',
        element: <UsersPage />,
      },
      {
        path: 'plugins',
        element: <PluginsPage />,
      },
    ],
  },
  {
    path: '*',
    element: <Navigate to="/" replace />,
  },
]);
