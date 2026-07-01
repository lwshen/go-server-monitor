// Pinia store for admin authentication. The JWT lives in localStorage so the
// axios request interceptor can read it; this store keeps a reactive mirror.

import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { clearToken, getToken, login as apiLogin, setToken } from '@/services/api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(getToken())

  const isAuthed = computed(() => !!token.value)

  /** Authenticate and persist the returned token. */
  async function login(username: string, password: string): Promise<void> {
    const res = await apiLogin(username, password)
    setToken(res.token)
    token.value = res.token
  }

  /** Clear the token locally (e.g. on logout or a 401). */
  function logout(): void {
    clearToken()
    token.value = null
  }

  /** Re-read the token from storage (e.g. after a 401 cleared it). */
  function sync(): void {
    token.value = getToken()
  }

  return { token, isAuthed, login, logout, sync }
})
