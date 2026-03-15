import { useEffect, useState, type ChangeEvent, type FormEvent } from 'react';
import { pluginsAPI } from '../services/api';
import type { Plugin, BuiltinPlugin } from '../services/types';
import { useToast } from '../components/Toast';

export default function PluginsPage() {
  const toast = useToast();
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [builtinPlugins, setBuiltinPlugins] = useState<BuiltinPlugin[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // 对话框状态
  const [showUploadDialog, setShowUploadDialog] = useState(false);
  const [showConfigDialog, setShowConfigDialog] = useState(false);
  const [showTestDialog, setShowTestDialog] = useState(false);
  const [selectedPlugin, setSelectedPlugin] = useState<Plugin | null>(null);

  // 表单状态
  const [uploadForm, setUploadForm] = useState({
    file: null as File | null,
    name: '',
    symbol: '',
    config: '',
  });

  const [testForm, setTestForm] = useState({
    input_format: 'deepseek',
    output_format: 'openclaw',
    test_data: '',
  });

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [testResult, setTestResult] = useState<any>(null);

  // 获取插件列表
  const fetchPlugins = async () => {
    setIsLoading(true);
    try {
      const [pluginsRes, builtinRes] = await Promise.all([
        pluginsAPI.list(),
        pluginsAPI.getBuiltin(),
      ]);
      setPlugins(pluginsRes.data.data);
      setBuiltinPlugins(builtinRes.data.data);
    } catch (error) {
      toast.error('获取插件列表失败');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchPlugins();
  }, []);

  // 上传插件
  const handleUpload = async (e: FormEvent) => {
    e.preventDefault();
    if (!uploadForm.file) {
      toast.error('请选择插件文件');
      return;
    }

    setIsSubmitting(true);
    try {
      await pluginsAPI.upload(uploadForm.file, {
        name: uploadForm.name || undefined,
        symbol: uploadForm.symbol || undefined,
        config: uploadForm.config || undefined,
      });
      toast.success('插件上传成功');
      setShowUploadDialog(false);
      setUploadForm({ file: null, name: '', symbol: '', config: '' });
      await fetchPlugins();
    } catch (error: any) {
      toast.error(error.response?.data?.error || '上传失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  // 启用插件
  const handleEnable = async (plugin: Plugin) => {
    setIsSubmitting(true);
    try {
      await pluginsAPI.enable(plugin.name);
      toast.success('插件已启用');
      await fetchPlugins();
    } catch (error: any) {
      toast.error(error.response?.data?.error || '启用失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  // 禁用插件
  const handleDisable = async (plugin: Plugin) => {
    setIsSubmitting(true);
    try {
      await pluginsAPI.disable(plugin.name);
      toast.success('插件已禁用');
      await fetchPlugins();
    } catch (error: any) {
      toast.error(error.response?.data?.error || '禁用失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  // 打开配置对话框
  const handleConfig = (plugin: Plugin) => {
    setSelectedPlugin(plugin);
    setShowConfigDialog(true);
  };

  // 保存配置
  const handleSaveConfig = async () => {
    if (!selectedPlugin) return;

    setIsSubmitting(true);
    try {
      await pluginsAPI.updateConfig(selectedPlugin.name, selectedPlugin.config);
      toast.success('配置已保存');
      setShowConfigDialog(false);
      setSelectedPlugin(null);
    } catch (error: any) {
      toast.error(error.response?.data?.error || '保存失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  // 测试插件
  const handleTest = async (e: FormEvent) => {
    e.preventDefault();
    if (!selectedPlugin) return;

    setIsSubmitting(true);
    try {
      const response = await pluginsAPI.test(selectedPlugin.name, testForm);
      setTestResult(response.data.data);
    } catch (error: any) {
      toast.error(error.response?.data?.error || '测试失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  // 获取插件类型标签
  const getTypeBadge = (type: string) => {
    switch (type) {
      case 'builtin':
        return 'bg-blue-100 text-blue-700';
      case 'so':
        return 'bg-purple-100 text-purple-700';
      default:
        return 'bg-gray-100 text-gray-700';
    }
  };

  // 下载插件
  const handleDownload = async (plugin: Plugin) => {
    try {
      const response = await pluginsAPI.download(plugin.name);
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.download = `${plugin.name}.so`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
      toast.success('插件已下载');
    } catch (error) {
      toast.error('下载失败');
    }
  };

  return (
    <div className="p-8">
      {/* 页面标题 */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">插件市场</h1>
          <p className="text-gray-600 mt-1">管理格式转换插件</p>
        </div>
        <button
          onClick={() => setShowUploadDialog(true)}
          className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors"
        >
          上传插件
        </button>
      </div>

      {/* 内置插件 */}
      <div className="mb-8">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">内置插件</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {builtinPlugins.map((plugin) => (
            <div
              key={plugin.name}
              className="bg-white rounded-lg shadow-sm border border-gray-200 p-4 hover:shadow-md transition-shadow"
            >
              <div className="flex items-start justify-between mb-3">
                <div>
                  <h3 className="font-semibold text-gray-900">{plugin.name}</h3>
                  <p className="text-sm text-gray-600 mt-1">{plugin.description}</p>
                </div>
                <span className={`text-xs px-2 py-1 rounded ${getTypeBadge(plugin.type)}`}>
                  {plugin.type}
                </span>
              </div>

              <div className="text-xs text-gray-500 space-y-1">
                <div>版本: {plugin.version}</div>
                <div>作者: {plugin.author}</div>
                <div className="flex gap-1 flex-wrap">
                  输入: {plugin.input_formats.join(', ')} → 输出: {plugin.output_formats.join(', ')}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* 已安装插件 */}
      <div>
        <h2 className="text-lg font-semibold text-gray-900 mb-4">已安装插件</h2>
        {isLoading ? (
          <div className="text-center py-12">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          </div>
        ) : plugins.length === 0 ? (
          <div className="text-center py-12 bg-white rounded-lg border border-dashed border-gray-300">
            <p className="text-gray-500">暂无已安装的插件</p>
          </div>
        ) : (
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
            <table className="w-full">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">名称</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">类型</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">状态</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {plugins.map((plugin) => (
                  <tr key={plugin.name} className="hover:bg-gray-50">
                    <td className="px-6 py-4 font-medium text-gray-900">{plugin.name}</td>
                    <td className="px-6 py-4">
                      <span className={`text-xs px-2 py-1 rounded ${getTypeBadge(plugin.type)}`}>
                        {plugin.type === 'builtin' ? '内置' : '外部'}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <span className={`text-xs px-2 py-1 rounded ${plugin.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-700'}`}>
                        {plugin.enabled ? '已启用' : '未启用'}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex gap-2">
                        <button
                          onClick={() => handleConfig(plugin)}
                          className="text-blue-600 hover:text-blue-700 text-sm"
                        >
                          配置
                        </button>
                        {plugin.enabled ? (
                          <button
                            onClick={() => handleDisable(plugin)}
                            className="text-red-600 hover:text-red-700 text-sm"
                          >
                            禁用
                          </button>
                        ) : (
                          <button
                            onClick={() => handleEnable(plugin)}
                            className="text-green-600 hover:text-green-700 text-sm"
                          >
                            启用
                          </button>
                        )}
                        <button
                          onClick={() => {
                            setSelectedPlugin(plugin);
                            setShowTestDialog(true);
                          }}
                          className="text-purple-600 hover:text-purple-700 text-sm"
                        >
                          测试
                        </button>
                        {plugin.type === 'so' && (
                          <button
                            onClick={() => handleDownload(plugin)}
                            className="text-gray-600 hover:text-gray-700 text-sm"
                          >
                            下载
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* 上传插件对话框 */}
      {showUploadDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">上传插件</h2>
            <form onSubmit={handleUpload} className="space-y-4">
              {/* 文件选择 */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  插件文件 (.so) <span className="text-red-500">*</span>
                </label>
                <input
                  type="file"
                  accept=".so"
                  onChange={(e: ChangeEvent<HTMLInputElement>) => {
                    const file = e.target.files?.[0];
                    if (file) {
                      setUploadForm({ ...uploadForm, file });
                    }
                  }}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  required
                />
                {uploadForm.file && (
                  <p className="text-xs text-gray-500 mt-1">已选择: {uploadForm.file.name}</p>
                )}
              </div>

              {/* 插件名称 */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  插件名称
                </label>
                <input
                  type="text"
                  value={uploadForm.name}
                  onChange={(e) => setUploadForm({ ...uploadForm, name: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="留空则使用文件名"
                />
              </div>

              {/* 符号名 */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  导出符号名
                </label>
                <input
                  type="text"
                  value={uploadForm.symbol}
                  onChange={(e) => setUploadForm({ ...uploadForm, symbol: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="例如: NewMyPlugin"
                />
              </div>

              {/* 配置 */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  配置 (JSON)
                </label>
                <textarea
                  value={uploadForm.config}
                  onChange={(e) => setUploadForm({ ...uploadForm, config: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm"
                  rows={3}
                  placeholder='{"key": "value"}'
                />
              </div>

              {/* 按钮 */}
              <div className="flex gap-2">
                <button
                  type="submit"
                  disabled={isSubmitting}
                  className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed transition-colors"
                >
                  {isSubmitting ? '上传中…' : '上传'}
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setShowUploadDialog(false);
                    setUploadForm({ file: null, name: '', symbol: '', config: '' });
                  }}
                  className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                >
                  取消
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* 配置对话框 */}
      {showConfigDialog && selectedPlugin && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">配置插件: {selectedPlugin.name}</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  配置 (JSON)
                </label>
                <textarea
                  value={JSON.stringify(selectedPlugin.config, null, 2)}
                  onChange={(e) => {
                    try {
                      const config = JSON.parse(e.target.value);
                      setSelectedPlugin({ ...selectedPlugin, config });
                    } catch {
                    }
                  }}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm"
                  rows={10}
                />
              </div>

              <div className="flex gap-2">
                <button
                  onClick={handleSaveConfig}
                  disabled={isSubmitting}
                  className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed transition-colors"
                >
                  {isSubmitting ? '保存中…' : '保存'}
                </button>
                <button
                  onClick={() => {
                    setShowConfigDialog(false);
                    setSelectedPlugin(null);
                  }}
                  className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                >
                  取消
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 测试对话框 */}
      {showTestDialog && selectedPlugin && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">测试插件: {selectedPlugin.name}</h2>
            <form onSubmit={handleTest} className="space-y-4">
              {/* 输入格式 */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  输入格式
                </label>
                <select
                  value={testForm.input_format}
                  onChange={(e) => setTestForm({ ...testForm, input_format: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="deepseek">DeepSeek</option>
                  <option value="openai-json">OpenAI JSON</option>
                  <option value="openclaw">OpenClaw</option>
                </select>
              </div>

              {/* 输出格式 */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  输出格式
                </label>
                <select
                  value={testForm.output_format}
                  onChange={(e) => setTestForm({ ...testForm, output_format: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="openclaw">OpenClaw</option>
                  <option value="json">JSON</option>
                </select>
              </div>

              {/* 测试数据 */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  测试数据
                </label>
                <textarea
                  value={testForm.test_data}
                  onChange={(e) => setTestForm({ ...testForm, test_data: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm"
                  rows={5}
                  placeholder='{"content": "test"}'
                  required
                />
              </div>

              {/* 测试结果 */}
              {testResult && (
                <div className={`p-3 rounded ${testResult.success ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}`}>
                  <div className="font-medium mb-1">
                    {testResult.success ? '✓ 测试成功' : '✗ 测试失败'}
                  </div>
                  {testResult.output && (
                    <pre className="text-xs mt-2 overflow-x-auto">{testResult.output}</pre>
                  )}
                  {testResult.error && (
                    <div className="text-sm mt-1">{testResult.error}</div>
                  )}
                </div>
              )}

              {/* 按钮 */}
              <div className="flex gap-2">
                <button
                  type="submit"
                  disabled={isSubmitting}
                  className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed transition-colors"
                >
                  {isSubmitting ? '测试中…' : '测试'}
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setShowTestDialog(false);
                    setSelectedPlugin(null);
                    setTestResult(null);
                    setTestForm({ input_format: 'deepseek', output_format: 'openclaw', test_data: '' });
                  }}
                  className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                >
                  关闭
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
