import { useEffect, useMemo, useState } from 'react'
import { Card, Form, Input, message, Modal, Select, Space, Table, Button } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { useDocumentStore } from '@/stores/documentStore'
import { useAuthStore } from '@/stores/authStore'
import { createDocument, deleteDocument } from '@/api/document'
import { listCases } from '@/api/case'
import { DocumentTypeOptions } from '@/constants/document'
import FileUploader from '@/components/common/FileUploader'
import { resolveCaseRelation } from '@/hooks/usePermission'
import { formatDateTime } from '@/utils/dateFormat'
import type { CaseItem, DocumentItem } from '@/types'

export default function Documents() {
  const store = useDocumentStore()
  const me = useAuthStore((s) => s.user)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [filters, setFilters] = useState<Record<string, unknown>>({})
  const [open, setOpen] = useState(false)
  const [url, setUrl] = useState('')
  const [cases, setCases] = useState<CaseItem[]>([])
  const [form] = Form.useForm()
  const selectedCaseId = Form.useWatch('case_id', form)

  useEffect(() => {
    store.fetchList({ page, page_size: pageSize, ...filters })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, filters])

  useEffect(() => {
    // 案件列表服务端已按成员关系过滤，这里取回后在前端计算各案件的维护权限。
    listCases({ page: 1, page_size: 200 }).then((res: any) => setCases(res.data.list || []))
  }, [])

  // 文档维护（上传/删除）：主办/协办/管理员；助理只读。
  const relationOf = useMemo(() => {
    const m = new Map<number, string>()
    cases.forEach((c) => m.set(c.id, resolveCaseRelation(me, c)))
    return m
  }, [cases, me])
  const canMaintain = (caseId: number) => ['admin', 'lead', 'co_lawyer'].includes(relationOf.get(caseId) || 'none')
  const maintainableCases = cases.filter((c) => canMaintain(c.id))
  const canUpload = me?.role === 'admin' || me?.role === 'lawyer'

  async function onCreate() {
    const values = await form.validateFields()
    if (!url) {
      message.warning('请先上传文件')
      return
    }
    await createDocument({ ...values, file_url: url })
    message.success('文档上传成功')
    setOpen(false)
    setUrl('')
    form.resetFields()
    setPage(1)
    store.fetchList({ page: 1, page_size: pageSize, ...filters })
  }

  return (
    <Card>
      <Space style={{ marginBottom: 16 }}>
        <Input.Search placeholder="搜索文档标题" allowClear style={{ width: 240 }} onSearch={(v) => { setFilters({ keyword: v }); setPage(1) }} />
        <Select placeholder="文件类型" allowClear style={{ width: 140 }} options={DocumentTypeOptions} onChange={(v) => { setFilters({ file_type: v }); setPage(1) }} />
        {canUpload && (
          <Button type="primary" icon={<PlusOutlined />} disabled={maintainableCases.length === 0} onClick={() => setOpen(true)}>上传文档</Button>
        )}
      </Space>
      <Table<DocumentItem>
        rowKey="id"
        dataSource={store.list}
        pagination={{ current: page, pageSize, total: store.total, onChange: (p, ps) => { setPage(p); setPageSize(ps) } }}
        columns={[
          { title: 'ID', dataIndex: 'id' },
          { title: '标题', dataIndex: 'title' },
          { title: '案件ID', dataIndex: 'case_id' },
          { title: '类型', dataIndex: 'file_type', render: (v) => DocumentTypeOptions.find((o) => o.value === v)?.label || v },
          { title: '上传时间', dataIndex: 'upload_time', render: (v) => formatDateTime(v) },
          {
            title: '操作',
            render: (_, row) => (
              <Space>
                <a href={row.file_url} target="_blank" rel="noreferrer">查看</a>
                {canMaintain(row.case_id) && (
                  <Button type="link" danger onClick={async () => { await deleteDocument(row.id); message.success('已删除'); store.fetchList({ page, page_size: pageSize, ...filters }) }}>删除</Button>
                )}
              </Space>
            ),
          },
        ]}
      />
      <Modal title="上传文档" open={open} onOk={onCreate} onCancel={() => setOpen(false)}>
        <Form form={form} layout="vertical">
          <Form.Item name="case_id" label="案件（仅可维护的案件）" rules={[{ required: true }]}>
            <Select
              showSearch
              optionFilterProp="label"
              options={maintainableCases.map((c) => ({ label: `${c.case_no} ${c.title}`, value: c.id }))}
              onChange={() => setUrl('')}
            />
          </Form.Item>
          <Form.Item name="title" label="文档标题" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="file_type" label="文件类型" rules={[{ required: true }]}><Select options={DocumentTypeOptions} /></Form.Item>
          <Form.Item label="文件">
            <FileUploader caseId={selectedCaseId} onUploaded={(u) => setUrl(u)} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}
