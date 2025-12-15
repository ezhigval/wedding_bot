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

export interface RegistrationStatus {
  registered: boolean
  in_group_chat?: boolean
  error?: string
}

export async function checkRegistration(): Promise<RegistrationStatus> {
  const config = await loadConfig()
  try {
    // ВРЕМЕННАЯ СИМУЛЯЦИЯ ДЛЯ ТЕСТА - УДАЛИТЬ ПОСЛЕ ПРОВЕРКИ
    // Симулируем user_id = 1034074077 для локального тестирования
    const TEST_USER_ID = 1034074077
    const isLocalhost = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
    if (isLocalhost) {
      console.log('[TEST MODE] Simulating user_id:', TEST_USER_ID)
      // Прямо проверяем регистрацию с тестовым user_id
      const params = new URLSearchParams({
        userId: TEST_USER_ID.toString(),
        firstName: '',
        lastName: '',
      })
      const url = `${config.apiUrl}/check?${params}`
      const response = await fetch(url)
      
      if (response.ok) {
        const data = await response.json()
        if (data.registered) {
          localStorage.setItem('telegram_user_id', TEST_USER_ID.toString())
        }
        return data
      }
    }
    // КОНЕЦ ВРЕМЕННОЙ СИМУЛЯЦИИ
    
    // Получаем данные пользователя из Telegram WebApp
    const tg = window.Telegram?.WebApp
    const initData = (tg as any)?.initData || ''
    
    // Пытаемся получить userId из Telegram WebApp
    let userId: number | null = null
    let firstName = ''
    let lastName = ''
    
    if (tg) {
      // Способ 1: Из initData (если доступен)
      if (initData) {
        try {
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
              userId = parsed.userId
              firstName = parsed.firstName || ''
              lastName = parsed.lastName || ''
              // Сохраняем в localStorage
              if (userId !== null) {
                localStorage.setItem('telegram_user_id', userId.toString())
              }
            }
          }
        } catch (error) {
          console.error('Error parsing initData:', error)
        }
      }
      
      // Способ 2: Из localStorage (если был сохранен ранее)
      if (!userId) {
        const savedUserId = localStorage.getItem('telegram_user_id')
        if (savedUserId) {
          userId = parseInt(savedUserId)
        }
      }
      
      // Способ 3: Из Telegram WebApp (если доступен напрямую)
      if (!userId && (tg as any).initDataUnsafe?.user) {
        userId = (tg as any).initDataUnsafe.user.id
        firstName = (tg as any).initDataUnsafe.user.first_name || ''
        lastName = (tg as any).initDataUnsafe.user.last_name || ''
      }
    }
    
    // Если нет userId, но есть имя/фамилия - пробуем поиск только по имени
    if (!userId && firstName && lastName) {
      try {
        const params = new URLSearchParams({
          firstName,
          lastName,
          searchByNameOnly: 'true',
        })
        const url = `${config.apiUrl}/check?${params}`
        const response = await fetch(url)
        
        if (response.ok) {
          const data = await response.json()
          return data
        }
      } catch (error) {
        console.error('Error checking by name only:', error)
      }
    }
    
    // Если нет userId вообще - возвращаем ошибку
    if (!userId) {
      return { registered: false, error: 'no_user_id' }
    }
    
    // Если userId есть - проверяем регистрацию
    if (userId === null) {
      return { registered: false, error: 'no_user_id' }
    }
    
    const params = new URLSearchParams({
      userId: userId.toString(),
      firstName,
      lastName,
    })
    const url = `${config.apiUrl}/check?${params}`
    const response = await fetch(url)
    
    if (response.ok) {
      const data = await response.json()
      
      // Если регистрация успешна, сохраняем userId в localStorage
      if (data.registered && userId !== null) {
        localStorage.setItem('telegram_user_id', userId.toString())
      }
      
      return data
    } else {
      return { registered: false, error: 'server_error' }
    }
  } catch (error) {
    console.error('Error checking registration:', error)
        return { registered: false, error: 'network_error' }
  }
}

export interface GameStats {
  user_id: number
  first_name?: string
  last_name?: string
  total_score: number
  dragon_score: number
  flappy_score: number
  crossword_score: number
  rank: string
}

export async function getGameStats(userId: number): Promise<GameStats> {
  const config = await loadConfig()
  try {
    const response = await fetch(`${config.apiUrl}/game-stats?userId=${userId}`)
    if (response.ok) {
      const data = await response.json()
      return data
    }
  } catch (error) {
    console.error('Error loading game stats:', error)
  }
  // Возвращаем дефолтные значения
  return {
    user_id: userId,
    first_name: '',
    last_name: '',
    total_score: 0,
    dragon_score: 0,
    flappy_score: 0,
    crossword_score: 0,
    rank: 'Незнакомец',
  }
}

export async function updateGameScore(
  userId: number,
  gameType: 'dragon' | 'flappy' | 'crossword' | 'wordle',
  score: number
): Promise<{ success: boolean; stats?: GameStats; error?: string }> {
  const config = await loadConfig()
  try {
    const response = await fetch(`${config.apiUrl}/game-score`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        userId,
        gameType,
        score,
      }),
    })

    if (response.ok) {
      const data = await response.json()
      return { success: true, stats: data.stats }
    } else {
      const data = await response.json()
      return { success: false, error: data.error || 'Ошибка обновления счета' }
    }
  } catch (error) {
    console.error('Error updating game score:', error)
    return { success: false, error: 'Ошибка сети' }
  }
}

export interface CrosswordWord {
  word: string
  description: string
}

export interface CrosswordData {
  words: CrosswordWord[]
  guessed_words: string[]
  crossword_index?: number
}

export async function getCrosswordData(userId: number): Promise<CrosswordData> {
  const config = await loadConfig()
  try {
    const response = await fetch(`${config.apiUrl}/crossword-data?userId=${userId}`)
    if (response.ok) {
      const data = await response.json()
      return data
    }
  } catch (error) {
    console.error('Error loading crossword data:', error)
  }
  return { words: [], guessed_words: [] }
}

export async function getWordleWord(): Promise<string | null> {
  const config = await loadConfig()
  try {
    const response = await fetch(`${config.apiUrl}/wordle/word`)
    if (response.ok) {
      const data = await response.json()
      return data.word || null
    }
  } catch (error) {
    console.error('Error loading Wordle word:', error)
  }
  return null
}

export async function getWordleProgress(): Promise<string[]> {
  const config = await loadConfig()
  try {
    const tg = window.Telegram?.WebApp
    let initData = (tg as any)?.initData || ''
    
    // Если initData нет, пытаемся использовать userId из localStorage
    if (!initData) {
      const savedUserId = localStorage.getItem('telegram_user_id')
      const isLocalhost = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
      
      if (isLocalhost && savedUserId) {
        const response = await fetch(`${config.apiUrl}/wordle/progress?userId=${savedUserId}`)
        if (response.ok) {
          const data = await response.json()
          return data.guessed_words || []
        }
      }
      return []
    }
    
    const response = await fetch(`${config.apiUrl}/wordle/progress?initData=${encodeURIComponent(initData)}`)
    if (response.ok) {
      const data = await response.json()
      return data.guessed_words || []
    }
  } catch (error) {
    console.error('Error loading Wordle progress:', error)
  }
  return []
}

export async function submitWordleGuess(word: string): Promise<{ success: boolean; message?: string; points?: number; already_guessed?: boolean }> {
  const config = await loadConfig()
  try {
    const tg = window.Telegram?.WebApp
    const initData = (tg as any)?.initData || ''
    let userId: number | null = null
    
    // Сначала пытаемся получить userId из initData
    if (initData) {
      try {
        const response = await fetch(`${config.apiUrl}/parse-init-data`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ initData }),
        })
        if (response.ok) {
          const parsed = await response.json()
          userId = parsed.userId
        }
      } catch (e) {
        console.error('Error parsing initData:', e)
      }
    }
    
    // Если не получилось, пытаемся использовать userId из localStorage
    if (!userId) {
      const savedUserId = localStorage.getItem('telegram_user_id')
      if (savedUserId) {
        userId = parseInt(savedUserId)
      }
    }
    
    if (!userId) {
      return { success: false, message: 'Не удалось получить данные пользователя' }
    }
    
    // Отправляем запрос с userId
    const response = await fetch(`${config.apiUrl}/wordle/guess`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ word, userId }),
    })
    
    if (response.ok) {
      const data = await response.json()
      return data
    } else {
      const error = await response.json()
      return { success: false, message: error.error || 'Ошибка' }
    }
  } catch (error) {
    console.error('Error submitting Wordle guess:', error)
    return { success: false, message: 'Ошибка сети' }
  }
}

export async function saveCrosswordProgress(
  userId: number,
  guessedWords: string[],
  crosswordIndex: number = 0
): Promise<{ success: boolean; error?: string }> {
  const config = await loadConfig()
  try {
    const response = await fetch(`${config.apiUrl}/crossword-progress`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        userId,
        guessedWords,
        crossword_index: crosswordIndex,
      }),
    })

    if (response.ok) {
      const data = await response.json()
      return { success: data.success || false }
    } else {
      const data = await response.json()
      return { success: false, error: data.error || 'Ошибка сохранения прогресса' }
    }
  } catch (error) {
    console.error('Error saving crossword progress:', error)
    return { success: false, error: 'Ошибка сети' }
  }
}

