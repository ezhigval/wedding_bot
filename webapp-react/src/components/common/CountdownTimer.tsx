import { useEffect, useRef } from 'react'
import { motion } from 'framer-motion'
import SectionCard from './SectionCard'
import SectionTitle from './SectionTitle'

interface CountdownTimerProps {
  weddingDate: string
}

export default function CountdownTimer({ weddingDate }: CountdownTimerProps) {
  const timerContainerRef = useRef<HTMLDivElement>(null)
  const timerIdRef = useRef<string>('')

  useEffect(() => {
    if (!timerContainerRef.current) return

    // Генерируем уникальный ID (как в оригинальном скрипте)
    const baseId = 'bf7a52277f481c3926925137e05d3351'
    let uniqueId = baseId
    while (document.getElementById('timer' + uniqueId)) {
      uniqueId = uniqueId + '0'
    }
    timerIdRef.current = uniqueId

    // Создаем контейнер для таймера (как в оригинале)
    const timerDiv = document.createElement('div')
    timerDiv.id = 'timer' + uniqueId
    // Адаптивные размеры - на мобильных уменьшаем
    const isMobile = window.innerWidth < 768
    timerDiv.style.minWidth = isMobile ? '100%' : '486px'
    timerDiv.style.width = isMobile ? '100%' : 'auto'
    timerDiv.style.height = '93px'
    timerDiv.style.maxWidth = '100%'
    timerContainerRef.current.appendChild(timerDiv)

    // Вычисляем UTC время для даты свадьбы
    const targetDate = new Date(weddingDate)
    const utcTime = targetDate.getTime()

    // Загружаем скрипт timer.min.js
    const script = document.createElement('script')
    script.src = '//megatimer.ru/timer/timer.min.js?v=1'
    
    // Функция инициализации (как в оригинале)
    const initTimer = (run: boolean = false) => {
      // Проверяем, что MegaTimer доступен
      if (typeof (window as any).MegaTimer === 'undefined') {
        console.error('MegaTimer не загружен')
        return
      }

      const MegaTimer = (window as any).MegaTimer
      const timer = new MegaTimer(uniqueId, {
        view: [1, 1, 1, 1], // Показываем все единицы времени
        type: {
          currentType: '1', // Обратный отсчет
          params: {
            usertime: true,
            tz: '3',
            utc: utcTime
          }
        },
        design: {
          type: 'plate',
          params: {
            round: '14',
            background: 'solid',
            'background-color': '#FFE9AD', // Наш cream цвет
            effect: 'flipchart', // Split-flap эффект
            space: '6',
            'separator-margin': '5',
            'number-font-family': {
              family: 'Cormorant Garamond',
              link: ''
            },
            'number-font-size': '60',
            'number-font-color': '#5A7C52', // Наш primary цвет
            padding: '9',
            'separator-on': true,
            'separator-text': ':',
            'text-on': true,
            'text-font-family': {
              family: 'Cormorant Garamond',
              link: ''
            },
            'text-font-size': '13',
            'text-font-color': '#5A7C52' // Наш primary цвет
          }
        },
        designId: 3,
        theme: 'white',
        width: window.innerWidth < 768 ? Math.min(window.innerWidth - 32, 486) : 486, // Адаптивная ширина
        height: 93
      })

      if (run) {
        timer.run()
      }
    }

    // Обработчики загрузки скрипта (как в оригинале)
    script.onload = () => initTimer(true)
    // Поддержка старых браузеров (IE)
    ;(script as any).onreadystatechange = function () {
      if ((script as any).readyState === 'loaded') {
        initTimer(true)
      }
    }

    // Добавляем скрипт в head
    const head = document.head || document.getElementsByTagName('head')[0]
    head.appendChild(script)

    return () => {
      // Очистка при размонтировании
      if (script.parentNode) {
        script.parentNode.removeChild(script)
      }
      if (timerDiv.parentNode) {
        timerDiv.parentNode.removeChild(timerDiv)
      }
    }
  }, [weddingDate])

  // Форматируем дату для отображения
  const formatDate = (dateString: string) => {
    const date = new Date(dateString)
    const monthNames = [
      'Январь',
      'Февраль',
      'Март',
      'Апрель',
      'Май',
      'Июнь',
      'Июль',
      'Август',
      'Сентябрь',
      'Октябрь',
      'Ноябрь',
      'Декабрь',
    ]
    const day = String(date.getDate()).padStart(2, '0')
    const year = date.getFullYear()
    return `${monthNames[date.getMonth()]} ${day} ${year}`
  }

  return (
    <section className="px-4 py-0 pb-0">
      <SectionCard>
        <SectionTitle>ДО СВАДЬБЫ ОСТАЛОСЬ</SectionTitle>
        
        {/* Контейнер для MegaTimer */}
        <div 
          ref={timerContainerRef}
          className="flex flex-col items-center justify-center w-full overflow-visible"
          style={{ minHeight: '93px', padding: '44px 0', marginBottom: '76px' }}
        />

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.5, delay: 0.2 }}
          className="text-center p-[3.2px] bg-cream/85 backdrop-blur-sm rounded-lg shadow-md"
          style={{ marginTop: '76px' }}
        >
          <div className="text-xl md:text-2xl font-secondary font-semibold text-primary leading-[1.2]">
            {formatDate(weddingDate)}
          </div>
        </motion.div>
      </SectionCard>

      <style>{`
        /* Стили для адаптивного отображения таймера */
        [id^="timerbf7a52277f481c3926925137e05d3351"] {
          max-width: 100% !important;
          width: 100% !important;
          overflow: visible !important;
        }
        
        @media (max-width: 768px) {
          [id^="timerbf7a52277f481c3926925137e05d3351"] {
            transform: scale(0.8);
            transform-origin: center;
          }
        }
        
        @media (max-width: 480px) {
          [id^="timerbf7a52277f481c3926925137e05d3351"] {
            transform: scale(0.65);
            transform-origin: center;
          }
        }
      `}</style>
    </section>
  )
}

