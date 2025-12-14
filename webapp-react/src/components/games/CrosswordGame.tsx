import { useEffect, useRef, useState } from 'react'
import { motion, AnimatePresence, useMotionValue, useTransform } from 'framer-motion'
import { generateCrossword, type CrosswordGrid, type CrosswordWord } from '../../utils/crosswordGenerator'
import { getCrosswordData, saveCrosswordProgress, updateGameScore, loadConfig } from '../../utils/api'
import { hapticFeedback } from '../../utils/telegram'
import type { Config } from '../../types'

interface CrosswordGameProps {
  onScore?: (score: number) => void
  onClose: () => void
}

interface Cell {
  letter: string
  isFilled: boolean
  isCorrect: boolean
  wordNumber?: number
  isSelected: boolean
  isPartOfSelectedWord: boolean
}

export default function CrosswordGame({ onClose }: CrosswordGameProps) {
  const [crossword, setCrossword] = useState<CrosswordGrid | null>(null)
  const [cells, setCells] = useState<Cell[][]>([])
  const [selectedWord, setSelectedWord] = useState<CrosswordWord | null>(null)
  const [selectedCell, setSelectedCell] = useState<{ row: number; col: number } | null>(null)
  const [guessedWords, setGuessedWords] = useState<Set<string>>(new Set())
  const [score, setScore] = useState(0)
  const [loading, setLoading] = useState(true)
  const [userId, setUserId] = useState<number | null>(null)
  const [config, setConfig] = useState<Config | null>(null)
  const [showOnboarding, setShowOnboarding] = useState(false)
  const [currentInput, setCurrentInput] = useState('')
  const [showKeyboard, setShowKeyboard] = useState(false)
  const [isDraggingKeyboard, setIsDraggingKeyboard] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const lastTapRef = useRef<number>(0)
  const keyboardStartY = useRef(0)
  const keyboardCurrentY = useRef(0)

  // Русская раскладка ЙЦУКЕН для виртуальной клавиатуры
  // Первый ряд: Й Ц У К Е Н Г Ш Щ З Х Ъ
  // Второй ряд: Ф Ы В А П Р О Л Д Ж Э
  // Третий ряд: Я Ч С М И Т Ь Б Ю Ё
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
          console.error('Cannot load crossword: userId not found')
          setLoading(false)
          return
        }

        setUserId(currentUserId)

        // Загружаем данные кроссворда
        const data = await getCrosswordData(currentUserId)
        
        if (data.words.length === 0) {
          console.warn('Нет слов для кроссворда')
          setLoading(false)
          return
        }

        // Генерируем кроссворд
        const generated = generateCrossword(data.words)
        const guessedSet = new Set(data.guessed_words.map((w: string) => w.toUpperCase()))
        setCrossword(generated)
        setGuessedWords(guessedSet)

        // Инициализируем клетки
        const newCells: Cell[][] = Array(generated.size)
          .fill(null)
          .map(() => Array(generated.size).fill(null).map(() => ({
            letter: '',
            isFilled: false,
            isCorrect: false,
            isSelected: false,
            isPartOfSelectedWord: false
          })))

        // Заполняем сетку и номера слов
        // Сначала помечаем все клетки как заполненные и добавляем номера
        generated.words.forEach(word => {
          for (let i = 0; i < word.word.length; i++) {
            const row = word.direction === 'down' ? word.row + i : word.row
            const col = word.direction === 'across' ? word.col + i : word.col

            newCells[row][col].isFilled = true
            
            if (i === 0) {
              newCells[row][col].wordNumber = word.number
            }
          }
        })

        // Затем заполняем буквы ТОЛЬКО для отгаданных слов
        generated.words.forEach(word => {
          const isGuessed = guessedSet.has(word.word.toUpperCase())
          
          if (isGuessed) {
            // Показываем буквы только для отгаданных слов
            for (let i = 0; i < word.word.length; i++) {
              const row = word.direction === 'down' ? word.row + i : word.row
              const col = word.direction === 'across' ? word.col + i : word.col
              
              newCells[row][col].letter = word.word[i]
              newCells[row][col].isCorrect = true
            }
          }
        })

        setCells(newCells)
        setScore(data.guessed_words.length)
        
        // Проверяем, нужно ли показать онбординг
        const hasSeenOnboarding = localStorage.getItem('crossword_onboarding_seen')
        if (!hasSeenOnboarding) {
          setShowOnboarding(true)
        }
      } catch (error) {
        console.error('Error loading crossword:', error)
      } finally {
        setLoading(false)
      }
    }

    loadGame()
  }, [config])

  const handleCellClick = (row: number, col: number) => {
    if (!crossword) return

    const cell = cells[row][col]
    if (!cell.isFilled) return

    // Находим слово, к которому относится эта клетка
    let word: CrosswordWord | null = null
    for (const w of crossword.words) {
      if (w.direction === 'across') {
        if (w.row === row && col >= w.col && col < w.col + w.word.length) {
          word = w
          break
        }
      } else {
        if (w.col === col && row >= w.row && row < w.row + w.word.length) {
          word = w
          break
        }
      }
    }

    if (word) {
      setSelectedWord(word)
      setSelectedCell({ row, col })
      
      // Обновляем выделение
      const newCells = cells.map((rowCells, r) =>
        rowCells.map((c, cIdx) => ({
          ...c,
          isSelected: r === row && cIdx === col,
          isPartOfSelectedWord: word && (
            (word.direction === 'across' && r === word.row && cIdx >= word.col && cIdx < word.col + word.word.length) ||
            (word.direction === 'down' && cIdx === word.col && r >= word.row && r < word.row + word.word.length)
          )
        }))
      )
      setCells(newCells)

      // Сбрасываем текущий ввод при выборе нового слова
      setCurrentInput('')
      // Автоматически показываем клавиатуру при выборе слова
      setShowKeyboard(true)
    }
  }

  const handleDoubleTap = () => {
    const now = Date.now()
    const timeSinceLastTap = now - lastTapRef.current
    if (timeSinceLastTap < 300) {
      // Двойной тап - показываем/скрываем клавиатуру
      setShowKeyboard(!showKeyboard)
      hapticFeedback('light')
    }
    lastTapRef.current = now
  }

  const keyboardDragY = useMotionValue(0)
  const keyboardY = useTransform(keyboardDragY, (value) => {
    // Ограничиваем движение только вниз (положительные значения)
    return Math.max(0, Math.min(200, value))
  })

  const handleKeyboardPointerDown = (e: React.PointerEvent) => {
    keyboardStartY.current = e.clientY
    keyboardCurrentY.current = keyboardStartY.current
    setIsDraggingKeyboard(true)
    e.preventDefault()
    e.stopPropagation()
    if (e.pointerType === 'touch') {
      e.currentTarget.setPointerCapture(e.pointerId)
    }
  }

  useEffect(() => {
    if (isDraggingKeyboard) {
      const handleGlobalPointerMove = (e: PointerEvent) => {
        keyboardCurrentY.current = e.clientY
        const deltaY = keyboardCurrentY.current - keyboardStartY.current
        keyboardDragY.set(deltaY)
      }

      const handleGlobalPointerUp = () => {
        setIsDraggingKeyboard(false)
        const deltaY = keyboardCurrentY.current - keyboardStartY.current

        if (deltaY > 30) {
          // Скрываем клавиатуру при перетаскивании вниз
          setShowKeyboard(false)
          hapticFeedback('medium')
          keyboardDragY.set(0)
        } else {
          // Возвращаем в исходное положение
          keyboardDragY.set(0)
        }
      }

      window.addEventListener('pointermove', handleGlobalPointerMove)
      window.addEventListener('pointerup', handleGlobalPointerUp)

      return () => {
        window.removeEventListener('pointermove', handleGlobalPointerMove)
        window.removeEventListener('pointerup', handleGlobalPointerUp)
      }
    }
  }, [isDraggingKeyboard, keyboardDragY])

  // Сбрасываем позицию при показе/скрытии клавиатуры
  useEffect(() => {
    if (!showKeyboard) {
      keyboardDragY.set(0)
    }
  }, [showKeyboard, keyboardDragY])

  const handleInput = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (!selectedWord || !selectedCell) return

    const value = e.target.value.toUpperCase().replace(/[^А-ЯЁ]/g, '').slice(0, selectedWord.word.length)
    setCurrentInput(value)
    updateCellsWithValue(value)

    // Автопроверка
    if (value.length === selectedWord.word.length) {
      checkWord(selectedWord, value)
    }
  }

  const handleKeyPress = (letter: string) => {
    if (!selectedWord || !selectedCell) return

    const newValue = (currentInput + letter).toUpperCase().slice(0, selectedWord.word.length)
    setCurrentInput(newValue)
    updateCellsWithValue(newValue)

    // Автопроверка
    if (newValue.length === selectedWord.word.length) {
      checkWord(selectedWord, newValue)
    }
  }

  const handleBackspace = () => {
    if (!selectedWord || !selectedCell) return

    const newValue = currentInput.slice(0, -1)
    setCurrentInput(newValue)
    updateCellsWithValue(newValue)
  }

  const updateCellsWithValue = (value: string) => {
    if (!selectedWord) return

    // Обновляем клетки - вводим буквы только в неотгаданные клетки
    const newCells = [...cells]
    for (let i = 0; i < selectedWord.word.length; i++) {
      const row = selectedWord.direction === 'down' ? selectedWord.row + i : selectedWord.row
      const col = selectedWord.direction === 'across' ? selectedWord.col + i : selectedWord.col
      
      // Если клетка уже отгадана, оставляем букву как есть
      if (newCells[row][col].isCorrect) {
        continue
      }
      
      if (i < value.length) {
        // Вводим букву
        newCells[row][col] = {
          ...newCells[row][col],
          letter: value[i],
          isFilled: true
        }
      } else {
        // Очищаем при удалении
        newCells[row][col] = {
          ...newCells[row][col],
          letter: '',
          isFilled: true
        }
      }
    }

    // Очищаем оставшиеся клетки слова, если ввод стал короче
    for (let i = value.length; i < selectedWord.word.length; i++) {
      const row = selectedWord.direction === 'down' ? selectedWord.row + i : selectedWord.row
      const col = selectedWord.direction === 'across' ? selectedWord.col + i : selectedWord.col

      if (row < newCells.length && col < newCells[row].length && !newCells[row][col].isCorrect) {
        newCells[row][col] = {
          ...newCells[row][col],
          letter: ''
        }
      }
    }

    setCells(newCells)
  }

  const checkWord = async (word: CrosswordWord, userInput: string) => {
    if (!userId || !crossword) return

    const isCorrect = userInput.toUpperCase() === word.word.toUpperCase()
    
    if (isCorrect && !guessedWords.has(word.word.toUpperCase())) {
      // Слово отгадано впервые
      const newGuessedWords = new Set([...guessedWords, word.word.toUpperCase()])
      setGuessedWords(newGuessedWords)
      setScore(newGuessedWords.size)
      
      // Обновляем клетки как правильные
      const newCells = [...cells]
      for (let i = 0; i < word.word.length; i++) {
        const row = word.direction === 'down' ? word.row + i : word.row
        const col = word.direction === 'across' ? word.col + i : word.col
        newCells[row][col] = {
          ...newCells[row][col],
          isCorrect: true,
          letter: word.word[i]
        }
      }
      setCells(newCells)
      setCurrentInput('') // Очищаем ввод после правильного ответа
      setShowKeyboard(false) // Скрываем клавиатуру после правильного ответа

      // Сохраняем прогресс
      await saveCrosswordProgress(userId, Array.from(newGuessedWords))
      
      // Обновляем счет в статистике (1 слово = 1 очко, баланс 5:1)
      const gamePoints = Math.floor(newGuessedWords.size / 5)
      await updateGameScore(userId, 'crossword', gamePoints)
      
      hapticFeedback('heavy')
    } else if (!isCorrect) {
      // Неправильное слово - очищаем ввод
      const newCells = [...cells]
      for (let i = 0; i < word.word.length; i++) {
        const row = word.direction === 'down' ? word.row + i : word.row
        const col = word.direction === 'across' ? word.col + i : word.col
        if (!newCells[row][col].isCorrect) {
          newCells[row][col] = {
            ...newCells[row][col],
            letter: ''
          }
        }
      }
      setCells(newCells)
      hapticFeedback('light')
    }
  }

  if (loading) {
    return (
      <div className="fixed inset-0 z-50 bg-[#F8F8F8] flex items-center justify-center" style={{ bottom: '80px' }}>
        <div className="text-center text-gray-500">Загрузка кроссворда...</div>
      </div>
    )
  }

  if (!crossword || crossword.words.length === 0) {
    return (
      <div className="fixed inset-0 z-50 bg-[#F8F8F8] flex items-center justify-center" style={{ bottom: '80px' }}>
        <div className="text-center p-4">
          <p className="text-gray-600 mb-4">Кроссворд пока не готов</p>
          <button
            onClick={onClose}
            className="px-4 py-2 bg-primary text-white rounded-lg font-semibold"
          >
            Назад
          </button>
        </div>
      </div>
    )
  }

  const cellSize = Math.min(28, (window.innerWidth - 64) / crossword.size)

  return (
    <div className="fixed inset-0 z-50 bg-[#F8F8F8] flex flex-col" style={{ bottom: '80px' }}>
      {/* Онбординг */}
      {showOnboarding && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="absolute inset-0 bg-black/60 flex items-center justify-center z-30"
          onClick={() => {
            setShowOnboarding(false)
            localStorage.setItem('crossword_onboarding_seen', 'true')
          }}
        >
          <motion.div
            initial={{ scale: 0.9, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            onClick={(e) => e.stopPropagation()}
            className="bg-white rounded-lg p-6 max-w-md mx-4 shadow-xl"
          >
            <h2 className="text-2xl font-bold text-primary mb-4">Как играть в кроссворд?</h2>
            <div className="space-y-3 text-gray-700 mb-6">
              <div className="flex items-start gap-3">
                <span className="text-2xl">1️⃣</span>
                <div>
                  <p className="font-semibold">Выберите вопрос</p>
                  <p className="text-sm text-gray-600">Нажмите на вопрос в списке или на клетку с номером в сетке</p>
                </div>
              </div>
              <div className="flex items-start gap-3">
                <span className="text-2xl">2️⃣</span>
                <div>
                  <p className="font-semibold">Введите ответ</p>
                  <p className="text-sm text-gray-600">Начните печатать слово на клавиатуре</p>
                </div>
              </div>
              <div className="flex items-start gap-3">
                <span className="text-2xl">3️⃣</span>
                <div>
                  <p className="font-semibold">Проверка</p>
                  <p className="text-sm text-gray-600">Слово автоматически проверяется при вводе. Правильные слова подсвечиваются зеленым</p>
                </div>
              </div>
            </div>
            <button
              onClick={() => {
                setShowOnboarding(false)
                localStorage.setItem('crossword_onboarding_seen', 'true')
              }}
              className="w-full px-4 py-2 bg-primary text-white rounded-lg font-semibold"
            >
              Понятно, начать!
            </button>
          </motion.div>
        </motion.div>
      )}

      {/* Кнопка назад */}
      <div className="absolute top-4 left-4 z-10">
        <motion.button
          onClick={onClose}
          whileTap={{ scale: 0.95 }}
          className="px-4 py-2 bg-primary text-white rounded-lg font-semibold shadow-lg"
        >
          ← Назад
        </motion.button>
      </div>

      {/* Счет */}
      <div className="absolute top-4 right-4 z-10">
        <div className="px-4 py-2 bg-white/90 backdrop-blur-sm rounded-lg shadow-lg border-2 border-primary/30">
          <div className="text-sm text-gray-600">Отгадано</div>
          <div className="text-2xl font-bold text-primary">{score} / {crossword.words.length}</div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-4 pt-20">
        <div className="max-w-4xl mx-auto">
          {/* Сетка кроссворда */}
          <div className="mb-6">
            <div
              className="inline-grid gap-0.5 bg-gray-800 p-1 rounded"
              style={{
                gridTemplateColumns: `repeat(${crossword.size}, ${cellSize}px)`,
                gridTemplateRows: `repeat(${crossword.size}, ${cellSize}px)`
              }}
            >
              {cells.map((row, rowIdx) =>
                row.map((cell, colIdx) => (
                  <div
                    key={`${rowIdx}-${colIdx}`}
                    onClick={() => handleCellClick(rowIdx, colIdx)}
                    className={`
                      relative border border-gray-600 rounded
                      ${cell.isFilled 
                        ? cell.isCorrect 
                          ? 'bg-green-100 text-green-800' 
                          : cell.isPartOfSelectedWord
                          ? 'bg-blue-100 text-blue-800'
                          : 'bg-white text-gray-800'
                        : 'bg-gray-300'
                      }
                      ${cell.isSelected ? 'ring-2 ring-primary ring-offset-1' : ''}
                      flex items-center justify-center font-bold text-sm
                      cursor-pointer transition-all
                    `}
                    style={{ width: cellSize, height: cellSize }}
                  >
                    {cell.wordNumber && (
                      <span className="absolute top-0 left-0.5 text-xs text-gray-600">
                        {cell.wordNumber}
                      </span>
                    )}
                    {cell.letter}
                  </div>
                ))
              )}
            </div>
          </div>

          {/* Список вопросов */}
          <div className="bg-white rounded-lg p-4 shadow-lg">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-bold text-primary">Вопросы:</h3>
              <button
                onClick={() => setShowOnboarding(true)}
                className="text-sm text-gray-500 hover:text-primary"
              >
                ❓ Как играть?
              </button>
            </div>
            <div className="space-y-3">
              {crossword.words.map((word) => (
                <div
                  key={`${word.number}-${word.direction}`}
                  onClick={() => {
                    const row = word.row
                    const col = word.col
                    handleCellClick(row, col)
                  }}
                  onDoubleClick={handleDoubleTap}
                  className={`
                    p-3 rounded-lg border-2 cursor-pointer transition-all
                    ${guessedWords.has(word.word.toUpperCase())
                      ? 'bg-green-50 border-green-300'
                      : selectedWord?.word === word.word && selectedWord?.direction === word.direction
                      ? 'bg-blue-50 border-blue-300'
                      : 'bg-gray-50 border-gray-200 hover:border-primary/50'
                    }
                  `}
                >
                  <div className="flex items-start gap-2">
                    <span className="font-bold text-primary">
                      {word.number}. {word.direction === 'across' ? '→' : '↓'}
                    </span>
                    <span className={guessedWords.has(word.word.toUpperCase()) ? 'line-through text-gray-500' : ''}>
                      {word.description}
                    </span>
                  </div>
                  {!guessedWords.has(word.word.toUpperCase()) && selectedWord?.word === word.word && selectedWord?.direction === word.direction && (
                    <div className="mt-2 flex items-center justify-between">
                      <div className="text-xs text-blue-600">
                        💡 Нажмите "Ввести ответ" для клавиатуры
                      </div>
                      <motion.button
                        onClick={() => {
                          hapticFeedback('light')
                          setShowKeyboard(!showKeyboard)
                        }}
                        whileTap={{ scale: 0.95 }}
                        className="px-3 py-1.5 bg-primary text-white rounded-lg text-xs font-semibold hover:bg-primary/80 transition-colors"
                      >
                        {showKeyboard ? '✕ Скрыть' : '⌨️ Ввести ответ'}
                      </motion.button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Виртуальная клавиатура */}
      <AnimatePresence>
        {selectedWord && !guessedWords.has(selectedWord.word.toUpperCase()) && showKeyboard && (
          <motion.div
            initial={{ y: 100, opacity: 0 }}
            animate={{ 
              y: isDraggingKeyboard ? undefined : 0, 
              opacity: 1 
            }}
            exit={{ y: 100, opacity: 0 }}
            transition={{ duration: isDraggingKeyboard ? 0 : 0.3 }}
            style={{ y: isDraggingKeyboard ? keyboardY : undefined }}
            className="fixed bottom-24 left-0 right-0 bg-white border-t-2 border-primary/30 shadow-lg z-20 p-2 max-h-[55vh] overflow-y-auto"
          >
          <div className="max-w-4xl mx-auto">
            {/* Полоска для скрытия и текущий ввод */}
            <div className="sticky top-0 bg-white pb-2 border-b border-gray-200">
              {/* Полоска для скрытия клавиатуры */}
              <div 
                className="w-full flex justify-center py-2 cursor-grab active:cursor-grabbing touch-none select-none"
                onPointerDown={handleKeyboardPointerDown}
              >
                <motion.div
                  className="relative"
                  animate={{
                    scale: isDraggingKeyboard ? 1.1 : 1,
                  }}
                  transition={{ duration: 0.2 }}
                >
                  <motion.div
                    className="w-20 h-1.5 bg-gradient-to-r from-gray-400 via-gray-500 to-gray-400 rounded-full shadow-md"
                    animate={{
                      width: isDraggingKeyboard ? '6rem' : '5rem',
                      opacity: isDraggingKeyboard ? 1 : 0.7,
                      boxShadow: isDraggingKeyboard 
                        ? '0 4px 12px rgba(0, 0, 0, 0.2)' 
                        : '0 2px 6px rgba(0, 0, 0, 0.1)',
                    }}
                    transition={{ duration: 0.2 }}
                  />
                </motion.div>
              </div>
              {/* Текущий ввод */}
              <div className="text-center">
                <div className="text-lg font-bold text-primary">
                  {currentInput || '_'.repeat(selectedWord.word.length)}
                </div>
              </div>
            </div>

            {/* Клавиатура */}
            <div className="mb-2">
              {/* Первый ряд */}
              <div className="grid grid-cols-12 gap-1 mb-1">
                {russianLetters.slice(0, 12).map((letter) => (
                  <motion.button
                    key={letter}
                    onClick={() => {
                      hapticFeedback('light')
                      handleKeyPress(letter)
                    }}
                    whileTap={{ scale: 0.9 }}
                    className="px-1 py-2 bg-primary text-white rounded-lg font-bold text-xs hover:bg-primary/80 active:bg-primary/60 transition-colors min-h-[40px] flex items-center justify-center"
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
                    onClick={() => {
                      hapticFeedback('light')
                      handleKeyPress(letter)
                    }}
                    whileTap={{ scale: 0.9 }}
                    className="px-1 py-2 bg-primary text-white rounded-lg font-bold text-xs hover:bg-primary/80 active:bg-primary/60 transition-colors min-h-[40px] flex items-center justify-center"
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
                    onClick={() => {
                      hapticFeedback('light')
                      handleKeyPress(letter)
                    }}
                    whileTap={{ scale: 0.9 }}
                    className="px-1 py-2 bg-primary text-white rounded-lg font-bold text-xs hover:bg-primary/80 active:bg-primary/60 transition-colors min-h-[40px] flex items-center justify-center"
                  >
                    {letter}
                  </motion.button>
                ))}
              </div>
            </div>

            {/* Кнопки управления */}
            <div className="flex gap-2">
              <motion.button
                onClick={() => {
                  hapticFeedback('light')
                  handleBackspace()
                }}
                whileTap={{ scale: 0.9 }}
                disabled={currentInput.length === 0}
                className={`flex-1 px-4 py-2 rounded-lg font-semibold transition-colors ${
                  currentInput.length > 0
                    ? 'bg-gray-500 text-white hover:bg-gray-600'
                    : 'bg-gray-300 text-gray-500 cursor-not-allowed'
                }`}
              >
                ⌫ Удалить
              </motion.button>
              <motion.button
                onClick={() => {
                  if (currentInput.length === selectedWord.word.length) {
                    hapticFeedback('medium')
                    checkWord(selectedWord, currentInput)
                  }
                }}
                whileTap={{ scale: 0.9 }}
                disabled={currentInput.length !== selectedWord.word.length}
                className={`flex-1 px-4 py-2 rounded-lg font-semibold transition-colors ${
                  currentInput.length === selectedWord.word.length
                    ? 'bg-green-500 text-white hover:bg-green-600'
                    : 'bg-gray-300 text-gray-500 cursor-not-allowed'
                }`}
              >
                ✓ Проверить
              </motion.button>
            </div>
          </div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Скрытое поле ввода для букв (резерв для десктопа) */}
      <input
        ref={inputRef}
        type="text"
        value=""
        onChange={handleInput}
        className="absolute opacity-0 pointer-events-none"
        autoFocus={false}
      />
    </div>
  )
}

