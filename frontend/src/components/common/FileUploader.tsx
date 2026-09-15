import { Button, Upload } from 'antd'
import { UploadOutlined } from '@ant-design/icons'
import { useFileUpload } from '@/hooks/useFileUpload'

interface Props {
  // 案件文件必须携带 caseId：服务端按案件成员范围 + 文档维护权限校验，未选择案件时禁止上传。
  caseId?: number
  onUploaded: (url: string) => void
}

export default function FileUploader({ caseId, onUploaded }: Props) {
  const { uploading, upload } = useFileUpload()
  const disabled = !caseId
  return (
    <Upload
      showUploadList={false}
      beforeUpload={(file) => {
        if (!caseId) return false
        upload(file, caseId).then((url) => {
          if (url) onUploaded(url)
        })
        return false
      }}
    >
      <Button icon={<UploadOutlined />} loading={uploading} disabled={disabled}>
        {disabled ? '请先选择案件' : '选择文件上传'}
      </Button>
    </Upload>
  )
}
