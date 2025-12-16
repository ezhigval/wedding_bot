import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import App from './App'
import './index.css'

// Создаем QueryClient для React Query
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
      staleTime: 5 * 60 * 1000, // 5 минут
    },
  },
})

// Инициализация Telegram Web App
declare global {
  interface Window {
    Telegram: {
      WebApp: {
        ready: () => void
        expand: () => void
        initData?: string
        initDataUnsafe?: {
          user?: {
            id: number
            first_name?: string
            last_name?: string
            username?: string
          }
        }
        HapticFeedback?: {
          impactOccurred: (style: 'light' | 'medium' | 'heavy' | 'rigid' | 'soft') => void
          notificationOccurred: (type: 'error' | 'success' | 'warning') => void
        }
        colorScheme: 'light' | 'dark'
        showAlert: (message: string) => void
        showConfirm: (message: string) => Promise<boolean>
      }
    }
  }
}

// Инициализация Telegram Web App (или симуляция для локальной разработки)
const tg = window.Telegram?.WebApp
const isLocalhost = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'

if (tg) {
  tg.ready()
  tg.expand()
} else if (isLocalhost) {
  // Симулируем Telegram WebApp для локальной разработки
  console.log('[LOCAL DEV] Simulating Telegram WebApp')
  window.Telegram = {
    WebApp: {
      ready: () => console.log('[LOCAL DEV] WebApp ready'),
      expand: () => console.log('[LOCAL DEV] WebApp expanded'),
      initData: '',
      colorScheme: 'light' as const,
      showAlert: (message: string) => alert(message),
      showConfirm: (message: string) => Promise.resolve(window.confirm(message)),
    }
  }
  // Сохраняем статичный user_id для локальной разработки
  localStorage.setItem('telegram_user_id', '1034074077')
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>,
)

