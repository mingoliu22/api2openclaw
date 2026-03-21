import { useEffect, useState, type FormEvent } from 'react';
import { keysAPI } from '../services/api';
import type { APIKey, CreateKeyRequest } from '../services/types';
import { useToast } from '../components/Toast';

interface QuotaStatus {
  key_id: string;
  label: string;
  daily_token_soft_limit: number | null;
  daily_token_hard_limit: number | null;
  priority: string;
}

export default function KeysPage() {
  const toast = useToast();
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [showQuotaDialog, setShowQuotaDialog] = useState(false);
  const [selectedKeyQuota, setSelectedKeyQuota] = useState<QuotaStatus | null>(null);
  const [createdKey, setCreatedKey] = useState<APIKey | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [copied, setCopied] = useState(false);

  // 表单状态
  const [formData, setFormData] = useState({
    label: '',
    model_alias: '',
    expires_at: '',
    note: '',
    daily_token_soft_limit: '',
    daily_token_hard_limit: '',
    priority: 'normal',
  });

  // 获取 Key 列表
  const fetchKeys = async () => {
    try {
      const response = await keysAPI.list(statusFilter);
      setKeys(response.data.data);
    } catch (error) {
      toast.error('获取 API Key 列表失败');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchKeys();
  }, [statusFilter]);

  // 打开创建弹窗
  const handleAdd = () => {
    setFormData({
      label: '',
      model_alias: '',
      expires_at: '',
      note: '',
      daily_token_soft_limit: '',
      daily_token_hard_limit: '',
      priority: 'normal',
    });
    setCreatedKey(null);
    setShowCreateDialog(true);
    setCopied(false);
  };

  // 查看配额详情
  const handleViewQuota = async (key: APIKey) => {
    try {
      const response = await keysAPI.getQuota(key.id);
      setSelectedKeyQuota(response.data.data);
      setShowQuotaDialog(true);
    } catch (error) {
      toast.error('获取配额信息失败');
    }
  };

  // 提交创建表单
  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();

    if (!formData.label) {
      toast.error('请输入标签');
      return;
    }

    setIsSubmitting(true);

    try {
      const createData: CreateKeyRequest = {
        label: formData.label,
        note: formData.note,
      };
      if (formData.model_alias) {
        createData.model_alias = formData.model_alias;
      }
      if (formData.expires_at) {
        createData.expires_at = formData.expires_at;
      }
      if (formData.daily_token_soft_limit) {
        createData.daily_token_soft_limit = parseInt(formData.daily_token_soft_limit);
      }
      if (formData.daily_token_hard_limit) {
        createData.daily_token_hard_limit = parseInt(formData.daily_token_hard_limit);
      }
      if (formData.priority) {
        createData.priority = formData.priority as 'high' | 'normal' | 'low';
      }

      const response = await keysAPI.create(createData);
      setCreatedKey(response.data.data);
      toast.success('API Key 已创建，请复制保存');
      await fetchKeys();
    } catch (error: any) {
      toast.error(error.response?.data?.error?.message || '创建失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  // 复制 Key
  const handleCopy = async () => {
    if (createdKey?.key) {
      try {
        await navigator.clipboard.writeText(createdKey.key);
        setCopied(true);
        setTimeout(() => setCopied(false), 3000);
      } catch (error) {
        toast.error('复制失败');
      }
    }
  };

  // 吊销 Key
  const handleRevoke = async (key: APIKey) => {
    if (!confirm(`确认吊销 API Key「${key.label}」？吊销后将无法恢复。`)) {
      return;
    }

    try {
      await keysAPI.revoke(key.id);
      toast.success('API Key 已吊销');
      await fetchKeys();
    } catch (error) {
      toast.error('吊销失败');
    }
  };

  // 获取状态筛选标签
  const statusTabs = [
    { value: '', label: '全部' },
    { value: 'active', label: '活跃' },
    { value: 'revoked', label: '已吊销' },
    { value: 'expired', label: '已过期' },
  ];

  // 状态颜色
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'bg-green-100 text-green-700';
      case 'revoked': return 'bg-red-100 text-red-700';
      case 'expired': return 'bg-gray-100 text-gray-700';
      default: return 'bg-gray-100 text-gray-700';
    }
  };

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'active': return '活跃';
      case 'revoked': return '已吊销';
      case 'expired': return '已过期';
      default: return status;
    }
  };

  return (
    <div className="p-8">
      {/* 页面标题 */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">API Key 管理</h1>
          <p className="text-gray-600 mt-1">管理 API 密钥和访问权限</p>
        </div>
        <button
          onClick={handleAdd}
          className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors"
        >
          创建 API Key
        </button>
      </div>

      {/* 状态筛选 */}
      <div className="mb-6 flex gap-2">
        {statusTabs.map((tab) => (
          <button
            key={tab.value}
            onClick={() => setStatusFilter(tab.value)}
            className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
              statusFilter === tab.value
                ? 'bg-blue-600 text-white'
                : 'bg-white text-gray-700 hover:bg-gray-100 border border-gray-300'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Key 列表 */}
      {isLoading ? (
        <div className="text-center py-12">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
        </div>
      ) : keys.length === 0 ? (
        <div className="text-center py-12 bg-white rounded-lg border border-dashed border-gray-300">
          <p className="text-gray-500">暂无 API Key</p>
          <button
            onClick={handleAdd}
            className="mt-4 text-blue-600 hover:text-blue-700 font-medium"
          >
            创建第一个 Key →
          </button>
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">标签</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Key 值</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">绑定模型</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">优先级</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">有效期</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">创建时间</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">状态</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {keys.map((key) => (
                <tr key={key.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 font-medium text-gray-900">{key.label}</td>
                  <td className="px-6 py-4 font-mono text-sm text-gray-600">{key.key_prefix}•••••••••</td>
                  <td className="px-6 py-4 text-gray-600">{key.model_alias || '全部'}</td>
                  <td className="px-6 py-4">
                    {key.priority && (
                      <span className={`text-xs px-2 py-1 rounded ${
                        key.priority === 'high' ? 'bg-purple-100 text-purple-700' :
                        key.priority === 'low' ? 'bg-gray-100 text-gray-700' :
                        'bg-blue-100 text-blue-700'
                      }`}>
                        {key.priority === 'high' ? '高' : key.priority === 'low' ? '低' : '普通'}
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-4 text-gray-600">
                    {key.expires_at ? new Date(key.expires_at).toLocaleDateString('zh-CN') : '永久'}
                  </td>
                  <td className="px-6 py-4 text-gray-600">
                    {new Date(key.created_at).toLocaleDateString('zh-CN')}
                  </td>
                  <td className="px-6 py-4">
                    <span className={`text-xs px-2 py-1 rounded ${getStatusColor(key.status)}`}>
                      {getStatusLabel(key.status)}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    {key.status === 'active' && (
                      <div className="flex gap-2">
                        <button
                          onClick={() => handleViewQuota(key)}
                          className="text-blue-600 hover:text-blue-700 text-sm"
                        >
                          配额
                        </button>
                        <button
                          onClick={() => handleRevoke(key)}
                          className="text-red-600 hover:text-red-700 text-sm"
                        >
                          吊销
                        </button>
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* 创建成功弹窗 */}
      {showCreateDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
            {!createdKey ? (
              <>
                <h2 className="text-xl font-semibold text-gray-900 mb-4">创建 API Key</h2>
                <form onSubmit={handleSubmit} className="space-y-4">
                  {/* 标签 */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      标签 <span className="text-red-500">*</span>
                    </label>
                    <input
                      type="text"
                      value={formData.label}
                      onChange={(e) => setFormData({ ...formData, label: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder="开发者-张三"
                    />
                    <p className="text-xs text-gray-500 mt-1">1-64 字符，用于识别 Key 用途</p>
                  </div>

                  {/* 绑定模型 */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      绑定模型
                    </label>
                    <select
                      value={formData.model_alias}
                      onChange={(e) => setFormData({ ...formData, model_alias: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                      <option value="">全部模型</option>
                      {/* TODO: 从实际模型列表加载 */}
                    </select>
                    <p className="text-xs text-gray-500 mt-1">不选择则可访问所有模型</p>
                  </div>

                  {/* 有效期 */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      有效期
                    </label>
                    <input
                      type="date"
                      value={formData.expires_at}
                      onChange={(e) => setFormData({ ...formData, expires_at: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                    <p className="text-xs text-gray-500 mt-1">不填则永久有效</p>
                  </div>

                  {/* 备注 */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      备注
                    </label>
                    <input
                      type="text"
                      value={formData.note}
                      onChange={(e) => setFormData({ ...formData, note: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder="用途说明"
                      maxLength={200}
                    />
                  </div>

                  {/* 配额设置 */}
                  <div className="border-t pt-4 mt-4">
                    <h3 className="text-sm font-medium text-gray-900 mb-3">配额设置（可选）</h3>

                    {/* 软上限 */}
                    <div className="mb-4">
                      <label className="block text-sm font-medium text-gray-700 mb-1">
                        每日软上限（Token）
                      </label>
                      <input
                        type="number"
                        value={formData.daily_token_soft_limit}
                        onChange={(e) => setFormData({ ...formData, daily_token_soft_limit: e.target.value })}
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                        placeholder="100000"
                        min="0"
                      />
                      <p className="text-xs text-gray-500 mt-1">超过此值将发送告警通知</p>
                    </div>

                    {/* 硬上限 */}
                    <div className="mb-4">
                      <label className="block text-sm font-medium text-gray-700 mb-1">
                        每日硬上限（Token）
                      </label>
                      <input
                        type="number"
                        value={formData.daily_token_hard_limit}
                        onChange={(e) => setFormData({ ...formData, daily_token_hard_limit: e.target.value })}
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                        placeholder="200000"
                        min="0"
                      />
                      <p className="text-xs text-gray-500 mt-1">超过此值将拒绝请求（429）</p>
                    </div>

                    {/* 优先级 */}
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">
                        优先级
                      </label>
                      <select
                        value={formData.priority}
                        onChange={(e) => setFormData({ ...formData, priority: e.target.value })}
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                      >
                        <option value="normal">普通</option>
                        <option value="high">高</option>
                        <option value="low">低</option>
                      </select>
                      <p className="text-xs text-gray-500 mt-1">资源紧张时的调度优先级</p>
                    </div>
                  </div>

                  {/* 按钮 */}
                  <div className="flex gap-2">
                    <button
                      type="submit"
                      disabled={isSubmitting}
                      className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed transition-colors"
                    >
                      {isSubmitting ? '创建中…' : '创建'}
                    </button>
                    <button
                      type="button"
                      onClick={() => setShowCreateDialog(false)}
                      className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                    >
                      取消
                    </button>
                  </div>
                </form>
              </>
            ) : (
              <>
                <div className="text-center">
                  <div className="w-12 h-12 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
                    <span className="text-green-600 text-2xl">✓</span>
                  </div>
                  <h2 className="text-xl font-semibold text-gray-900 mb-2">API Key 创建成功</h2>
                  <p className="text-gray-600 mb-4">请立即复制以下 Key，关闭此弹窗后将无法再次查看完整内容。</p>

                  {/* Key 值 */}
                  <div className="bg-gray-50 rounded-lg p-4 mb-4">
                    <code className="text-sm font-mono text-gray-900 break-all">{createdKey.key}</code>
                  </div>

                  {/* 复制按钮 */}
                  <button
                    onClick={handleCopy}
                    className={`w-full mb-4 px-4 py-2 rounded-lg transition-colors ${
                      copied
                        ? 'bg-green-600 text-white'
                        : 'bg-blue-600 text-white hover:bg-blue-700'
                    }`}
                  >
                    {copied ? '已复制 ✓' : '复制 Key'}
                  </button>

                  <button
                    onClick={() => setShowCreateDialog(false)}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                  >
                    我已复制，关闭
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* 配额详情弹窗 */}
      {showQuotaDialog && selectedKeyQuota && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">配额状态</h2>

            <div className="space-y-4">
              {/* Key 信息 */}
              <div className="bg-gray-50 rounded-lg p-4">
                <p className="text-sm text-gray-600">Key 标签</p>
                <p className="text-lg font-semibold text-gray-900">{selectedKeyQuota.label}</p>
              </div>

              {/* 优先级 */}
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600">优先级</span>
                <span className={`text-xs px-2 py-1 rounded ${
                  selectedKeyQuota.priority === 'high' ? 'bg-purple-100 text-purple-700' :
                  selectedKeyQuota.priority === 'low' ? 'bg-gray-100 text-gray-700' :
                  'bg-blue-100 text-blue-700'
                }`}>
                  {selectedKeyQuota.priority === 'high' ? '高' : selectedKeyQuota.priority === 'low' ? '低' : '普通'}
                </span>
              </div>

              {/* 软上限 */}
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600">每日软上限</span>
                <span className="font-medium text-gray-900">
                  {selectedKeyQuota.daily_token_soft_limit
                    ? selectedKeyQuota.daily_token_soft_limit.toLocaleString()
                    : '未设置'}
                </span>
              </div>

              {/* 硬上限 */}
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600">每日硬上限</span>
                <span className="font-medium text-gray-900">
                  {selectedKeyQuota.daily_token_hard_limit
                    ? selectedKeyQuota.daily_token_hard_limit.toLocaleString()
                    : '未设置'}
                </span>
              </div>

              {/* 说明 */}
              <div className="bg-blue-50 rounded-lg p-3 text-sm text-blue-800">
                <p className="font-medium mb-1">配额说明：</p>
                <ul className="list-disc list-inside space-y-1 text-xs">
                  <li>软上限：超过时发送告警通知，不拦截请求</li>
                  <li>硬上限：超过时直接拒绝请求，返回 429 错误</li>
                  <li>每日 00:00 重置配额</li>
                </ul>
              </div>
            </div>

            <button
              onClick={() => setShowQuotaDialog(false)}
              className="w-full mt-6 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              关闭
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
