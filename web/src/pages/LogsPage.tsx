import { useEffect, useState } from 'react';
import { logsAPI } from '../services/api';
import type { RequestLog } from '../services/types';
import { useToast } from '../components/Toast';

export default function LogsPage() {
  const toast = useToast();
  const [logs, setLogs] = useState<RequestLog[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [totalCount, setTotalCount] = useState(0);

  // 分页和筛选状态
  const [page, setPage] = useState(1);
  const limit = 50;
  const [filters, setFilters] = useState({
    model_alias: '',
    status_code: '',
    from: '',
    to: '',
  });

  // 获取日志列表
  const fetchLogs = async () => {
    setIsLoading(true);
    try {
      const params: any = {
        page,
        limit,
      };

      if (filters.model_alias) params.model_alias = filters.model_alias;
      if (filters.status_code) params.status_code = parseInt(filters.status_code);
      if (filters.from) params.from = filters.from;
      if (filters.to) params.to = filters.to;

      const response = await logsAPI.list(params);
      setLogs(response.data.data);
      setTotalCount(response.data.total);
    } catch (error) {
      toast.error('获取日志失败');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchLogs();
  }, [page, limit, filters]);

  // 导出 CSV
  const handleExport = async () => {
    try {
      const params: any = {};
      if (filters.model_alias) params.model_alias = filters.model_alias;
      if (filters.status_code) params.status_code = parseInt(filters.status_code);
      if (filters.from) params.from = filters.from;
      if (filters.to) params.to = filters.to;

      const response = await logsAPI.export(params);
      const url = window.URL.createObjectURL(new Blob([response.data], { type: 'text/csv' }));
      const link = document.createElement('a');
      link.href = url;
      link.download = `request_logs_${new Date().toISOString().replace(/[:.]/g, '').slice(0, 15)}.csv`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
      toast.success('日志已导出');
    } catch (error) {
      toast.error('导出失败');
    }
  };

  // 状态码颜色
  const getStatusColor = (statusCode: number) => {
    if (statusCode >= 200 && statusCode < 300) return 'text-green-600 bg-green-50';
    if (statusCode >= 400 && statusCode < 500) return 'text-yellow-600 bg-yellow-50';
    if (statusCode >= 500 && statusCode < 600) return 'text-red-600 bg-red-50';
    return 'text-gray-600';
  };

  // 分页计算
  const totalPages = Math.ceil(totalCount / limit);

  return (
    <div className="p-8">
      {/* 页面标题 */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">调用日志</h1>
          <p className="text-gray-600 mt-1">查看 API 请求记录和统计数据</p>
        </div>
        <button
          onClick={handleExport}
          className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors"
        >
          导出 CSV
        </button>
      </div>

      {/* 筛选器 */}
      <div className="mb-6 bg-white rounded-lg shadow-sm border border-gray-200 p-4">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {/* 模型筛选 */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              模型别名
            </label>
            <input
              type="text"
              value={filters.model_alias}
              onChange={(e) => setFilters({ ...filters, model_alias: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
              placeholder="全部模型"
            />
          </div>

          {/* 状态码筛选 */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              状态码
            </label>
            <input
              type="text"
              value={filters.status_code}
              onChange={(e) => setFilters({ ...filters, status_code: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
              placeholder="例如: 200"
            />
          </div>

          {/* 开始时间 */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              开始时间
            </label>
            <input
              type="datetime-local"
              value={filters.from}
              onChange={(e) => setFilters({ ...filters, from: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
            />
          </div>

          {/* 结束时间 */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              结束时间
            </label>
            <input
              type="datetime-local"
              value={filters.to}
              onChange={(e) => setFilters({ ...filters, to: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
            />
          </div>
        </div>

        {/* 清除筛选按钮 */}
        <div className="mt-4 flex gap-2">
          <button
            onClick={() => setFilters({ model_alias: '', status_code: '', from: '', to: '' })}
            className="text-sm text-blue-600 hover:text-blue-700"
          >
            清除筛选
          </button>
        </div>
      </div>

      {/* 日志列表 */}
      {isLoading ? (
        <div className="text-center py-12">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
        </div>
      ) : logs.length === 0 ? (
        <div className="text-center py-12 bg-white rounded-lg border border-dashed border-gray-300">
          <p className="text-gray-500">暂无日志记录</p>
        </div>
      ) : (
        <>
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden mb-4">
            <table className="w-full">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">时间</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">模型</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Token 数</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">延迟</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">状态码</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {logs.map((log) => (
                  <tr key={log.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3 text-sm text-gray-600">
                      {new Date(log.created_at).toLocaleString('zh-CN')}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-900">{log.model_alias}</td>
                    <td className="px-4 py-3 text-sm text-gray-600">
                      {log.total_tokens.toLocaleString()}
                      <span className="text-gray-400">({log.prompt_tokens}/{log.completion_tokens})</span>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600">{log.latency_ms}ms</td>
                    <td className="px-4 py-3">
                      <span className={`text-xs px-2 py-1 rounded ${getStatusColor(log.status_code)}`}>
                        {log.status_code}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* 分页 */}
          <div className="flex items-center justify-between">
            <div className="text-sm text-gray-600">
              共 {totalCount} 条记录，第 {page} / {totalPages} 页
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed text-sm"
              >
                上一页
              </button>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed text-sm"
              >
                下一页
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
