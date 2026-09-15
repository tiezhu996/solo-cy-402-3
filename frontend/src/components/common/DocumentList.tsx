import { Empty, List, message, Popconfirm } from 'antd'
import DocumentCard from './DocumentCard'
import { deleteDocument } from '@/api/document'
import type { DocumentItem } from '@/types'

interface Props {
  documents: DocumentItem[]
  // maintainable 为 true 时展示删除入口（主办/协办/管理员）；onDeleted 用于删除后刷新。
  maintainable?: boolean
  onDeleted?: () => void
}

export default function DocumentList({ documents, maintainable = false, onDeleted }: Props) {
  if (documents.length === 0) return <Empty description="暂无文档" />

  async function onDelete(id: number) {
    await deleteDocument(id)
    message.success('已删除')
    onDeleted?.()
  }

  return (
    <List
      grid={{ gutter: 16, column: 2 }}
      dataSource={documents}
      renderItem={(doc) => (
        <List.Item>
          <DocumentCard
            item={doc}
            extra={maintainable ? (
              <Popconfirm title="确认删除该文档？" onConfirm={() => onDelete(doc.id)}>
                <a>删除</a>
              </Popconfirm>
            ) : undefined}
          />
        </List.Item>
      )}
    />
  )
}
