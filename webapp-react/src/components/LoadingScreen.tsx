import { useEffect, useRef } from 'react'
import lottie, { type AnimationItem } from 'lottie-web'

interface LoadingScreenProps {
  onComplete: () => void
}

export default function LoadingScreen({ onComplete }: LoadingScreenProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const animationRef = useRef<AnimationItem | null>(null)
  const timeoutRef = useRef<number | null>(null)

  useEffect(() => {
    if (!containerRef.current) return

    // Функция показа основного содержимого
    const showApp = () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
        timeoutRef.current = null
      }
      onComplete()
    }

    // Инициализация Lottie анимации
    const initLottieOrFallback = () => {
      if (!containerRef.current) {
        setTimeout(showApp, 300)
        return
      }

      // Загружаем Lottie анимацию из локального файла ring_animation.json
      const animationPath = '/ring_animation.json'

      try {
        const anim = lottie.loadAnimation({
          container: containerRef.current,
          renderer: 'svg',
          loop: false,
          autoplay: true,
          path: animationPath,
        })

        animationRef.current = anim

        // Обработка событий анимации
        anim.addEventListener('data_ready', () => {
          console.log('Lottie animation loaded')
        })

        // Если данные не загрузились — сразу показываем сайт
        anim.addEventListener('data_failed', () => {
          console.error('Lottie animation data_failed, showing site')
          showApp()
        })

        anim.addEventListener('complete', () => {
          // Анимация завершена, скрываем загрузчик и показываем сайт
          showApp()
        })
      } catch (error) {
        console.error('Error loading Lottie animation:', error)
        showApp()
      }

      // Жёсткий таймаут: через ~7 секунд обязательно показываем сайт,
      // даже если анимация зависла или не успела загрузиться
      timeoutRef.current = setTimeout(() => {
        console.warn('Lottie animation timeout, showing site')
        showApp()
      }, 7000)
    }

    // Стартуем Lottie/фоллбек
    initLottieOrFallback()

    return () => {
      if (animationRef.current) {
        animationRef.current.destroy()
        animationRef.current = null
      }
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
        timeoutRef.current = null
      }
    }
  }, [onComplete])

  return (
    <div
      className="fixed inset-0 z-[10000] flex items-center justify-center bg-white"
      style={{
        opacity: 1,
        transition: 'opacity 0.8s ease-out',
        pointerEvents: 'all',
      }}
    >
      <div
        ref={containerRef}
        className="lottie-container w-full max-w-[420px] aspect-square h-auto flex items-center justify-center overflow-hidden"
        style={{
          width: '100vw',
          maxWidth: '420px',
          aspectRatio: '1 / 1',
        }}
      />
    </div>
  )
}

