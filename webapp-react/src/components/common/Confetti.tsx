import { useEffect, useRef } from 'react'

interface ConfettiProps {
  trigger: boolean
  duration?: number
}

interface Particle {
  x: number
  y: number
  vx: number
  vy: number
  size: number
  color: string
  rotation: number
  rotationSpeed: number
}

export default function Confetti({ trigger, duration = 2000 }: ConfettiProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const particlesRef = useRef<Particle[]>([])
  const animationFrameRef = useRef<number>()

  // Цвета из палитры проекта
  const colors = ['#5A7C52', '#FFE9AD', '#DBD0C4', '#C8A067', '#4A6B42']

  useEffect(() => {
    if (!trigger) return

    const canvas = canvasRef.current
    if (!canvas) return

    const ctx = canvas.getContext('2d')
    if (!ctx) return

    // Устанавливаем размер canvas
    canvas.width = window.innerWidth
    canvas.height = window.innerHeight

    // Создаем частицы
    const particleCount = 100
    particlesRef.current = []

    for (let i = 0; i < particleCount; i++) {
      particlesRef.current.push({
        x: canvas.width / 2,
        y: canvas.height / 2,
        vx: (Math.random() - 0.5) * 8,
        vy: (Math.random() - 0.5) * 8 - 2, // Немного вверх
        size: Math.random() * 8 + 4,
        color: colors[Math.floor(Math.random() * colors.length)],
        rotation: Math.random() * Math.PI * 2,
        rotationSpeed: (Math.random() - 0.5) * 0.2,
      })
    }

    let startTime = Date.now()

    const animate = () => {
      const elapsed = Date.now() - startTime
      if (elapsed > duration) {
        particlesRef.current = []
        return
      }

      ctx.clearRect(0, 0, canvas.width, canvas.height)

      particlesRef.current.forEach((particle) => {
        // Обновляем позицию
        particle.x += particle.vx
        particle.y += particle.vy
        particle.vy += 0.15 // Гравитация
        particle.rotation += particle.rotationSpeed

        // Рисуем частицу
        ctx.save()
        ctx.translate(particle.x, particle.y)
        ctx.rotate(particle.rotation)
        ctx.fillStyle = particle.color
        ctx.fillRect(-particle.size / 2, -particle.size / 2, particle.size, particle.size)
        ctx.restore()
      })

      animationFrameRef.current = requestAnimationFrame(animate)
    }

    animate()

    return () => {
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current)
      }
    }
  }, [trigger, duration])

  if (!trigger) return null

  return (
    <canvas
      ref={canvasRef}
      className="fixed inset-0 pointer-events-none z-50"
      style={{ top: 0, left: 0 }}
    />
  )
}

