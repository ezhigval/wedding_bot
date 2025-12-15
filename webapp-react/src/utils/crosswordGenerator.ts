// Утилита для автоматического составления кроссворда

export interface CrosswordWord {
  word: string
  description: string
  row: number
  col: number
  direction: 'across' | 'down'
  number: number
}

export interface CrosswordGrid {
  size: number
  grid: string[][] // Пустая строка = пустая клетка, буква = буква, '#' = черная клетка
  words: CrosswordWord[]
}

/**
 * Автоматически составляет кроссворд из списка слов
 */
export function generateCrossword(words: Array<{ word: string; description: string }>): CrosswordGrid {
  // Оптимальный размер для мобильных (5+ дюймов) - 12x12
  const size = 12
  const grid: string[][] = Array(size).fill(null).map(() => Array(size).fill(''))
  
  // Сортируем слова по длине (самые длинные первыми)
  const sortedWords = [...words].sort((a, b) => b.word.length - a.word.length)
  
  const placedWords: CrosswordWord[] = []
  let wordNumber = 1
  
  // Размещаем первое слово по центру горизонтально
  if (sortedWords.length > 0) {
    const firstWord = sortedWords[0]
    const startCol = Math.floor((size - firstWord.word.length) / 2)
    const startRow = Math.floor(size / 2)
    
    for (let i = 0; i < firstWord.word.length; i++) {
      grid[startRow][startCol + i] = firstWord.word[i]
    }
    
    placedWords.push({
      word: firstWord.word,
      description: firstWord.description,
      row: startRow,
      col: startCol,
      direction: 'across',
      number: wordNumber++
    })
    
    sortedWords.shift()
  }
  
  // Размещаем остальные слова, пересекаясь с уже размещенными
  for (const wordData of sortedWords) {
    const word = wordData.word
    let placed = false
    
    // Пробуем разместить слово
    for (const placedWord of placedWords) {
      // Ищем общие буквы
      for (let i = 0; i < word.length; i++) {
        for (let j = 0; j < placedWord.word.length; j++) {
          if (word[i] === placedWord.word[j]) {
            // Пробуем разместить перпендикулярно
            let newRow: number
            let newCol: number
            let newDirection: 'across' | 'down'
            
            if (placedWord.direction === 'across') {
              // Размещаем вертикально
              newRow = placedWord.row - i
              newCol = placedWord.col + j
              newDirection = 'down'
              
              // Проверяем, что слово помещается
              if (newRow >= 0 && newRow + word.length <= size && newCol >= 0 && newCol < size) {
                // Проверяем, что не пересекается с другими словами (кроме точки пересечения)
                let canPlace = true
                for (let k = 0; k < word.length; k++) {
                  const checkRow = newRow + k
                  const checkCol = newCol
                  
                  if (k !== i) { // Не проверяем точку пересечения
                    if (grid[checkRow][checkCol] !== '' && grid[checkRow][checkCol] !== word[k]) {
                      canPlace = false
                      break
                    }
                    // Проверяем, что слева и справа нет букв (для вертикального слова)
                    if (checkCol > 0 && grid[checkRow][checkCol - 1] !== '') {
                      canPlace = false
                      break
                    }
                    if (checkCol < size - 1 && grid[checkRow][checkCol + 1] !== '') {
                      canPlace = false
                      break
                    }
                  }
                }
                
                if (canPlace) {
                  // Размещаем слово
                  for (let k = 0; k < word.length; k++) {
                    grid[newRow + k][newCol] = word[k]
                  }
                  
                  placedWords.push({
                    word: word,
                    description: wordData.description,
                    row: newRow,
                    col: newCol,
                    direction: newDirection,
                    number: wordNumber++
                  })
                  
                  placed = true
                  break
                }
              }
            } else {
              // Размещаем горизонтально
              newRow = placedWord.row + j
              newCol = placedWord.col - i
              newDirection = 'across'
              
              // Проверяем, что слово помещается
              if (newCol >= 0 && newCol + word.length <= size && newRow >= 0 && newRow < size) {
                // Проверяем, что не пересекается с другими словами
                let canPlace = true
                for (let k = 0; k < word.length; k++) {
                  const checkRow = newRow
                  const checkCol = newCol + k
                  
                  if (k !== i) { // Не проверяем точку пересечения
                    if (grid[checkRow][checkCol] !== '' && grid[checkRow][checkCol] !== word[k]) {
                      canPlace = false
                      break
                    }
                    // Проверяем, что сверху и снизу нет букв (для горизонтального слова)
                    if (checkRow > 0 && grid[checkRow - 1][checkCol] !== '') {
                      canPlace = false
                      break
                    }
                    if (checkRow < size - 1 && grid[checkRow + 1][checkCol] !== '') {
                      canPlace = false
                      break
                    }
                  }
                }
                
                if (canPlace) {
                  // Размещаем слово
                  for (let k = 0; k < word.length; k++) {
                    grid[newRow][newCol + k] = word[k]
                  }
                  
                  placedWords.push({
                    word: word,
                    description: wordData.description,
                    row: newRow,
                    col: newCol,
                    direction: newDirection,
                    number: wordNumber++
                  })
                  
                  placed = true
                  break
                }
              }
            }
            
            if (placed) break
          }
        }
        if (placed) break
      }
      if (placed) break
    }
    
    // Если не удалось разместить, пропускаем слово
    if (!placed) {
      console.warn(`Не удалось разместить слово: ${word}`)
    }
  }
  
  return {
    size,
    grid,
    words: placedWords
  }
}

