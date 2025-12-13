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
const allNavItems: Array<{ id: TabName; label: string }> = [
  { id: 'home', label: 'Главная' },
  { id: 'dresscode', label: 'Дресс-Код' },
  { id: 'timeline', label: 'План-сетка' },
  { id: 'menu', label: 'Меню' },
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
    window.scrollTo({ top: 0, behavior: 'smooth' })
    // Закрываем навбар после выбора вкладки
    if (isExpanded) {
      setIsExpanded(false)
    }
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

  const renderNavButton = (item: { id: TabName; label: string }) => (
    <motion.button
      key={item.id}
      onClick={() => handleTabClick(item.id)}
      className="flex flex-col items-center justify-center gap-0.5 px-1 py-2 h-20 min-w-0 transition-colors"
      whileTap={{ scale: 0.95 }}
    >
      <motion.div
        animate={{ scale: activeTab === item.id ? 1.1 : 1 }}
        transition={{ duration: 0.2 }}
        className="w-6 h-6"
      >
        <NavIcon
          name={item.id as 'home' | 'dresscode' | 'timeline' | 'seating' | 'menu' | 'wishes'}
          isActive={activeTab === item.id}
          className="w-full h-full"
        />
      </motion.div>
      <span
        className={`text-xs font-main transition-colors ${
          activeTab === item.id
            ? 'text-primary font-semibold'
            : 'text-gray-600'
        }`}
      >
        {item.label}
      </span>
    </motion.button>
  )

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
        {/* Видимые кнопки */}
        {visibleItems.map(renderNavButton)}
        
        {/* Скрытые кнопки (показываются при вытягивании) */}
        {hiddenItems.length > 0 && (
          <>
            {isExpanded && hiddenItems.map(renderNavButton)}
          </>
        )}
      </div>
    </motion.nav>
  )
}

