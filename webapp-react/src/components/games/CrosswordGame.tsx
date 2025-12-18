import { useEffect, useRef, useState } from 'react'
import { motion, AnimatePresence, useMotionValue, useTransform } from 'framer-motion'
import { generateCrossword, type CrosswordGrid, type CrosswordWord } from '../../utils/crosswordGenerator'
import { getCrosswordData, saveCrosswordProgress, updateGameScore, loadConfig } from '../../utils/api'
import { hapticFeedback } from '../../utils/telegram'
import Confetti from '../common/Confetti'
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
  const [wrongWords, setWrongWords] = useState<Set<string>>(new Set()) // Неправильные слова после завершения
  const [score, setScore] = useState(0)
  const [loading, setLoading] = useState(true)
  const [userId, setUserId] = useState<number | null>(null)
  const [config, setConfig] = useState<Config | null>(null)
  const [crosswordIndex, setCrosswordIndex] = useState<number>(0)
  const [crosswordStartDate, setCrosswordStartDate] = useState<string>('')
  const [timeUntilNextCrossword, setTimeUntilNextCrossword] = useState<{ hours: number; minutes: number; seconds: number } | null>(null)
  const [showConfetti, setShowConfetti] = useState(false)
  const timerIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const [showOnboarding, setShowOnboarding] = useState(false)
  const [currentInput, setCurrentInput] = useState('')
  const [showKeyboard, setShowKeyboard] = useState(false)
  const [isDraggingKeyboard, setIsDraggingKeyboard] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const lastTapRef = useRef<number>(0)
  const keyboardStartY = useRef(0)
  const keyboardCurrentY = useRef(0)
  
  // Проверяем, завершен ли кроссворд (все клетки заполнены)
  const isCompleted = crossword ? crossword.words.every(word => {
    // Проверяем, что все клетки слова заполнены
    for (let i = 0; i < word.word.length; i++) {
      const row = word.direction === 'down' ? word.row + i : word.row
      const col = word.direction === 'across' ? word.col + i : word.col
      if (!cells[row] || !cells[row][col] || !cells[row][col].letter) {
        return false
      }
    }
    return true
  }) : false
  
  // Кроссворд решен, если завершен и все слова правильные
  const isSolved = isCompleted && crossword ? crossword.words.every(word => guessedWords.has(word.word.toUpperCase())) : false

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

        // Сохраняем индекс кроссворда
        if (data.crossword_index !== undefined) {
          setCrosswordIndex(data.crossword_index)
        }

        // Генерируем кроссворд
        const generated = generateCrossword(data.words)
        const guessedSet = new Set(data.guessed_words.map((w: string) => w.toUpperCase()))
        const wrongSet = new Set((data.wrong_words || []).map((w: string) => w.toUpperCase()))
        setCrossword(generated)
        setGuessedWords(guessedSet)
        setWrongWords(wrongSet)
        
        // Устанавливаем дату начала кроссворда
        if (data.start_date) {
          setCrosswordStartDate(data.start_date)
        } else {
          // Если даты нет, устанавливаем сегодня
          const today = new Date().toISOString().split('T')[0]
          setCrosswordStartDate(today)
        }
        
        // Запускаем таймер обратного отсчета
        if (data.start_date) {
          startCountdownTimer(data.start_date)
        } else {
          const today = new Date().toISOString().split('T')[0]
          startCountdownTimer(today)
        }

        // Проверяем, завершен ли кроссворд (все клетки заполнены и слова проверены)
        const allFilled = data.cell_letters && Object.keys(data.cell_letters).length > 0 && 
          generated.words.every(w => {
            for (let i = 0; i < w.word.length; i++) {
              const row = w.direction === 'down' ? w.row + i : w.row
              const col = w.direction === 'across' ? w.col + i : w.col
              const key = `${row},${col}`
              if (!data.cell_letters || !data.cell_letters[key]) return false
            }
            return true
          })
        const allWordsChecked = generated.words.every(w => 
          guessedSet.has(w.word.toUpperCase()) || wrongSet.has(w.word.toUpperCase())
        )
        
        if (allFilled && allWordsChecked) {
          setShowKeyboard(false)
          setSelectedWord(null)
        }

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

        // Заполняем буквы из сохраненного состояния (cell_letters)
        // После завершения правильные слова будут зелеными, неправильные желтыми
        generated.words.forEach(word => {
          const isGuessed = guessedSet.has(word.word.toUpperCase())
          const isWrong = wrongSet.has(word.word.toUpperCase())
          
          for (let i = 0; i < word.word.length; i++) {
            const row = word.direction === 'down' ? word.row + i : word.row
            const col = word.direction === 'across' ? word.col + i : word.col
            const key = `${row},${col}`
            
            // Если слово отгадано - показываем правильную букву и помечаем как правильное
            if (isGuessed) {
              newCells[row][col].letter = word.word[i]
              newCells[row][col].isCorrect = true
            } 
            // Если слово неправильное после завершения - помечаем как неправильное
            else if (isWrong && data.cell_letters && data.cell_letters[key]) {
              newCells[row][col].letter = data.cell_letters[key]
              newCells[row][col].isCorrect = false
            }
          }
        })

        // Восстанавливаем сохраненные буквы в клетках (если есть и кроссворд не завершен)
        if (data.cell_letters && typeof data.cell_letters === 'object') {
          const cellLetters = data.cell_letters
          const allFilled = generated.words.every(w => {
            for (let i = 0; i < w.word.length; i++) {
              const row = w.direction === 'down' ? w.row + i : w.row
              const col = w.direction === 'across' ? w.col + i : w.col
              const key = `${row},${col}`
              if (!cellLetters[key]) return false
            }
            return true
          })
          
          // Если кроссворд не завершен, восстанавливаем буквы
          if (!allFilled) {
            Object.keys(cellLetters).forEach(key => {
              const [row, col] = key.split(',').map(Number)
              if (row >= 0 && row < newCells.length && col >= 0 && col < newCells[row].length) {
                const cell = newCells[row][col]
                // Восстанавливаем букву только если клетка не отгадана
                if (!cell.isCorrect && cell.isFilled && cellLetters[key]) {
                  newCells[row][col].letter = cellLetters[key]
                }
              }
            })
          }
        }

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
    if (!crossword || isCompleted) return // Блокируем клики после завершения

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
    if (!selectedWord || !selectedCell || isCompleted) return

    const value = e.target.value.toUpperCase().replace(/[^А-ЯЁ]/g, '').slice(0, selectedWord.word.length)
    setCurrentInput(value)
    updateCellsWithValue(value)
    
    // Проверяем завершение после обновления клеток
    setTimeout(() => checkCompletion(), 100)
  }

  const handleKeyPress = (letter: string) => {
    if (!selectedWord || !selectedCell || isCompleted) return

    const newValue = (currentInput + letter).toUpperCase().slice(0, selectedWord.word.length)
    setCurrentInput(newValue)
    updateCellsWithValue(newValue)
    
    // Проверяем завершение после обновления клеток
    setTimeout(() => checkCompletion(), 100)
  }

  const handleBackspace = () => {
    if (!selectedWord || !selectedCell || isCompleted) return

    const newValue = currentInput.slice(0, -1)
    setCurrentInput(newValue)
    updateCellsWithValue(newValue)
    
    // Проверяем завершение после обновления клеток
    setTimeout(() => checkCompletion(), 100)
  }

  const updateCellsWithValue = (value: string) => {
    if (!selectedWord || !crossword) return

    // Обновляем клетки - вводим буквы только в незавершенные клетки
    const newCells = cells.map(row => [...row])
    
    for (let i = 0; i < selectedWord.word.length; i++) {
      const row = selectedWord.direction === 'down' ? selectedWord.row + i : selectedWord.row
      const col = selectedWord.direction === 'across' ? selectedWord.col + i : selectedWord.col
      
      // Если кроссворд завершен, не позволяем изменять
      if (isCompleted) {
        continue
      }
      
      if (i < value.length) {
        const newLetter = value[i]
        const existingLetter = newCells[row][col].letter
        
        // Проверяем пересечения: если в этой клетке уже есть буква из другого слова,
        // она должна совпадать с новой буквой
        if (existingLetter && existingLetter !== newLetter) {
          // Находим все слова, которые проходят через эту клетку
          const intersectingWords = crossword.words.filter(w => {
            if (w.word === selectedWord.word && w.direction === selectedWord.direction) {
              return false // Исключаем текущее слово
            }
            for (let j = 0; j < w.word.length; j++) {
              const wRow = w.direction === 'down' ? w.row + j : w.row
              const wCol = w.direction === 'across' ? w.col + j : w.col
              if (wRow === row && wCol === col) {
                return true
              }
            }
            return false
          })
          
          // Если есть пересекающиеся слова с другой буквой - не позволяем ввод
          const hasConflict = intersectingWords.some(w => {
            for (let j = 0; j < w.word.length; j++) {
              const wRow = w.direction === 'down' ? w.row + j : w.row
              const wCol = w.direction === 'across' ? w.col + j : w.col
              if (wRow === row && wCol === col) {
                const cellLetter = newCells[wRow][wCol].letter
                // Если в пересекающемся слове уже есть буква, она должна совпадать
                if (cellLetter && cellLetter !== newLetter) {
                  return true
                }
              }
            }
            return false
          })
          
          if (hasConflict) {
            hapticFeedback('heavy')
            alert('Буква в этой клетке уже используется другим словом!')
            return // Не обновляем клетку
          }
        }
        
        // Вводим букву
        newCells[row][col] = {
          ...newCells[row][col],
          letter: newLetter,
          isFilled: true
        }
      } else {
        // Очищаем при удалении (только если нет конфликтов с другими словами)
        const intersectingWords = crossword.words.filter(w => {
          if (w.word === selectedWord.word && w.direction === selectedWord.direction) {
            return false
          }
          for (let j = 0; j < w.word.length; j++) {
            const wRow = w.direction === 'down' ? w.row + j : w.row
            const wCol = w.direction === 'across' ? w.col + j : w.col
            if (wRow === row && wCol === col) {
              return true
            }
          }
          return false
        })
        
        // Очищаем только если нет пересекающихся слов с буквами
        const hasIntersectingLetter = intersectingWords.some(w => {
          for (let j = 0; j < w.word.length; j++) {
            const wRow = w.direction === 'down' ? w.row + j : w.row
            const wCol = w.direction === 'across' ? w.col + j : w.col
            if (wRow === row && wCol === col && newCells[wRow][wCol].letter) {
              return true
            }
          }
          return false
        })
        
        if (!hasIntersectingLetter) {
          newCells[row][col] = {
            ...newCells[row][col],
            letter: ''
          }
        }
      }
    }

    setCells(newCells)
    
    // Сохраняем состояние клеток при изменении
    if (userId !== null) {
      saveCrosswordCellState()
    }
  }
  
  // Проверка завершения кроссворда и финальная проверка всех слов
  const checkCompletion = async () => {
    if (!crossword || !userId) return
    
    // Проверяем, все ли клетки заполнены (используем текущее состояние cells)
    const allFilled = crossword.words.every(word => {
      for (let i = 0; i < word.word.length; i++) {
        const row = word.direction === 'down' ? word.row + i : word.row
        const col = word.direction === 'across' ? word.col + i : word.col
        if (!cells[row] || !cells[row][col] || !cells[row][col].letter) {
          return false
        }
      }
      return true
    })
    
    if (!allFilled) return
    
    // Проверяем, не завершен ли уже (чтобы не проверять повторно)
    // Если все слова уже проверены (в guessedWords или wrongWords) - не проверяем снова
    const allWordsChecked = crossword.words.every(word => 
      guessedWords.has(word.word.toUpperCase()) || wrongWords.has(word.word.toUpperCase())
    )
    
    if (allWordsChecked) return
    
    // Все клетки заполнены - проверяем все слова
    const newGuessedWords = new Set<string>()
    const newWrongWords = new Set<string>()
    
    crossword.words.forEach(word => {
      // Собираем буквы из клеток
      let userWord = ''
      for (let i = 0; i < word.word.length; i++) {
        const row = word.direction === 'down' ? word.row + i : word.row
        const col = word.direction === 'across' ? word.col + i : word.col
        userWord += cells[row][col].letter || ''
      }
      
      // Проверяем правильность
      if (userWord.toUpperCase() === word.word.toUpperCase()) {
        newGuessedWords.add(word.word.toUpperCase())
      } else {
        newWrongWords.add(word.word.toUpperCase())
      }
    })
    
    // Обновляем состояние
    setGuessedWords(newGuessedWords)
    setWrongWords(newWrongWords)
    setScore(newGuessedWords.size)
    
    // Обновляем клетки: правильные - зеленые, неправильные - желтые
    const newCells = cells.map((row, rowIdx) =>
      row.map((cell, colIdx) => {
        // Находим все слова, которые проходят через эту клетку
        const wordsInCell = crossword.words.filter(w => {
          for (let i = 0; i < w.word.length; i++) {
            const wRow = w.direction === 'down' ? w.row + i : w.row
            const wCol = w.direction === 'across' ? w.col + i : w.col
            if (wRow === rowIdx && wCol === colIdx) {
              return true
            }
          }
          return false
        })
        
        // Если хотя бы одно слово правильное - клетка правильная (зеленая)
        // Если все слова неправильные - клетка неправильная (желтая)
        const hasCorrectWord = wordsInCell.some(w => newGuessedWords.has(w.word.toUpperCase()))
        
        return {
          ...cell,
          isCorrect: hasCorrectWord,
          // Если есть правильное слово - зеленый, если только неправильные - желтый
        }
      })
    )
    
    setCells(newCells)
    
    // Если все слова правильные - запускаем салют
    if (newGuessedWords.size === crossword.words.length) {
      setShowConfetti(true)
      setTimeout(() => setShowConfetti(false), 2000)
      hapticFeedback('heavy')
    } else {
      hapticFeedback('medium')
    }
    
    // Скрываем клавиатуру
    setShowKeyboard(false)
    setSelectedWord(null)
    
    // Сохраняем финальное состояние
    const cellLetters: { [key: string]: string } = {}
    newCells.forEach((row, rowIndex) => {
      row.forEach((cell, colIndex) => {
        if (cell.isFilled && cell.letter) {
          cellLetters[`${rowIndex},${colIndex}`] = cell.letter
        }
      })
    })
    
    try {
      await saveCrosswordProgress(userId, Array.from(newGuessedWords), crosswordIndex, cellLetters, Array.from(newWrongWords), crosswordStartDate)
      
      // Начисляем очки (1 слово = 1 очко, баланс 5:1)
      const gamePoints = Math.floor(newGuessedWords.size / 5)
      await updateGameScore(userId, 'crossword', gamePoints)
    } catch (error) {
      console.error('Error saving completed crossword:', error)
    }
  }

  // Функция для сохранения состояния клеток
  const saveCrosswordCellState = async () => {
    if (!userId || !crossword || isCompleted) return
    
    // Собираем текущие буквы в клетках
    const cellLetters: { [key: string]: string } = {}
    cells.forEach((row, rowIndex) => {
      row.forEach((cell, colIndex) => {
        if (cell.isFilled && cell.letter) {
          cellLetters[`${rowIndex},${colIndex}`] = cell.letter
        }
      })
    })
    
    // Сохраняем прогресс с текущими буквами
    try {
      await saveCrosswordProgress(userId, Array.from(guessedWords), crosswordIndex, cellLetters, Array.from(wrongWords), crosswordStartDate)
    } catch (error) {
      console.error('Error saving crossword cell state:', error)
    }
  }
  
  // Функция для расчета времени до следующего кроссворда
  const calculateTimeUntilNext = (startDate: string) => {
    const startDateObj = new Date(startDate + 'T00:00:00')
    const nextDateObj = new Date(startDateObj)
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
  const startCountdownTimer = (startDate: string) => {
    // Очищаем предыдущий таймер
    if (timerIntervalRef.current) {
      clearInterval(timerIntervalRef.current)
    }
    
    // Обновляем сразу
    setTimeUntilNextCrossword(calculateTimeUntilNext(startDate))
    
    // Обновляем каждую секунду
    timerIntervalRef.current = setInterval(() => {
      const time = calculateTimeUntilNext(startDate)
      setTimeUntilNextCrossword(time)
      
      // Если время истекло, перезагружаем игру
      if (time.hours === 0 && time.minutes === 0 && time.seconds === 0) {
        if (timerIntervalRef.current) {
          clearInterval(timerIntervalRef.current)
        }
        // Перезагружаем игру для получения нового кроссворда
        window.location.reload()
      }
    }, 1000)
  }

  // Сохраняем состояние при выходе
  useEffect(() => {
    return () => {
      if (userId && crossword) {
        saveCrosswordCellState().catch(console.error)
      }
      if (timerIntervalRef.current) {
        clearInterval(timerIntervalRef.current)
      }
    }
  }, [])

  // Старая функция checkWord удалена - теперь проверка происходит только при завершении кроссворда

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
      <Confetti trigger={showConfetti} duration={2000} />
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
          {/* Таймер до следующего кроссворда */}
          {timeUntilNextCrossword && (
            <div className={`mb-4 p-3 rounded-lg text-center text-sm border ${
              isCompleted
                ? 'bg-[#FFE9AD] text-[#5A7C52] border-[#5A7C52]/20' 
                : 'bg-[#FDFBF5] text-[#5A7C52] border-[#5A7C52]/20'
            }`}>
              {isCompleted ? (
                <div>
                  <div className="font-semibold mb-2">
                    {isSolved ? 'Кроссворд решен!' : 'Кроссворд завершен!'}
                  </div>
                  <div>Следующий кроссворд через: {timeUntilNextCrossword.hours}ч {timeUntilNextCrossword.minutes}м {timeUntilNextCrossword.seconds}с</div>
                </div>
              ) : (
                <div>Следующий кроссворд через: {timeUntilNextCrossword.hours}ч {timeUntilNextCrossword.minutes}м {timeUntilNextCrossword.seconds}с</div>
              )}
            </div>
          )}
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
                        ? isCompleted
                          ? cell.isCorrect 
                            ? 'bg-green-100 text-green-800' // Правильные слова - зеленые
                            : 'bg-yellow-100 text-yellow-800' // Неправильные слова - желтые
                          : cell.isPartOfSelectedWord
                          ? 'bg-blue-100 text-blue-800'
                          : 'bg-white text-gray-800'
                        : 'bg-gray-300'
                      }
                      ${cell.isSelected ? 'ring-2 ring-primary ring-offset-1' : ''}
                      flex items-center justify-center font-bold text-sm
                      ${isCompleted ? 'cursor-default' : 'cursor-pointer'} transition-all
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
                    if (!isCompleted) {
                      const row = word.row
                      const col = word.col
                      handleCellClick(row, col)
                    }
                  }}
                  onDoubleClick={() => {
                    if (!isCompleted) {
                      handleDoubleTap()
                    }
                  }}
                  className={`
                    p-3 rounded-lg border-2 transition-all
                    ${isCompleted
                      ? guessedWords.has(word.word.toUpperCase())
                        ? 'bg-green-50 border-green-300 cursor-default'
                        : wrongWords.has(word.word.toUpperCase())
                        ? 'bg-yellow-50 border-yellow-300 cursor-default'
                        : 'bg-gray-50 border-gray-200 cursor-default'
                      : guessedWords.has(word.word.toUpperCase())
                      ? 'bg-green-50 border-green-300 cursor-pointer'
                      : selectedWord?.word === word.word && selectedWord?.direction === word.direction
                      ? 'bg-blue-50 border-blue-300 cursor-pointer'
                      : 'bg-gray-50 border-gray-200 hover:border-primary/50 cursor-pointer'
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
        {selectedWord && !isCompleted && showKeyboard && (
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
              {/* Кнопка "Проверить" удалена - проверка происходит автоматически при заполнении всех клеток кроссворда */}
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
