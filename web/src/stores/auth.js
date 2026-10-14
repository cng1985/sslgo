import { defineStore } from 'pinia'
import http from '../api/http'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('ssl_token') || '',
    profile: null
  }),
  getters: {
    can: (state) => (key) => (state.profile?.permissions || []).includes(key),
    name: (state) => state.profile?.user?.displayName || state.profile?.user?.username || ''
  },
  actions: {
    async login(form) {
      const body = await http.post('/auth/login', form)
      this.token = body.data.token
      this.profile = body.data.profile
      localStorage.setItem('ssl_token', this.token)
    },
    async loadProfile() {
      const body = await http.post('/auth/profile', {})
      this.profile = body.data
    },
    logout() {
      this.token = ''
      this.profile = null
      localStorage.removeItem('ssl_token')
    }
  }
})
