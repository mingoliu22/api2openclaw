import { useState, useEffect, type FormEvent } from 'react';
import { costAPI, modelsAPI } from '../services/api';
import { useToast } from './Toast';

interface CostConfig {
  id?: string;
  model_id: string;
  model_alias: string;
  gpu_count: number;
  power_per_gpu_w: number;
  electricity_price_per_kwh: number;
  depreciation_per_gpu_month: number;
  pue: number;
  effective_from: string;
  created_at: string;
}

interface CostConfigDialogProps {
  modelId: string;
  modelAlias: string;
  onClose: () => void;
}

export default function CostConfigDialog({ modelId, modelAlias, onClose }: CostConfigDialogProps) {
  const toast = useToast();
  const [configs, setConfigs] = useState<CostConfig[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // 表单状态
  const [formData, setFormData] = useState({
    gpu_count: 4,
    power_per_gpu_w: 400,
    electricity_price_per_kwh: 0.8,
    depreciation_per_gpu_month: 5000,
    pue: 1.3,
    effective_from: new Date().toISOString().slice(0, 16),
  });

  // 获取成本配置列表
  const fetchConfigs = async () => {
    try {
      const response = await costAPI.getModelConfigs(modelId);
      setConfigs(response.data.data || []);
    } catch (error) {
      toast.error('获取成本配置失败');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchConfigs();
  }, [modelId]);

  // 提交表单
  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();

    setIsSubmitting(true);

    try {
      await costAPI.createConfig({
        model_id: modelId,
        ...formData,
        effective_from: new Date(formData.effective_from).toISOString(),
      });

      toast.success('成本配置已创建');
      await fetchConfigs();
      setShowCreateForm(false);

      // 重置表单
      setFormData({
        gpu_count: 4,
        power_per_gpu_w: 400,
        electricity_price_per_kwh: 0.8,
        depreciation_per_gpu_month: 5000,
        pue: 1.3,
        effective_from: new Date().toISOString().slice(0, 16),
      });
    } catch (error: any) {
      toast.error(error.response?.data?.error?.message || '创建失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  // 删除配置
  const handleDelete = async (id: string) => {
    if (!confirm('确认删除此成本配置？')) {
      return;
    }

    try {
      await costAPI.deleteConfig(id);
      toast.success('成本配置已删除');
      await fetchConfigs();
    } catch (error) {
      toast.error('删除失败');
    }
  };

  // 计算预计每千 token 成本
  const calculateCostPer1k = () => {
    const hourlyCost =
      (formData.gpu_count * formData.power_per_gpu_w * formData.pue * formData.electricity_price_per_kwh / 1000) +
      (formData.gpu_count * formData.depreciation_per_gpu_month / 720);

    // 假设每小时产出 360K tokens (100 tokens/s)
    const tokensPerHour = 360000;
    const costPer1k = hourlyCost / (tokensPerHour / 1000);

    return costPer1k.toFixed(4);
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full mx-4 max-h-[90vh] overflow-hidden flex flex-col">
        {/* 标题 */}
        <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-gray-900">成本配置</h2>
            <p className="text-sm text-gray-600 mt-1">模型: {modelAlias}</p>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
          >
            ✕
          </button>
        </div>

        {/* 内容 */}
        <div className="flex-1 overflow-y-auto p-6">
          {!showCreateForm ? (
            <>
              {/* 现有配置列表 */}
              {isLoading ? (
                <div className="text-center py-8">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
                </div>
              ) : configs.length === 0 ? (
                <div className="text-center py-12 bg-gray-50 rounded-lg border-2 border-dashed border-gray-300">
                  <p className="text-gray-500 mb-4">暂无成本配置</p>
                  <button
                    onClick={() => setShowCreateForm(true)}
                    className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
                  >
                    创建配置
                  </button>
                </div>
              ) : (
                <div className="space-y-4">
                  {configs.map((config) => (
                    <div key={config.id} className="bg-gray-50 rounded-lg p-4">
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-2">
                            <span className="text-sm font-medium text-gray-900">
                              生效时间: {new Date(config.effective_from).toLocaleString('zh-CN')}
                            </span>
                            <span className="text-xs px-2 py-1 bg-blue-100 text-blue-700 rounded">
                              {config.id === configs[0].id ? '当前生效' : '历史版本'}
                            </span>
                          </div>
                          <div className="grid grid-cols-2 md:grid-cols-3 gap-4 text-sm">
                            <div>
                              <span className="text-gray-600">GPU 数量:</span>
                              <span className="ml-2 font-medium">{config.gpu_count} 张</span>
                            </div>
                            <div>
                              <span className="text-gray-600">单卡功耗:</span>
                              <span className="ml-2 font-medium">{config.power_per_gpu_w} W</span>
                            </div>
                            <div>
                              <span className="text-gray-600">电费:</span>
                              <span className="ml-2 font-medium">¥{config.electricity_price_per_kwh}/度</span>
                            </div>
                            <div>
                              <span className="text-gray-600">单卡折旧:</span>
                              <span className="ml-2 font-medium">¥{config.depreciation_per_gpu_month}/月</span>
                            </div>
                            <div>
                              <span className="text-gray-600">PUE 系数:</span>
                              <span className="ml-2 font-medium">{config.pue}</span>
                            </div>
                          </div>
                        </div>
                        <button
                          onClick={() => config.id && handleDelete(config.id)}
                          className="ml-4 text-red-600 hover:text-red-700 text-sm"
                        >
                          删除
                        </button>
                      </div>
                    </div>
                  ))}
                  <button
                    onClick={() => setShowCreateForm(true)}
                    className="w-full py-3 border-2 border-dashed border-gray-300 rounded-lg text-gray-600 hover:border-blue-500 hover:text-blue-600 transition-colors"
                  >
                    + 添加新版本
                  </button>
                </div>
              )}
            </>
          ) : (
            <>
              {/* 创建表单 */}
              <h3 className="text-lg font-semibold text-gray-900 mb-4">创建成本配置</h3>
              <form onSubmit={handleSubmit} className="space-y-4">
                {/* GPU 数量 */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    GPU 数量（张） <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="number"
                    value={formData.gpu_count}
                    onChange={(e) => setFormData({ ...formData, gpu_count: parseInt(e.target.value) || 0 })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                    min="1"
                    required
                  />
                  <p className="text-xs text-gray-500 mt-1">该模型使用的 GPU 数量</p>
                </div>

                {/* 单卡功耗 */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    单卡满载功耗（瓦特） <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="number"
                    value={formData.power_per_gpu_w}
                    onChange={(e) => setFormData({ ...formData, power_per_gpu_w: parseInt(e.target.value) || 0 })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                    min="1"
                    required
                  />
                  <p className="text-xs text-gray-500 mt-1">例如: A100=400W, H100=700W</p>
                </div>

                {/* 电费单价 */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    电费单价（元/度） <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="number"
                    step="0.01"
                    value={formData.electricity_price_per_kwh}
                    onChange={(e) => setFormData({ ...formData, electricity_price_per_kwh: parseFloat(e.target.value) || 0 })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                    min="0"
                    required
                  />
                  <p className="text-xs text-gray-500 mt-1">典型值: 0.6~1.2 元/度</p>
                </div>

                {/* 硬件折旧 */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    单卡月折旧（元） <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="number"
                    value={formData.depreciation_per_gpu_month}
                    onChange={(e) => setFormData({ ...formData, depreciation_per_gpu_month: parseInt(e.target.value) || 0 })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                    min="0"
                    required
                  />
                  <p className="text-xs text-gray-500 mt-1">按 36 个月摊销，H100 约 5000 元/月/张</p>
                </div>

                {/* PUE 系数 */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    PUE 系数 <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="number"
                    step="0.01"
                    value={formData.pue}
                    onChange={(e) => setFormData({ ...formData, pue: parseFloat(e.target.value) || 1 })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                    min="1"
                    max="3"
                    step="0.01"
                    required
                  />
                  <p className="text-xs text-gray-500 mt-1">数据中心电力使用效率，典型值 1.2~1.5</p>
                </div>

                {/* 生效时间 */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    生效时间 <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="datetime-local"
                    value={formData.effective_from}
                    onChange={(e) => setFormData({ ...formData, effective_from: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                    required
                  />
                  <p className="text-xs text-gray-500 mt-1">此配置的生效时间（支持版本管理）</p>
                </div>

                {/* 成本预估 */}
                <div className="bg-blue-50 rounded-lg p-4">
                  <h4 className="text-sm font-medium text-blue-900 mb-2">成本预估</h4>
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <span className="text-blue-700">每小时电费:</span>
                      <span className="ml-2 font-medium">
                        ¥{((formData.gpu_count * formData.power_per_gpu_w * formData.pue * formData.electricity_price_per_kwh / 1000)).toFixed(2)}
                      </span>
                    </div>
                    <div>
                      <span className="text-blue-700">每小时折旧:</span>
                      <span className="ml-2 font-medium">
                        ¥{((formData.gpu_count * formData.depreciation_per_gpu_month / 720)).toFixed(2)}
                      </span>
                    </div>
                    <div className="col-span-2">
                      <span className="text-blue-700">预计每千 Token 成本:</span>
                      <span className="ml-2 font-bold text-lg">
                        ¥{calculateCostPer1k()}
                      </span>
                      <span className="text-xs text-blue-600 ml-2">(假设 100 tokens/s)</span>
                    </div>
                  </div>
                </div>

                {/* 按钮 */}
                <div className="flex gap-2 pt-4">
                  <button
                    type="submit"
                    disabled={isSubmitting}
                    className="flex-1 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed transition-colors"
                  >
                    {isSubmitting ? '创建中…' : '创建配置'}
                  </button>
                  <button
                    type="button"
                    onClick={() => setShowCreateForm(false)}
                    className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                  >
                    取消
                  </button>
                </div>
              </form>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
