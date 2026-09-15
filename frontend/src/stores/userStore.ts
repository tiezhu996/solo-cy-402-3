import { create } from 'zustand'
import { getMe, updateProfile, listLawyers, listAssistants } from '@/api/user'
import type { User } from '@/types'

interface UserState {
  me: User | null
  lawyers: User[]
  assistants: User[]
  fetchMe: () => Promise<User>
  updateMe: (data: Partial<User>) => Promise<User>
  fetchLawyers: () => Promise<void>
  fetchAssistants: () => Promise<void>
}

export const useUserStore = create<UserState>((set) => ({
  me: null,
  lawyers: [],
  assistants: [],
  async fetchMe() {
    const res: any = await getMe()
    set({ me: res.data })
    return res.data as User
  },
  async updateMe(data: Partial<User>) {
    const res: any = await updateProfile(data)
    set({ me: res.data })
    return res.data as User
  },
  async fetchLawyers() {
    const res: any = await listLawyers()
    set({ lawyers: res.data })
  },
  async fetchAssistants() {
    const res: any = await listAssistants()
    set({ assistants: res.data })
  },
}))
