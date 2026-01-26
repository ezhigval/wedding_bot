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
  const { userId, isLoading: userIdLoading } = useUser()

  const refreshRegistration = async () => {
    // Ждем, пока user_id будет получен
    if (userIdLoading || !userId) {
      return
    }

    setIsLoading(true)
    try {
      const status: RegistrationStatus = await checkRegistration(userId)
      setIsRegistered(status.registered || false)
    } catch (error) {
      console.error('Error checking registration:', error)
      setIsRegistered(false)
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    refreshRegistration()
  }, [userId, userIdLoading])

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

