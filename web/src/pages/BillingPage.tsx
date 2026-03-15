import { useState, useEffect } from 'react';
import { billingAPI, keysAPI } from '../services/api';
import type {
  BillingRule,
  Invoice,
  InvoiceDetail,
  CreateBillingRuleRequest,
  UsageStats,
  APIKey,
} from '../services/types';

type Tab = 'rules' | 'invoices' | 'usage';

export default function BillingPage() {
  const [activeTab, setActiveTab] = useState<Tab>('rules');
  const [rules, setRules] = useState<BillingRule[]>([]);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [invoicesTotal, setInvoicesTotal] = useState(0);
  const [invoicesPage, setInvoicesPage] = useState(1);
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(false);

  // 规则表单状态
  const [showRuleForm, setShowRuleForm] = useState(false);
  const [editingRule, setEditingRule] = useState<BillingRule | null>(null);
  const [ruleForm, setRuleForm] = useState<CreateBillingRuleRequest>({
    name: '',
    rule_type: 'token_based',
    unit_price: 0.0001,
    currency: 'CNY',
    free_quota: 0,
    is_active: true,
    valid_from: new Date().toISOString().split('T')[0],
  });

  // 生成账单状态
  const [showGenerateDialog, setShowGenerateDialog] = useState(false);
  const [generateForm, setGenerateForm] = useState({
    key_id: '',
    start_date: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
    end_date: new Date().toISOString().split('T')[0],
  });

  // 用量统计状态
  const [usageStats, setUsageStats] = useState<UsageStats | null>(null);
  const [usageKeyId, setUsageKeyId] = useState('');

  // 账单详情
  const [selectedInvoice, setSelectedInvoice] = useState<InvoiceDetail | null>(null);
  const [showInvoiceDetail, setShowInvoiceDetail] = useState(false);

  useEffect(() => {
    loadRules();
    loadInvoices();
    loadApiKeys();
  }, [invoicesPage]);

  const loadRules = async () => {
    try {
      const res = await billingAPI.listRules();
      setRules(res.data.rules || []);
    } catch (error) {
      console.error('Failed to load rules:', error);
    }
  };

  const loadInvoices = async () => {
    setLoading(true);
    try {
      const res = await billingAPI.listInvoices({ page: invoicesPage, limit: 20 });
      setInvoices(res.data.invoices || []);
      setInvoicesTotal(res.data.total || 0);
    } catch (error) {
      console.error('Failed to load invoices:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadApiKeys = async () => {
    try {
      const res = await keysAPI.list();
      setApiKeys(res.data.data || []);
    } catch (error) {
      console.error('Failed to load API keys:', error);
    }
  };

  const loadUsageStats = async () => {
    if (!usageKeyId) return;
    try {
      const res = await billingAPI.getUsage({ key_id: usageKeyId });
      setUsageStats(res.data);
    } catch (error) {
      console.error('Failed to load usage stats:', error);
    }
  };

  const handleSaveRule = async () => {
    try {
      if (editingRule) {
        await billingAPI.updateRule(editingRule.id, ruleForm);
      } else {
        await billingAPI.createRule(ruleForm);
      }
      setShowRuleForm(false);
      setEditingRule(null);
      setRuleForm({
        name: '',
        rule_type: 'token_based',
        unit_price: 0.0001,
        currency: 'CNY',
        free_quota: 0,
        is_active: true,
        valid_from: new Date().toISOString().split('T')[0],
      });
      loadRules();
    } catch (error) {
      console.error('Failed to save rule:', error);
      alert('保存规则失败');
    }
  };

  const handleDeleteRule = async (id: number) => {
    if (!confirm('确定要删除这条规则吗？')) return;
    try {
      await billingAPI.deleteRule(id);
      loadRules();
    } catch (error) {
      console.error('Failed to delete rule:', error);
      alert('删除规则失败');
    }
  };

  const handleEditRule = (rule: BillingRule) => {
    setEditingRule(rule);
    setRuleForm({
      name: rule.name,
      description: rule.description,
      rule_type: rule.rule_type,
      model_alias: rule.model_alias,
      key_id: rule.key_id,
      unit_price: rule.unit_price,
      currency: rule.currency,
      free_quota: rule.free_quota,
      tier_threshold: rule.tier_threshold,
      tier_price: rule.tier_price,
      is_active: rule.is_active,
      valid_from: rule.valid_from.split('T')[0],
      valid_until: rule.valid_until?.split('T')[0],
    });
    setShowRuleForm(true);
  };

  const handleGenerateInvoice = async () => {
    try {
      await billingAPI.generateInvoice(generateForm);
      setShowGenerateDialog(false);
      loadInvoices();
      alert('账单生成成功');
    } catch (error) {
      console.error('Failed to generate invoice:', error);
      alert('生成账单失败');
    }
  };

  const handleViewInvoice = async (invoice: Invoice) => {
    try {
      const res = await billingAPI.getInvoice(invoice.id);
      setSelectedInvoice(res.data);
      setShowInvoiceDetail(true);
    } catch (error) {
      console.error('Failed to load invoice detail:', error);
    }
  };

  const handleExportInvoice = async (id: number, invoiceNumber: string) => {
    try {
      const res = await billingAPI.exportInvoice(id);
      const url = window.URL.createObjectURL(new Blob([res.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', `${invoiceNumber}.csv`);
      document.body.appendChild(link);
      link.click();
      link.remove();
    } catch (error) {
      console.error('Failed to export invoice:', error);
    }
  };

  const handleUpdateInvoiceStatus = async (id: number, status: string) => {
    try {
      await billingAPI.updateInvoiceStatus(id, status as any);
      loadInvoices();
      if (selectedInvoice?.invoice.id === id) {
        setSelectedInvoice({
          ...selectedInvoice,
          invoice: { ...selectedInvoice.invoice, status: status as any },
        });
      }
    } catch (error) {
      console.error('Failed to update invoice status:', error);
      alert('更新状态失败');
    }
  };

  const getStatusBadge = (status: string) => {
    const styles: Record<string, string> = {
      pending: 'bg-yellow-100 text-yellow-800',
      paid: 'bg-green-100 text-green-800',
      overdue: 'bg-red-100 text-red-800',
      cancelled: 'bg-gray-100 text-gray-800',
      active: 'bg-green-100 text-green-800',
      inactive: 'bg-gray-100 text-gray-800',
    };
    return styles[status] || 'bg-gray-100 text-gray-800';
  };

  const getRuleTypeBadge = (type: string) => {
    const styles: Record<string, { text: string; class: string }> = {
      token_based: { text: 'Token计费', class: 'bg-blue-100 text-blue-800' },
      request_based: { text: '请求计费', class: 'bg-purple-100 text-purple-800' },
      tier: { text: '阶梯计费', class: 'bg-green-100 text-green-800' },
    };
    const style = styles[type] || { text: type, class: 'bg-gray-100 text-gray-800' };
    return <span className={`px-2 py-1 rounded text-xs ${style.class}`}>{style.text}</span>;
  };

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">计费管理</h1>
        <p className="text-gray-600 mt-1">管理计费规则、账单和用量统计</p>
      </div>

      {/* 标签页导航 */}
      <div className="border-b border-gray-200 mb-6">
        <nav className="-mb-px flex space-x-8">
          {[
            { key: 'rules' as Tab, label: '计费规则' },
            { key: 'invoices' as Tab, label: '账单管理' },
            { key: 'usage' as Tab, label: '用量统计' },
          ].map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`py-4 px-1 border-b-2 font-medium text-sm ${
                activeTab === tab.key
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* 计费规则 */}
      {activeTab === 'rules' && (
        <div>
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-lg font-semibold">计费规则列表</h2>
            <button
              onClick={() => {
                setEditingRule(null);
                setRuleForm({
                  name: '',
                  rule_type: 'token_based',
                  unit_price: 0.0001,
                  currency: 'CNY',
                  free_quota: 0,
                  is_active: true,
                  valid_from: new Date().toISOString().split('T')[0],
                });
                setShowRuleForm(true);
              }}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              新建规则
            </button>
          </div>

          <div className="bg-white rounded-lg shadow overflow-hidden">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">名称</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">类型</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">单价</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">适用范围</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">状态</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">有效期</th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">操作</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {rules.map((rule) => (
                  <tr key={rule.id}>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm font-medium text-gray-900">{rule.name}</div>
                      <div className="text-sm text-gray-500">{rule.description || '-'}</div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">{getRuleTypeBadge(rule.rule_type)}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {rule.unit_price.toFixed(4)} {rule.currency}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {rule.model_alias || '全部模型'}
                      {rule.key_id && ` / ${rule.key_id.slice(0, 8)}...`}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`px-2 py-1 rounded text-xs ${rule.is_active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'}`}>
                        {rule.is_active ? '启用' : '禁用'}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {rule.valid_from.split('T')[0]} ~ {rule.valid_until?.split('T')[0] || '无限期'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <button onClick={() => handleEditRule(rule)} className="text-blue-600 hover:text-blue-900 mr-3">编辑</button>
                      <button onClick={() => handleDeleteRule(rule.id)} className="text-red-600 hover:text-red-900">删除</button>
                    </td>
                  </tr>
                ))}
                {rules.length === 0 && (
                  <tr>
                    <td colSpan={7} className="px-6 py-12 text-center text-gray-500">暂无计费规则</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* 账单管理 */}
      {activeTab === 'invoices' && (
        <div>
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-lg font-semibold">账单列表</h2>
            <div className="space-x-2">
              <button
                onClick={() => setShowGenerateDialog(true)}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
              >
                生成账单
              </button>
              <button
                onClick={() => billingAPI.exportInvoicesCSV().then(res => {
                  const url = window.URL.createObjectURL(new Blob([res.data]));
                  const link = document.createElement('a');
                  link.href = url;
                  link.setAttribute('download', 'invoices.csv');
                  document.body.appendChild(link);
                  link.click();
                })}
                className="px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700"
              >
                导出CSV
              </button>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow overflow-hidden">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">账单号</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">账期</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">金额</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">状态</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">到期日</th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">操作</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {invoices.map((invoice) => (
                  <tr key={invoice.id}>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{invoice.invoice_number}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {invoice.billing_period_start.split('T')[0]} ~ {invoice.billing_period_end.split('T')[0]}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {invoice.total.toFixed(2)} {invoice.currency}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`px-2 py-1 rounded text-xs ${getStatusBadge(invoice.status)}`}>
                        {invoice.status === 'pending' ? '待支付' : invoice.status === 'paid' ? '已支付' : invoice.status === 'overdue' ? '逾期' : '已取消'}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {invoice.due_date?.split('T')[0] || '-'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <button onClick={() => handleViewInvoice(invoice)} className="text-blue-600 hover:text-blue-900 mr-3">详情</button>
                      <button onClick={() => handleExportInvoice(invoice.id, invoice.invoice_number)} className="text-gray-600 hover:text-gray-900">导出</button>
                    </td>
                  </tr>
                ))}
                {invoices.length === 0 && (
                  <tr>
                    <td colSpan={6} className="px-6 py-12 text-center text-gray-500">暂无账单</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {/* 分页 */}
          {invoicesTotal > 20 && (
            <div className="mt-4 flex justify-center">
              <div className="flex space-x-2">
                <button
                  onClick={() => setInvoicesPage(Math.max(1, invoicesPage - 1))}
                  disabled={invoicesPage === 1}
                  className="px-4 py-2 border rounded-lg disabled:opacity-50"
                >
                  上一页
                </button>
                <span className="px-4 py-2">
                  第 {invoicesPage} 页，共 {Math.ceil(invoicesTotal / 20)} 页
                </span>
                <button
                  onClick={() => setInvoicesPage(Math.min(Math.ceil(invoicesTotal / 20), invoicesPage + 1))}
                  disabled={invoicesPage >= Math.ceil(invoicesTotal / 20)}
                  className="px-4 py-2 border rounded-lg disabled:opacity-50"
                >
                  下一页
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* 用量统计 */}
      {activeTab === 'usage' && (
        <div>
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-2">选择 API Key</label>
            <select
              value={usageKeyId}
              onChange={(e) => setUsageKeyId(e.target.value)}
              className="px-4 py-2 border rounded-lg w-64"
            >
              <option value="">请选择</option>
              {apiKeys.map((key) => (
                <option key={key.id} value={key.id}>
                  {key.label} ({key.key_prefix}...)
                </option>
              ))}
            </select>
            <button
              onClick={loadUsageStats}
              disabled={!usageKeyId}
              className="ml-4 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
            >
              查询用量
            </button>
          </div>

          {usageStats && (
            <div className="bg-white rounded-lg shadow p-6">
              <h3 className="text-lg font-semibold mb-4">用量统计</h3>
              <div className="grid grid-cols-3 gap-4 mb-6">
                <div className="bg-blue-50 p-4 rounded-lg">
                  <div className="text-sm text-gray-600">总请求数</div>
                  <div className="text-2xl font-bold text-blue-600">{usageStats.total_requests.toLocaleString()}</div>
                </div>
                <div className="bg-green-50 p-4 rounded-lg">
                  <div className="text-sm text-gray-600">总 Token 数</div>
                  <div className="text-2xl font-bold text-green-600">{usageStats.total_tokens.toLocaleString()}</div>
                </div>
                <div className="bg-purple-50 p-4 rounded-lg">
                  <div className="text-sm text-gray-600">预估费用</div>
                  <div className="text-2xl font-bold text-purple-600">¥{usageStats.total_cost.toFixed(2)}</div>
                </div>
              </div>

              <h4 className="font-semibold mb-3">按模型分组</h4>
              <div className="border rounded-lg overflow-hidden">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">模型</th>
                      <th className="px-4 py-2 text-right text-xs font-medium text-gray-500 uppercase">请求数</th>
                      <th className="px-4 py-2 text-right text-xs font-medium text-gray-500 uppercase">Token数</th>
                      <th className="px-4 py-2 text-right text-xs font-medium text-gray-500 uppercase">费用</th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {Object.entries(usageStats.cost_by_model).map(([model, info]) => (
                      <tr key={model}>
                        <td className="px-4 py-2 text-sm font-medium text-gray-900">{model}</td>
                        <td className="px-4 py-2 text-sm text-right text-gray-900">{info.request_count.toLocaleString()}</td>
                        <td className="px-4 py-2 text-sm text-right text-gray-900">{info.token_count.toLocaleString()}</td>
                        <td className="px-4 py-2 text-sm text-right text-gray-900">¥{info.cost.toFixed(2)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>
      )}

      {/* 规则表单弹窗 */}
      {showRuleForm && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
            <h3 className="text-lg font-semibold mb-4">{editingRule ? '编辑规则' : '新建规则'}</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">规则名称</label>
                <input
                  type="text"
                  value={ruleForm.name}
                  onChange={(e) => setRuleForm({ ...ruleForm, name: e.target.value })}
                  className="w-full px-3 py-2 border rounded-lg"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">描述</label>
                <textarea
                  value={ruleForm.description || ''}
                  onChange={(e) => setRuleForm({ ...ruleForm, description: e.target.value })}
                  className="w-full px-3 py-2 border rounded-lg"
                  rows={2}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">计费类型</label>
                  <select
                    value={ruleForm.rule_type}
                    onChange={(e) => setRuleForm({ ...ruleForm, rule_type: e.target.value as any })}
                    className="w-full px-3 py-2 border rounded-lg"
                  >
                    <option value="token_based">按Token计费</option>
                    <option value="request_based">按请求计费</option>
                    <option value="tier">阶梯计费</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">币种</label>
                  <input
                    type="text"
                    value={ruleForm.currency}
                    onChange={(e) => setRuleForm({ ...ruleForm, currency: e.target.value })}
                    className="w-full px-3 py-2 border rounded-lg"
                  />
                </div>
              </div>
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">单价</label>
                  <input
                    type="number"
                    step="0.0001"
                    value={ruleForm.unit_price}
                    onChange={(e) => setRuleForm({ ...ruleForm, unit_price: parseFloat(e.target.value) })}
                    className="w-full px-3 py-2 border rounded-lg"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">免费配额</label>
                  <input
                    type="number"
                    value={ruleForm.free_quota}
                    onChange={(e) => setRuleForm({ ...ruleForm, free_quota: parseInt(e.target.value) })}
                    className="w-full px-3 py-2 border rounded-lg"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">状态</label>
                  <select
                    value={ruleForm.is_active ? 'true' : 'false'}
                    onChange={(e) => setRuleForm({ ...ruleForm, is_active: e.target.value === 'true' })}
                    className="w-full px-3 py-2 border rounded-lg"
                  >
                    <option value="true">启用</option>
                    <option value="false">禁用</option>
                  </select>
                </div>
              </div>
              {ruleForm.rule_type === 'tier' && (
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">阶梯阈值</label>
                    <input
                      type="number"
                      value={ruleForm.tier_threshold || ''}
                      onChange={(e) => setRuleForm({ ...ruleForm, tier_threshold: parseInt(e.target.value) })}
                      className="w-full px-3 py-2 border rounded-lg"
                      placeholder="超过此数量使用阶梯价格"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">阶梯价格</label>
                    <input
                      type="number"
                      step="0.0001"
                      value={ruleForm.tier_price || ''}
                      onChange={(e) => setRuleForm({ ...ruleForm, tier_price: parseFloat(e.target.value) })}
                      className="w-full px-3 py-2 border rounded-lg"
                    />
                  </div>
                </div>
              )}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">生效日期</label>
                  <input
                    type="date"
                    value={ruleForm.valid_from}
                    onChange={(e) => setRuleForm({ ...ruleForm, valid_from: e.target.value })}
                    className="w-full px-3 py-2 border rounded-lg"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">失效日期（可选）</label>
                  <input
                    type="date"
                    value={ruleForm.valid_until || ''}
                    onChange={(e) => setRuleForm({ ...ruleForm, valid_until: e.target.value || undefined })}
                    className="w-full px-3 py-2 border rounded-lg"
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">适用模型（可选）</label>
                  <input
                    type="text"
                    value={ruleForm.model_alias || ''}
                    onChange={(e) => setRuleForm({ ...ruleForm, model_alias: e.target.value || undefined })}
                    className="w-full px-3 py-2 border rounded-lg"
                    placeholder="留空表示全部模型"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">适用Key（可选）</label>
                  <input
                    type="text"
                    value={ruleForm.key_id || ''}
                    onChange={(e) => setRuleForm({ ...ruleForm, key_id: e.target.value || undefined })}
                    className="w-full px-3 py-2 border rounded-lg"
                    placeholder="留空表示全部Key"
                  />
                </div>
              </div>
            </div>
            <div className="flex justify-end space-x-3 mt-6">
              <button
                onClick={() => setShowRuleForm(false)}
                className="px-4 py-2 border rounded-lg hover:bg-gray-50"
              >
                取消
              </button>
              <button
                onClick={handleSaveRule}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
              >
                保存
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 生成账单弹窗 */}
      {showGenerateDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h3 className="text-lg font-semibold mb-4">生成账单</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">API Key</label>
                <select
                  value={generateForm.key_id}
                  onChange={(e) => setGenerateForm({ ...generateForm, key_id: e.target.value })}
                  className="w-full px-3 py-2 border rounded-lg"
                >
                  <option value="">请选择</option>
                  {apiKeys.map((key) => (
                    <option key={key.id} value={key.id}>
                      {key.label} ({key.key_prefix}...)
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">开始日期</label>
                <input
                  type="date"
                  value={generateForm.start_date}
                  onChange={(e) => setGenerateForm({ ...generateForm, start_date: e.target.value })}
                  className="w-full px-3 py-2 border rounded-lg"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">结束日期</label>
                <input
                  type="date"
                  value={generateForm.end_date}
                  onChange={(e) => setGenerateForm({ ...generateForm, end_date: e.target.value })}
                  className="w-full px-3 py-2 border rounded-lg"
                />
              </div>
            </div>
            <div className="flex justify-end space-x-3 mt-6">
              <button
                onClick={() => setShowGenerateDialog(false)}
                className="px-4 py-2 border rounded-lg hover:bg-gray-50"
              >
                取消
              </button>
              <button
                onClick={handleGenerateInvoice}
                disabled={!generateForm.key_id}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
              >
                生成
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 账单详情弹窗 */}
      {showInvoiceDetail && selectedInvoice && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-3xl max-h-[90vh] overflow-y-auto">
            <div className="flex justify-between items-start mb-4">
              <h3 className="text-lg font-semibold">账单详情</h3>
              <button onClick={() => setShowInvoiceDetail(false)} className="text-gray-400 hover:text-gray-600">✕</button>
            </div>

            <div className="mb-6">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div><span className="text-gray-500">账单号：</span>{selectedInvoice.invoice.invoice_number}</div>
                <div><span className="text-gray-500">状态：</span>
                  <span className={`px-2 py-1 rounded text-xs ${getStatusBadge(selectedInvoice.invoice.status)}`}>
                    {selectedInvoice.invoice.status === 'pending' ? '待支付' : selectedInvoice.invoice.status === 'paid' ? '已支付' : selectedInvoice.invoice.status === 'overdue' ? '逾期' : '已取消'}
                  </span>
                </div>
                <div><span className="text-gray-500">账期：</span>
                  {selectedInvoice.invoice.billing_period_start.split('T')[0]} ~ {selectedInvoice.invoice.billing_period_end.split('T')[0]}
                </div>
                <div><span className="text-gray-500">到期日：</span>{selectedInvoice.invoice.due_date?.split('T')[0] || '-'}</div>
              </div>
            </div>

            <div className="mb-6">
              <h4 className="font-semibold mb-3">账单明细</h4>
              <table className="w-full text-sm">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-3 py-2 text-left">模型</th>
                    <th className="px-3 py-2 text-right">请求数</th>
                    <th className="px-3 py-2 text-right">Token数</th>
                    <th className="px-3 py-2 text-right">单价</th>
                    <th className="px-3 py-2 text-right">小计</th>
                  </tr>
                </thead>
                <tbody>
                  {selectedInvoice.items.map((item) => (
                    <tr key={item.id} className="border-t">
                      <td className="px-3 py-2">{item.model_alias}</td>
                      <td className="px-3 py-2 text-right">{item.request_count.toLocaleString()}</td>
                      <td className="px-3 py-2 text-right">{item.token_count.toLocaleString()}</td>
                      <td className="px-3 py-2 text-right">{item.unit_price.toFixed(4)}</td>
                      <td className="px-3 py-2 text-right">¥{item.line_total.toFixed(2)}</td>
                    </tr>
                  ))}
                </tbody>
                <tfoot className="bg-gray-50 font-medium">
                  <tr>
                    <td colSpan={4} className="px-3 py-2 text-right">小计</td>
                    <td className="px-3 py-2 text-right">¥{selectedInvoice.invoice.subtotal.toFixed(2)}</td>
                  </tr>
                  <tr>
                    <td colSpan={4} className="px-3 py-2 text-right">税</td>
                    <td className="px-3 py-2 text-right">¥{selectedInvoice.invoice.tax.toFixed(2)}</td>
                  </tr>
                  <tr>
                    <td colSpan={4} className="px-3 py-2 text-right">折扣</td>
                    <td className="px-3 py-2 text-right">-¥{selectedInvoice.invoice.discount.toFixed(2)}</td>
                  </tr>
                  <tr>
                    <td colSpan={4} className="px-3 py-2 text-right">总计</td>
                    <td className="px-3 py-2 text-right text-lg">¥{selectedInvoice.invoice.total.toFixed(2)}</td>
                  </tr>
                </tfoot>
              </table>
            </div>

            {selectedInvoice.payments.length > 0 && (
              <div className="mb-6">
                <h4 className="font-semibold mb-3">付款记录</h4>
                <table className="w-full text-sm">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-3 py-2 text-left">金额</th>
                      <th className="px-3 py-2 text-left">方式</th>
                      <th className="px-3 py-2 text-left">状态</th>
                      <th className="px-3 py-2 text-left">支付时间</th>
                    </tr>
                  </thead>
                  <tbody>
                    {selectedInvoice.payments.map((payment) => (
                      <tr key={payment.id} className="border-t">
                        <td className="px-3 py-2">¥{payment.amount.toFixed(2)}</td>
                        <td className="px-3 py-2">{payment.payment_method}</td>
                        <td className="px-3 py-2">
                          <span className={`px-2 py-1 rounded text-xs ${getStatusBadge(payment.status)}`}>
                            {payment.status}
                          </span>
                        </td>
                        <td className="px-3 py-2">{payment.paid_at?.split('T')[0] || '-'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            <div className="flex justify-end space-x-3">
              {selectedInvoice.invoice.status === 'pending' && (
                <>
                  <button
                    onClick={() => handleUpdateInvoiceStatus(selectedInvoice.invoice.id, 'paid')}
                    className="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700"
                  >
                    标记为已支付
                  </button>
                  <button
                    onClick={() => handleUpdateInvoiceStatus(selectedInvoice.invoice.id, 'overdue')}
                    className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
                  >
                    标记为逾期
                  </button>
                </>
              )}
              <button
                onClick={() => handleExportInvoice(selectedInvoice.invoice.id, selectedInvoice.invoice.invoice_number)}
                className="px-4 py-2 border rounded-lg hover:bg-gray-50"
              >
                导出CSV
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
