import { useState, useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { getWordleWord, getWordleProgress, submitWordleGuess } from '../../utils/api'
import { hapticFeedback } from '../../utils/telegram'
import Confetti from '../common/Confetti'

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
  const inputRef = useRef<HTMLInputElement>(null)

  // Русская раскладка клавиатуры
  const keyboardRows = [
    ['Й', 'Ц', 'У', 'К', 'Е', 'Н', 'Г', 'Ш', 'Щ', 'З', 'Х', 'Ъ'],
    ['Ф', 'Ы', 'В', 'А', 'П', 'Р', 'О', 'Л', 'Д', 'Ж', 'Э'],
    ['Я', 'Ч', 'С', 'М', 'И', 'Т', 'Ь', 'Б', 'Ю']
  ]

  useEffect(() => {
    // Загружаем актуальное слово из Google Sheets
    const loadWord = async () => {
      setLoading(true)
      try {
        const word = await getWordleWord()
        const progress = await getWordleProgress()
        
        if (word) {
          setTargetWord(word.toUpperCase())
          setGuessedWords(progress.map(w => w.toUpperCase()))
          
          // Проверяем, не отгадано ли уже это слово
          if (progress.map(w => w.toUpperCase()).includes(word.toUpperCase())) {
            setAlreadyGuessed(true)
          }
        } else {
          // Fallback на случайное слово из резервного списка
          const randomWord = FALLBACK_WORDS[Math.floor(Math.random() * FALLBACK_WORDS.length)].toUpperCase()
          setTargetWord(randomWord)
        }
        
        setGuesses(Array(MAX_ATTEMPTS).fill(null).map(() => 
          Array(WORD_LENGTH).fill(null).map(() => ({ letter: '', state: 'empty' }))
        ))
      } catch (error) {
        console.error('Error loading Wordle word:', error)
        // Fallback
        const randomWord = FALLBACK_WORDS[Math.floor(Math.random() * FALLBACK_WORDS.length)].toUpperCase()
        setTargetWord(randomWord)
        setGuesses(Array(MAX_ATTEMPTS).fill(null).map(() => 
          Array(WORD_LENGTH).fill(null).map(() => ({ letter: '', state: 'empty' }))
        ))
      } finally {
        setLoading(false)
        // Фокус на input при загрузке
        if (inputRef.current) {
          inputRef.current.focus()
        }
      }
    }
    
    loadWord()
  }, [])

  const handleKeyPress = (key: string) => {
    if (gameOver) return
    
    if (key === 'ENTER') {
      handleSubmit()
    } else if (key === 'BACKSPACE') {
      setCurrentGuess(prev => prev.slice(0, -1))
    } else if (currentGuess.length < WORD_LENGTH && /[А-ЯЁ]/.test(key)) {
      setCurrentGuess(prev => prev + key.toUpperCase())
    }
    
    hapticFeedback('light')
  }

  const handleSubmit = async () => {
    if (currentGuess.length !== WORD_LENGTH) return
    
    // Проверяем, не отгадано ли уже это слово
    if (alreadyGuessed || guessedWords.includes(currentGuess)) {
      hapticFeedback('heavy')
      alert('Это слово уже было отгадано!')
      return
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
    const currentAttempt = guesses.findIndex(row => row[0].state === 'empty')
    if (currentAttempt !== -1) {
      newGuesses[currentAttempt] = newGuess
      setGuesses(newGuesses)
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
      // Отправляем слово на сервер для проверки и начисления очков
      submitWordleGuess(currentGuess).then(result => {
        if (result.success) {
          setGameOver('win')
          setScore(5) // Каждое отгаданное слово = 5 очков
          setAlreadyGuessed(true)
          setGuessedWords(prev => [...prev, currentGuess])
          setShowConfetti(true) // Запускаем салют
          hapticFeedback('heavy')
          if (onScore) {
            onScore(5) // Передаем 5 очков
          }
          // Скрываем салют через 2 секунды
          setTimeout(() => setShowConfetti(false), 2000)
        } else if (result.already_guessed) {
          setAlreadyGuessed(true)
          hapticFeedback('heavy')
          alert(result.message || 'Это слово уже было отгадано!')
        } else {
          hapticFeedback('heavy')
          alert(result.message || 'Ошибка при проверке слова')
        }
      }).catch(error => {
        console.error('Error submitting word:', error)
        hapticFeedback('heavy')
        alert('Ошибка при отправке слова')
      })
    } else if (currentAttempt === MAX_ATTEMPTS - 1) {
      setGameOver('lose')
      hapticFeedback('heavy')
    } else {
      setCurrentGuess('')
    }
  }

  const handleRestart = async () => {
    setLoading(true)
    try {
      const word = await getWordleWord()
      const progress = await getWordleProgress()
      
      if (word) {
        setTargetWord(word.toUpperCase())
        setGuessedWords(progress.map(w => w.toUpperCase()))
        if (progress.map(w => w.toUpperCase()).includes(word.toUpperCase())) {
          setAlreadyGuessed(true)
        } else {
          setAlreadyGuessed(false)
        }
      } else {
        const randomWord = FALLBACK_WORDS[Math.floor(Math.random() * FALLBACK_WORDS.length)].toUpperCase()
        setTargetWord(randomWord)
        setAlreadyGuessed(false)
      }
      
      setCurrentGuess('')
      setGameOver(null)
      setScore(0)
      setUsedLetters(new Map())
      setGuesses(Array(MAX_ATTEMPTS).fill(null).map(() => 
        Array(WORD_LENGTH).fill(null).map(() => ({ letter: '', state: 'empty' }))
      ))
    } catch (error) {
      console.error('Error reloading word:', error)
      const randomWord = FALLBACK_WORDS[Math.floor(Math.random() * FALLBACK_WORDS.length)].toUpperCase()
      setTargetWord(randomWord)
      setCurrentGuess('')
      setGameOver(null)
      setScore(0)
      setUsedLetters(new Map())
      setGuessedWords([])
      setAlreadyGuessed(false)
    } finally {
      setLoading(false)
      if (inputRef.current) {
        inputRef.current.focus()
      }
    }
  }


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
        return 'bg-[#5A7C52] text-white'
      case 'present':
        return 'bg-[#FFE9AD] text-[#5A7C52]'
      case 'absent':
        return 'bg-gray-400 text-white'
      default:
        return 'bg-gray-200 text-gray-800 hover:bg-gray-300'
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
          <div className="mb-4 p-3 bg-[#FFE9AD] text-[#5A7C52] rounded-lg text-center font-semibold">
            Это слово уже отгадано! Ждите новое слово в таблице.
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

        {/* Виртуальная клавиатура */}
        <div className="space-y-1 mb-4">
          {keyboardRows.map((row, rowIndex) => (
            <div key={rowIndex} className="flex gap-0.5 justify-center flex-wrap">
              {row.map((letter) => (
                <button
                  key={letter}
                  onClick={() => handleKeyPress(letter)}
                  className={`px-2 py-1.5 rounded-md font-semibold text-xs min-w-[28px] ${getKeyColor(letter)} transition-colors`}
                >
                  {letter}
                </button>
              ))}
            </div>
          ))}
          
          {/* Кнопки управления */}
          <div className="flex gap-1.5 justify-center mt-1">
            <button
              onClick={() => handleKeyPress('BACKSPACE')}
              className="px-3 py-1.5 bg-gray-300 text-gray-800 rounded-md font-semibold text-xs hover:bg-gray-400 transition-colors"
            >
              ⌫
            </button>
            <button
              onClick={() => handleKeyPress('ENTER')}
              className="px-4 py-1.5 bg-[#5A7C52] text-white rounded-md font-semibold text-xs hover:bg-[#4A6B42] transition-colors"
            >
              ВВОД
            </button>
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
                    onClick={handleRestart}
                    className="px-6 py-2 bg-[#5A7C52] text-white rounded-lg font-semibold hover:bg-[#4A6B42] transition-colors"
                  >
                    Играть снова
                  </button>
                  <button
                    onClick={onClose}
                    className="px-6 py-2 bg-gray-300 text-gray-800 rounded-lg font-semibold hover:bg-gray-400 transition-colors"
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

