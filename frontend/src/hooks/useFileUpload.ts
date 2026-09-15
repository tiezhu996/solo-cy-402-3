import { useState } from 'react'
import { message } from 'antd'
import { uploadFile, uploadAvatar } from '@/api/upload'

export function useFileUpload() {
  const [uploading, setUploading] = useState(false)
  const [url, setUrl] = useState('')

  // 案件文件上传：caseId 必填，权限校验在服务端按案件成员关系即时判定。
  async function upload(file: File, caseId: number) {
    setUploading(true)
    try {
      const res: any = await uploadFile(file, caseId)
      setUrl(res.data.url)
      message.success('上传成功')
      return res.data.url as string
    } catch {
      return ''
    } finally {
      setUploading(false)
    }
  }

  async function uploadAvatarFile(file: File) {
    setUploading(true)
    try {
      const res: any = await uploadAvatar(file)
      setUrl(res.data.url)
      message.success('上传成功')
      return res.data.url as string
    } catch {
      return ''
    } finally {
      setUploading(false)
    }
  }

  return { uploading, url, upload, uploadAvatarFile }
}
