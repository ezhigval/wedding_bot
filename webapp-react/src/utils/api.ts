import type { Config, TimelineItem } from '../types'

const DEFAULT_CONFIG: Config = {
  weddingDate: '2026-06-05',
  groomName: 'Валентин',
  brideName: 'Мария',
  groomTelegram: 'ezhigval',
  brideTelegram: '',
  weddingAddress: 'Ресторан Марсала, Большой проспект Петроградской стороны, 84, Санкт-Петербург',
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

/**
 * Отправляет форму RSVP
 * @param userId - user_id из UserContext (централизованно получен при открытии приложения)
 * @param formData - данные формы
 */
export async function submitRSVP(
  userId: number,
  formData: {
    lastName: string
    firstName: string
    category: string
    side: string
    guests: Array<{ firstName: string; lastName: string; telegram?: string }>
  }
): Promise<{ success: boolean; error?: string }> {
  if (!userId) {
    return { success: false, error: 'user_id required' }
  }

  const config = await loadConfig()
  try {
    const response = await fetch(`${config.apiUrl}/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        ...formData,
        userId,
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

/**
 * Проверяет регистрацию пользователя
 * @param userId - user_id из UserContext (централизованно получен при открытии приложения)
 */
export async function checkRegistration(userId: number): Promise<RegistrationStatus> {
  if (!userId) {
    return { registered: false, error: 'no_user_id' }
  }

  const config = await loadConfig()
  const checkUrl = `${config.apiUrl}/check-registration`
  
  try {
    const params = new URLSearchParams({
      userId: userId.toString(),
    })
    const url = `${checkUrl}?${params.toString()}`
    const response = await fetch(url, { method: 'POST' })
    
    if (response.ok) {
      const data = await response.json()
      return data
    } else {
      const errorText = await response.text()
      console.error('[checkRegistration] Failed:', response.status, errorText)
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
  cell_letters?: { [key: string]: string }
  crossword_index?: number
  start_date?: string
  wrong_attempts?: string[] // Для обратной совместимости
  wrong_words?: string[] // Неправильные слова после завершения
}

export async function getCrosswordData(userId: number): Promise<CrosswordData> {
  const config = await loadConfig()
  try {
    const response = await fetch(`${config.apiUrl}/crossword-data?userId=${userId}`)
    if (response.ok) {
      const data = await response.json()
      console.log('Crossword data loaded:', { 
        wordsCount: data.words?.length || 0, 
        crosswordIndex: data.crossword_index,
        guessedWords: data.guessed_words?.length || 0 
      })
      return data
    } else {
      const errorData = await response.json().catch(() => ({}))
      console.error('Error loading crossword data:', response.status, errorData)
    }
  } catch (error) {
    console.error('Error loading crossword data:', error)
  }
  return { words: [], guessed_words: [] }
}

/**
 * Получает слово для Wordle
 * @param userId - user_id из UserContext
 */
export async function getWordleWord(userId: number): Promise<string | null> {
  if (!userId) {
    return null
  }

  const config = await loadConfig()
  try {
    const response = await fetch(`${config.apiUrl}/wordle/word?userId=${userId}`)
    if (response.ok) {
      const data = await response.json()
      return data.word || null
    }
  } catch (error) {
    console.error('Error loading Wordle word:', error)
  }
  return null
}

export interface WordleState {
  current_word: string | null
  attempts: Array<Array<{ letter: string; state: string }>>
  current_guess?: string
  last_word_date: string | null
}

export async function getWordleState(userId: number): Promise<WordleState | null> {
  const config = await loadConfig()
  try {
    const response = await fetch(`${config.apiUrl}/wordle/state?userId=${userId}`)
    if (response.ok) {
      const data = await response.json()
      return data
    }
  } catch (error) {
    console.error('Error loading Wordle state:', error)
  }
  return null
}

export async function saveWordleState(
  userId: number,
  currentWord: string,
  attempts: Array<Array<{ letter: string; state: string }>>,
  lastWordDate: string,
  currentGuess: string = ''
): Promise<{ success: boolean; error?: string }> {
  const config = await loadConfig()
  try {
    const response = await fetch(`${config.apiUrl}/wordle/state`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        userId,
        current_word: currentWord,
        attempts,
        current_guess: currentGuess,
        last_word_date: lastWordDate,
      }),
    })

    if (response.ok) {
      const data = await response.json()
      return { success: data.success || false }
    } else {
      const data = await response.json()
      return { success: false, error: data.error || 'Ошибка сохранения состояния' }
    }
  } catch (error) {
    console.error('Error saving Wordle state:', error)
    return { success: false, error: 'Ошибка сети' }
  }
}

/**
 * Получает прогресс в Wordle
 * @param userId - user_id из UserContext
 */
export async function getWordleProgress(userId: number): Promise<string[]> {
  if (!userId) {
    return []
  }

  const config = await loadConfig()
  try {
    const response = await fetch(`${config.apiUrl}/wordle/progress?userId=${userId}`)
    if (response.ok) {
      const data = await response.json()
      return data.guessed_words || []
    }
  } catch (error) {
    console.error('Error loading Wordle progress:', error)
  }
  return []
}

/**
 * Отправляет отгаданное слово в Wordle
 * @param userId - user_id из UserContext
 * @param word - отгаданное слово
 */
export async function submitWordleGuess(
  userId: number,
  word: string
): Promise<{ success: boolean; message?: string; points?: number; already_guessed?: boolean }> {
  if (!userId) {
    return { success: false, message: 'Не удалось получить данные пользователя' }
  }

  const config = await loadConfig()
  try {
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
  crosswordIndex: number = 0,
  cellLetters?: { [key: string]: string },
  wrongAttempts?: string[],
  startDate?: string
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
        cell_letters: cellLetters,
        wrong_attempts: wrongAttempts,
        start_date: startDate,
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
