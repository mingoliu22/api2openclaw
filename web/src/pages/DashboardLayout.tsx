// 仪表盘布局 - 占位符
import { Outlet } from 'react-router-dom';

export default function DashboardLayout() {
  return (
    <div className="min-h-screen bg-gray-50">
      <div className="flex">
        {/* 侧边栏占位 */}
        <aside className="w-64 bg-gray-800 text-white min-h-screen p-4">
          <h1 className="text-xl font-bold mb-8">api2openclaw</h1>
          <nav>
            <a href="/dashboard/overview" className="block py-2 hover:bg-gray-700 rounded">概览</a>
            <a href="/dashboard/models" className="block py-2 hover:bg-gray-700 rounded">模型配置</a>
            <a href="/dashboard/keys" className="block py-2 hover:bg-gray-700 rounded">API Keys</a>
            <a href="/dashboard/logs" className="block py-2 hover:bg-gray-700 rounded">日志</a>
          </nav>
        </aside>
        {/* 主内容区 */}
        <main className="flex-1 p-8">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
