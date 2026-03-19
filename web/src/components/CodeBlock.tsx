import { useState } from 'react';

interface CodeBlockProps {
  code: string;
  language?: string;
  title?: string;
  showCopy?: boolean;
}

export default function CodeBlock({ code, language = 'bash', title, showCopy = true }: CodeBlockProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  const highlightCode = (code: string) => {
    const lines = code.split('\n');
    return lines.map((line, index) => {
      let className = 'text-[#94C6F0]'; // 默认浅蓝

      // 注释行
      if (line.trim().startsWith('#')) {
        className = 'text-[#4B6A8A]';
      }
      // CLI 参数行（包含 --）
      else if (line.includes('--')) {
        // 高亮参数名
        const parts = line.split(/( --[\w-]+)/g);
        return (
          <span key={index}>
            {parts.map((part, i) => {
              if (part.startsWith('--')) {
                return <span key={i} className="text-[#7DD3FC]">{part}</span>;
              }
              return <span key={i} className="text-[#94C6F0]">{part}</span>;
            })}
            {'\n'}
          </span>
        );
      }
      // 环境变量行
      else if (/^[A-Z_]+=/.test(line.trim())) {
        const parts = line.split(/([A-Z_]+)=/);
        return (
          <span key={index}>
            {parts.map((part, i) => {
              if (i === 1) {
                // 变量名
                return <span key={i} className="text-[#FBBF24]">{part}=</span>;
              }
              return <span key={i} className="text-[#94C6F0]">{part}</span>;
            })}
            {'\n'}
          </span>
        );
      }

      return <span className={className}>{line}{'\n'}</span>;
    });
  };

  return (
    <div className="relative my-4">
      {/* 标题栏 */}
      {(title || showCopy) && (
        <div className="flex items-center justify-between px-4 py-2 bg-[#0B1623] border-b border-gray-700 rounded-t-lg">
          {title && (
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-[#22D3EE]"></span>
              <span className="text-sm text-gray-400">{title}</span>
            </div>
          )}
          {showCopy && (
            <button
              onClick={handleCopy}
              aria-label={copied ? '已复制' : '复制'}
              className="px-3 py-1 text-xs bg-gray-700 hover:bg-gray-600 text-gray-300 rounded transition-colors"
            >
              {copied ? '✓ 已复制' : '复制'}
            </button>
          )}
        </div>
      )}

      {/* 代码内容 */}
      <pre
        className={`p-4 bg-[#0B1623] text-sm font-mono overflow-x-auto ${
          title ? 'rounded-t-none' : 'rounded-t-lg'
        } rounded-b-lg`}
      >
        <code>{highlightCode(code)}</code>
      </pre>
    </div>
  );
}
