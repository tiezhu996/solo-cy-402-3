import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Card, Descriptions, Tabs, Button, Select, Space, message, Tag, Modal, Form, Input, Result } from 'antd'
import { getCase, changeCaseStatus, assignLawyer, getCaseMembers, updateCaseMembers } from '@/api/case'
import { getClient } from '@/api/client'
import { createDocument } from '@/api/document'
import DocumentList from '@/components/common/DocumentList'
import BillingCard from '@/components/common/BillingCard'
import StatusBadge from '@/components/common/StatusBadge'
import TimelineItem from '@/components/common/TimelineItem'
import FileUploader from '@/components/common/FileUploader'
import { useDocumentStore } from '@/stores/documentStore'
import { useBillingStore } from '@/stores/billingStore'
import { useUserStore } from '@/stores/userStore'
import { usePermission } from '@/hooks/usePermission'
import { DocumentTypeOptions } from '@/constants/document'
import { CaseStatusOptions, CaseTypeOptions } from '@/constants/case'
import type { CaseItem, CaseMembers, Client } from '@/types'

export default function CaseDetail() {
  const { id } = useParams()
  const caseId = Number(id)
  const [item, setItem] = useState<CaseItem | null>(null)
  const [members, setMembers] = useState<CaseMembers | null>(null)
  const [client, setClient] = useState<Client | null>(null)
  const [denied, setDenied] = useState(false)
  const [status, setStatus] = useState('')
  const [handover, setHandover] = useState<number>()
  const [coSel, setCoSel] = useState<number[]>([])
  const [assistantSel, setAssistantSel] = useState<number[]>([])
  const [docOpen, setDocOpen] = useState(false)
  const [docUrl, setDocUrl] = useState('')
  const [docForm] = Form.useForm()
  const docStore = useDocumentStore()
  const billingStore = useBillingStore()
  const userStore = useUserStore()
  const { canManageCase, canViewBilling, canMaintainDocuments } = usePermission()

  useEffect(() => {
    userStore.fetchLawyers()
    userStore.fetchAssistants()
    load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [caseId])

  async function load() {
    try {
      const res: any = await getCase(caseId)
      const kase: CaseItem = res.data
      setItem(kase)
      setStatus(kase.status)
      const mr: any = await getCaseMembers(caseId)
      setMembers(mr.data)
      setCoSel(mr.data.co_lawyer_ids || [])
      setAssistantSel(mr.data.assistant_ids || [])
      if (kase.client_id) {
        const cr: any = await getClient(kase.client_id)
        setClient(cr.data.client)
      }
      docStore.fetchByCase(caseId)
      if (canViewBilling(kase)) {
        billingStore.fetchByCase(caseId)
      }
    } catch {
      setDenied(true)
    }
  }

  async function onStatusChange() {
    await changeCaseStatus(caseId, status)
    message.success('状态已更新')
    load()
  }

  async function onHandover() {
    if (!handover) return
    await assignLawyer(caseId, { lead_lawyer_id: handover })
    message.success('主办律师已交接，管理权同步转移')
    setHandover(undefined)
    load()
  }

  async function onSaveMembers() {
    await updateCaseMembers(caseId, { co_lawyer_ids: coSel, assistant_ids: assistantSel })
    message.success('案件成员已更新')
    load()
  }

  async function onUploadDoc() {
    const values = await docForm.validateFields()
    if (!docUrl) {
      message.warning('请先上传文件')
      return
    }
    await createDocument({ ...values, case_id: caseId, file_url: docUrl })
    message.success('文档上传成功')
    setDocOpen(false)
    setDocUrl('')
    docForm.resetFields()
    docStore.fetchByCase(caseId)
  }

  if (denied) {
    return <Result status="403" title="无权访问" subTitle="您不是该案件成员，无法查看案件资料" />
  }
  if (!item) return null

  const manageable = canManageCase(item)
  const maintainable = canMaintainDocuments(item)
  const lawyerOptions = userStore.lawyers.map((l) => ({ label: l.real_name || l.username, value: l.id }))
  const coOptions = lawyerOptions.filter((o) => o.value !== item.lead_lawyer_id)
  const handoverOptions = lawyerOptions.filter((o) => o.value !== item.lead_lawyer_id)
  const assistantOptions = userStore.assistants.map((a) => ({ label: a.real_name || a.username, value: a.id }))

  return (
    <Card>
      <Space style={{ marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>{item.case_no} {item.title}</h2>
        <StatusBadge status={item.status} />
        <Tag>{CaseTypeOptions.find((o) => o.value === item.case_type)?.label || item.case_type}</Tag>
      </Space>
      <Tabs
        items={[
          {
            key: 'info',
            label: '案件信息',
            children: (
              <>
                <Descriptions bordered column={2} size="small">
                  <Descriptions.Item label="案号">{item.case_no}</Descriptions.Item>
                  <Descriptions.Item label="状态">{item.status}</Descriptions.Item>
                  <Descriptions.Item label="类型">{item.case_type}</Descriptions.Item>
                  <Descriptions.Item label="主办律师">{members?.lead_lawyer?.real_name || `#${item.lead_lawyer_id}`}</Descriptions.Item>
                  <Descriptions.Item label="协办律师">
                    {members && members.co_lawyers.length > 0 ? members.co_lawyers.map((m) => m.real_name || m.username).join('、') : '-'}
                  </Descriptions.Item>
                  <Descriptions.Item label="助理">
                    {members && members.assistants.length > 0 ? members.assistants.map((m) => m.real_name || m.username).join('、') : '-'}
                  </Descriptions.Item>
                  <Descriptions.Item label="受理日期">{item.accept_date || '-'}</Descriptions.Item>
                  <Descriptions.Item label="结案日期">{item.close_date || '-'}</Descriptions.Item>
                  <Descriptions.Item label="摘要" span={2}>{item.summary || '-'}</Descriptions.Item>
                </Descriptions>
                {manageable && (
                  <>
                    <Space style={{ marginTop: 16 }}>
                      <Select value={status} style={{ width: 150 }} options={CaseStatusOptions} onChange={setStatus} />
                      <Button type="primary" onClick={onStatusChange}>更新状态</Button>
                    </Space>
                    <Card size="small" title="协作成员管理" style={{ marginTop: 16 }}>
                      <Space direction="vertical" style={{ width: '100%' }} size={12}>
                        <Space wrap>
                          <span style={{ width: 72, display: 'inline-block' }}>协办律师</span>
                          <Select
                            mode="multiple"
                            placeholder="选择协办律师"
                            style={{ minWidth: 320 }}
                            value={coSel}
                            onChange={setCoSel}
                            options={coOptions}
                          />
                        </Space>
                        <Space wrap>
                          <span style={{ width: 72, display: 'inline-block' }}>助理</span>
                          <Select
                            mode="multiple"
                            placeholder="选择助理"
                            style={{ minWidth: 320 }}
                            value={assistantSel}
                            onChange={setAssistantSel}
                            options={assistantOptions}
                          />
                        </Space>
                        <Button type="primary" onClick={onSaveMembers}>保存成员</Button>
                        <Space wrap>
                          <span style={{ width: 72, display: 'inline-block' }}>主办交接</span>
                          <Select
                            placeholder="交接给律师"
                            style={{ width: 180 }}
                            value={handover}
                            onChange={setHandover}
                            options={handoverOptions}
                          />
                          <Button onClick={onHandover}>交接</Button>
                        </Space>
                      </Space>
                    </Card>
                  </>
                )}
              </>
            ),
          },
          {
            key: 'client',
            label: '关联客户',
            children: client ? (
              <Descriptions bordered column={1} size="small">
                <Descriptions.Item label="姓名">{client.name}</Descriptions.Item>
                <Descriptions.Item label="证件号">{client.id_number}</Descriptions.Item>
                <Descriptions.Item label="联系方式">{client.contact}</Descriptions.Item>
                <Descriptions.Item label="地址">{client.address}</Descriptions.Item>
              </Descriptions>
            ) : null,
          },
          {
            key: 'docs',
            label: '文档',
            children: (
              <>
                {maintainable && (
                  <Button type="primary" style={{ marginBottom: 16 }} onClick={() => setDocOpen(true)}>上传文档</Button>
                )}
                <DocumentList documents={docStore.byCase} maintainable={maintainable} onDeleted={() => docStore.fetchByCase(caseId)} />
              </>
            ),
          },
          ...(canViewBilling(item)
            ? [{
                key: 'billings',
                label: '账单',
                children: billingStore.byCase.map((b) => <BillingCard key={b.id} item={b} />),
              }]
            : []),
          {
            key: 'timeline',
            label: '时间线',
            children: (
              <TimelineItem
                items={[
                  { id: 1, time: item.created_at, text: `案件创建（${item.case_no}）` },
                  { id: 2, time: item.accept_date || item.created_at, text: '案件受理' },
                  { id: 3, time: item.close_date || item.created_at, text: item.close_date ? '案件结案' : '案件进行中' },
                ]}
              />
            ),
          },
        ]}
      />
      <Modal title="上传文档" open={docOpen} onOk={onUploadDoc} onCancel={() => setDocOpen(false)}>
        <Form form={docForm} layout="vertical">
          <Form.Item name="title" label="文档标题" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="file_type" label="文件类型" rules={[{ required: true }]}>
            <Select options={DocumentTypeOptions} />
          </Form.Item>
          <Form.Item label="文件">
            <FileUploader onUploaded={(u) => setDocUrl(u)} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}
