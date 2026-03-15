import { useEffect, useState, type FormEvent } from 'react';
import { keysAPI } from '../services/api';
import type { APIKey, CreateKeyRequest } from '../services/types';
import { useToast } from '../components/Toast';

export default function KeysPage() {
  const toast = useToast();
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [createdKey, setCreatedKey] = useState<APIKey | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [copied, setCopied] = useState(false);

  // 表单状态
  const [formData, setFormData] = useState({
    label: '',
    model_alias: '',
    expires_at: '',
    note: '',
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
    });
    setCreatedKey(null);
    setShowCreateDialog(true);
    setCopied(false);
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
                    {key.status === 'active' ? (
                      <button
                        onClick={() => handleRevoke(key)}
                        className="text-red-600 hover:text-red-700 text-sm"
                      >
                        吊销
                      </button>
                    ) : (
                      <span className="text-gray-400 text-sm">已失效</span>
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
    </div>
  );
}
