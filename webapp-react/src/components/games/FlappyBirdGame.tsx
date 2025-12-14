import { useEffect, useRef, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

interface FlappyBirdGameProps {
  onScore: (score: number) => void
  onClose: () => void
}

// Реализация игры Flappy Bird
export default function FlappyBirdGame({ onScore, onClose }: FlappyBirdGameProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [score, setScore] = useState(0)
  const [gameOver, setGameOver] = useState(false)
  const [gameStarted, setGameStarted] = useState(false)
  const gameLoopRef = useRef<number>()
  const birdRef = useRef({ x: 100, y: 200, width: 40, height: 30, velocityY: 0 })
  const pipesRef = useRef<Array<{ x: number; topHeight: number; gap: number }>>([])
  const gameSpeedRef = useRef(3)
  const scoreRef = useRef(0)
  const birdFaceImageRef = useRef<HTMLImageElement | null>(null)
  const [faceImageLoaded, setFaceImageLoaded] = useState(false)

  // Загружаем изображение лица
  useEffect(() => {
    const img = new Image()
    img.onload = () => {
      console.log('Изображение лица птички загружено успешно')
      birdFaceImageRef.current = img
      setFaceImageLoaded(true)
    }
    img.onerror = (e) => {
      console.warn('Не удалось загрузить изображение лица птички, используем стандартную графику', e)
      setFaceImageLoaded(false)
    }
    // Пробуем разные варианты пути
    const imagePath = '/res/flappy_face.png'
    console.log('Пытаемся загрузить изображение:', imagePath)
    img.src = imagePath
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
      // Обновляем позицию птички при ресайзе
      if (birdRef.current.y > 0) {
        birdRef.current.y = canvas.height / 2
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
      if (!gameStarted && !gameOver) {
        setGameStarted(true)
        // Первый прыжок при старте
        birdRef.current.velocityY = -8
      }
      if (gameStarted && !gameOver) {
        birdRef.current.velocityY = -8
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

      // Рисуем фон (небо)
      ctx.fillStyle = '#87CEEB'
      ctx.fillRect(0, 0, canvas.width, canvas.height)

      // Рисуем землю
      ctx.fillStyle = '#8B4513'
      ctx.fillRect(0, canvas.height - 40, canvas.width, 40)
      // Трава на земле
      ctx.fillStyle = '#228B22'
      ctx.fillRect(0, canvas.height - 40, canvas.width, 5)

      // Получаем ссылку на птичку (объявляем один раз для всего цикла)
      const bird = birdRef.current

      // Обновляем птичку только если игра началась
      if (gameStarted) {
        bird.velocityY += 0.5 // Гравитация
        bird.y += bird.velocityY

        // Ограничения по экрану
        if (bird.y < 0) {
          bird.y = 0
          bird.velocityY = 0
        }
        if (bird.y + bird.height > canvas.height - 40) {
          setGameOver(true)
          onScore(scoreRef.current)
          return
        }

        // Обновляем трубы
        pipesRef.current = pipesRef.current.map(pipe => ({
          ...pipe,
          x: pipe.x - gameSpeedRef.current
        })).filter(pipe => pipe.x > -60)

        // Добавляем новые трубы - увеличиваем расстояние между столбами и высоту пролета
        const minDistance = 350 // Увеличено расстояние между столбами (было 200)
        const gap = 220 // Увеличена высота пролета (было 150)
        const lastPipe = pipesRef.current[pipesRef.current.length - 1]
        const distanceFromLast = lastPipe 
          ? canvas.width - lastPipe.x 
          : Infinity
        
        if (pipesRef.current.length === 0 || distanceFromLast > minDistance) {
          const topHeight = Math.random() * (canvas.height - gap - 100) + 50
          pipesRef.current.push({
            x: canvas.width,
            topHeight,
            gap
          })
        }

        // Проверка столкновений с трубами
        pipesRef.current.forEach((pipe: any) => {
          // Верхняя труба
          if (
            bird.x < pipe.x + 60 &&
            bird.x + bird.width > pipe.x &&
            bird.y < pipe.topHeight
          ) {
            setGameOver(true)
            onScore(scoreRef.current)
            return
          }
          // Нижняя труба
          if (
            bird.x < pipe.x + 60 &&
            bird.x + bird.width > pipe.x &&
            bird.y + bird.height > pipe.topHeight + pipe.gap
          ) {
            setGameOver(true)
            onScore(scoreRef.current)
            return
          }
          // Увеличиваем счет при прохождении трубы
          if (pipe.x + 60 < bird.x && !pipe.passed) {
            scoreRef.current += 1
            setScore(scoreRef.current)
            pipe.passed = true
          }
        })

        // Увеличиваем скорость
        if (scoreRef.current % 10 === 0 && scoreRef.current > 0) {
          gameSpeedRef.current = 3 + Math.floor(scoreRef.current / 10) * 0.5
        }
      } else {
        // До первого тапа птичка не падает - гравитация не применяется
        bird.velocityY = 0
      }

      // Рисуем трубы
      pipesRef.current.forEach(pipe => {
        ctx.fillStyle = '#228B22'
        // Верхняя труба
        ctx.fillRect(pipe.x, 0, 60, pipe.topHeight)
        // Нижняя труба
        ctx.fillRect(pipe.x, pipe.topHeight + pipe.gap, 60, canvas.height - (pipe.topHeight + pipe.gap))
        // Обводка труб
        ctx.strokeStyle = '#006400'
        ctx.lineWidth = 3
        ctx.strokeRect(pipe.x, 0, 60, pipe.topHeight)
        ctx.strokeRect(pipe.x, pipe.topHeight + pipe.gap, 60, canvas.height - (pipe.topHeight + pipe.gap))
      })

      // Рисуем птичку (bird уже объявлена выше)
      if (birdFaceImageRef.current && faceImageLoaded && birdFaceImageRef.current.complete) {
        // Рисуем тело птички (простой овал)
        ctx.fillStyle = '#FFD700'
        ctx.beginPath()
        ctx.ellipse(bird.x + bird.width/2, bird.y + bird.height/2, bird.width/2, bird.height/2, 0, 0, 2 * Math.PI)
        ctx.fill()
        ctx.strokeStyle = '#FFA500'
        ctx.lineWidth = 2
        ctx.stroke()

        // Рисуем лицо поверх
        const faceSize = 80 // Увеличено в 2 раза
        const faceX = bird.x + bird.width/2 - faceSize/2
        const faceY = bird.y + bird.height/2 - faceSize/2

        ctx.save()
        ctx.beginPath()
        ctx.ellipse(bird.x + bird.width/2, bird.y + bird.height/2, faceSize/2, faceSize/2, 0, 0, 2 * Math.PI)
        ctx.clip()
        ctx.drawImage(birdFaceImageRef.current, faceX, faceY, faceSize, faceSize)
        ctx.restore()

        // Крылья
        ctx.fillStyle = '#FFA500'
        ctx.beginPath()
        ctx.ellipse(bird.x - 5, bird.y + bird.height/2, 8, 15, -0.3, 0, 2 * Math.PI)
        ctx.fill()
        ctx.beginPath()
        ctx.ellipse(bird.x + bird.width + 5, bird.y + bird.height/2, 8, 15, 0.3, 0, 2 * Math.PI)
        ctx.fill()
      } else {
        // Стандартная графика птички
        ctx.fillStyle = '#FFD700'
        ctx.beginPath()
        ctx.ellipse(bird.x + bird.width/2, bird.y + bird.height/2, bird.width/2, bird.height/2, 0, 0, 2 * Math.PI)
        ctx.fill()
        ctx.strokeStyle = '#FFA500'
        ctx.lineWidth = 2
        ctx.stroke()

        // Глаз
        ctx.fillStyle = '#000'
        ctx.beginPath()
        ctx.arc(bird.x + bird.width/2 + 5, bird.y + bird.height/2 - 5, 3, 0, 2 * Math.PI)
        ctx.fill()

        // Клюв
        ctx.fillStyle = '#FF8C00'
        ctx.beginPath()
        ctx.moveTo(bird.x + bird.width, bird.y + bird.height/2)
        ctx.lineTo(bird.x + bird.width + 10, bird.y + bird.height/2 - 3)
        ctx.lineTo(bird.x + bird.width + 10, bird.y + bird.height/2 + 3)
        ctx.closePath()
        ctx.fill()

        // Крылья
        ctx.fillStyle = '#FFA500'
        ctx.beginPath()
        ctx.ellipse(bird.x - 5, bird.y + bird.height/2, 8, 15, -0.3, 0, 2 * Math.PI)
        ctx.fill()
        ctx.beginPath()
        ctx.ellipse(bird.x + bird.width + 5, bird.y + bird.height/2, 8, 15, 0.3, 0, 2 * Math.PI)
        ctx.fill()
      }

      // Показываем счет
      ctx.fillStyle = '#000'
      ctx.font = 'bold 32px Arial'
      ctx.fillText(scoreRef.current.toString(), canvas.width / 2 - 20, 50)

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
  }, [gameOver, gameStarted, onScore, faceImageLoaded])

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
                    setGameStarted(false)
                    setScore(0)
                    scoreRef.current = 0
                    gameSpeedRef.current = 3
                    const canvas = canvasRef.current
                    if (canvas) {
                      birdRef.current = { x: 100, y: canvas.height / 2, width: 40, height: 30, velocityY: 0 }
                    } else {
                      birdRef.current = { x: 100, y: 200, width: 40, height: 30, velocityY: 0 }
                    }
                    // Сбрасываем скорость
                    gameSpeedRef.current = 3
                    pipesRef.current = []
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
    </div>
  )
}

