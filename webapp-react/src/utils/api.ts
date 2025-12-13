import type { Config, TimelineItem } from '../types'

const DEFAULT_CONFIG: Config = {
  weddingDate: '2026-06-05',
  groomName: 'Валентин',
  brideName: 'Мария',
  groomTelegram: 'ezhigval',
  brideTelegram: '',
  weddingAddress: 'Санкт-Петербург',
  apiUrl: window.location.origin + '/api',
}

let cachedConfig: Config | null = null

export async function loadConfig(): Promise<Config> {
  if (cachedConfig) {
    return cachedConfig
  }

  let config: Config = { ...DEFAULT_CONFIG }

  try {
    const response = await fetch(`${DEFAULT_CONFIG.apiUrl}/config`)
    if (response.ok) {
      const data = await response.json()
      config = { ...DEFAULT_CONFIG, ...data }
    }
  } catch (error) {
    console.log('Используем конфигурацию по умолчанию:', error)
  }

  cachedConfig = config
  return config
}

export async function loadTimeline(): Promise<TimelineItem[]> {
  const config = await loadConfig()
  try {
    const response = await fetch(`${config.apiUrl}/timeline`)
    if (response.ok) {
      const data = await response.json()
      return data.timeline || []
    }
  } catch (error) {
    console.error('Error loading timeline:', error)
  }
  return []
}

export async function submitRSVP(formData: {
  lastName: string
  firstName: string
  category: string
  side: string
  guests: Array<{ firstName: string; lastName: string; telegram?: string }>
}): Promise<{ success: boolean; error?: string }> {
  const config = await loadConfig()
  try {
    // Получаем user_id из Telegram WebApp
    const tg = window.Telegram?.WebApp
    const initData = (tg as any)?.initData || ''
    
    const response = await fetch(`${config.apiUrl}/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        ...formData,
        initData, // Telegram WebApp initData для проверки подлинности
      }),
    })

    if (response.ok) {
      return { success: true }
    } else {
      const data = await response.json()
      return { success: false, error: data.error || 'Ошибка регистрации' }
    }
  } catch (error) {
    console.error('Error submitting RSVP:', error)
    return { success: false, error: 'Ошибка сети' }
  }
}

export async function cancelInvitation(): Promise<{ success: boolean; error?: string }> {
  const config = await loadConfig()
  try {
    const response = await fetch(`${config.apiUrl}/cancel`, {
      method: 'POST',
    })

    if (response.ok) {
      return { success: true }
    } else {
      const data = await response.json()
      return { success: false, error: data.error || 'Ошибка отмены' }
    }
  } catch (error) {
    console.error('Error canceling invitation:', error)
    return { success: false, error: 'Ошибка сети' }
  }
}

