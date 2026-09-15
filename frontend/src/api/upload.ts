import request from '@/utils/request'

// 上传案件文件：必须携带 case_id，服务端校验案件成员范围与文档维护权限后才写盘。
export function uploadFile(file: File, caseId: number) {
  const form = new FormData()
  form.append('file', file)
  form.append('case_id', String(caseId))
  return request.post('/upload/file', form, { headers: { 'Content-Type': 'multipart/form-data' } })
}

// 上传头像：个人资料场景，与案件无关。
export function uploadAvatar(file: File) {
  const form = new FormData()
  form.append('file', file)
  return request.post('/upload/avatar', form, { headers: { 'Content-Type': 'multipart/form-data' } })
}
