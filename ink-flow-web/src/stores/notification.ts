import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import type { UnreadCountMap } from '@/types/notification.ts'
import { unreadCount } from '@/service/notification.ts'
import { useUserStore } from '@/stores/user.ts'

const defaultUnreadMap = (): UnreadCountMap => ({
  reply: 0,
  like: 0,
  follow: 0,
  system: 0,
  mention: 0,
  subscribe: 0,
})

export const useUnreadNotificationStore = defineStore('unread-notification-store', () => {
  const unreadMap = ref<UnreadCountMap>(defaultUnreadMap())
  let timer: ReturnType<typeof setInterval> | null = null

  const refresh = async () => {
    try {
      const res = await unreadCount()
      unreadMap.value = { ...defaultUnreadMap(), ...res }
    } catch (err) {
      console.error('[notification] unreadCount failed:', err)
    }
  }

  const startPolling = () => {
    if (timer) return
    refresh()
    timer = setInterval(refresh, 1000 * 60)
  }

  const stopPolling = () => {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
    unreadMap.value = defaultUnreadMap()
  }

  const userStore = useUserStore()
  watch(
    () => userStore.getActiveUser()?.user?.id,
    (userId) => {
      if (userId) {
        startPolling()
      } else {
        stopPolling()
      }
    },
    { immediate: true }
  )

  return { unreadMap, refresh, startPolling, stopPolling }
})
