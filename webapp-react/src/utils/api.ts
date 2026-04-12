import type { Config, TimelineItem } from '../types'
import { getInitData } from './telegram'

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

function firstNonEmptyString(...values: Array<string | null | undefined>): string | undefined {
  for (const value of values) {
    if (typeof value !== 'string') {
      continue
    }
    const clean = value.trim()
    if (clean !== '' && clean !== 'undefined' && clean !== 'null') {
      return clean
    }
  }
  return undefined
}

function normalizeApiUrl(rawApiUrl: string | undefined): string {
  const fallback = DEFAULT_CONFIG.apiUrl
  const candidate = firstNonEmptyString(rawApiUrl)
  if (!candidate) {
    return fallback
  }

  try {
    const normalized = new URL(candidate, window.location.origin)
    return normalized.toString().replace(/\/+$/, '')
  } catch {
    return fallback
  }
}

export async function loadConfig(): Promise<Config> {
  if (cachedConfig) {
    return cachedConfig
  }

  let config: Config = { ...DEFAULT_CONFIG }

  try {
    const response = await fetch(`${DEFAULT_CONFIG.apiUrl}/config`)
    if (response.ok) {
      const data = await response.json()
      const mapped: Partial<Config> = {}
      const weddingDate = firstNonEmptyString(data.weddingDate, data.wedding_date)
      const groomName = firstNonEmptyString(data.groomName, data.groom_name)
      const brideName = firstNonEmptyString(data.brideName, data.bride_name)
      const groomTelegram = firstNonEmptyString(data.groomTelegram, data.groom_telegram)
      const brideTelegram = firstNonEmptyString(data.brideTelegram, data.bride_telegram)
      const weddingAddress = firstNonEmptyString(data.weddingAddress, data.wedding_address)

      if (weddingDate) {
        mapped.weddingDate = weddingDate
      }
      if (groomName) {
        mapped.groomName = groomName
      }
      if (brideName) {
        mapped.brideName = brideName
      }
      if (groomTelegram) {
        mapped.groomTelegram = groomTelegram
      }
      if (brideTelegram) {
        mapped.brideTelegram = brideTelegram
      }
      if (weddingAddress) {
        mapped.weddingAddress = weddingAddress
      }
      mapped.apiUrl = normalizeApiUrl(firstNonEmptyString(data.apiUrl, data.api_url))

      config = { ...DEFAULT_CONFIG, ...mapped }
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

export interface ApiAuth {
  userId?: number | null
  username?: string | null
  initData?: string
}

interface ResolvedApiAuth {
  userId: number
  username: string
  initData: string
}

function normalizeApiUsername(username?: string | null): string {
  return (username || '').trim().replace(/^@/, '').toLowerCase()
}

function readStoredUserId(): number {
  const sessionUserId = sessionStorage.getItem('telegram_user_id_session')
  if (sessionUserId) {
    const parsed = parseInt(sessionUserId, 10)
    if (!isNaN(parsed) && parsed > 0) {
      return parsed
    }
  }

  const persistentUserId = localStorage.getItem('telegram_user_id')
  if (persistentUserId) {
    const parsed = parseInt(persistentUserId, 10)
    if (!isNaN(parsed) && parsed > 0) {
      return parsed
    }
  }

  return 0
}

function readStoredUsername(): string {
  const sessionUsername = normalizeApiUsername(sessionStorage.getItem('telegram_username_session'))
  if (sessionUsername) {
    return sessionUsername
  }

  const manualUsername = normalizeApiUsername(localStorage.getItem('manual_username'))
  if (manualUsername) {
    return manualUsername
  }

  return normalizeApiUsername(localStorage.getItem('telegram_username'))
}

function resolveApiAuth(auth?: ApiAuth): ResolvedApiAuth {
  let userId = auth?.userId && auth.userId > 0 ? auth.userId : 0
  let username = normalizeApiUsername(auth?.username)

  if (!userId) {
    userId = readStoredUserId()
  }

  if (!username) {
    username = readStoredUsername()
  }

  return {
    userId,
    username,
    initData: auth?.initData ?? getInitData(),
  }
}

function buildAuthQuery(auth?: ApiAuth): URLSearchParams {
  const resolved = resolveApiAuth(auth)
  const params = new URLSearchParams()

  if (resolved.userId > 0) {
    params.set('userId', resolved.userId.toString())
  }

  if (resolved.username) {
    params.set('username', resolved.username)
  }

  return params
}

function withQuery(baseUrl: string, params: URLSearchParams): string {
  const query = params.toString()
  return query ? `${baseUrl}?${query}` : baseUrl
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
  },
  auth?: { initData?: string; username?: string }
): Promise<{ success: boolean; error?: string }> {
  if (!userId && !auth?.initData && !auth?.username) {
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
        initData: auth?.initData || '',
        username: auth?.username || '',
      }),
    })

    const contentType = response.headers.get('content-type') || ''
    if (!contentType.includes('application/json')) {
      return { success: false, error: 'Некорректный ответ сервера. Обновите страницу и попробуйте снова.' }
    }

    const data = await response.json()
    if (!response.ok || data?.success === false) {
      return { success: false, error: data.error || 'Ошибка регистрации' }
    }

    return { success: true }
  } catch (error) {
    console.error('Error submitting RSVP:', error)
    return { success: false, error: 'Ошибка сети' }
  }
}

export async function cancelInvitation(auth?: {
  userId?: number | null
  username?: string | null
  initData?: string
}): Promise<{ success: boolean; error?: string }> {
  const config = await loadConfig()
  try {
    const initData = auth?.initData ?? getInitData()
    const effectiveUserId = auth?.userId && auth.userId > 0 ? auth.userId : 0
    const effectiveUsername = (auth?.username || '').trim().replace(/^@/, '').toLowerCase()

    const response = await fetch(`${config.apiUrl}/cancel-registration`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        userId: effectiveUserId || 0,
        initData,
        username: effectiveUsername,
      }),
    })

    const contentType = response.headers.get('content-type') || ''
    if (!contentType.includes('application/json')) {
      return { success: false, error: 'Некорректный ответ сервера. Обновите страницу и попробуйте снова.' }
    }

    const data = await response.json()
    if (!response.ok || data?.success === false) {
      return { success: false, error: data.error || 'Ошибка отмены' }
    }

    return { success: true }
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
export async function checkRegistration(userId: number, username?: string): Promise<RegistrationStatus> {
  const resolvedAuth = resolveApiAuth({ userId, username })
  const initData = resolvedAuth.initData
  const effectiveUserId = resolvedAuth.userId
  const effectiveUsername = resolvedAuth.username

  const config = await loadConfig()
  const checkUrl = `${config.apiUrl}/check-registration`
  
  try {
    const params = new URLSearchParams()
    if (effectiveUserId) {
      params.set('userId', effectiveUserId.toString())
    }
    if (effectiveUsername) {
      params.set('username', effectiveUsername)
    }
    const url = `${checkUrl}?${params.toString()}`
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        userId: effectiveUserId || 0,
        username: effectiveUsername,
        initData,
      }),
    })
    
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
  wordle_score: number
  rank: string
}

export async function getGameStats(auth?: ApiAuth): Promise<GameStats> {
  const config = await loadConfig()
  const resolvedAuth = resolveApiAuth(auth)
  try {
    const response = await fetch(withQuery(`${config.apiUrl}/game-stats`, buildAuthQuery(auth)))
    if (response.ok) {
      const data = await response.json()
      return data
    }
  } catch (error) {
    console.error('Error loading game stats:', error)
  }
  // Возвращаем дефолтные значения
  return {
    user_id: resolvedAuth.userId,
    first_name: '',
    last_name: '',
    total_score: 0,
    dragon_score: 0,
    flappy_score: 0,
    crossword_score: 0,
    wordle_score: 0,
    rank: 'Незнакомец',
  }
}

export async function updateGameScore(
  auth: ApiAuth | undefined,
  gameType: 'dragon' | 'flappy' | 'crossword' | 'wordle',
  score: number
): Promise<{ success: boolean; stats?: GameStats; error?: string }> {
  const config = await loadConfig()
  const resolvedAuth = resolveApiAuth(auth)
  try {
    const response = await fetch(`${config.apiUrl}/update-game-score`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        userId: resolvedAuth.userId,
        username: resolvedAuth.username,
        gameType,
        score,
        initData: resolvedAuth.initData,
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

export async function getCrosswordData(auth?: ApiAuth): Promise<CrosswordData> {
  const config = await loadConfig()
  try {
    const response = await fetch(withQuery(`${config.apiUrl}/crossword/data`, buildAuthQuery(auth)))
    if (response.ok) {
      const data = await response.json()
      const normalized: CrosswordData = {
        ...data,
        words: data.words || [],
        guessed_words: data.guessed_words || [],
        start_date: data.start_date || data.crossword_start_date || '',
      }
      console.log('Crossword data loaded:', { 
        wordsCount: normalized.words?.length || 0, 
        crosswordIndex: normalized.crossword_index,
        guessedWords: normalized.guessed_words?.length || 0 
      })
      return normalized
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
export async function getWordleWord(auth?: ApiAuth): Promise<string | null> {
  const config = await loadConfig()
  try {
    const response = await fetch(withQuery(`${config.apiUrl}/wordle/word`, buildAuthQuery(auth)))
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

export async function getWordleState(auth?: ApiAuth): Promise<WordleState | null> {
  const config = await loadConfig()
  try {
    const response = await fetch(withQuery(`${config.apiUrl}/wordle/state`, buildAuthQuery(auth)))
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
  auth: ApiAuth | undefined,
  currentWord: string,
  attempts: Array<Array<{ letter: string; state: string }>>,
  lastWordDate: string,
  currentGuess: string = ''
): Promise<{ success: boolean; error?: string }> {
  const config = await loadConfig()
  const resolvedAuth = resolveApiAuth(auth)
  try {
    const response = await fetch(`${config.apiUrl}/wordle/state`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        userId: resolvedAuth.userId,
        username: resolvedAuth.username,
        current_word: currentWord,
        attempts,
        current_guess: currentGuess,
        last_word_date: lastWordDate,
        initData: resolvedAuth.initData,
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
export async function getWordleProgress(auth?: ApiAuth): Promise<string[]> {
  const config = await loadConfig()
  try {
    const response = await fetch(withQuery(`${config.apiUrl}/wordle/progress`, buildAuthQuery(auth)))
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
  auth: ApiAuth | undefined,
  word: string
): Promise<{ success: boolean; message?: string; points?: number; already_guessed?: boolean }> {
  const config = await loadConfig()
  const resolvedAuth = resolveApiAuth(auth)
  try {
    const response = await fetch(`${config.apiUrl}/wordle/guess`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        word,
        userId: resolvedAuth.userId,
        username: resolvedAuth.username,
        initData: resolvedAuth.initData,
      }),
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
  auth: ApiAuth | undefined,
  guessedWords: string[],
  crosswordIndex: number = 0,
  cellLetters?: { [key: string]: string },
  wrongAttempts?: string[],
  startDate?: string
): Promise<{ success: boolean; error?: string }> {
  const config = await loadConfig()
  const resolvedAuth = resolveApiAuth(auth)
  try {
    const response = await fetch(`${config.apiUrl}/crossword/progress`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        userId: resolvedAuth.userId,
        username: resolvedAuth.username,
        guessed_words: guessedWords,
        crossword_index: crosswordIndex,
        cell_letters: cellLetters,
        wrong_attempts: wrongAttempts,
        crossword_start_date: startDate,
        initData: resolvedAuth.initData,
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
