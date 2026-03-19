import { useState, useMemo } from 'react';
import { RUNTIMES, getRuntimeById, type RuntimeConfig, type ModelConfig } from '../data/deployRuntimes';
import CodeBlock from '../components/CodeBlock';
import { ChevronRight, Check, Info } from 'lucide-react';

type TabType = 'steps' | 'params' | 'gateway';

export default function DeployGuidePage() {
  const [selectedRuntimeId, setSelectedRuntimeId] = useState<string>('vllm');
  const [selectedModelIndex, setSelectedModelIndex] = useState<number>(0);
  const [activeTab, setActiveTab] = useState<TabType>('steps');

  const selectedRuntime = useMemo(() => getRuntimeById(selectedRuntimeId), [selectedRuntimeId]);
  const selectedModel = useMemo(
    () => selectedRuntime?.models[selectedModelIndex],
    [selectedRuntime, selectedModelIndex]
  );

  // 切换框架时重置选择
  const handleRuntimeChange = (runtimeId: string) => {
    setSelectedRuntimeId(runtimeId);
    setSelectedModelIndex(0);
    setActiveTab('steps');
  };

  // 切换模型时重置 Tab
  const handleModelChange = (index: number) => {
    setSelectedModelIndex(index);
    setActiveTab('steps');
  };

  if (!selectedRuntime) return null;

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      {/* 面包屑导航 */}
      <nav className="mb-6 text-sm text-gray-600">
        <span className="hover:text-gray-900 cursor-pointer">管理控制台</span>
        <ChevronRight className="inline w-4 h-4 mx-1" />
        <span className="hover:text-gray-900 cursor-pointer">模型配置</span>
        <ChevronRight className="inline w-4 h-4 mx-1" />
        <span className="text-gray-900 font-medium">部署指南</span>
      </nav>

      {/* 页头 */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900">本地模型部署指南</h1>
        <p className="text-gray-600 mt-2">
          按照推理框架完成本地模型部署，然后接入 api2openclaw 网关
        </p>
      </div>

      {/* 框架选择卡片组 */}
      <div className="grid grid-cols-4 gap-4 mb-8">
        {RUNTIMES.map((runtime) => (
          <button
            key={runtime.id}
            onClick={() => handleRuntimeChange(runtime.id)}
            className={`
              relative p-4 rounded-lg border-2 transition-all
              ${
                selectedRuntimeId === runtime.id
                  ? `${runtime.accentLight} border-[${runtime.accentColor}] shadow-lg shadow-${runtime.accentColor}/20`
                  : 'bg-white border-gray-200 hover:border-gray-300 hover:shadow-md'
              }
            `}
            style={
              selectedRuntimeId === runtime.id
                ? { backgroundColor: runtime.accentLight, borderColor: runtime.accentColor }
                : undefined
            }
          >
            <div className="text-3xl mb-2">{runtime.icon}</div>
            <div className="font-semibold text-gray-900">{runtime.name}</div>
            <div
              className={`text-xs mt-1 px-2 py-0.5 rounded-full inline-block ${
                selectedRuntimeId === runtime.id ? 'bg-white' : 'bg-gray-100'
              }`}
            >
              {runtime.badge}
            </div>
            <div className="text-xs text-gray-500 mt-2">
              端口: {runtime.apiPort}
            </div>
          </button>
        ))}
      </div>

      {/* 主体内容：左侧边栏 + 右侧详情 */}
      <div className="flex gap-6">
        {/* 左侧边栏 */}
        <aside className="w-72 flex-shrink-0">
          {/* 框架信息卡片 */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-5 mb-4">
            <div className="flex items-center gap-2 mb-3">
              <span className="text-2xl">{selectedRuntime.icon}</span>
              <h3 className="text-xl font-bold" style={{ color: selectedRuntime.accentColor }}>
                {selectedRuntime.name}
              </h3>
            </div>
            <p className="text-sm text-gray-600 mb-3">{selectedRuntime.tagline}</p>

            {/* 能力标签 */}
            <div className="flex flex-wrap gap-1.5 mb-4">
              {selectedRuntime.features.map((feature) => (
                <span
                  key={feature}
                  className="text-xs px-2 py-1 rounded"
                  style={{ backgroundColor: selectedRuntime.accentLight, color: selectedRuntime.accentColor }}
                >
                  {feature}
                </span>
              ))}
            </div>

            {/* 环境要求 */}
            <div className="text-xs text-gray-500 space-y-1">
              <div>OS: {selectedRuntime.requirements.os}</div>
              {selectedRuntime.requirements.python && (
                <div>Python: {selectedRuntime.requirements.python}</div>
              )}
              {selectedRuntime.requirements.cuda && (
                <div>CUDA: {selectedRuntime.requirements.cuda}</div>
              )}
              <div>GPU: {selectedRuntime.requirements.gpu}</div>
            </div>
          </div>

          {/* 模型选择列表 */}
          <div className="space-y-2">
            <h4 className="text-sm font-semibold text-gray-700 mb-2">选择模型</h4>
            {selectedRuntime.models.map((model, index) => (
              <button
                key={model.id}
                onClick={() => handleModelChange(index)}
                className={`w-full text-left p-3 rounded-lg border-2 transition-all ${
                  selectedModelIndex === index
                    ? selectedRuntime.accentLight + ' border-2'
                    : 'bg-white border-gray-200 hover:border-gray-300'
                }`}
                style={
                  selectedModelIndex === index ? { borderColor: selectedRuntime.accentColor } : undefined
                }
              >
                <div className="font-medium text-gray-900">{model.name}</div>
                <div className="text-xs text-gray-500 mt-1 flex gap-2">
                  <span>显存: {model.vram}</span>
                  <span>精度: {model.precision}</span>
                </div>
              </button>
            ))}
          </div>
        </aside>

        {/* 右侧详情面板 */}
        <main className="flex-1 bg-white rounded-lg shadow-sm border border-gray-200">
          {/* Tab 导航 */}
          <div className="border-b border-gray-200 px-6 py-4">
            <div className="flex items-center gap-4">
              <div className="flex gap-1">
                <TabButton
                  active={activeTab === 'steps'}
                  onClick={() => setActiveTab('steps')}
                  icon="📋"
                  label="部署步骤"
                />
                <TabButton
                  active={activeTab === 'params'}
                  onClick={() => setActiveTab('params')}
                  icon="⚙️"
                  label="启动参数"
                />
                <TabButton
                  active={activeTab === 'gateway'}
                  onClick={() => setActiveTab('gateway')}
                  icon="🔗"
                  label="接入网关"
                />
              </div>
              <div className="ml-auto text-sm text-gray-500">
                {selectedModel?.name} · {selectedRuntime.name}:{selectedRuntime.apiPort}
              </div>
            </div>
          </div>

          {/* Tab 内容 */}
          <div className="p-6">
            {activeTab === 'steps' && selectedModel && (
              <StepsTab runtime={selectedRuntime} model={selectedModel} />
            )}
            {activeTab === 'params' && selectedRuntime.params && (
              <ParamsTab params={selectedRuntime.params} accentColor={selectedRuntime.accentColor} />
            )}
            {activeTab === 'gateway' && selectedModel && (
              <GatewayTab runtime={selectedRuntime} model={selectedModel} />
            )}
          </div>
        </main>
      </div>

      {/* 底部框架对比条 */}
      <div className="mt-8 pt-6 border-t border-gray-200">
        <div className="grid grid-cols-4 gap-4">
          {RUNTIMES.map((runtime) => (
            <button
              key={runtime.id}
              onClick={() => handleRuntimeChange(runtime.id)}
              className={`text-center p-4 rounded-lg transition-all ${
                selectedRuntimeId === runtime.id
                  ? 'bg-blue-50 border-2 border-blue-500'
                  : 'bg-gray-50 hover:bg-gray-100'
              }`}
            >
              <div className="text-2xl mb-1">{runtime.icon}</div>
              <div className="font-semibold text-sm text-gray-900">{runtime.badge}</div>
              <div className="text-xs text-gray-500 mt-1">
                {runtime.id === 'vllm' && '高并发 · PagedAttention'}
                {runtime.id === 'sglang' && 'KV 缓存复用 · 低延迟'}
                {runtime.id === 'xinference' && 'Web UI · 多后端'}
                {runtime.id === 'ollama' && '零配置 · 最易上手'}
              </div>
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}

// Tab 按钮组件
function TabButton({
  active,
  onClick,
  icon,
  label,
}: {
  active: boolean;
  onClick: () => void;
  icon: string;
  label: string;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-2 px-4 py-2 rounded-lg transition-colors ${
        active ? 'bg-blue-100 text-blue-700' : 'text-gray-600 hover:bg-gray-100'
      }`}
    >
      <span>{icon}</span>
      <span className="font-medium">{label}</span>
    </button>
  );
}

// 部署步骤 Tab
function StepsTab({ runtime, model }: { runtime: RuntimeConfig; model: ModelConfig }) {
  const quickVerifyCmd = `curl http://localhost:${runtime.apiPort}/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${model.hfId}",
    "messages": [{"role": "user", "content": "你好"}],
    "stream": false
  }`;

  return (
    <div>
      {/* 第零步：安装运行时 */}
      <div className="mb-6">
        <div className="flex items-center gap-2 mb-3">
          <span className="flex items-center justify-center w-6 h-6 rounded-full bg-blue-500 text-white text-sm font-bold">
            0
          </span>
          <h3 className="text-lg font-semibold text-gray-900">安装运行时</h3>
        </div>
        <CodeBlock code={runtime.install.code} />
        {runtime.install.note && (
          <div className="flex items-start gap-2 mt-2 text-sm text-amber-700">
            <Info className="w-4 h-4 mt-0.5 flex-shrink-0" />
            <span>{runtime.install.note}</span>
          </div>
        )}
      </div>

      {/* 部署步骤 */}
      {model.steps.map((step, index) => (
        <div key={index} className="mb-6">
          <div className="flex items-center gap-2 mb-3">
            <span
              className="flex items-center justify-center w-6 h-6 rounded-full text-white text-sm font-bold"
              style={{ backgroundColor: runtime.accentColor }}
            >
              {index + 1}
            </span>
            <h3 className="text-lg font-semibold text-gray-900">{step.title}</h3>
          </div>
          <CodeBlock code={step.code} />
          {step.note && (
            <div className="flex items-start gap-2 mt-2 text-sm text-amber-700">
              <Info className="w-4 h-4 mt-0.5 flex-shrink-0" />
              <span>{step.note}</span>
            </div>
          )}
        </div>
      ))}

      {/* 快速验证 */}
      <div className="mt-8 pt-6 border-t border-gray-200">
        <h4 className="font-semibold text-gray-900 mb-3">快速验证</h4>
        <CodeBlock code={quickVerifyCmd} />
      </div>
    </div>
  );
}

// 启动参数 Tab
function ParamsTab({ params, accentColor }: { params: Array<{name: string; default: string; desc: string}>; accentColor: string }) {
  return (
    <div>
      <h3 className="text-lg font-semibold text-gray-900 mb-4">启动参数说明</h3>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr style={{ backgroundColor: accentColor + '20' }}>
              <th className="px-4 py-3 text-left font-semibold text-gray-900">参数名</th>
              <th className="px-4 py-3 text-left font-semibold text-gray-900">默认值</th>
              <th className="px-4 py-3 text-left font-semibold text-gray-900">说明</th>
            </tr>
          </thead>
          <tbody>
            {params.map((param, index) => (
              <tr
                key={param.name}
                className={index % 2 === 0 ? 'bg-[#FAFBFC]' : 'bg-white'}
              >
                <td className="px-4 py-3">
                  <code
                    className="text-sm px-2 py-1 rounded"
                    style={{ backgroundColor: accentColor + '20', color: accentColor }}
                  >
                    {param.name}
                  </code>
                </td>
                <td className="px-4 py-3">
                  <code className="text-sm text-gray-700">{param.default}</code>
                </td>
                <td className="px-4 py-3 text-sm text-gray-600">{param.desc}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// 接入网关 Tab
function GatewayTab({ runtime, model }: { runtime: RuntimeConfig; model: ModelConfig }) {
  const configSnippet = `# configs/config.yaml 中添加以下内容
models:
  - alias: "${model.gatewayAlias}"
    model_id: "${model.hfId}"
    base_url: "http://YOUR_HOST:${runtime.apiPort}/v1"
    capabilities:
      streaming: true
      tool_use: true
      json_mode: true`;

  const gatewayVerifyCmd = `curl http://YOUR_GATEWAY:8080/v1/chat/completions \\
  -H "Authorization: Bearer sk-a2oc-YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${model.gatewayAlias}",
    "messages": [{"role": "user", "content": "你好"}],
    "stream": false
  }'`;

  return (
    <div>
      {/* 配置片段 */}
      <div className="mb-8">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">配置片段</h3>
        <CodeBlock code={configSnippet} title="configs/config.yaml 片段" />

        {/* 注意事项 */}
        <div className="mt-4 p-4 bg-amber-50 border border-amber-200 rounded-lg">
          <div className="flex items-start gap-2">
            <span className="text-amber-600 font-semibold">⚠ 填写前确认</span>
          </div>
          <ul className="mt-2 text-sm text-amber-800 space-y-1">
            <li>• 将 YOUR_HOST 替换为运行推理服务的实际 IP 地址</li>
            <li>• 服务与网关同机时使用 127.0.0.1，跨机时使用局域网 IP</li>
            <li>• 确保防火墙已放行端口 {runtime.apiPort}</li>
            {runtime.id === 'xinference' && (
              <li>• Xinference 用户请通过 xinference list 查询 model_uid 填入 model_id 字段</li>
            )}
          </ul>
        </div>
      </div>

      {/* 网关验证命令 */}
      <div className="mb-8">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">通过网关验证</h3>
        <CodeBlock code={gatewayVerifyCmd} />
      </div>

      {/* Xinference UID 查询按钮 */}
      {runtime.id === 'xinference' && (
        <div className="mb-8">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">查询 Xinference 模型 UID</h3>
          <button
            onClick={() => {
              const cmd = `curl http://localhost:9997/v1/models`;
              navigator.clipboard.writeText(cmd);
            }}
            className="px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors"
          >
            复制查询命令
          </button>
          <p className="text-sm text-gray-500 mt-2">
            执行后获取响应中的 <code className="px-1 py-0.5 bg-gray-100 rounded">id</code> 字段填入配置
          </p>
        </div>
      )}

      {/* 能力矩阵 */}
      <div>
        <h3 className="text-lg font-semibold text-gray-900 mb-4">能力支持</h3>
        <div className="grid grid-cols-2 gap-4">
          {[
            { name: 'SSE 流式输出', supported: true },
            { name: 'Tool Use (函数调用)', supported: runtime.id !== 'ollama' || model.id.includes('tool') },
            { name: 'JSON Mode (结构化输出)', supported: true },
            { name: '多 GPU 张量并行', supported: runtime.id !== 'ollama' },
          ].map((capability) => (
            <div
              key={capability.name}
              className={`p-4 rounded-lg border-2 ${
                capability.supported
                  ? 'bg-green-50 border-green-200'
                  : 'bg-orange-50 border-orange-200'
              }`}
            >
              <div className="flex items-center gap-2">
                {capability.supported ? (
                  <Check className="w-5 h-5 text-green-600" />
                ) : (
                  <span className="text-orange-600 font-semibold">⚠</span>
                )}
                <span className="font-medium text-gray-900">{capability.name}</span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
