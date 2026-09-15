import { useEffect, useMemo, useState } from 'react'
import { Button, Card, Form, Input, InputNumber, message, Modal, Select, Space, Table } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import AmountSummary from '@/components/common/AmountSummary'
import StatusBadge from '@/components/common/StatusBadge'
import { useBillingStore } from '@/stores/billingStore'
import { useAuthStore } from '@/stores/authStore'
import { createBilling, markPaid, markInvoiced, voidBilling } from '@/api/billing'
import { listCases } from '@/api/case'
import { BillingStatusOptions, BillingTypeOptions, BillingTypeText } from '@/constants/billing'
import { formatAmount } from '@/utils/amountFormatter'
import type { Billing, CaseItem } from '@/types'

export default function Billing() {
  const store = useBillingStore()
  const me = useAuthStore((s) => s.user)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [filters, setFilters] = useState<Record<string, unknown>>({})
  const [open, setOpen] = useState(false)
  const [ledCases, setLedCases] = useState<CaseItem[]>([])
  const [form] = Form.useForm()

  useEffect(() => {
    store.fetchList({ page, page_size: pageSize, ...filters })
    store.fetchSummary()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, filters])

  useEffect(() => {
    // 可维护账单的案件集合：管理员为全部案件，律师仅其主办的案件。
    const params: Record<string, unknown> = { page: 1, page_size: 200 }
    if (me?.role !== 'admin' && me?.id) {
      params.lead_lawyer_id = me.id
    }
    listCases(params).then((res: any) => setLedCases(res.data.list || []))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [me?.id])

  // 仅主办律师/管理员可改动账单；行级按钮按案件判定。
  const ledCaseIds = useMemo(() => new Set(ledCases.map((c) => c.id)), [ledCases])
  const canModify = (caseId: number) => ledCaseIds.has(caseId)

  async function onCreate() {
    const values = await form.validateFields()
    const kase = ledCases.find((c) => c.id === values.case_id)
    await createBilling({ ...values, client_id: kase?.client_id })
    message.success('账单创建成功')
    setOpen(false)
    form.resetFields()
    setPage(1)
    store.fetchList({ page: 1, page_size: pageSize, ...filters })
    store.fetchSummary()
  }

  async function run(action: () => Promise<unknown>, msg: string) {
    await action()
    message.success(msg)
    store.fetchList({ page, page_size: pageSize, ...filters })
    store.fetchSummary()
  }

  return (
    <Card>
      <AmountSummary receivable={store.summary.receivable} received={store.summary.received} pending={store.summary.pending} />
      <Space style={{ marginBottom: 16 }}>
        <Select placeholder="账单状态" allowClear style={{ width: 140 }} options={BillingStatusOptions} onChange={(v) => { setFilters({ status: v }); setPage(1) }} />
        <Button type="primary" icon={<PlusOutlined />} disabled={ledCases.length === 0} onClick={() => setOpen(true)}>创建账单</Button>
      </Space>
      <Table<Billing>
        rowKey="id"
        dataSource={store.list}
        pagination={{ current: page, pageSize, total: store.total, onChange: (p, ps) => { setPage(p); setPageSize(ps) } }}
        columns={[
          { title: '单号', dataIndex: 'bill_no' },
          { title: '类型', dataIndex: 'billing_type', render: (v) => BillingTypeText[v] || v },
          { title: '金额', dataIndex: 'amount', render: (v) => formatAmount(v) },
          { title: '状态', dataIndex: 'status', render: (v) => <StatusBadge status={v} kind="billing" /> },
          { title: '案件ID', dataIndex: 'case_id' },
          { title: '客户ID', dataIndex: 'client_id' },
          {
            title: '操作',
            render: (_, row) => (
              <Space>
                {canModify(row.case_id) && row.status === 'pending' && (
                  <Button size="small" type="primary" onClick={() => run(() => markPaid(row.id), '已标记支付')}>标记支付</Button>
                )}
                {canModify(row.case_id) && row.status === 'paid' && (
                  <Button size="small" onClick={() => run(() => markInvoiced(row.id), '已开票')}>开票</Button>
                )}
                {canModify(row.case_id) && row.status !== 'void' && (
                  <Button size="small" danger onClick={() => run(() => voidBilling(row.id), '已作废')}>作废</Button>
                )}
              </Space>
            ),
          },
        ]}
      />
      <Modal title="创建账单" open={open} onOk={onCreate} onCancel={() => setOpen(false)}>
        <Form form={form} layout="vertical">
          <Form.Item name="case_id" label="案件（仅可为您主办的案件）" rules={[{ required: true }]}>
            <Select
              showSearch
              optionFilterProp="label"
              options={ledCases.map((c) => ({ label: `${c.case_no} ${c.title}`, value: c.id }))}
            />
          </Form.Item>
          <Form.Item name="billing_type" label="费用类型" rules={[{ required: true }]}><Select options={BillingTypeOptions} /></Form.Item>
          <Form.Item name="amount" label="金额" rules={[{ required: true }]}><InputNumber style={{ width: '100%' }} min={0} precision={2} /></Form.Item>
          <Form.Item name="invoice_info" label="发票信息"><Input /></Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}
