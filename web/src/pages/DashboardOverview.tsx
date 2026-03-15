import { useEffect, useState } from 'react';
import { systemAPI, logsAPI } from '../services/api';
import type { UsageStats } from '../services/types';

export default function DashboardOverview() {
  const [healthStatus, setHealthStatus] = useState<'healthy' | 'error'>('healthy');
  const [stats, setStats] = useState<UsageStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // 获取健康状态和统计数据
  const fetchData = async () => {
    try {
      const [healthRes, usageRes] = await Promise.all([
        systemAPI.getHealth(),
        logsAPI.getUsage({}),
      ]);

      setHealthStatus(healthRes.data.status === 'ok' ? 'healthy' : 'error');
      setStats(usageRes.data.data);
    } catch (error) {
      setHealthStatus('error');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchData();

    // 每 30 秒刷新一次健康状态
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, []);

  // 统计卡片数据
  const cards = [
    {
      title: '总请求数（今日）',
      value: stats?.total_requests?.toLocaleString() || '-',
      icon: '📊',
      color: 'bg-blue-500',
    },
    {
      title: 'Token 消耗（今日）',
      value: stats?.total_tokens?.toLocaleString() || '-',
      icon: '🔥',
      color: 'bg-orange-500',
    },
    {
      title: '在线模型数',
      value: stats?.active_models || '-',
      icon: '🤖',
      color: 'bg-green-500',
    },
    {
      title: '活跃 API Key 数',
      value: stats?.active_keys || '-',
      icon: '🔑',
      color: 'bg-purple-500',
    },
  ];

  return (
    <div className="p-8">
      {/* 加载状态 */}
      {isLoading ? (
        <div className="text-center py-12">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
        </div>
      ) : (
        <>
          {/* 页面标题 */}
          <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">仪表盘概览</h1>
        <p className="text-gray-600 mt-1">系统运行状态和关键指标</p>
      </div>

      {/* 健康状态指示器 */}
      <div className="mb-6 flex items-center gap-2">
        <span className={`w-3 h-3 rounded-full ${healthStatus === 'healthy' ? 'bg-green-500' : 'bg-red-500'}`} />
        <span className="text-sm text-gray-600">
          服务状态: {healthStatus === 'healthy' ? '正常' : '异常'}
        </span>
      </div>

      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        {cards.map((card, index) => (
          <div
            key={index}
            className="bg-white rounded-lg shadow-sm border border-gray-200 p-6 hover:shadow-md transition-shadow"
          >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600 mb-1">{card.title}</p>
                <p className="text-2xl font-bold text-gray-900">{card.value}</p>
              </div>
              <div className={`${card.color} w-12 h-12 rounded-full flex items-center justify-center text-2xl`}>
                {card.icon}
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* 快速操作 */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">快速操作</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <a
            href="/dashboard/models"
            className="flex items-center gap-3 p-4 border border-gray-200 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-colors"
          >
            <span className="text-2xl">➕</span>
            <span className="font-medium text-gray-900">添加模型</span>
          </a>
          <a
            href="/dashboard/keys"
            className="flex items-center gap-3 p-4 border border-gray-200 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-colors"
          >
            <span className="text-2xl">🔑</span>
            <span className="font-medium text-gray-900">创建 API Key</span>
          </a>
          <a
            href="/dashboard/logs"
            className="flex items-center gap-3 p-4 border border-gray-200 rounded-lg hover:border-blue-500 hover:bg-blue-50 transition-colors"
          >
            <span className="text-2xl">📋</span>
            <span className="font-medium text-gray-900">查看日志</span>
          </a>
        </div>
      </div>

      {/* 系统信息 */}
      <div className="mt-6 bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">系统信息</h2>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="text-gray-600">版本:</span>
            <span className="ml-2 text-gray-900 font-medium">v0.2.0</span>
          </div>
          <div>
            <span className="text-gray-600">构建:</span>
            <span className="ml-2 text-gray-900 font-medium">dev</span>
          </div>
        </div>
      </div>
        </>
      )}
    </div>
  );
}
