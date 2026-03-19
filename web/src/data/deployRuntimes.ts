// 本地模型部署指南 - 运行时配置数据
// v0.3.1

export interface RuntimeConfig {
  id: string;
  name: string;
  tagline: string;
  description: string;
  badge: string;
  badgeColor: string;
  accentColor: string;
  accentLight: string;
  icon: string;
  features: string[];
  requirements: {
    os: string;
    python?: string;
    cuda?: string;
    gpu: string;
  };
  apiPort: number;
  install: {
    code: string;
    note?: string;
  };
  models: ModelConfig[];
  params?: ParamConfig[];
}

export interface ModelConfig {
  id: string;
  name: string;
  family: string;
  vram: string;
  precision: string;
  hfId: string;
  steps: StepConfig[];
  gatewayAlias: string;
}

export interface StepConfig {
  title: string;
  code: string;
  note?: string;
}

export interface ParamConfig {
  name: string;
  default: string;
  desc: string;
}

// vLLM 配置
const vllmConfig: RuntimeConfig = {
  id: 'vllm',
  name: 'vLLM',
  tagline: '生产首选',
  description: '高吞吐推理引擎，专为多用户并发生产环境优化。PagedAttention 技术减少 50%+ 显存碎片，支持 Speculative Decoding 加速推理。',
  badge: '生产首选',
  badgeColor: 'blue',
  accentColor: '#3B82F6',
  accentLight: '#EFF6FF',
  icon: '⚡',
  features: ['PagedAttention', '连续批处理', 'Speculative Decoding'],
  requirements: {
    os: 'Linux',
    python: '>=3.8',
    cuda: '>=11.8',
    gpu: 'NVIDIA (A100/H100/RTX 40xx)',
  },
  apiPort: 8000,
  install: {
    code: 'pip install vllm',
  },
  models: [
    {
      id: 'qwen2.5-72b-instruct',
      name: 'Qwen2.5-72B-Instruct',
      family: 'Qwen',
      vram: '≥48GB',
      precision: 'BF16',
      hfId: 'Qwen/Qwen2.5-72B-Instruct',
      gatewayAlias: 'qwen-72b',
      steps: [
        {
          title: '下载模型',
          code: `# 从 HuggingFace 下载模型
huggingface-cli download Qwen/Qwen2.5-72B-Instruct \
  --local-dir /path/to/models/Qwen2.5-72B-Instruct`,
          note: '首次下载需要较长时间，请确保网络稳定。国内用户可配置 HF_ENDPOINT=https://hf-mirror.com',
        },
        {
          title: '启动推理服务',
          code: `vllm serve Qwen/Qwen2.5-72B-Instruct \\
  --tensor-parallel-size 4 \\
  --host 0.0.0.0 \\
  --port 8000 \\
  --dtype auto`,
          note: 'tensor-parallel-size 需根据 GPU 数量调整，4 卡显存总和需 ≥ 48GB',
        },
      ],
    },
    {
      id: 'deepseek-v3',
      name: 'DeepSeek-V3',
      family: 'DeepSeek',
      vram: '≥8×80GB',
      precision: 'BF16',
      hfId: 'deepseek-ai/DeepSeek-V3',
      gatewayAlias: 'deepseek-v3',
      steps: [
        {
          title: '下载模型',
          code: `huggingface-cli download deepseek-ai/DeepSeek-V3 \\
  --local-dir /path/to/models/DeepSeek-V3`,
          note: 'DeepSeek-V3 为 MoE 架构超大模型，需要 8 张 H100 (80GB) 或等效算力',
        },
        {
          title: '启动推理服务',
          code: `vllm serve deepseek-ai/DeepSeek-V3 \\
  --tensor-parallel-size 8 \\
  --host 0.0.0.0 \\
  --port 8000 \\
  --max-model-len 8192 \\
  --dtype auto`,
        },
      ],
    },
    {
      id: 'llama-3.3-70b-instruct',
      name: 'Llama-3.3-70B-Instruct',
      family: 'Llama',
      vram: '≥40GB',
      precision: 'BF16',
      hfId: 'meta-llama/Llama-3.3-70B-Instruct',
      gatewayAlias: 'llama-70b',
      steps: [
        {
          title: '下载模型',
          code: `huggingface-cli download meta-llama/Llama-3.3-70B-Instruct \\
  --local-dir /path/to/models/Llama-3.3-70B-Instruct`,
          note: 'Llama 模型需访问 HuggingFace 并同意使用条款',
        },
        {
          title: '启动推理服务',
          code: `vllm serve meta-llama/Llama-3.3-70B-Instruct \\
  --tensor-parallel-size 4 \\
  --host 0.0.0.0 \\
  --port 8000 \\
  --dtype auto`,
        },
      ],
    },
  ],
  params: [
    { name: '--host', default: '0.0.0.0', desc: '监听地址，0.0.0.0 表示所有网卡' },
    { name: '--port', default: '8000', desc: '监听端口' },
    { name: '--tensor-parallel-size', default: '1', desc: '张量并行度，多 GPU 推理时使用' },
    { name: '--dtype', default: 'auto', desc: '数据类型：auto/bfloat16/float16' },
    { name: '--max-model-len', default: '4096', desc: '最大上下文长度' },
    { name: '--gpu-memory-utilization', default: '0.9', desc: 'GPU 显存利用率 (0-1)' },
    { name: '--trust-remote-code', default: 'false', desc: '是否信任远程代码' },
    { name: '--enable-prefix-caching', default: 'false', desc: '启用前缀缓存优化' },
    { name: '--quantization', default: '-', desc: '量化方式：awq/gptq/squeezellm' },
    { name: '--served-model-name', default: 'model_name', desc: '对外暴露的模型名称' },
  ],
};

// SGLang 配置
const sglangConfig: RuntimeConfig = {
  id: 'sglang',
  name: 'SGLang',
  tagline: 'Agent 场景',
  description: '专为多轮对话、RAG、Agent 推理优化。RadixAttention 实现自动 prefix cache 复用，大幅降低重复计算，首 token 延迟极低。',
  badge: 'Agent场景',
  badgeColor: 'purple',
  accentColor: '#8B5CF6',
  accentLight: '#F5F3FF',
  icon: '🧠',
  features: ['RadixAttention', 'Prefix Cache', 'Low Latency'],
  requirements: {
    os: 'Linux',
    python: '>=3.10',
    cuda: '>=11.8',
    gpu: 'NVIDIA (A100/H100)',
  },
  apiPort: 30000,
  install: {
    code: 'pip install --upgrade pip && pip install uv && uv pip install "sglang[all]"',
    note: '使用 uv 可大幅加速依赖安装',
  },
  models: [
    {
      id: 'qwen3-32b',
      name: 'Qwen3-32B',
      family: 'Qwen',
      vram: '≥24GB',
      precision: 'BF16',
      hfId: 'Qwen/Qwen2.5-32B-Instruct',
      gatewayAlias: 'qwen-32b',
      steps: [
        {
          title: '下载模型',
          code: `huggingface-cli download Qwen/Qwen2.5-32B-Instruct \\
  --local-dir /path/to/models/Qwen2.5-32B-Instruct`,
        },
        {
          title: '启动推理服务',
          code: `python -m sglang.launch_server \\
  --model-path Qwen/Qwen2.5-32B-Instruct \\
  --host 0.0.0.0 \\
  --port 30000`,
        },
      ],
    },
    {
      id: 'deepseek-v3-fp8',
      name: 'DeepSeek-V3 (FP8)',
      family: 'DeepSeek',
      vram: '≥8×80GB',
      precision: 'FP8',
      hfId: 'deepseek-ai/DeepSeek-V3',
      gatewayAlias: 'deepseek-fp8',
      steps: [
        {
          title: '启动推理服务（FP8 量化）',
          code: `python -m sglang.launch_server \\
  --model-path deepseek-ai/DeepSeek-V3 \\
  --host 0.0.0.0 \\
  --port 30000 \\
  --tp 8 \\
  --quantization fp8`,
          note: 'FP8 量化可降低显存需求，轻微损失精度',
        },
      ],
    },
    {
      id: 'llama-3.1-8b-instruct',
      name: 'Llama-3.1-8B-Instruct',
      family: 'Llama',
      vram: '≥16GB',
      precision: 'BF16',
      hfId: 'meta-llama/Llama-3.1-8B-Instruct',
      gatewayAlias: 'llama-8b',
      steps: [
        {
          title: '启动推理服务',
          code: `python -m sglang.launch_server \\
  --model-path meta-llama/Llama-3.1-8B-Instruct \\
  --host 0.0.0.0 \\
  --port 30000`,
        },
      ],
    },
  ],
  params: [
    { name: '--model-path', default: 'required', desc: '模型路径（必需）' },
    { name: '--host', default: '0.0.0.0', desc: '监听地址' },
    { name: '--port', default: '30000', desc: '监听端口' },
    { name: '--tp', default: '1', desc: '张量并行度' },
    { name: '--context-length', default: '4096', desc: '上下文长度' },
    { name: '--quantization', default: '-', desc: '量化：fp8/int8' },
    { name: '--dtype', default: 'auto', desc: '数据类型' },
    { name: '--mem-fraction-static', default: '0.9', desc: '静态显存分配比例' },
    { name: '--chunked-prefill-size', default: '4096', desc: '预填充分块大小' },
    { name: '--radix-cache-config', default: '-', desc: 'RadixCache 配置' },
  ],
};

// Xinference 配置
const xinferenceConfig: RuntimeConfig = {
  id: 'xinference',
  name: 'Xinference',
  tagline: '多模型管理',
  description: '统一管理多个模型和推理框架。支持 vLLM/SGLang/llama.cpp/Transformers/MLX 多后端，自带 Web UI 管理界面。',
  badge: '多模型管理',
  badgeColor: 'green',
  accentColor: '#10B981',
  accentLight: '#ECFDF5',
  icon: '🎛',
  features: ['Web UI', 'Multi Backend', 'RESTful API'],
  requirements: {
    os: 'Linux/Mac/Windows',
    python: '>=3.9',
    gpu: 'NVIDIA (可选)',
  },
  apiPort: 9997,
  install: {
    code: 'pip install "xinference[all]"',
    note: '也可指定后端：pip install "xinference[vllm]"',
  },
  models: [
    {
      id: 'qwen2.5-instruct-vllm',
      name: 'Qwen2.5-Instruct (vLLM)',
      family: 'Qwen',
      vram: '≥8GB (7B)',
      precision: 'BF16/AWQ',
      hfId: 'Qwen/Qwen2.5-7B-Instruct',
      gatewayAlias: 'qwen-vllm',
      steps: [
        {
          title: '启动 Xinference 服务',
          code: `xinference-local --host 0.0.0.0 --port 9997`,
          note: '首次启动会自动下载默认模型',
        },
        {
          title: '启动模型（vLLM 后端）',
          code: `xinference launch --model-name qwen2.5-instruct \\
  --model-format pytorch \\
  --size-in-billions 7 \\
  --backend vllm`,
        },
        {
          title: '查询 model_uid',
          code: `curl http://localhost:9997/v1/models`,
          note: '获取响应中的 id 字段作为 model_id 填入网关配置',
        },
      ],
    },
    {
      id: 'deepseek-r1-14b-sglang',
      name: 'DeepSeek-R1-Distill-14B (SGLang)',
      family: 'DeepSeek',
      vram: '≥16GB',
      precision: 'BF16',
      hfId: 'deepseek-ai/DeepSeek-R1-Distill-Qwen-14B',
      gatewayAlias: 'deepseek-sglang',
      steps: [
        {
          title: '启动模型（SGLang 后端）',
          code: `xinference launch --model-name deepseek-r1 \\
  --model-format pytorch \\
  --size-in-billions 14 \\
  --backend sglang`,
        },
        {
          title: '查询 model_uid',
          code: `curl http://localhost:9997/v1/models`,
        },
      ],
    },
    {
      id: 'llama-3.1-8b-gguf',
      name: 'Llama-3.1-8B (GGUF)',
      family: 'Llama',
      vram: '≥6GB (Q4)',
      precision: 'GGUF Q4_K_M',
      hfId: 'meta-llama/Llama-3.1-8B-Instruct-GGUF',
      gatewayAlias: 'llama-gguf',
      steps: [
        {
          title: '启动模型（llama.cpp 后端）',
          code: `xinference launch --model-name llama-3.1-8b-instruct \\
  --model-format gguf \\
  --backend llama.cpp`,
        },
        {
          title: '查询 model_uid',
          code: `curl http://localhost:9997/v1/models`,
        },
      ],
    },
  ],
  params: [
    { name: '--host', default: '0.0.0.0', desc: '监听地址' },
    { name: '--port', default: '9997', desc: 'Web UI 端口' },
    { name: '--model-name', default: 'required', desc: '模型名称（必需）' },
    { name: '--model-format', default: 'pytorch', desc: '模型格式：pytorch/gguf/safetensors' },
    { name: '--size-in-billions', default: '-', desc: '模型参数量：7/14/32/70' },
    { name: '--backend', default: 'auto', desc: '推理后端：vllm/sglang/llama.cpp/transformers' },
    { name: '--gpu-index', default: '0', desc: '指定 GPU 索引' },
    { name: 'XINFERENCE_HOME', default: '~/.xinference', desc: '环境变量：模型存储目录' },
  ],
};

// Ollama 配置
const ollamaConfig: RuntimeConfig = {
  id: 'ollama',
  name: 'Ollama',
  tagline: '最易上手',
  description: '零配置、一行命令安装运行。支持 Mac/Linux/Windows，个人开发和快速原型验证首选。内置模型库，自动管理模型下载和版本。',
  badge: '最易上手',
  badgeColor: 'orange',
  accentColor: '#F97316',
  accentLight: '#FFF7ED',
  icon: '🦙',
  features: ['Zero Config', 'Cross Platform', 'RESTful API'],
  requirements: {
    os: 'Linux/Mac/Windows',
    gpu: 'NVIDIA/Apple Silicon/AMD',
  },
  apiPort: 11434,
  install: {
    code: 'curl -fsSL https://ollama.com/install.sh | sh',
    note: 'Windows 用户访问 https://ollama.com/download 下载安装包',
  },
  models: [
    {
      id: 'qwen2.5-72b',
      name: 'qwen2.5:72b',
      family: 'Qwen',
      vram: '≥48GB',
      precision: 'Q4_K_M',
      hfId: '',
      gatewayAlias: 'qwen-72b',
      steps: [
        {
          title: '拉取并运行模型',
          code: `ollama run qwen2.5:72b`,
          note: '首次运行会自动下载模型，约 43GB',
        },
      ],
    },
    {
      id: 'deepseek-r1-14b',
      name: 'deepseek-r1:14b',
      family: 'DeepSeek',
      vram: '≥10GB',
      precision: 'Q4_K_M',
      hfId: '',
      gatewayAlias: 'deepseek-14b',
      steps: [
        {
          title: '拉取并运行模型',
          code: `ollama run deepseek-r1:14b`,
        },
      ],
    },
    {
      id: 'llama-3.3-70b',
      name: 'llama3.3:70b',
      family: 'Llama',
      vram: '≥40GB',
      precision: 'Q4_K_M',
      hfId: '',
      gatewayAlias: 'llama-70b',
      steps: [
        {
          title: '拉取并运行模型',
          code: `ollama run llama3.3:70b`,
        },
      ],
    },
  ],
  params: [
    { name: 'OLLAMA_HOST', default: '127.0.0.1:11434', desc: '服务监听地址' },
    { name: 'OLLAMA_MODELS', default: '~/.ollama/models', desc: '模型存储目录' },
    { name: 'OLLAMA_NUM_PARALLEL', default: '1', desc: '并行请求数' },
    { name: 'OLLAMA_MAX_QUEUE', default: '512', desc: '请求队列长度' },
    { name: 'OLLAMA_LOAD_TIMEOUT', default: '5m', desc: '模型加载超时' },
    { name: 'OLLAMA_KEEP_ALIVE', default: '5m', desc: '模型存活时间' },
    { name: 'OLLAMA_DEBUG', default: '0', desc: '调试模式：1 启用' },
    { name: '--gpu', default: 'auto', desc: '指定 GPU（运行时参数）' },
    { name: '--num-gpu', default: '99', desc: '使用 GPU 层数（运行时参数）' },
    { name: '--context-length', default: '2048', desc: '上下文长度（运行时参数）' },
  ],
};

// 运行时配置导出
export const RUNTIMES: RuntimeConfig[] = [
  vllmConfig,
  sglangConfig,
  xinferenceConfig,
  ollamaConfig,
];

// 框架能力矩阵
export const CAPABILITY_MATRIX = {
  vllm: {
    streaming: true,
    toolUse: true,
    jsonMode: true,
    multiGPU: true,
  },
  sglang: {
    streaming: true,
    toolUse: true,
    jsonMode: true,
    multiGPU: true,
  },
  xinference: {
    streaming: true,
    toolUse: true, // 取决于后端
    jsonMode: true,
    multiGPU: true, // 取决于后端
  },
  ollama: {
    streaming: true,
    toolUse: true, // 部分模型
    jsonMode: true,
    multiGPU: false,
  },
};

// 根据 ID 获取运行时配置
export function getRuntimeById(id: string): RuntimeConfig | undefined {
  return RUNTIMES.find(r => r.id === id);
}

// 获取默认运行时（vLLM）
export function getDefaultRuntime(): RuntimeConfig {
  return vllmConfig;
}
