import { useEffect, useRef, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

interface DragonGameProps {
  onScore: (score: number) => void
  onClose: () => void
}

// Простая реализация игры "Дракончик" (Chrome Dino)
export default function DragonGame({ onScore, onClose }: DragonGameProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [score, setScore] = useState(0)
  const [gameOver, setGameOver] = useState(false)
  const gameLoopRef = useRef<number>()
  const dinoRef = useRef({ x: 50, y: 0, width: 50, height: 50, velocityY: 0, jumping: false })
  const obstaclesRef = useRef<Array<{ x: number; y: number; width: number; height: number }>>([])
  const gameSpeedRef = useRef(3) // Уменьшена начальная скорость
  const scoreRef = useRef(0)
  const dinoFaceImageRef = useRef<HTMLImageElement | null>(null)
  const [faceImageLoaded, setFaceImageLoaded] = useState(false)

  // Загружаем изображение лица
  useEffect(() => {
    const img = new Image()
    img.onload = () => {
      dinoFaceImageRef.current = img
      setFaceImageLoaded(true)
    }
    img.onerror = () => {
      console.warn('Не удалось загрузить изображение лица динозавра, используем стандартную графику')
      setFaceImageLoaded(false)
    }
    // Путь к изображению - поддерживает PNG, JPEG, JPG
    // Если файл называется по-другому, измените путь здесь
    img.src = '/res/dino_face.png'
  }, [])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return

    const ctx = canvas.getContext('2d')
    if (!ctx) return

    // Функция обновления размера canvas
    const updateCanvasSize = () => {
      canvas.width = window.innerWidth
      canvas.height = window.innerHeight - 80 // Вычитаем высоту навбара
      // Обновляем позицию динозавра при ресайзе
      const groundY = canvas.height / 2 - dinoRef.current.height
      if (dinoRef.current.y > 0) {
        dinoRef.current.y = groundY
      }
    }

    // Настройка canvas - 100% ширины и высоты экрана
    updateCanvasSize()

    // Обработчик ресайза
    const handleResize = () => {
      updateCanvasSize()
    }
    window.addEventListener('resize', handleResize)

    // Обработка прыжка
    const handleJump = () => {
      if (!dinoRef.current.jumping && !gameOver) {
        dinoRef.current.jumping = true
        dinoRef.current.velocityY = -15
      }
    }

    const handleKeyPress = (e: KeyboardEvent) => {
      if (e.code === 'Space' || e.key === 'ArrowUp' || e.key === ' ') {
        e.preventDefault()
        handleJump()
      }
    }

    const handleTouch = (e: TouchEvent | MouseEvent) => {
      e.preventDefault()
      handleJump()
    }

    // Обработчики на весь экран для удобства тапов
    const container = canvas.parentElement
    window.addEventListener('keydown', handleKeyPress)
    if (container) {
      container.addEventListener('touchstart', handleTouch, { passive: false })
      container.addEventListener('click', handleTouch)
    }

    // Игровой цикл
    const gameLoop = () => {
      if (gameOver) return

      ctx.clearRect(0, 0, canvas.width, canvas.height)

      // Рисуем небо (верхняя половина)
      ctx.fillStyle = '#87CEEB'
      ctx.fillRect(0, 0, canvas.width, canvas.height / 2)

      // Рисуем землю (нижняя половина)
      ctx.fillStyle = '#8B4513'
      const groundHeight = canvas.height / 2
      ctx.fillRect(0, canvas.height / 2, canvas.width, groundHeight)
      
      // Трава на границе
      ctx.fillStyle = '#228B22'
      ctx.fillRect(0, canvas.height / 2, canvas.width, 5)

      // Обновляем динозавра
      const dino = dinoRef.current
      const groundY = canvas.height / 2 - dino.height // Динозавр на границе земли
      
      // Инициализация позиции при первом запуске
      if (dino.y === 0) {
        dino.y = groundY
      }
      
      dino.velocityY += 0.8 // Гравитация
      dino.y += dino.velocityY

      // Ограничение по земле
      if (dino.y >= groundY) {
        dino.y = groundY
        dino.velocityY = 0
        dino.jumping = false
      }

      // Рисуем динозавра
      const dinoX = dino.x
      const dinoY = dino.y
      const dinoW = dino.width
      const dinoH = dino.height
      
      // Если есть изображение лица, используем его
      if (dinoFaceImageRef.current && faceImageLoaded) {
        // Минимальная графика - простые формы
        ctx.fillStyle = '#535353'
        
        // Простое тело (прямоугольник)
        ctx.fillRect(dinoX, dinoY + 20, dinoW, dinoH - 20)
        
        // Рисуем лицо поверх головы
        const faceSize = 60 // Размер области лица (увеличено в 2 раза)
        const faceX = dinoX + dinoW/2 + 8 - faceSize/2
        const faceY = dinoY + 15 - faceSize/2
        
        // Сначала рисуем круглую голову как фон
        ctx.fillStyle = '#535353'
        ctx.beginPath()
        ctx.arc(dinoX + dinoW/2 + 8, dinoY + 15, faceSize/2, 0, 2 * Math.PI)
        ctx.fill()
        
        // Затем рисуем лицо поверх
        ctx.save()
        ctx.beginPath()
        ctx.arc(dinoX + dinoW/2 + 8, dinoY + 15, faceSize/2, 0, 2 * Math.PI)
        ctx.clip()
        ctx.drawImage(dinoFaceImageRef.current, faceX, faceY, faceSize, faceSize)
        ctx.restore()
        
        // Простые лапы
        ctx.fillRect(dinoX + 8, dinoY + dinoH - 12, 8, 12)
        ctx.fillRect(dinoX + 32, dinoY + dinoH - 12, 8, 12)
      } else {
        // Стандартная графика, если изображение не загружено
        ctx.fillStyle = '#535353'
        ctx.strokeStyle = '#535353'
        ctx.lineWidth = 2
        
        // Тело (овальное)
        ctx.beginPath()
        ctx.ellipse(dinoX + dinoW/2, dinoY + dinoH/2 + 5, dinoW/2 - 5, dinoH/2 - 5, 0, 0, 2 * Math.PI)
        ctx.fill()
        
        // Голова (округлая)
        ctx.beginPath()
        ctx.arc(dinoX + dinoW/2 + 8, dinoY + 15, 12, 0, 2 * Math.PI)
        ctx.fill()
        
        // Глаз
        ctx.fillStyle = '#fff'
        ctx.beginPath()
        ctx.arc(dinoX + dinoW/2 + 10, dinoY + 13, 3, 0, 2 * Math.PI)
        ctx.fill()
        
        // Рот
        ctx.fillStyle = '#535353'
        ctx.beginPath()
        ctx.arc(dinoX + dinoW/2 + 12, dinoY + 18, 4, 0, Math.PI, false)
        ctx.stroke()
        
        // Шея
        ctx.fillStyle = '#535353'
        ctx.fillRect(dinoX + dinoW/2 - 3, dinoY + 20, 6, 8)
        
        // Передняя лапа
        ctx.fillStyle = '#535353'
        ctx.fillRect(dinoX + 8, dinoY + dinoH - 15, 10, 12)
        // Пальцы передней лапы
        ctx.fillRect(dinoX + 6, dinoY + dinoH - 3, 3, 5)
        ctx.fillRect(dinoX + 10, dinoY + dinoH - 3, 3, 5)
        ctx.fillRect(dinoX + 14, dinoY + dinoH - 3, 3, 5)
        
        // Задняя лапа
        ctx.fillRect(dinoX + 30, dinoY + dinoH - 12, 10, 12)
        // Пальцы задней лапы
        ctx.fillRect(dinoX + 28, dinoY + dinoH - 3, 3, 5)
        ctx.fillRect(dinoX + 32, dinoY + dinoH - 3, 3, 5)
        ctx.fillRect(dinoX + 36, dinoY + dinoH - 3, 3, 5)
        
        // Хвост
        ctx.beginPath()
        ctx.moveTo(dinoX - 5, dinoY + dinoH/2 + 5)
        ctx.lineTo(dinoX - 15, dinoY + dinoH/2 - 5)
        ctx.lineTo(dinoX - 10, dinoY + dinoH/2 + 5)
        ctx.closePath()
        ctx.fill()
        
        // Спинной гребень (небольшие треугольники)
        ctx.beginPath()
        ctx.moveTo(dinoX + 15, dinoY + 10)
        ctx.lineTo(dinoX + 20, dinoY + 5)
        ctx.lineTo(dinoX + 25, dinoY + 10)
        ctx.closePath()
        ctx.fill()
        
        ctx.beginPath()
        ctx.moveTo(dinoX + 25, dinoY + 15)
        ctx.lineTo(dinoX + 30, dinoY + 10)
        ctx.lineTo(dinoX + 35, dinoY + 15)
        ctx.closePath()
        ctx.fill()
      }

      // Обновляем препятствия
      obstaclesRef.current = obstaclesRef.current.map(obstacle => ({
        ...obstacle,
        x: obstacle.x - gameSpeedRef.current
      })).filter(obstacle => obstacle.x > -obstacle.width)

      // Добавляем новые препятствия - более предсказуемая логика
      // Минимальное расстояние между кактусами
      const minDistance = 300
      const lastObstacle = obstaclesRef.current[obstaclesRef.current.length - 1]
      const distanceFromLast = lastObstacle 
        ? canvas.width - lastObstacle.x 
        : Infinity
      
      // Если нет препятствий или последнее достаточно далеко, добавляем новое
      if (obstaclesRef.current.length === 0 || distanceFromLast > minDistance) {
        // Вероятность увеличивается со временем и уменьшается при наличии препятствий
        const baseProbability = obstaclesRef.current.length === 0 ? 0.02 : 0.01
        const speedFactor = Math.min(gameSpeedRef.current / 3, 2) // Увеличиваем вероятность с ростом скорости
        const probability = baseProbability * speedFactor
        
        if (Math.random() < probability) {
          obstaclesRef.current.push({
            x: canvas.width,
            y: canvas.height / 2 - 35, // Препятствия на границе земли
            width: 20,
            height: 35
          })
        }
      }

      // Рисуем препятствия (кактусы) на границе земли
      obstaclesRef.current.forEach(obstacle => {
        ctx.fillStyle = '#535353'
        ctx.strokeStyle = '#3a3a3a'
        ctx.lineWidth = 1
        
        const cactusX = obstacle.x
        const cactusY = obstacle.y
        const cactusW = obstacle.width
        const cactusH = obstacle.height
        
        // Основной ствол
        ctx.fillRect(cactusX + 6, cactusY, cactusW - 12, cactusH)
        
        // Левая ветка (верхняя)
        ctx.fillRect(cactusX, cactusY + 8, 8, 12)
        // Левая ветка (нижняя)
        ctx.fillRect(cactusX + 2, cactusY + cactusH - 15, 6, 10)
        
        // Правая ветка (верхняя)
        ctx.fillRect(cactusX + cactusW - 8, cactusY + 10, 8, 12)
        // Правая ветка (нижняя)
        ctx.fillRect(cactusX + cactusW - 8, cactusY + cactusH - 12, 6, 8)
        
        // Колючки (маленькие линии)
        ctx.strokeStyle = '#2a2a2a'
        ctx.lineWidth = 1
        // Левая сторона
        ctx.beginPath()
        ctx.moveTo(cactusX + 6, cactusY + 5)
        ctx.lineTo(cactusX + 2, cactusY + 5)
        ctx.stroke()
        ctx.beginPath()
        ctx.moveTo(cactusX + 6, cactusY + 15)
        ctx.lineTo(cactusX + 2, cactusY + 15)
        ctx.stroke()
        // Правая сторона
        ctx.beginPath()
        ctx.moveTo(cactusX + cactusW - 6, cactusY + 8)
        ctx.lineTo(cactusX + cactusW - 2, cactusY + 8)
        ctx.stroke()
        ctx.beginPath()
        ctx.moveTo(cactusX + cactusW - 6, cactusY + 18)
        ctx.lineTo(cactusX + cactusW - 2, cactusY + 18)
        ctx.stroke()
      })

      // Проверка столкновений
      obstaclesRef.current.forEach(obstacle => {
        if (
          dino.x < obstacle.x + obstacle.width &&
          dino.x + dino.width > obstacle.x &&
          dino.y < obstacle.y + obstacle.height &&
          dino.y + dino.height > obstacle.y
        ) {
          setGameOver(true)
          onScore(scoreRef.current)
          return
        }
      })

      // Увеличиваем счет
      scoreRef.current += 1
      setScore(scoreRef.current)

      // Увеличиваем скорость медленнее и реже
      // Каждые 200 очков увеличиваем на 0.3 (вместо каждые 100 на 0.5)
      if (scoreRef.current > 0 && scoreRef.current % 200 === 0) {
        gameSpeedRef.current = Math.min(gameSpeedRef.current + 0.3, 8) // Максимальная скорость 8
      }

      gameLoopRef.current = requestAnimationFrame(gameLoop)
    }

    gameLoopRef.current = requestAnimationFrame(gameLoop)

    return () => {
      window.removeEventListener('keydown', handleKeyPress)
      window.removeEventListener('resize', handleResize)
      const container = canvas.parentElement
      if (container) {
        container.removeEventListener('touchstart', handleTouch)
        container.removeEventListener('click', handleTouch)
      }
      if (gameLoopRef.current) {
        cancelAnimationFrame(gameLoopRef.current)
      }
    }
  }, [gameOver, onScore, faceImageLoaded])

  return (
    <div className="fixed inset-0 z-50 bg-[#F8F8F8] flex flex-col" style={{ bottom: '80px' }}>
      {/* Кнопка назад */}
      <div className="absolute top-4 left-4 z-10">
        <motion.button
          onClick={() => {
            // Обновляем счет перед выходом
            if (scoreRef.current > 0) {
              onScore(scoreRef.current)
            }
            onClose()
          }}
          whileTap={{ scale: 0.95 }}
          className="px-4 py-2 bg-primary text-white rounded-lg font-semibold shadow-lg"
        >
          ← Назад
        </motion.button>
      </div>

      {/* Счет */}
      <div className="absolute top-4 right-4 z-10">
        <div className="px-4 py-2 bg-white/90 backdrop-blur-sm rounded-lg shadow-lg border-2 border-primary/30">
          <div className="text-sm text-gray-600">Счет</div>
          <div className="text-2xl font-bold text-primary">{score}</div>
        </div>
      </div>

      {/* Игровое поле - на весь экран */}
      <div className="flex-1 overflow-hidden" style={{ width: '100%', height: '100%' }}>
        <canvas
          ref={canvasRef}
          className="block"
          style={{ width: '100%', height: '100%', display: 'block' }}
        />
      </div>

      {/* Экран окончания игры */}
      <AnimatePresence>
        {gameOver && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0 bg-black/50 flex items-center justify-center z-20"
          >
            <motion.div
              initial={{ scale: 0.8, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              className="bg-white rounded-lg p-8 text-center max-w-md mx-4"
            >
              <h2 className="text-3xl font-bold text-primary mb-4">Игра окончена!</h2>
              <p className="text-xl text-gray-700 mb-2">Ваш счет: <span className="font-bold text-primary">{score}</span></p>
              <div className="flex gap-3 mt-6">
                <button
                  onClick={() => {
                    // Обновляем счет перед перезапуском
                    if (scoreRef.current > 0) {
                      onScore(scoreRef.current)
                    }
                    setGameOver(false)
                    setScore(0)
                    scoreRef.current = 0
                    gameSpeedRef.current = 3 // Сбрасываем на начальную скорость
                    const canvas = canvasRef.current
                    if (canvas) {
                      const groundY = canvas.height / 2 - 50
                      dinoRef.current = { x: 50, y: groundY, width: 50, height: 50, velocityY: 0, jumping: false }
                    } else {
                      dinoRef.current = { x: 50, y: 0, width: 50, height: 50, velocityY: 0, jumping: false }
                    }
                    obstaclesRef.current = []
                  }}
                  className="flex-1 px-4 py-2 bg-primary text-white rounded-lg font-semibold"
                >
                  Играть снова
                </button>
                <button
                  onClick={() => {
                    // Обновляем счет перед выходом
                    if (scoreRef.current > 0) {
                      onScore(scoreRef.current)
                    }
                    onClose()
                  }}
                  className="flex-1 px-4 py-2 bg-gray-300 text-gray-700 rounded-lg font-semibold"
                >
                  Выйти
                </button>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Инструкция */}
      {!gameOver && score === 0 && (
        <div className="absolute bottom-4 left-1/2 transform -translate-x-1/2 px-4 w-full max-w-md">
          <div className="bg-white/90 backdrop-blur-sm rounded-lg p-3 shadow-lg border-2 border-primary/30">
            <p className="text-sm text-center text-gray-600">
              Нажмите пробел или коснитесь экрана, чтобы прыгнуть
            </p>
          </div>
        </div>
      )}
    </div>
  )
}

