import { useState, useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { getWordleWord, getWordleProgress, submitWordleGuess, getWordleState, saveWordleState, loadConfig } from '../../utils/api'
import { hapticFeedback } from '../../utils/telegram'
import Confetti from '../common/Confetti'
import type { Config } from '../../types'

interface WordleGameProps {
  onScore?: (score: number) => void
  onClose: () => void
}

const WORD_LENGTH = 5
const MAX_ATTEMPTS = 6

// Резервный список слов на случай, если API недоступен
const FALLBACK_WORDS = [
  'ГОСТИ', 'ТАНЕЦ', 'БУКЕТ', 'ЖЕНИХ', 'ВИДЕО',
  'СЕМЬЯ', 'ВЕНЕЦ', 'БРАК', 'СОЮЗ', 'ПАРА'
].filter(word => word.length === 5)

type LetterState = 'empty' | 'correct' | 'present' | 'absent'

interface Cell {
  letter: string
  state: LetterState
}

export default function WordleGame({ onScore, onClose }: WordleGameProps) {
  const [targetWord, setTargetWord] = useState<string>('')
  const [guesses, setGuesses] = useState<Cell[][]>([])
  const [currentGuess, setCurrentGuess] = useState<string>('')
  const [gameOver, setGameOver] = useState<'win' | 'lose' | null>(null)
  const [score, setScore] = useState(0)
  const [usedLetters, setUsedLetters] = useState<Map<string, LetterState>>(new Map())
  const [loading, setLoading] = useState(true)
  const [guessedWords, setGuessedWords] = useState<string[]>([])
  const [alreadyGuessed, setAlreadyGuessed] = useState(false)
  const [showConfetti, setShowConfetti] = useState(false)
  const [userId, setUserId] = useState<number | null>(null)
  const [config, setConfig] = useState<Config | null>(null)
  const [lastWordDate, setLastWordDate] = useState<string>('')
  const [timeUntilNextWord, setTimeUntilNextWord] = useState<{ hours: number; minutes: number; seconds: number } | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const saveStateTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const timerIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // Русская раскладка клавиатуры ЙЦУКЕН (как в кроссворде)
  const russianLetters = [
    'Й', 'Ц', 'У', 'К', 'Е', 'Н', 'Г', 'Ш', 'Щ', 'З', 'Х', 'Ъ',
    'Ф', 'Ы', 'В', 'А', 'П', 'Р', 'О', 'Л', 'Д', 'Ж', 'Э',
    'Я', 'Ч', 'С', 'М', 'И', 'Т', 'Ь', 'Б', 'Ю', 'Ё'
  ]

  useEffect(() => {
    loadConfig().then(setConfig)
  }, [])

  useEffect(() => {
    if (!config) return

    const loadGame = async () => {
      setLoading(true)
      try {
        // Получаем userId
        const tg = window.Telegram?.WebApp
        const initData = (tg as any)?.initData || ''
        let currentUserId: number | null = null

        if (initData) {
          try {
            const response = await fetch(`${config.apiUrl}/parse-init-data`, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ initData }),
            })
            if (response.ok) {
              const parsed = await response.json()
              currentUserId = parsed.userId
            }
          } catch (e) {
            console.error('Error parsing initData:', e)
          }
        }

        if (!currentUserId) {
          const savedUserId = localStorage.getItem('telegram_user_id')
          if (savedUserId) {
            currentUserId = parseInt(savedUserId)
          }
        }

        if (!currentUserId) {
          console.error('Cannot load Wordle: userId not found')
          setLoading(false)
          return
        }

        setUserId(currentUserId)

        // Загружаем состояние игры
        const state = await getWordleState(currentUserId)
        const progress = await getWordleProgress()
        
        if (state && state.current_word && state.attempts && state.last_word_date) {
          // Восстанавливаем состояние из сохраненного
          const word = state.current_word.toUpperCase()
          setTargetWord(word)
          setLastWordDate(state.last_word_date)
          setGuessedWords(progress.map(w => w.toUpperCase()))
          
          // Восстанавливаем попытки
          if (state.attempts && state.attempts.length > 0) {
            const restoredGuesses: Cell[][] = state.attempts.map((attempt: any[]) => 
              attempt.map((cell: any) => ({
                letter: cell.letter || '',
                state: (cell.state || 'empty') as LetterState
              }))
            )
            
            // Заполняем пустые попытки
            while (restoredGuesses.length < MAX_ATTEMPTS) {
              restoredGuesses.push(Array(WORD_LENGTH).fill(null).map(() => ({ letter: '', state: 'empty' })))
            }
            
            setGuesses(restoredGuesses)
            
            // Восстанавливаем текущий ввод (current_guess)
            if (state.current_guess) {
              setCurrentGuess(state.current_guess.toUpperCase())
            } else {
              // Если нет сохраненного current_guess, определяем из первой пустой попытки
              const currentAttemptIndex = restoredGuesses.findIndex(row => row[0].state === 'empty' || !row[0].letter)
              if (currentAttemptIndex !== -1) {
                const currentAttempt = restoredGuesses[currentAttemptIndex]
                const currentGuessStr = currentAttempt.map(c => c.letter).join('')
                setCurrentGuess(currentGuessStr)
              }
            }
            
            // Проверяем, не закончена ли игра
            const hasWin = restoredGuesses.some(row => row.every(cell => cell.state === 'correct'))
            const hasLose = restoredGuesses.every(row => row[0].letter !== '' && !row.every(cell => cell.state === 'correct'))
            
            if (hasWin) {
              setGameOver('win')
            } else if (hasLose && restoredGuesses.filter(row => row[0].letter !== '').length === MAX_ATTEMPTS) {
              setGameOver('lose')
            }
          } else {
            // Нет сохраненных попыток - начинаем заново
            setGuesses(Array(MAX_ATTEMPTS).fill(null).map(() => 
              Array(WORD_LENGTH).fill(null).map(() => ({ letter: '', state: 'empty' }))
            ))
          }
          
          // Проверяем, не отгадано ли уже это слово
          if (progress.map(w => w.toUpperCase()).includes(word)) {
            setAlreadyGuessed(true)
            setGameOver('win') // Фиксируем победу, чтобы не стиралось поле
          }
          
          // Запускаем таймер обратного отсчета
          startCountdownTimer(state.last_word_date)
        } else {
          // Нет сохраненного состояния - загружаем новое слово
          const word = await getWordleWord(currentUserId)
          
          if (word) {
            setTargetWord(word.toUpperCase())
            setGuessedWords(progress.map(w => w.toUpperCase()))
            
            if (progress.map(w => w.toUpperCase()).includes(word.toUpperCase())) {
              setAlreadyGuessed(true)
            }
          } else {
            // Fallback
            const randomWord = FALLBACK_WORDS[Math.floor(Math.random() * FALLBACK_WORDS.length)].toUpperCase()
            setTargetWord(randomWord)
          }
          
          setGuesses(Array(MAX_ATTEMPTS).fill(null).map(() => 
            Array(WORD_LENGTH).fill(null).map(() => ({ letter: '', state: 'empty' }))
          ))
          
          // Получаем дату для нового слова
          const today = new Date().toISOString().split('T')[0]
          setLastWordDate(today)
          
          // Запускаем таймер обратного отсчета
          startCountdownTimer(today)
          
          // Сохраняем начальное состояние
          setTimeout(() => {
            if (currentUserId && word) {
              saveWordleState(currentUserId, word.toUpperCase(), [], today, '').catch(console.error)
            }
          }, 1000)
        }
      } catch (error) {
        console.error('Error loading Wordle:', error)
        // Fallback
        const randomWord = FALLBACK_WORDS[Math.floor(Math.random() * FALLBACK_WORDS.length)].toUpperCase()
        setTargetWord(randomWord)
        setGuesses(Array(MAX_ATTEMPTS).fill(null).map(() => 
          Array(WORD_LENGTH).fill(null).map(() => ({ letter: '', state: 'empty' }))
        ))
      } finally {
        setLoading(false)
        if (inputRef.current) {
          inputRef.current.focus()
        }
      }
    }
    
    loadGame()
  }, [config])

  // Используем ref для предотвращения двойных нажатий
  const lastKeyPressTime = useRef<number>(0)
  const lastKey = useRef<string>('')
  
  const handleKeyPress = (key: string) => {
    if (gameOver) return
    
    // Защита от двойных нажатий (debounce 150ms)
    const now = Date.now()
    if (now - lastKeyPressTime.current < 150 && lastKey.current === key) {
      return
    }
    lastKeyPressTime.current = now
    lastKey.current = key
    
    hapticFeedback('light')
    
    if (key === 'ENTER') {
      handleSubmit()
    } else if (key === 'BACKSPACE') {
      setCurrentGuess(prev => {
        const newGuess = prev.slice(0, -1)
        // Сохраняем состояние после изменения
        setTimeout(() => saveGameState(), 500)
        return newGuess
      })
    } else if (currentGuess.length < WORD_LENGTH && /[А-ЯЁ]/.test(key)) {
      setCurrentGuess(prev => {
        const newGuess = prev + key.toUpperCase()
        // Сохраняем состояние после изменения
        setTimeout(() => saveGameState(), 500)
        return newGuess
      })
    }
  }

  // Функция сохранения состояния игры
  const saveGameState = async () => {
    if (!userId || !targetWord || !lastWordDate) return
    
    // Отменяем предыдущий таймер, если он есть
    if (saveStateTimeoutRef.current) {
      clearTimeout(saveStateTimeoutRef.current)
    }
    
    // Сохраняем с небольшой задержкой, чтобы не спамить запросами
    saveStateTimeoutRef.current = setTimeout(async () => {
      try {
        // Преобразуем guesses в формат для сохранения (только отправленные попытки)
        const attemptsToSave = guesses
          .filter(row => row.some(cell => cell.state !== 'empty' && cell.letter !== ''))
          .map(row => 
            row.map(cell => ({
              letter: cell.letter,
              state: cell.state
            }))
          )
        
        // Сохраняем текущий ввод (currentGuess) отдельно
        await saveWordleState(userId, targetWord, attemptsToSave, lastWordDate, currentGuess)
      } catch (error) {
        console.error('Error saving Wordle state:', error)
      }
    }, 1000)
  }

  // Сохраняем состояние при изменении guesses или currentGuess
  useEffect(() => {
    if (userId && targetWord && !loading) {
      saveGameState()
    }
  }, [guesses, currentGuess, userId, targetWord, loading])

  // Функция для расчета времени до следующего слова
  const calculateTimeUntilNext = (lastDate: string) => {
    const lastDateObj = new Date(lastDate + 'T00:00:00')
    const nextDateObj = new Date(lastDateObj)
    nextDateObj.setDate(nextDateObj.getDate() + 1)
    
    const now = new Date()
    const diff = nextDateObj.getTime() - now.getTime()
    
    if (diff <= 0) {
      return { hours: 0, minutes: 0, seconds: 0 }
    }
    
    const hours = Math.floor(diff / (1000 * 60 * 60))
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))
    const seconds = Math.floor((diff % (1000 * 60)) / 1000)
    
    return { hours, minutes, seconds }
  }

  // Запуск таймера обратного отсчета
  const startCountdownTimer = (lastDate: string) => {
    // Очищаем предыдущий таймер
    if (timerIntervalRef.current) {
      clearInterval(timerIntervalRef.current)
    }
    
    // Обновляем сразу
    setTimeUntilNextWord(calculateTimeUntilNext(lastDate))
    
    // Обновляем каждую секунду
    timerIntervalRef.current = setInterval(() => {
      const time = calculateTimeUntilNext(lastDate)
      setTimeUntilNextWord(time)
      
      // Если время истекло, перезагружаем игру
      if (time.hours === 0 && time.minutes === 0 && time.seconds === 0) {
        if (timerIntervalRef.current) {
          clearInterval(timerIntervalRef.current)
        }
        // Перезагружаем игру для получения нового слова
        window.location.reload()
      }
    }, 1000)
  }

  // Сохраняем состояние при выходе из компонента
  useEffect(() => {
    return () => {
      if (userId && targetWord && lastWordDate) {
        // Сохраняем только отправленные попытки
        const attemptsToSave = guesses
          .filter(row => row.some(cell => cell.state !== 'empty' && cell.letter !== ''))
          .map(row => 
            row.map(cell => ({
              letter: cell.letter,
              state: cell.state
            }))
          )
        saveWordleState(userId, targetWord, attemptsToSave, lastWordDate, currentGuess).catch(console.error)
      }
      if (saveStateTimeoutRef.current) {
        clearTimeout(saveStateTimeoutRef.current)
      }
      if (timerIntervalRef.current) {
        clearInterval(timerIntervalRef.current)
      }
    }
  }, [userId, targetWord, lastWordDate, guesses, currentGuess])

  const handleSubmit = async () => {
    if (currentGuess.length !== WORD_LENGTH) return
    
    // Проверяем, не отгадано ли уже это слово
    if (alreadyGuessed || guessedWords.includes(currentGuess)) {
      hapticFeedback('heavy')
      alert('Это слово уже было отгадано!')
      return
    }

    // Если слово не совпадает с целевым, проверяем его по словарю перед отправкой
    if (currentGuess !== targetWord) {
      // Проверяем слово по словарю через API перед показом результата
      try {
        const config = await loadConfig()
        const tg = window.Telegram?.WebApp
        const initData = (tg as any)?.initData || ''
        
        const checkResponse = await fetch(`${config.apiUrl}/wordle/guess`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ 
            word: currentGuess, 
            userId: userId,
            initData: initData 
          }),
        })
        
        if (checkResponse.ok) {
          const checkResult = await checkResponse.json()
          if (checkResult.invalid_word || checkResult.message === 'Слово не найдено в словаре') {
            hapticFeedback('heavy')
            alert('Слово не найдено в словаре')
            setCurrentGuess('') // Очищаем ввод
            return
          }
          // Если слово валидно, но не совпадает - продолжаем с обычной логикой показа результата
        }
      } catch (error) {
        console.error('Error checking word:', error)
        // Продолжаем, если проверка не удалась
      }
    }

    // Правильная логика определения состояния букв
    const newGuess: Cell[] = []
    const targetLetters = targetWord.split('')
    const guessLetters = currentGuess.split('')
    const usedTargetIndices = new Set<number>()
    
    // Сначала помечаем правильные буквы (зеленые)
    guessLetters.forEach((letter, index) => {
      if (letter === targetLetters[index]) {
        newGuess.push({ letter, state: 'correct' })
        usedTargetIndices.add(index)
      } else {
        newGuess.push({ letter, state: 'empty' }) // Временно
      }
    })
    
    // Затем помечаем буквы, которые есть, но не на своем месте (желтые)
    guessLetters.forEach((letter, guessIndex) => {
      if (newGuess[guessIndex].state === 'correct') {
        return // Уже помечено как правильное
      }
      
      // Ищем букву в целевом слове, которая еще не использована
      let found = false
      for (let i = 0; i < targetLetters.length; i++) {
        if (targetLetters[i] === letter && !usedTargetIndices.has(i)) {
          newGuess[guessIndex] = { letter, state: 'present' }
          usedTargetIndices.add(i)
          found = true
          break
        }
      }
      
      // Если не найдено, помечаем как отсутствующее
      if (!found) {
        newGuess[guessIndex] = { letter, state: 'absent' }
      }
    })

        const newGuesses = [...guesses]
        const currentAttempt = guesses.findIndex(row => row[0].state === 'empty' || !row[0].letter)
        if (currentAttempt !== -1) {
          newGuesses[currentAttempt] = newGuess
          setGuesses(newGuesses)
          
          // Сохраняем состояние после отправки попытки
          setTimeout(() => saveGameState(), 500)
        }

    // Обновляем использованные буквы
    const newUsedLetters = new Map(usedLetters)
    newGuess.forEach(({ letter, state }) => {
      const existingState = newUsedLetters.get(letter)
      if (!existingState || state === 'correct' || (state === 'present' && existingState === 'absent')) {
        newUsedLetters.set(letter, state)
      }
    })
    setUsedLetters(newUsedLetters)

    // Проверяем победу
    if (currentGuess === targetWord) {
      // Сначала показываем визуальный результат (зеленые клетки уже установлены)
      // Затем отправляем на сервер для проверки и начисления очков
      submitWordleGuess(currentGuess).then(result => {
        if (result.success) {
          setGameOver('win')
          setScore(5) // Каждое отгаданное слово = 5 очков
          setAlreadyGuessed(true)
          setGuessedWords(prev => [...prev, currentGuess])
          setShowConfetti(true) // Запускаем салют
          setCurrentGuess('') // Очищаем строку ввода, чтобы не дублировать
          hapticFeedback('heavy')
          // Запускаем таймер, если не запущен
          const today = new Date().toISOString().split('T')[0]
          startCountdownTimer(lastWordDate || today)
          if (onScore) {
            onScore(5) // Передаем 5 очков
          }
          // Скрываем салют через 2 секунды
          setTimeout(() => setShowConfetti(false), 2000)
        } else if (result.already_guessed) {
          setAlreadyGuessed(true)
          setGameOver('win') // Показываем как выигрыш, если уже отгадано
          hapticFeedback('heavy')
        } else {
          // Если ошибка, но слово правильное - все равно показываем победу
          // (может быть проблема с сетью, но визуально слово угадано)
          console.error('Error submitting word, but word is correct:', result.message)
          setGameOver('win')
          setScore(5)
          setAlreadyGuessed(true)
          setShowConfetti(true)
          hapticFeedback('heavy')
          setTimeout(() => setShowConfetti(false), 2000)
        }
      }).catch(error => {
        console.error('Error submitting word:', error)
        // Если ошибка сети, но слово правильное - все равно показываем победу
        setGameOver('win')
        setScore(5)
        setAlreadyGuessed(true)
        setShowConfetti(true)
        hapticFeedback('heavy')
        setTimeout(() => setShowConfetti(false), 2000)
      })
    } else if (currentAttempt === MAX_ATTEMPTS - 1) {
      setGameOver('lose')
      hapticFeedback('heavy')
    } else {
      setCurrentGuess('')
    }
  }

  // Функция handleRestart удалена - игроки могут отгадать только одно слово в день
  // При повторном входе показывается та же выигранная игра до сброса в 00:00


  const getCellColor = (state: LetterState) => {
    switch (state) {
      case 'correct':
        return 'bg-[#5A7C52] text-white' // primary - зеленый
      case 'present':
        return 'bg-[#FFE9AD] text-[#5A7C52]' // cream - желтый
      case 'absent':
        return 'bg-gray-300 text-gray-600'
      default:
        return 'bg-white border-2 border-gray-300 text-gray-800'
    }
  }

  const getKeyColor = (letter: string) => {
    const state = usedLetters.get(letter)
    switch (state) {
      case 'correct':
        return 'bg-[#5A7C52] text-white hover:bg-[#4A6B42] active:bg-[#4A6B42]'
      case 'present':
        return 'bg-[#FFE9AD] text-[#5A7C52] hover:bg-[#FFE099] active:bg-[#FFE099]'
      case 'absent':
        return 'bg-gray-400 text-white hover:bg-gray-500 active:bg-gray-500'
      default:
        return 'bg-primary text-white hover:bg-primary/80 active:bg-primary/60'
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-[#F8F8F8] px-4 py-6 pb-24 flex items-center justify-center">
        <div className="text-center">
          <div className="text-2xl font-bold text-[#5A7C52] mb-4">Загрузка слова...</div>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-[#F8F8F8] px-4 py-6 pb-24">
      <Confetti trigger={showConfetti} duration={2000} />
      <div className="max-w-md mx-auto">
        {/* Заголовок */}
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-2xl font-bold text-[#5A7C52]">ВОРДЛИ</h2>
          <button
            onClick={onClose}
            className="px-4 py-2 bg-[#5A7C52] text-white rounded-lg font-semibold hover:bg-[#4A6B42] transition-colors"
          >
            Назад
          </button>
        </div>
        
        {/* Сообщение о том, что слово уже отгадано */}
        {alreadyGuessed && (
          <div className="mb-4 p-3 bg-[#FFE9AD] text-[#5A7C52] rounded-lg text-center">
            <div className="font-semibold mb-2">Это слово уже отгадано!</div>
            {timeUntilNextWord && (
              <div className="text-sm">
                Следующее слово через: {timeUntilNextWord.hours}ч {timeUntilNextWord.minutes}м {timeUntilNextWord.seconds}с
              </div>
            )}
          </div>
        )}
        
        {/* Таймер для неотгаданного слова */}
        {!alreadyGuessed && timeUntilNextWord && (
          <div className="mb-4 p-3 bg-[#FDFBF5] text-[#5A7C52] rounded-lg text-center text-sm border border-[#5A7C52]/20">
            Следующее слово через: {timeUntilNextWord.hours}ч {timeUntilNextWord.minutes}м {timeUntilNextWord.seconds}с
          </div>
        )}

        {/* Игровое поле */}
        <div className="space-y-1.5 mb-4">
          {guesses.map((row, rowIndex) => {
            const isCurrentRow = rowIndex === guesses.findIndex(r => r[0].state === 'empty')
            const displayWord = isCurrentRow ? currentGuess : ''
            
            return (
              <div key={rowIndex} className="flex gap-2 justify-center">
                {row.map((cell, cellIndex) => {
                  const letter = isCurrentRow && displayWord[cellIndex] ? displayWord[cellIndex] : cell.letter
                  const state = isCurrentRow ? 'empty' : cell.state
                  
                  return (
                    <motion.div
                      key={cellIndex}
                      initial={false}
                      animate={{
                        scale: letter ? [1, 1.1, 1] : 1,
                        rotate: letter && state !== 'empty' ? [0, 5, -5, 0] : 0,
                      }}
                      transition={{ duration: 0.3 }}
                      className={`w-10 h-10 flex items-center justify-center rounded-md font-bold text-base ${getCellColor(state)}`}
                    >
                      {letter}
                    </motion.div>
                  )
                })}
              </div>
            )
          })}
        </div>

        {/* Скрытый input для ввода */}
        <input
          ref={inputRef}
          type="text"
          value={currentGuess}
          onChange={(e) => {
            const value = e.target.value.toUpperCase().replace(/[^А-ЯЁ]/g, '').slice(0, WORD_LENGTH)
            setCurrentGuess(value)
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              handleSubmit()
            } else if (e.key === 'Backspace') {
              setCurrentGuess(prev => prev.slice(0, -1))
            }
          }}
          className="absolute opacity-0 pointer-events-none"
          autoFocus
        />

        {/* Виртуальная клавиатура (как в кроссворде) */}
        <div className="mb-4">
          {/* Первый ряд */}
          <div className="grid grid-cols-12 gap-1 mb-1">
            {russianLetters.slice(0, 12).map((letter) => (
              <motion.button
                key={letter}
                onClick={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                  handleKeyPress(letter)
                }}
                onTouchStart={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                  handleKeyPress(letter)
                }}
                onTouchEnd={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                }}
                whileTap={{ scale: 0.9 }}
                className={`px-1 py-2 rounded-lg font-bold text-xs min-h-[40px] flex items-center justify-center transition-colors touch-none select-none ${getKeyColor(letter)}`}
              >
                {letter}
              </motion.button>
            ))}
          </div>
          {/* Второй ряд (с небольшим смещением влево, как на реальной клавиатуре) */}
          <div className="grid grid-cols-11 gap-1 mb-1 ml-[4.17%]">
            {russianLetters.slice(12, 23).map((letter) => (
              <motion.button
                key={letter}
                onClick={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                  handleKeyPress(letter)
                }}
                onTouchStart={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                  handleKeyPress(letter)
                }}
                onTouchEnd={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                }}
                whileTap={{ scale: 0.9 }}
                className={`px-1 py-2 rounded-lg font-bold text-xs min-h-[40px] flex items-center justify-center transition-colors touch-none select-none ${getKeyColor(letter)}`}
              >
                {letter}
              </motion.button>
            ))}
          </div>
          {/* Третий ряд (с большим смещением влево) */}
          <div className="grid grid-cols-10 gap-1 mb-1 ml-[8.33%]">
            {russianLetters.slice(23).map((letter) => (
              <motion.button
                key={letter}
                onClick={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                  handleKeyPress(letter)
                }}
                onTouchStart={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                  handleKeyPress(letter)
                }}
                onTouchEnd={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                }}
                whileTap={{ scale: 0.9 }}
                className={`px-1 py-2 rounded-lg font-bold text-xs min-h-[40px] flex items-center justify-center transition-colors touch-none select-none ${getKeyColor(letter)}`}
              >
                {letter}
              </motion.button>
            ))}
          </div>
          
          {/* Кнопки управления */}
          <div className="flex gap-2 mt-2">
            <motion.button
              onClick={(e) => {
                e.preventDefault()
                e.stopPropagation()
                handleKeyPress('BACKSPACE')
              }}
              onTouchStart={(e) => {
                e.preventDefault()
                e.stopPropagation()
                handleKeyPress('BACKSPACE')
              }}
              onTouchEnd={(e) => {
                e.preventDefault()
                e.stopPropagation()
              }}
              whileTap={{ scale: 0.9 }}
              className="flex-1 px-4 py-2 bg-gray-500 text-white rounded-lg font-semibold hover:bg-gray-600 transition-colors min-h-[40px] flex items-center justify-center touch-none select-none"
            >
              ⌫ Удалить
            </motion.button>
            <motion.button
              onClick={(e) => {
                e.preventDefault()
                e.stopPropagation()
                handleKeyPress('ENTER')
              }}
              onTouchStart={(e) => {
                e.preventDefault()
                e.stopPropagation()
                handleKeyPress('ENTER')
              }}
              onTouchEnd={(e) => {
                e.preventDefault()
                e.stopPropagation()
              }}
              whileTap={{ scale: 0.9 }}
              disabled={currentGuess.length !== WORD_LENGTH}
              className={`flex-1 px-4 py-2 rounded-lg font-semibold transition-colors min-h-[40px] flex items-center justify-center touch-none select-none ${
                currentGuess.length === WORD_LENGTH
                  ? 'bg-[#5A7C52] text-white hover:bg-[#4A6B42]'
                  : 'bg-gray-300 text-gray-500 cursor-not-allowed'
              }`}
            >
              ✓ ВВОД
            </motion.button>
          </div>
        </div>

        {/* Результат игры */}
        <AnimatePresence>
          {gameOver && (
            <motion.div
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.8 }}
              className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
            >
              <motion.div
                initial={{ y: 20 }}
                animate={{ y: 0 }}
                className="bg-white rounded-lg p-6 max-w-sm w-full text-center"
              >
                {gameOver === 'win' ? (
                  <>
                    <div className="text-4xl mb-4">🎉</div>
                    <h3 className="text-2xl font-bold text-[#5A7C52] mb-2">Поздравляем!</h3>
                    <p className="text-gray-600 mb-4">Вы угадали слово: <strong>{targetWord}</strong></p>
                    <p className="text-lg font-semibold text-[#5A7C52] mb-4">Очки: {score}</p>
                    {timeUntilNextWord && (
                      <p className="text-sm text-gray-500 mb-4">
                        Следующее слово через: {timeUntilNextWord.hours}ч {timeUntilNextWord.minutes}м {timeUntilNextWord.seconds}с
                      </p>
                    )}
                  </>
                ) : (
                  <>
                    <div className="text-4xl mb-4">😔</div>
                    <h3 className="text-2xl font-bold text-gray-800 mb-2">Не повезло</h3>
                    <p className="text-gray-600 mb-4">Загаданное слово: <strong>{targetWord}</strong></p>
                  </>
                )}
                <div className="flex gap-3 justify-center">
                  <button
                    onClick={onClose}
                    className="px-6 py-2 bg-[#5A7C52] text-white rounded-lg font-semibold hover:bg-[#4A6B42] transition-colors"
                  >
                    Выйти
                  </button>
                </div>
              </motion.div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  )
}
