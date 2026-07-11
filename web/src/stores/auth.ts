import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api/auth'
import type { User, AuthResponse } from '@/api/types'

export const useAuthStore = defineStore('auth', () => {
  // The JWT lives only in an HttpOnly cookie set by the server, so it is never
  // readable from JavaScript (XSS can't exfiltrate it). We cache only the profile
  // for instant UI; the cookie authenticates every request, and a stale cache is
  // corrected by the 401 interceptor.
  const user = ref<User | null>(JSON.parse(localStorage.getItem('mb_user') || 'null'))

  const isAuthenticated = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  function setAuth(data: AuthResponse) {
    user.value = data.user || null
    localStorage.setItem('mb_user', JSON.stringify(user.value))
  }

  // login returns true when authentication completed, or false when the account
  // has 2FA enabled and a TOTP/recovery code is still required. Pass the code on
  // the second call to finish signing in. The session cookie is set by the server.
  async function login(identifier: string, password: string, twoFactorCode?: string) {
    const res = await authApi.login(identifier, password, twoFactorCode)
    if (res.data.data?.two_factor_required) {
      return false
    }
    if (!res.data.data?.user) {
      throw new Error('Login failed: unexpected server response')
    }
    setAuth(res.data.data)
    return true
  }

  // clearSession drops all local auth state WITHOUT calling the API. Safe to call
  // repeatedly — used by the 401 interceptor, so it must never itself make a
  // request (that would 401 and re-trigger the interceptor in a loop). The cookie
  // is cleared server-side by logout(); here we only drop the cached profile.
  function clearSession() {
    user.value = null
    localStorage.removeItem('mb_user')
    localStorage.removeItem('mb_workspace_id')
  }

  function logout() {
    // Best-effort server-side revoke, then clear local state regardless.
    authApi.logout().catch(() => {})
    clearSession()
  }

  async function fetchUser() {
    try {
      const res = await authApi.me()
      user.value = res.data.data
      localStorage.setItem('mb_user', JSON.stringify(res.data.data))
    } catch {
      // A failed /me means the session is gone; drop local state only (a 401 is
      // already handled by the API interceptor, which redirects to /login).
      clearSession()
    }
  }

  // Updates display name (and optionally the username handle), refreshing the cache.
  async function updateProfile(name: string, username?: string) {
    const res = await authApi.updateProfile(name, username)
    user.value = res.data.data
    localStorage.setItem('mb_user', JSON.stringify(res.data.data))
  }
  // updateName is a thin back-compat wrapper over updateProfile.
  async function updateName(name: string) {
    await updateProfile(name)
  }

  // Hides the getting-started checklist permanently; refreshes the cache so it survives a reload.
  async function dismissOnboarding() {
    const res = await authApi.dismissOnboarding()
    user.value = res.data.data
    localStorage.setItem('mb_user', JSON.stringify(res.data.data))
  }

  return { user, isAuthenticated, isAdmin, login, logout, clearSession, fetchUser, updateName, updateProfile, dismissOnboarding }
})
