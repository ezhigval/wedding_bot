import { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { loadConfig } from '../utils/api'
import { getInitDataAny } from '../utils/telegram'

interface UserContextType {
  userId: number | null
  isLoading: boolean
  error: string | null
  refreshUserId: () => Promise<void>
  manualUsername: string | null
  setManualUsername: (username: string | null) => void
}

const UserContext = createContext<UserContextType | undefined>(undefined)

interface ExtractedIdentity {
  userId: number | null
  username: string | null
}

function normalizeUsername(username?: string | null): string | null {
  const normalized = (username || '').trim().replace(/^@/, '').toLowerCase()
  return normalized || null
}

function parseIdentityFromInitData(initData: string): ExtractedIdentity {
  if (!initData) {
    return { userId: null, username: null }
  }

  try {
    const params = new URLSearchParams(initData)
    const userRaw = params.get('user')
    if (!userRaw) {
      return { userId: null, username: null }
    }

    let decoded = userRaw
    try {
      decoded = decodeURIComponent(userRaw)
    } catch {
      // userRaw уже декодирован
    }

    const user = JSON.parse(decoded) as { id?: number; username?: string }
    return {
      userId: typeof user.id === 'number' ? user.id : null,
      username: normalizeUsername(user.username),
    }
  } catch {
    return { userId: null, username: null }
  }
}

/**
 * Централизованное получение user_id один раз при открытии приложения
 * Используется для всех API запросов в течение сессии
 */
export function UserProvider({ children }: { children: ReactNode }) {
  const [userId, setUserId] = useState<number | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [manualUsername, setManualUsernameState] = useState<string | null>(null)

  const setManualUsername = (username: string | null) => {
    const cleaned = normalizeUsername(username)
    if (!cleaned) {
      localStorage.removeItem('manual_username')
      setManualUsernameState(null)
      return
    }
    localStorage.setItem('manual_username', cleaned)
    setManualUsernameState(cleaned)
  }

  const extractIdentity = async (): Promise<ExtractedIdentity> => {
    const tg = window.Telegram?.WebApp
    const initData = getInitDataAny()
    const unsafeUser = tg?.initDataUnsafe?.user

    // Способ 1: Из initDataUnsafe (самый надежный способ)
    if (unsafeUser?.id) {
      const id = unsafeUser.id
      const username = normalizeUsername(unsafeUser.username)
      console.log('[UserContext] Got identity from initDataUnsafe:', { id, username })
      return { userId: id, username }
    }

    // Способ 2: Локальный парсинг initData
    if (initData) {
      const localParsed = parseIdentityFromInitData(initData)
      if (localParsed.userId || localParsed.username) {
        console.log('[UserContext] Got identity from local initData parse:', localParsed)
        return localParsed
      }
    }

    // Способ 3: Из initData через backend API
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
          const id = typeof parsed.userId === 'number' ? parsed.userId : null
          const username = normalizeUsername(parsed.username)
          if (id || username) {
            console.log('[UserContext] Got identity from parse-init-data:', { id, username })
            return { userId: id, username }
          }
        } else {
          const errorText = await response.text()
          console.error('[UserContext] Error parsing initData:', response.status, errorText)
        }
      } catch (error) {
        console.error('[UserContext] Error parsing initData:', error)
      }
    }

    // Способ 4: Из localStorage (кэш с прошлой успешной сессии)
    const savedUserId = localStorage.getItem('telegram_user_id')
    const savedUsername = normalizeUsername(localStorage.getItem('manual_username'))
    let id: number | null = null
    if (savedUserId) {
      const parsedId = parseInt(savedUserId, 10)
      if (!isNaN(parsedId)) {
        id = parsedId
        console.log('[UserContext] Got user_id from localStorage:', id)
      }
    }

    return { userId: id, username: savedUsername }
  }

  const refreshUserId = async () => {
    setIsLoading(true)
    setError(null)

    try {
      // Telegram WebApp иногда инициализируется не сразу (особенно iOS/desktop) — делаем несколько попыток.
      const delays = [50, 200, 600, 1200, 2000]
      let identity: ExtractedIdentity = { userId: null, username: null }
      for (const d of delays) {
        await new Promise(resolve => setTimeout(resolve, d))
        const current = await extractIdentity()
        if (!identity.username && current.username) {
          identity.username = current.username
        }
        if (current.userId) {
          identity = current
          break
        }
      }

      if (identity.username) {
        setManualUsername(identity.username)
      }

      if (identity.userId) {
        setUserId(identity.userId)
        // Сохраняем в localStorage для fallback
        localStorage.setItem('telegram_user_id', identity.userId.toString())
        setError(null)
      } else {
        setUserId(null)
        if (!identity.username) {
          setError('user_id_not_found')
        } else {
          setError(null)
        }
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
    const savedManual = localStorage.getItem('manual_username')
    if (savedManual) {
      setManualUsernameState(savedManual)
    }
    refreshUserId()
  }, [])

  useEffect(() => {
    // Всплывающий fallback для браузерного режима без Telegram user_id
    if (!isLoading && !userId && !manualUsername && error === 'user_id_not_found') {
      const key = 'manual_username_prompted'
      if (sessionStorage.getItem(key) === '1') {
        return
      }
      sessionStorage.setItem(key, '1')

      const entered = window.prompt('Не удалось получить Telegram user_id. Введите ваш Telegram username (например, @username):')
      if (entered && entered.trim()) {
        setManualUsername(entered)
      }
    }
  }, [isLoading, userId, manualUsername, error])

  return (
    <UserContext.Provider value={{ userId, isLoading, error, refreshUserId, manualUsername, setManualUsername }}>
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
