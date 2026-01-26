import { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { loadConfig } from '../utils/api'

interface UserContextType {
  userId: number | null
  isLoading: boolean
  error: string | null
  refreshUserId: () => Promise<void>
}

const UserContext = createContext<UserContextType | undefined>(undefined)

/**
 * Централизованное получение user_id один раз при открытии приложения
 * Используется для всех API запросов в течение сессии
 */
export function UserProvider({ children }: { children: ReactNode }) {
  const [userId, setUserId] = useState<number | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const extractUserId = async (): Promise<number | null> => {
    const tg = window.Telegram?.WebApp
    const initData = tg?.initData || ''

    // Способ 1: Из initDataUnsafe (самый надежный способ)
    if (tg?.initDataUnsafe?.user?.id) {
      const id = tg.initDataUnsafe.user.id
      console.log('[UserContext] Got user_id from initDataUnsafe:', id)
      return id
    }

    // Способ 2: Из initData через API (если initDataUnsafe недоступен)
    if (initData) {
      try {
        const config = await loadConfig()
        const response = await fetch(`${config.apiUrl}/parse-init-data`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ initData }),
        })

        if (response.ok) {
          const parsed = await response.json()
          if (parsed.userId) {
            console.log('[UserContext] Got user_id from parse-init-data:', parsed.userId)
            return parsed.userId
          }
        } else {
          const errorText = await response.text()
          console.error('[UserContext] Error parsing initData:', response.status, errorText)
        }
      } catch (error) {
        console.error('[UserContext] Error parsing initData:', error)
      }
    }

    // Способ 3: Из localStorage (fallback для разработки)
    const savedUserId = localStorage.getItem('telegram_user_id')
    if (savedUserId) {
      const id = parseInt(savedUserId, 10)
      if (!isNaN(id)) {
        console.log('[UserContext] Got user_id from localStorage:', id)
        return id
      }
    }

    return null
  }

  const refreshUserId = async () => {
    setIsLoading(true)
    setError(null)

    try {
      // Ждем немного, чтобы Telegram WebApp успел инициализироваться
      await new Promise(resolve => setTimeout(resolve, 100))

      const id = await extractUserId()
      
      if (id) {
        setUserId(id)
        // Сохраняем в localStorage для fallback
        localStorage.setItem('telegram_user_id', id.toString())
      } else {
        setError('user_id_not_found')
        console.warn('[UserContext] Could not extract user_id. Available:', {
          hasTg: !!window.Telegram?.WebApp,
          hasInitData: !!window.Telegram?.WebApp?.initData,
          hasInitDataUnsafe: !!window.Telegram?.WebApp?.initDataUnsafe,
          hasInitDataUnsafeUser: !!window.Telegram?.WebApp?.initDataUnsafe?.user,
        })
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'unknown_error'
      setError(errorMessage)
      console.error('[UserContext] Error extracting user_id:', err)
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    refreshUserId()
  }, [])

  return (
    <UserContext.Provider value={{ userId, isLoading, error, refreshUserId }}>
      {children}
    </UserContext.Provider>
  )
}

export function useUser() {
  const context = useContext(UserContext)
  if (context === undefined) {
    throw new Error('useUser must be used within a UserProvider')
  }
  return context
}

