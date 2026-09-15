import { useAuthStore } from '@/stores/authStore'

// 案件成员关系：与后端 CaseAccessLevel 一一对应。
export type CaseRelation = 'admin' | 'lead' | 'co_lawyer' | 'assistant' | 'none'

export interface CaseMembershipLike {
  lead_lawyer_id: number
  co_lawyer_ids?: number[] | null
  assistant_ids?: number[] | null
}

export function resolveCaseRelation(
  user: { id: number; role: string } | null | undefined,
  kase: CaseMembershipLike | null | undefined,
): CaseRelation {
  if (!user) return 'none'
  if (user.role === 'admin') return 'admin'
  if (!kase) return 'none'
  if (kase.lead_lawyer_id === user.id) return 'lead'
  if ((kase.co_lawyer_ids || []).includes(user.id)) return 'co_lawyer'
  if ((kase.assistant_ids || []).includes(user.id)) return 'assistant'
  return 'none'
}

export function usePermission() {
  const user = useAuthStore((s) => s.user)
  const role = user?.role || ''
  const hasRole = (...roles: string[]) => roles.includes(role)

  const caseRelation = (kase: CaseMembershipLike | null | undefined): CaseRelation =>
    resolveCaseRelation(user, kase)

  // 查看案件/客户/文档：任意成员
  const canViewCase = (kase: CaseMembershipLike | null | undefined) => caseRelation(kase) !== 'none'
  // 维护文档、查看账单：主办/协办/管理员
  const canMaintainDocuments = (kase: CaseMembershipLike | null | undefined) =>
    ['admin', 'lead', 'co_lawyer'].includes(caseRelation(kase))
  const canViewBilling = (kase: CaseMembershipLike | null | undefined) =>
    ['admin', 'lead', 'co_lawyer'].includes(caseRelation(kase))
  // 调整成员、流转状态、交接主办、维护账单：主办/管理员
  const canManageCase = (kase: CaseMembershipLike | null | undefined) =>
    ['admin', 'lead'].includes(caseRelation(kase))
  const canModifyBilling = canManageCase

  return { role, hasRole, caseRelation, canViewCase, canMaintainDocuments, canViewBilling, canManageCase, canModifyBilling }
}
