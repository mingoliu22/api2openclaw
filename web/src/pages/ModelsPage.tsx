import { useEffect, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { modelsAPI } from '../services/api';
import type { Model, CreateModelRequest, UpdateModelRequest } from '../services/types';
import { useToast } from '../components/Toast';
import CostConfigDialog from '../components/CostConfigDialog';

export default function ModelsPage() {
  const toast = useToast();
  const navigate = useNavigate();
  const [models, setModels] = useState<Model[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [showCostConfig, setShowCostConfig] = useState(false);
  const [selectedModelForCost, setSelectedModelForCost] = useState<Model | null>(null);
  const [editingModel, setEditingModel] = useState<Model | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // 表单状态
  const [formData, setFormData] = useState({
    alias: '',
    model_id: '',
    base_url: '',
    api_key: '',
    note: '',
    // v0.3.0: 模型能力字段
    supports_streaming: true,
    supports_tool_use: false,
    supports_json_mode: false,
    context_window: 4096,
    model_family: 'other',
  });

  // 连通性测试状态
  const [testingConnection, setTestingConnection] = useState(false);
  const [connectionResult, setConnectionResult] = useState<{
    ok: boolean;
    latency_ms?: number;
    error?: string;
  } | null>(null);

  // 获取模型列表
  const fetchModels = async () => {
    try {
      const response = await modelsAPI.list(false);
      setModels(response.data.data);
    } catch (error) {
      toast.error('获取模型列表失败');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchModels();
    // 每 30 秒刷新一次在线状态
    const interval = setInterval(fetchModels, 30000);
    return () => clearInterval(interval);
  }, []);

  // 打开添加表单
  const handleAdd = () => {
    setEditingModel(null);
    setFormData({
      alias: '',
      model_id: '',
      base_url: '',
      api_key: '',
      note: '',
      supports_streaming: true,
      supports_tool_use: false,
      supports_json_mode: false,
      context_window: 4096,
      model_family: 'other',
    });
    setConnectionResult(null);
    setShowForm(true);
  };

  // 打开编辑表单
  const handleEdit = (model: Model) => {
    setEditingModel(model);
    setFormData({
      alias: model.alias,
      model_id: model.model_id,
      base_url: model.base_url,
      api_key: '', // API Key 不回填
      note: model.note || '',
      supports_streaming: model.supports_streaming ?? true,
      supports_tool_use: model.supports_tool_use ?? false,
      supports_json_mode: model.supports_json_mode ?? false,
      context_window: model.context_window ?? 4096,
      model_family: model.model_family || 'other',
    });
    setConnectionResult(null);
    setShowForm(true);
  };

  // 测试连通性
  const handleTestConnection = async () => {
    if (!formData.base_url) {
      return;
    }

    setTestingConnection(true);
    setConnectionResult(null);

    try {
      const response = await modelsAPI.testConnection({
        base_url: formData.base_url,
        api_key: formData.api_key || undefined,
      });
      setConnectionResult(response.data);
    } catch (error) {
      setConnectionResult({
        ok: false,
        error: '连接测试失败',
      });
    } finally {
      setTestingConnection(false);
    }
  };

  // 提交表单
  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();

    // 前端校验
    if (!formData.alias || !formData.model_id || !formData.base_url) {
      toast.error('请填写所有必填字段');
      return;
    }

    // 校验别名格式
    const aliasRegex = /^[a-zA-Z0-9._-]+$/;
    if (!aliasRegex.test(formData.alias)) {
      toast.error('别名只能包含字母、数字、.-_');
      return;
    }

    // 校验 URL 格式
    try {
      new URL(formData.base_url);
    } catch {
      toast.error('请输入有效的 URL');
      return;
    }

    setIsSubmitting(true);

    try {
      if (editingModel) {
        // 更新模型
        const updateData: UpdateModelRequest = {
          alias: formData.alias,
          model_id: formData.model_id,
          base_url: formData.base_url,
          note: formData.note,
          supports_streaming: formData.supports_streaming,
          supports_tool_use: formData.supports_tool_use,
          supports_json_mode: formData.supports_json_mode,
          context_window: formData.context_window,
          model_family: formData.model_family,
        };
        if (formData.api_key) {
          updateData.api_key = formData.api_key;
        }
        await modelsAPI.update(editingModel.id, updateData);
        toast.success('模型配置已更新并生效');
      } else {
        // 创建模型
        const createData: CreateModelRequest = {
          alias: formData.alias,
          model_id: formData.model_id,
          base_url: formData.base_url,
          note: formData.note,
          supports_streaming: formData.supports_streaming,
          supports_tool_use: formData.supports_tool_use,
          supports_json_mode: formData.supports_json_mode,
          context_window: formData.context_window,
          model_family: formData.model_family,
        };
        if (formData.api_key) {
          createData.api_key = formData.api_key;
        }
        await modelsAPI.create(createData);
        toast.success('模型配置已保存并生效');
      }

      setShowForm(false);
      await fetchModels();
    } catch (error: any) {
      if (error.response?.status === 409) {
        toast.error('此别名已存在');
      } else {
        toast.error(error.response?.data?.error?.message || '保存失败');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  // 删除模型
  const handleDelete = async (model: Model) => {
    if (!confirm(`确认删除模型「${model.alias}」？删除后路由至此模型的请求将立即返回错误。此操作不可恢复。`)) {
      return;
    }

    try {
      await modelsAPI.delete(model.id);
      toast.success('模型已删除');
      await fetchModels();
    } catch (error) {
      toast.error('删除失败');
    }
  };

  // 切换启用状态
  const handleToggleActive = async (model: Model) => {
    try {
      await modelsAPI.toggleActive(model.id, !model.is_active);
      toast.success(model.is_active ? '模型已禁用' : '模型已启用');
      await fetchModels();
    } catch (error) {
      toast.error('操作失败');
    }
  };

  // 打开成本配置
  const handleCostConfig = (model: Model) => {
    setSelectedModelForCost(model);
    setShowCostConfig(true);
  };

  return (
    <div className="p-8">
      {/* 页面标题 */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">模型配置</h1>
          <p className="text-gray-600 mt-1">管理本地模型实例配置</p>
        </div>
        <button
          onClick={handleAdd}
          className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors"
        >
          添加模型
        </button>
      </div>

      {/* 添加/编辑表单 */}
      {showForm && (
        <div className="mb-6 bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">
            {editingModel ? '编辑模型' : '添加模型'}
          </h2>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              {/* 对外别名 */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  对外别名 <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={formData.alias}
                  onChange={(e) => setFormData({ ...formData, alias: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="gpt-4"
                />
                <p className="text-xs text-gray-500 mt-1">1-64 字符，仅允许字母、数字、.-_</p>
              </div>

              {/* 实际模型标识 */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  实际模型标识 <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={formData.model_id}
                  onChange={(e) => setFormData({ ...formData, model_id: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="qwen2.5-72b-instruct"
                />
                <p className="text-xs text-gray-500 mt-1">1-128 字符，与后端 model 字段一致</p>
              </div>

              {/* BaseURL */}
              <div>
                <div className="flex items-center justify-between mb-1">
                  <label className="block text-sm font-medium text-gray-700">
                    BaseURL <span className="text-red-500">*</span>
                  </label>
                  <button
                    type="button"
                    onClick={() => navigate('/dashboard/models/deploy-guide')}
                    className="text-sm text-blue-600 hover:text-blue-700"
                  >
                    查看部署指南 →
                  </button>
                </div>
                <input
                  type="url"
                  value={formData.base_url}
                  onChange={(e) => setFormData({ ...formData, base_url: e.target.value })}
                  onBlur={handleTestConnection}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="http://localhost:11434/v1"
                />
                {/* 连通性测试状态 */}
                {testingConnection && (
                  <div className="flex items-center gap-2 mt-1 text-sm text-gray-500">
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600"></div>
                    <span>检测中...</span>
                  </div>
                )}
                {connectionResult && (
                  <div className={`flex items-center gap-2 mt-1 text-sm ${
                    connectionResult.ok ? 'text-green-600' : 'text-red-600'
                  }`}>
                    {connectionResult.ok ? (
                      <>
                        <span>✓</span>
                        <span>连接成功 ({connectionResult.latency_ms}ms)</span>
                      </>
                    ) : (
                      <>
                        <span>✕</span>
                        <span>{connectionResult.error}</span>
                      </>
                    )}
                  </div>
                )}
                <p className="text-xs text-gray-500 mt-1">失焦时自动检测连通性</p>
              </div>

              {/* API Key */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  API Key
                </label>
                <input
                  type="password"
                  value={formData.api_key}
                  onChange={(e) => setFormData({ ...formData, api_key: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="sk-xxxx（可选）"
                />
                <p className="text-xs text-gray-500 mt-1">若后端无需鉴权可留空</p>
              </div>
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
                placeholder="模型用途说明"
                maxLength={200}
              />
              <p className="text-xs text-gray-500 mt-1">最多 200 字符</p>
            </div>

            {/* v0.3.0: 模型能力配置 */}
            <div className="border-t border-gray-200 pt-4">
              <h3 className="text-sm font-semibold text-gray-900 mb-3">模型能力</h3>
              <div className="grid grid-cols-3 gap-4">
                {/* Streaming 支持 */}
                <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div>
                    <label className="text-sm font-medium text-gray-700">Streaming</label>
                    <p className="text-xs text-gray-500">SSE 流式输出</p>
                  </div>
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      checked={formData.supports_streaming}
                      onChange={(e) => setFormData({ ...formData, supports_streaming: e.target.checked })}
                      className="sr-only peer"
                    />
                    <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                  </label>
                </div>

                {/* Tool Use 支持 */}
                <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div>
                    <label className="text-sm font-medium text-gray-700">Tool Use</label>
                    <p className="text-xs text-gray-500">函数调用</p>
                  </div>
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      checked={formData.supports_tool_use}
                      onChange={(e) => setFormData({ ...formData, supports_tool_use: e.target.checked })}
                      className="sr-only peer"
                    />
                    <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                  </label>
                </div>

                {/* JSON Mode 支持 */}
                <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div>
                    <label className="text-sm font-medium text-gray-700">JSON Mode</label>
                    <p className="text-xs text-gray-500">结构化输出</p>
                  </div>
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      checked={formData.supports_json_mode}
                      onChange={(e) => setFormData({ ...formData, supports_json_mode: e.target.checked })}
                      className="sr-only peer"
                    />
                    <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                  </label>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4 mt-4">
                {/* 上下文窗口 */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    上下文窗口（tokens）
                  </label>
                  <input
                    type="number"
                    value={formData.context_window}
                    onChange={(e) => setFormData({ ...formData, context_window: parseInt(e.target.value) || 4096 })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                    min="1024"
                    max="2000000"
                    step="1024"
                  />
                  <p className="text-xs text-gray-500 mt-1">模型的上下文窗口大小</p>
                </div>

                {/* 模型家族 */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    模型家族
                  </label>
                  <select
                    value={formData.model_family}
                    onChange={(e) => setFormData({ ...formData, model_family: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="other">Other</option>
                    <option value="qwen">Qwen (通义千问)</option>
                    <option value="deepseek">DeepSeek</option>
                    <option value="llama">Llama</option>
                    <option value="openai">OpenAI</option>
                  </select>
                  <p className="text-xs text-gray-500 mt-1">用于格式归一化策略选择</p>
                </div>
              </div>
            </div>

            {/* 表单按钮 */}
            <div className="flex gap-2">
              <button
                type="submit"
                disabled={isSubmitting}
                className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed transition-colors"
              >
                {isSubmitting ? '保存中…' : '保存'}
              </button>
              <button
                type="button"
                onClick={() => setShowForm(false)}
                className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
              >
                取消
              </button>
            </div>
          </form>
        </div>
      )}

      {/* 模型列表 */}
      {isLoading ? (
        <div className="text-center py-12">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
        </div>
      ) : models.length === 0 ? (
        <div className="text-center py-12 bg-white rounded-lg border border-dashed border-gray-300">
          <p className="text-gray-500">暂无模型配置</p>
          <button
            onClick={handleAdd}
            className="mt-4 text-blue-600 hover:text-blue-700 font-medium"
          >
            添加第一个模型 →
          </button>
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">对外别名</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">实际模型</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">BaseURL</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">API Key</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">在线状态</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">状态</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {models.map((model) => (
                <tr key={model.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 font-medium text-gray-900">{model.alias}</td>
                  <td className="px-6 py-4 text-gray-600">{model.model_id}</td>
                  <td className="px-6 py-4 text-gray-600 text-sm font-mono">{model.base_url}</td>
                  <td className="px-6 py-4 text-gray-600">
                    {model.note ? `•••••••••` : '(无)'}
                  </td>
                  <td className="px-6 py-4">
                    {model.health_status?.status === 'healthy' ? (
                      <span className="inline-flex items-center gap-1 text-green-600">
                        <span className="w-2 h-2 bg-green-500 rounded-full"></span>
                        在线
                      </span>
                    ) : model.health_status?.status === 'unhealthy' ? (
                      <span className="inline-flex items-center gap-1 text-red-600">
                        <span className="w-2 h-2 bg-red-500 rounded-full"></span>
                        离线
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 text-gray-400">
                        <span className="w-2 h-2 bg-gray-300 rounded-full"></span>
                        未知
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-4">
                    <button
                      onClick={() => handleToggleActive(model)}
                      className={`text-xs px-2 py-1 rounded ${
                        model.is_active
                          ? 'bg-green-100 text-green-700 hover:bg-green-200'
                          : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                      }`}
                    >
                      {model.is_active ? '已启用' : '已禁用'}
                    </button>
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => handleEdit(model)}
                        className="text-blue-600 hover:text-blue-700 text-sm"
                      >
                        编辑
                      </button>
                      <button
                        onClick={() => handleCostConfig(model)}
                        className="text-green-600 hover:text-green-700 text-sm"
                      >
                        成本
                      </button>
                      <button
                        onClick={() => handleDelete(model)}
                        className="text-red-600 hover:text-red-700 text-sm"
                      >
                        删除
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* 成本配置弹窗 */}
      {showCostConfig && selectedModelForCost && (
        <CostConfigDialog
          modelId={selectedModelForCost.id}
          modelAlias={selectedModelForCost.alias}
          onClose={() => {
            setShowCostConfig(false);
            setSelectedModelForCost(null);
          }}
        />
      )}
    </div>
  );
}
