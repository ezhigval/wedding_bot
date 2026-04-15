import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react'
import { checkRegistration, type RegistrationStatus } from '../utils/api'
import { useUser } from './UserContext'

interface RegistrationRefreshOptions {
  userId?: number | null
  username?: string | null
}

interface RegistrationContextType {
  isRegistered: boolean
  isLoading: boolean
  refreshRegistration: (options?: RegistrationRefreshOptions) => Promise<void>
}

const RegistrationContext = createContext<RegistrationContextType | undefined>(undefined)

function normalizeRegistrationUsername(username?: string | null): string {
  return (username || '').trim().replace(/^@/, '').toLowerCase()
}

export function RegistrationProvider({ children }: { children: ReactNode }) {
  const [isRegistered, setIsRegistered] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const { userId, manualUsername, isLoading: userIdLoading } = useUser()

  const refreshRegistration = useCallback(async (options?: RegistrationRefreshOptions) => {
    if (userIdLoading) {
      setIsLoading(true)
      return
    }

    setIsLoading(true)
    try {
      const effectiveUserId =
        typeof options?.userId === 'number' && options.userId > 0 ? options.userId : userId || 0
      const effectiveUsername = normalizeRegistrationUsername(options?.username) || manualUsername || ''

      const attemptCheck = async (): Promise<RegistrationStatus> => {
        return checkRegistration(effectiveUserId, effectiveUsername)
      }

      let status: RegistrationStatus = await attemptCheck()

      // Короткий повторный запрос при временном сбое.
      if (!status.registered && (status.error === 'network_error' || status.error === 'server_error')) {
        await new Promise((resolve) => setTimeout(resolve, 400))
        status = await attemptCheck()
      }

      setIsRegistered(status.registered || false)
    } catch (error) {
      console.error('Error checking registration:', error)
      setIsRegistered(false)
    } finally {
      setIsLoading(false)
    }
  }, [manualUsername, userId, userIdLoading])

  useEffect(() => {
    void refreshRegistration()
  }, [refreshRegistration])

  return (
    <RegistrationContext.Provider value={{ isRegistered, isLoading, refreshRegistration }}>
      {children}
    </RegistrationContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useRegistration() {
  const context = useContext(RegistrationContext)
  if (context === undefined) {
    throw new Error('useRegistration must be used within a RegistrationProvider')
  }
  return context
}
