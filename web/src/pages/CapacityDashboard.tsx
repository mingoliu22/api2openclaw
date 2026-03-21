import { useEffect, useState, useRef } from 'react';
import { statsAPI, costAPI } from '../services/api';

// 实时统计类型
interface RealtimeStats {
  tokens_per_sec: number;
  tokens_today: number;
  tokens_yesterday: number;
  online_models: number;
  total_models: number;
  active_keys_1h: number;
  threshold: number;
  threshold_status: 'normal' | 'warning' | 'alert';
}

// 模型统计类型
interface ModelStat {
  model_alias: string;
  total_tokens: number;
  requests_count: number;
  avg_tokens_per_req: number;
}

// 每日趋势类型
interface DailyTrend {
  date: string;
  tokens: number;
}

// 成本汇总类型
interface CostSummary {
  total_electricity: number;
  total_depreciation: number;
  total_cost: number;
  total_tokens: number;
  active_models: number;
  cost_per_1k_tokens: number;
  period_days: number;
}

export default function CapacityDashboard() {
  const [realtimeStats, setRealtimeStats] = useState<RealtimeStats | null>(null);
  const [modelStats, setModelStats] = useState<ModelStat[]>([]);
  const [dailyTrend, setDailyTrend] = useState<DailyTrend[]>([]);
  const [costSummary, setCostSummary] = useState<CostSummary | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const pollingRef = useRef<number | null>(null);

  // 获取数据
  const fetchData = async () => {
    try {
      setError(null);

      // 并行获取所有数据
      const [overviewRes, trendRes, costRes] = await Promise.all([
        statsAPI.getOverview(),
        statsAPI.getDailyChart(7),
        costAPI.getStats(7),
      ]);

      setRealtimeStats(overviewRes.data.data.realtime);
      setModelStats(overviewRes.data.data.models || []);
      setDailyTrend(trendRes.data.data || []);
      setCostSummary(costRes.data.data.summary || null);
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : '获取数据失败';
      setError(errorMessage);
      console.error('Failed to fetch dashboard data:', err);
    } finally {
      setIsLoading(false);
    }
  };

  // 启动自动轮询
  const startPolling = () => {
    if (pollingRef.current) {
      clearInterval(pollingRef.current);
    }
    // 每 10 秒刷新一次
    pollingRef.current = setInterval(fetchData, 10000);
  };

  // 停止轮询
  const stopPolling = () => {
    if (pollingRef.current) {
      clearInterval(pollingRef.current);
      pollingRef.current = null;
    }
  };

  useEffect(() => {
    fetchData();
    startPolling();

    return () => {
      stopPolling();
    };
  }, []);

  // 获取阈值状态样式
  const getThresholdStatusStyle = (status: string) => {
    switch (status) {
      case 'alert':
        return 'bg-red-100 text-red-800 border-red-300';
      case 'warning':
        return 'bg-yellow-100 text-yellow-800 border-yellow-300';
      default:
        return 'bg-green-100 text-green-800 border-green-300';
    }
  };

  // 格式化数字
  const formatNumber = (num: number) => {
    if (num >= 1000000) {
      return `${(num / 1000000).toFixed(1)}M`;
    }
    if (num >= 1000) {
      return `${(num / 1000).toFixed(1)}K`;
    }
    return num.toLocaleString();
  };

  return (
    <div className="p-8">
      {/* 页面标题 */}
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">产能仪表盘</h1>
        <p className="text-gray-600 mt-1">Token 工厂 - 实时监控与成本分析</p>
      </div>

      {/* 加载状态 */}
      {isLoading ? (
        <div className="text-center py-12">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">加载中...</p>
        </div>
      ) : error ? (
        <div className="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
          <p className="text-red-800">{error}</p>
          <button
            onClick={fetchData}
            className="mt-4 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
          >
            重试
          </button>
        </div>
      ) : (
        <>
          {/* 实时统计卡片 */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
            {/* Token 产量 */}
            <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-medium text-gray-600">今日 Token 产量</h3>
                <span className="text-2xl">🔥</span>
              </div>
              <p className="text-3xl font-bold text-gray-900">
                {formatNumber(realtimeStats?.tokens_today || 0)}
              </p>
              <p className="text-sm text-gray-500 mt-2">
                昨日: {formatNumber(realtimeStats?.tokens_yesterday || 0)}
              </p>
            </div>

            {/* 实时速率 */}
            <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-medium text-gray-600">实时产出速率</h3>
                <span className="text-2xl">📊</span>
              </div>
              <p className="text-3xl font-bold text-gray-900">
                {realtimeStats?.tokens_per_sec?.toFixed(1) || '0'}
                <span className="text-lg font-normal text-gray-600 ml-1">tokens/s</span>
              </p>
              <div className={`mt-3 inline-flex items-center px-3 py-1 rounded-full text-xs font-medium border ${getThresholdStatusStyle(realtimeStats?.threshold_status || 'normal')}`}>
                状态: {realtimeStats?.threshold_status === 'alert' ? '告警' : realtimeStats?.threshold_status === 'warning' ? '警告' : '正常'}
              </div>
            </div>

            {/* 在线模型 */}
            <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-medium text-gray-600">在线模型数</h3>
                <span className="text-2xl">🤖</span>
              </div>
              <p className="text-3xl font-bold text-gray-900">
                {realtimeStats?.online_models || 0} <span className="text-lg font-normal text-gray-600">/ {realtimeStats?.total_models || 0}</span>
              </p>
              <p className="text-sm text-gray-500 mt-2">
                总模型数: {realtimeStats?.total_models || 0}
              </p>
            </div>

            {/* 活跃 Key */}
            <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-medium text-gray-600">活跃 Key (1h)</h3>
                <span className="text-2xl">🔑</span>
              </div>
              <p className="text-3xl font-bold text-gray-900">
                {realtimeStats?.active_keys_1h || 0}
              </p>
              <p className="text-sm text-gray-500 mt-2">
                过去 1 小时有请求的 Key 数
              </p>
            </div>
          </div>

          {/* 模型分布 */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6 mb-8">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">模型 Token 分布</h3>
            {modelStats.length > 0 ? (
              <div className="space-y-4">
                {modelStats.map((model, index) => {
                  const maxTokens = Math.max(...modelStats.map(m => m.total_tokens));
                  const percentage = maxTokens > 0 ? (model.total_tokens / maxTokens) * 100 : 0;

                  return (
                    <div key={index} className="flex items-center gap-4">
                      <div className="w-32 text-sm text-gray-700 truncate" title={model.model_alias}>
                        {model.model_alias}
                      </div>
                      <div className="flex-1 bg-gray-100 rounded-full h-6 overflow-hidden">
                        <div
                          className="bg-blue-600 h-full rounded-full transition-all duration-500"
                          style={{ width: `${percentage}%` }}
                        />
                      </div>
                      <div className="w-24 text-right text-sm text-gray-700">
                        {formatNumber(model.total_tokens)} tokens
                      </div>
                      <div className="w-20 text-right text-sm text-gray-500">
                        {model.requests_count} 请求
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <p className="text-center text-gray-500 py-8">暂无数据</p>
            )}
          </div>

          {/* 每日趋势图 */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6 mb-8">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">近 7 日趋势</h3>
            {dailyTrend.length > 0 ? (
              <div className="flex items-end justify-between gap-2 h-48">
                {dailyTrend.map((day, index) => {
                  const maxTokens = Math.max(...dailyTrend.map(d => d.tokens));
                  const height = maxTokens > 0 ? (day.tokens / maxTokens) * 100 : 0;

                  return (
                    <div key={index} className="flex-1 flex flex-col items-center gap-2">
                      <div
                        className="w-full bg-blue-600 rounded-t transition-all duration-500 hover:bg-blue-700"
                        style={{ height: `${height}%` }}
                        title={`${day.date}: ${formatNumber(day.tokens)} tokens`}
                      />
                      <div className="text-xs text-gray-600 transform -rotate-45 origin-top-left mt-2">
                        {day.date.slice(5)}
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <p className="text-center text-gray-500 py-8">暂无数据</p>
            )}
          </div>

          {/* 成本汇总 */}
          {costSummary && (
            <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">成本汇总（近 {costSummary.period_days} 日）</h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <div className="bg-gray-50 rounded-lg p-4">
                  <p className="text-sm text-gray-600 mb-1">电费成本</p>
                  <p className="text-2xl font-bold text-gray-900">
                    ¥{costSummary.total_electricity.toFixed(2)}
                  </p>
                </div>
                <div className="bg-gray-50 rounded-lg p-4">
                  <p className="text-sm text-gray-600 mb-1">折旧成本</p>
                  <p className="text-2xl font-bold text-gray-900">
                    ¥{costSummary.total_depreciation.toFixed(2)}
                  </p>
                </div>
                <div className="bg-blue-50 rounded-lg p-4">
                  <p className="text-sm text-gray-600 mb-1">总成本</p>
                  <p className="text-2xl font-bold text-blue-900">
                    ¥{costSummary.total_cost.toFixed(2)}
                  </p>
                </div>
              </div>
              <div className="mt-6 pt-6 border-t border-gray-200">
                <div className="flex justify-between items-center">
                  <div>
                    <p className="text-sm text-gray-600">每千 Token 成本</p>
                    <p className="text-lg font-semibold text-gray-900">
                      ¥{costSummary.cost_per_1k_tokens.toFixed(4)}
                    </p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm text-gray-600">总 Token 数</p>
                    <p className="text-lg font-semibold text-gray-900">
                      {formatNumber(costSummary.total_tokens)}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* 刷新时间提示 */}
          <p className="text-center text-sm text-gray-500 mt-8">
            数据每 10 秒自动刷新
          </p>
        </>
      )}
    </div>
  );
}
