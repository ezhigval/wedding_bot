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
    if (userIdLoading) {
      setIsLoading(true)
      return
    }

    setIsLoading(true)
    try {
      const attemptCheck = async (): Promise<RegistrationStatus> => {
        return checkRegistration(userId || 0, manualUsername || '')
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
  }

  useEffect(() => {
    if (userIdLoading) {
      setIsLoading(true)
      return
    }
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
