import { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { checkRegistration, type RegistrationStatus } from '../utils/api'
import { useUser } from './UserContext'

interface RegistrationContextType {
  isRegistered: boolean
  isLoading: boolean
  refreshRegistration: () => Promise<void>
}

const RegistrationContext = createContext<RegistrationContextType | undefined>(undefined)

export function RegistrationProvider({ children }: { children: ReactNode }) {
  const [isRegistered, setIsRegistered] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const { userId, manualUsername, isLoading: userIdLoading } = useUser()

  const refreshRegistration = async () => {
    setIsLoading(true)
    try {
      // Проверяем по user_id и/или username.
      // Это важно, потому что в колонке F может храниться как user_id, так и username.
      if (!userIdLoading && (userId || manualUsername)) {
        const status: RegistrationStatus = await checkRegistration(userId || 0, manualUsername || '')
        setIsRegistered(status.registered || false)
        return
      }

      // Нет идентификатора — не зависаем в Loading, просто считаем незарегистрированным
      setIsRegistered(false)
    } catch (error) {
      console.error('Error checking registration:', error)
      setIsRegistered(false)
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    refreshRegistration()
  }, [userId, userIdLoading, manualUsername])

  return (
    <RegistrationContext.Provider value={{ isRegistered, isLoading, refreshRegistration }}>
      {children}
    </RegistrationContext.Provider>
  )
}

export function useRegistration() {
  const context = useContext(RegistrationContext)
  if (context === undefined) {
    throw new Error('useRegistration must be used within a RegistrationProvider')
  }
  return context
}
