import request from '@/utils/request'

export function listDocuments(params: { page?: number; page_size?: number; file_type?: string; keyword?: string }) {
  return request.get('/documents', { params })
}

export function listDocumentsByCase(caseId: number) {
  return request.get(`/documents/by-case/${caseId}`)
}

export function createDocument(data: { case_id: number; title: string; file_type: string; file_url: string }) {
  return request.post('/documents', data)
}

export function deleteDocument(id: number) {
  return request.delete(`/documents/${id}`)
}

// 下载案件文件：服务端按文档记录校验案件成员关系，成员被移出后立即 403。
export function downloadDocument(id: number) {
  return request.get(`/documents/${id}/download`, { responseType: 'blob' })
}

// 以新窗口预览案件文件（携带 JWT 的 blob 下载后转对象 URL）。
export async function openDocument(id: number) {
  try {
    const blob: any = await downloadDocument(id)
    window.open(URL.createObjectURL(blob), '_blank')
  } catch {
    // 拦截器已提示错误（如无权限）
  }
}
