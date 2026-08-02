import { describe, it, expect, beforeEach } from 'vitest'
import { useAuthStore } from '@/store/authStore'
import type { User } from '@/types'

const mockUser: User = {
  id: '123e4567-e89b-12d3-a456-426614174000',
  name: 'Test User',
  email: 'test@example.com',
  created_at: new Date().toISOString(),
}

describe('useAuthStore', () => {
  beforeEach(() => {
    useAuthStore.setState({ user: null, token: null, isAuthenticated: false })
    localStorage.clear()
  })

  it('starts unauthenticated', () => {
    const state = useAuthStore.getState()
    expect(state.isAuthenticated).toBe(false)
    expect(state.user).toBeNull()
    expect(state.token).toBeNull()
  })

  it('sets auth state on setAuth', () => {
    useAuthStore.getState().setAuth(mockUser, 'my-token')
    const state = useAuthStore.getState()
    expect(state.isAuthenticated).toBe(true)
    expect(state.user).toEqual(mockUser)
    expect(state.token).toBe('my-token')
    expect(localStorage.getItem('token')).toBe('my-token')
  })

  it('clears auth state on clearAuth', () => {
    useAuthStore.getState().setAuth(mockUser, 'my-token')
    useAuthStore.getState().clearAuth()
    const state = useAuthStore.getState()
    expect(state.isAuthenticated).toBe(false)
    expect(state.user).toBeNull()
    expect(state.token).toBeNull()
    expect(localStorage.getItem('token')).toBeNull()
  })
})
