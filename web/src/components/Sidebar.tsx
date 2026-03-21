import { NavLink, useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';

interface NavItem {
  path: string;
  label: string;
  icon: string;
}

const navItems: NavItem[] = [
  { path: '/dashboard/overview', label: '概览', icon: '📊' },
  { path: '/dashboard/models', label: '模型配置', icon: '🤖' },
  { path: '/dashboard/models/deploy-guide', label: '部署指南', icon: '📖' },
  { path: '/dashboard/keys', label: 'API Keys', icon: '🔑' },
  { path: '/dashboard/logs', label: '调用日志', icon: '📋' },
  { path: '/dashboard/capacity', label: '产能仪表盘', icon: '⚡' },
  { path: '/dashboard/users', label: '用户管理', icon: '👥' },
  { path: '/dashboard/plugins', label: '插件市场', icon: '🧩' },
  { path: '/dashboard/billing', label: '计费管理', icon: '💰' },
];

export default function Sidebar() {
  const navigate = useNavigate();
  const { user, logout } = useAuth();

  const handleLogout = async () => {
    try {
      await logout();
      navigate('/');
    } catch (error) {
      console.error('登出失败:', error);
    }
  };


  return (
    <aside className="w-64 bg-gray-900 text-white min-h-screen flex flex-col">
      {/* Logo 和产品名称 */}
      <div className="p-6 border-b border-gray-800">
        <h1 className="text-xl font-bold text-white mb-1">api2openclaw</h1>
        <p className="text-xs text-gray-400">管理控制台 v0.2.0</p>
      </div>

      {/* 健康状态 */}
      <div className="px-4 py-3 border-b border-gray-800">
        <div className="flex items-center gap-2 text-sm">
          <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
          <span className="text-gray-400">运行中</span>
        </div>
      </div>

      {/* 导航菜单 */}
      <nav className="flex-1 p-4 space-y-1">
        {navItems.map((item) => (
          <NavLink
            key={item.path}
            to={item.path}
            end={item.path === '/dashboard/overview'}
            className={({ isActive }) =>
              `flex items-center gap-3 px-4 py-2.5 rounded-lg transition-colors ${
                isActive
                  ? 'bg-blue-600 text-white'
                  : 'text-gray-300 hover:bg-gray-800 hover:text-white'
              }`
            }
          >
            <span className="text-lg">{item.icon}</span>
            <span>{item.label}</span>
          </NavLink>
        ))}
      </nav>

      {/* 底部用户信息和登出 */}
      <div className="p-4 border-t border-gray-800">
        <div className="flex items-center justify-between mb-3">
          <div className="text-sm">
            <div className="text-gray-400">当前用户</div>
            <div className="text-white font-medium">{user?.username}</div>
          </div>
        </div>
        <button
          onClick={handleLogout}
          className="w-full flex items-center justify-center gap-2 px-4 py-2 text-sm text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
        >
          <span>🚪</span>
          <span>退出登录</span>
        </button>
      </div>
    </aside>
  );
}
