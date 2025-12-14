import { useState, useRef, useEffect } from 'react'
import { motion, useMotionValue, useTransform } from 'framer-motion'
import type { TabName } from '../types'
import { hapticFeedback } from '../utils/telegram'
import NavIcon from './NavIcon'

interface BottomNavbarProps {
  activeTab: TabName
  onTabChange: (tab: TabName) => void
}

// Все кнопки в одном массиве для сетки 4xN
// Первый ряд: Главная, Испытание, Меню, Сделать фото
// Второй ряд: План-сетка, Дресс-код, Рассадка, Пожелания
const allNavItems: Array<{ id: TabName; label: string; isSpecial?: boolean }> = [
  { id: 'home', label: 'Главная' },
  { id: 'challenge', label: 'Испытание' },
  { id: 'menu', label: 'Меню' },
  { id: 'photo', label: 'Сделать фото', isSpecial: true }, // Особенная кнопка
  { id: 'timeline', label: 'План-сетка' },
  { id: 'dresscode', label: 'Дресс-Код' },
  { id: 'seating', label: 'Рассадка' },
  { id: 'wishes', label: 'Пожелания' },
]

const ITEMS_PER_ROW = 4 // Количество кнопок в одном ряду
const ROW_HEIGHT = 80 // Высота одного ряда в пикселях
const DRAG_INDICATOR_HEIGHT = 24 // Высота индикатора для вытягивания
const DRAG_THRESHOLD = 30 // Порог для открытия/закрытия при перетаскивании

// Вычисляем количество рядов
const totalRows = Math.ceil(allNavItems.length / ITEMS_PER_ROW)
const visibleRows = 1 // Первый ряд (4 кнопки)
const hiddenRows = totalRows - visibleRows // Скрытые ряды

export default function BottomNavbar({ activeTab, onTabChange }: BottomNavbarProps) {
  const [isExpanded, setIsExpanded] = useState(false)
  const [isDragging, setIsDragging] = useState(false)
  const navRef = useRef<HTMLDivElement>(null)
  const startY = useRef(0)
  const currentY = useRef(0)

  const dragY = useMotionValue(0)
  const y = useTransform(dragY, (value) => {
    // Ограничиваем движение только вверх (отрицательные значения)
    const maxDelta = ROW_HEIGHT * hiddenRows
    return Math.max(-maxDelta, Math.min(0, value))
  })

  const handleTabClick = (tab: TabName) => {
    hapticFeedback('light')
    onTabChange(tab)
    // Закрываем навбар после выбора вкладки
    if (isExpanded) {
      setIsExpanded(false)
    }
    // Скролл будет сброшен в App.tsx через useEffect при изменении activeTab
  }

  const handlePointerDown = (e: React.PointerEvent) => {
    startY.current = e.clientY
    currentY.current = startY.current
    setIsDragging(true)
    e.preventDefault()
    // Для тач-устройств блокируем скролл
    if (e.pointerType === 'touch') {
      e.currentTarget.setPointerCapture(e.pointerId)
    }
  }

  // Синхронизируем состояние с анимацией
  useEffect(() => {
    if (isExpanded) {
      dragY.set(-ROW_HEIGHT * hiddenRows)
    } else {
      dragY.set(0)
    }
  }, [isExpanded, dragY])

  // Обработка глобальных событий pointer
  useEffect(() => {
    if (isDragging) {
      const handleGlobalPointerMove = (e: PointerEvent) => {
        currentY.current = e.clientY
        const deltaY = startY.current - currentY.current
        // Устанавливаем значение dragY (отрицательное при движении вверх)
        dragY.set(-deltaY)
      }

      const handleGlobalPointerUp = () => {
        setIsDragging(false)
        const deltaY = startY.current - currentY.current

        if (deltaY > DRAG_THRESHOLD) {
          // Вытягиваем навбар
          setIsExpanded(true)
          dragY.set(-ROW_HEIGHT * hiddenRows)
          hapticFeedback('light')
        } else if (deltaY < -DRAG_THRESHOLD && isExpanded) {
          // Сворачиваем навбар
          setIsExpanded(false)
          dragY.set(0)
          hapticFeedback('light')
        } else {
          // Возвращаем в исходное положение
          if (isExpanded) {
            dragY.set(-ROW_HEIGHT * hiddenRows)
          } else {
            dragY.set(0)
          }
        }
      }

      window.addEventListener('pointermove', handleGlobalPointerMove)
      window.addEventListener('pointerup', handleGlobalPointerUp)

      return () => {
        window.removeEventListener('pointermove', handleGlobalPointerMove)
        window.removeEventListener('pointerup', handleGlobalPointerUp)
      }
    }
  }, [isDragging, isExpanded, dragY])

  const renderNavButton = (item: { id: TabName; label: string; isSpecial?: boolean }) => {
    const isSpecial = item.isSpecial || false
    const isActive = activeTab === item.id
    
    if (isSpecial) {
      // Круглая кнопка для "Сделать фото"
      return (
        <motion.button
          key={item.id}
          onClick={() => handleTabClick(item.id)}
          className="flex flex-col items-center justify-center gap-0.5 h-20 min-w-0 transition-all"
          whileTap={{ scale: 0.95 }}
        >
          <motion.div
            className={`w-14 h-14 bg-[#FFE9AD] rounded-full shadow-lg hover:shadow-xl flex items-center justify-center transition-all relative ${
              isActive ? 'bg-[#F5D98A] shadow-xl ring-2 ring-[#FFE9AD] ring-offset-1' : ''
            }`}
            whileHover={{ scale: 1.05 }}
            animate={{ 
              boxShadow: isActive 
                ? '0 10px 25px rgba(255, 233, 173, 0.4)' 
                : '0 5px 15px rgba(255, 233, 173, 0.3)'
            }}
          >
            <motion.div
              className="absolute inset-0 bg-[#FFE9AD] rounded-full opacity-20"
              animate={{ 
                scale: [1, 1.1, 1],
                opacity: [0.2, 0.3, 0.2]
              }}
              transition={{ 
                duration: 2,
                repeat: Infinity,
                ease: "easeInOut"
              }}
            />
            <motion.div
              animate={{ scale: isActive ? 1.1 : 1 }}
              transition={{ duration: 0.2 }}
              className="relative z-10 w-7 h-7"
            >
              <NavIcon
                name={item.id as 'home' | 'challenge' | 'menu' | 'photo' | 'timeline' | 'dresscode' | 'seating' | 'wishes'}
                isActive={true}
                className="w-full h-full"
              />
            </motion.div>
          </motion.div>
          <span className="text-xs font-main text-primary-dark font-bold drop-shadow-sm">
            {item.label}
          </span>
        </motion.button>
      )
    }
    
    // Обычная кнопка
    return (
      <motion.button
        key={item.id}
        onClick={() => handleTabClick(item.id)}
        className="flex flex-col items-center justify-center gap-0.5 px-1 py-2 h-20 min-w-0 transition-colors"
        whileTap={{ scale: 0.95 }}
      >
        <motion.div
          animate={{ scale: isActive ? 1.1 : 1 }}
          transition={{ duration: 0.2 }}
          className="w-6 h-6"
        >
          <NavIcon
            name={item.id as 'home' | 'challenge' | 'menu' | 'photo' | 'timeline' | 'dresscode' | 'seating' | 'wishes'}
            isActive={isActive}
            className="w-full h-full"
          />
        </motion.div>
        <span
          className={`text-xs font-main transition-colors ${
            isActive
              ? 'text-primary font-semibold'
              : 'text-gray-600'
          }`}
        >
          {item.label}
        </span>
      </motion.button>
    )
  }

  // Разделяем кнопки на видимые и скрытые
  const visibleItems = allNavItems.slice(0, visibleRows * ITEMS_PER_ROW)
  const hiddenItems = allNavItems.slice(visibleRows * ITEMS_PER_ROW)

  return (
    <motion.nav
      ref={navRef}
      style={{
        y,
      }}
      animate={{
        height: isExpanded 
          ? DRAG_INDICATOR_HEIGHT + ROW_HEIGHT * totalRows 
          : DRAG_INDICATOR_HEIGHT + ROW_HEIGHT * visibleRows,
      }}
      initial={{ y: 0, height: DRAG_INDICATOR_HEIGHT + ROW_HEIGHT * visibleRows }}
      className="fixed bottom-0 left-0 right-0 z-50 bg-white border-t-2 border-primary/30 shadow-lg backdrop-blur-sm overflow-hidden"
      transition={{ type: 'spring', damping: 25, stiffness: 300 }}
    >
      {/* Индикатор для вытягивания */}
      <div
        className="w-full flex justify-center py-1 cursor-grab active:cursor-grabbing touch-none select-none"
        onPointerDown={handlePointerDown}
      >
        <div className="w-12 h-1 bg-gray-300 rounded-full" />
      </div>

      {/* Сетка кнопок 4xN */}
      <div className="grid grid-cols-4 gap-0">
        {/* Видимые кнопки (первый ряд) */}
        {visibleItems.map(renderNavButton)}
        
        {/* Скрытые кнопки (показываются при вытягивании) */}
        {isExpanded && hiddenItems.map(renderNavButton)}
      </div>
    </motion.nav>
  )
}

