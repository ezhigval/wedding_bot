import { useEffect, useRef, useState } from 'react'
import { motion } from 'framer-motion'
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
  const inputRef = useRef<HTMLInputElement>(null)

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

        // Загружаем данные кроссвода
        const data = await getCrosswordData(currentUserId)
        
        if (data.words.length === 0) {
          console.warn('Нет слов для кроссвода')
          setLoading(false)
          return
        }

        // Генерируем кроссвод
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
        generated.words.forEach(word => {
          const isGuessed = guessedSet.has(word.word.toUpperCase())
          
          for (let i = 0; i < word.word.length; i++) {
            const row = word.direction === 'down' ? word.row + i : word.row
            const col = word.direction === 'across' ? word.col + i : word.col

            if (newCells[row][col].letter === '') {
              newCells[row][col].letter = word.word[i]
              newCells[row][col].isFilled = true
              newCells[row][col].isCorrect = isGuessed
            }

            if (i === 0) {
              newCells[row][col].wordNumber = word.number
            }
          }
        })

        setCells(newCells)
        setScore(data.guessed_words.length)
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

      // Фокусируемся на вводе
      setTimeout(() => inputRef.current?.focus(), 100)
    }
  }

  const handleInput = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (!selectedWord || !selectedCell) return

    const value = e.target.value.toUpperCase().replace(/[^А-ЯЁ]/g, '').slice(0, selectedWord.word.length)
    
    // Обновляем клетки
    const newCells = [...cells]
    for (let i = 0; i < selectedWord.word.length; i++) {
      const row = selectedWord.direction === 'down' ? selectedWord.row + i : selectedWord.row
      const col = selectedWord.direction === 'across' ? selectedWord.col + i : selectedWord.col
      
      if (i < value.length) {
        newCells[row][col] = {
          ...newCells[row][col],
          letter: value[i],
          isFilled: true
        }
      } else {
        newCells[row][col] = {
          ...newCells[row][col],
          letter: newCells[row][col].isCorrect ? newCells[row][col].letter : '',
          isFilled: true
        }
      }
    }
    setCells(newCells)

    // Автопроверка
    if (value.length === selectedWord.word.length) {
      checkWord(selectedWord, value)
    }
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
        <div className="text-center text-gray-500">Загрузка кроссвода...</div>
      </div>
    )
  }

  if (!crossword || crossword.words.length === 0) {
    return (
      <div className="fixed inset-0 z-50 bg-[#F8F8F8] flex items-center justify-center" style={{ bottom: '80px' }}>
        <div className="text-center p-4">
          <p className="text-gray-600 mb-4">Кроссвод пока не готов</p>
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
          {/* Сетка кроссвода */}
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
            <h3 className="text-lg font-bold text-primary mb-4">Вопросы:</h3>
            <div className="space-y-3">
              {crossword.words.map((word) => (
                <div
                  key={`${word.number}-${word.direction}`}
                  onClick={() => {
                    const row = word.row
                    const col = word.col
                    handleCellClick(row, col)
                  }}
                  className={`
                    p-3 rounded-lg border-2 cursor-pointer transition-all
                    ${guessedWords.has(word.word.toUpperCase())
                      ? 'bg-green-50 border-green-300'
                      : selectedWord?.word === word.word && selectedWord?.direction === word.direction
                      ? 'bg-blue-50 border-blue-300'
                      : 'bg-gray-50 border-gray-200'
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
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Скрытое поле ввода для букв */}
      <input
        ref={inputRef}
        type="text"
        value=""
        onChange={handleInput}
        className="absolute opacity-0 pointer-events-none"
        autoFocus
      />
    </div>
  )
}

