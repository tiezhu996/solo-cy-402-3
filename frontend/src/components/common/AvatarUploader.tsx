import { Upload } from 'antd'
import { useFileUpload } from '@/hooks/useFileUpload'

export default function AvatarUploader({ onUploaded }: { onUploaded: (url: string) => void }) {
  const { uploading, uploadAvatarFile } = useFileUpload()
  return (
    <Upload
      showUploadList={false}
      accept="image/*"
      beforeUpload={(file) => {
        uploadAvatarFile(file).then((url) => {
          if (url) onUploaded(url)
        })
        return false
      }}
    >
      <span style={{ color: uploading ? '#999' : '#1677ff' }}>{uploading ? '上传中...' : '点击上传头像'}</span>
    </Upload>
  )
}
