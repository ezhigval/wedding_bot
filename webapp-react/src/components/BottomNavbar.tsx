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
const DRAG_INDICATOR_HEIGHT = 40 // Высота индикатора для вытягивания (увеличена)
const DRAG_THRESHOLD = 20 // Порог для открытия/закрытия при перетаскивании (уменьшен для более отзывчивости)
const CLICK_THRESHOLD = 5 // Порог для различения клика и драга

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
  const startTime = useRef(0)
  const hasMoved = useRef(false)

  const dragY = useMotionValue(0)
  const y = useTransform(dragY, (value) => {
    // Ограничиваем движение только вверх (отрицательные значения)
    const maxDelta = ROW_HEIGHT * hiddenRows
    return Math.max(-maxDelta, Math.min(0, value))
  })

  const handleTabClick = (tab: TabName, e?: React.MouseEvent) => {
    // Проверяем, что это был клик, а не драг
    if (hasMoved.current || isDragging) {
      e?.preventDefault()
      e?.stopPropagation()
      return
    }
    hapticFeedback('light')
    onTabChange(tab)
    // Закрываем навбар после выбора вкладки
    if (isExpanded) {
      setIsExpanded(false)
    }
    // Скролл будет сброшен в App.tsx через useEffect при изменении activeTab
  }

  const handlePointerDown = (e: React.PointerEvent) => {
    // Игнорируем клики по кнопкам - они обрабатываются отдельно
    const target = e.target as HTMLElement
    if (target.closest('button')) {
      return
    }

    startY.current = e.clientY
    currentY.current = startY.current
    startTime.current = Date.now()
    hasMoved.current = false
    setIsDragging(true)
    e.preventDefault()
    e.stopPropagation()
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
        
        // Отмечаем, что произошло движение
        if (Math.abs(deltaY) > CLICK_THRESHOLD) {
          hasMoved.current = true
        }
        
        // Устанавливаем значение dragY (отрицательное при движении вверх)
        dragY.set(-deltaY)
      }

      const handleGlobalPointerUp = () => {
        const deltaY = startY.current - currentY.current

        if (deltaY > DRAG_THRESHOLD) {
          // Вытягиваем навбар
          setIsExpanded(true)
          dragY.set(-ROW_HEIGHT * hiddenRows)
          hapticFeedback('medium')
        } else if (deltaY < -DRAG_THRESHOLD && isExpanded) {
          // Сворачиваем навбар
          setIsExpanded(false)
          dragY.set(0)
          hapticFeedback('medium')
        } else {
          // Возвращаем в исходное положение
          if (isExpanded) {
            dragY.set(-ROW_HEIGHT * hiddenRows)
          } else {
            dragY.set(0)
          }
        }
        
        setIsDragging(false)
        
        // Сбрасываем флаг движения после небольшой задержки
        setTimeout(() => {
          hasMoved.current = false
        }, 150)
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
          onClick={(e) => handleTabClick(item.id, e)}
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
        onClick={(e) => handleTabClick(item.id, e)}
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
      className="fixed bottom-0 left-0 right-0 z-50 bg-white border-t-2 border-primary/30 shadow-lg backdrop-blur-sm overflow-hidden flex flex-col justify-end"
      transition={{ type: 'spring', damping: 25, stiffness: 300 }}
      onPointerDown={handlePointerDown}
    >
      {/* Индикатор для вытягивания - увеличенный и более заметный */}
      <div className="w-full flex justify-center py-2 cursor-grab active:cursor-grabbing touch-none select-none drag-handle flex-shrink-0">
        <motion.div
          className="relative"
          animate={{
            scale: isDragging ? 1.1 : 1,
          }}
          transition={{ duration: 0.2 }}
        >
          <motion.div
            className="w-20 h-1.5 bg-gradient-to-r from-gray-400 via-gray-500 to-gray-400 rounded-full shadow-md"
            animate={{
              width: isDragging ? '6rem' : '5rem',
              opacity: isDragging ? 1 : 0.7,
              boxShadow: isDragging 
                ? '0 4px 12px rgba(0, 0, 0, 0.2)' 
                : '0 2px 6px rgba(0, 0, 0, 0.1)',
            }}
            transition={{ duration: 0.2 }}
          />
          {/* Анимированная подсказка */}
          {!isExpanded && (
            <motion.div
              className="absolute -top-6 left-1/2 transform -translate-x-1/2 text-xs text-gray-500 whitespace-nowrap"
              animate={{
                opacity: [0.5, 0.8, 0.5],
                y: [0, -2, 0],
              }}
              transition={{
                duration: 2,
                repeat: Infinity,
                ease: "easeInOut"
              }}
            >
              Потяните вверх
            </motion.div>
          )}
        </motion.div>
      </div>

      {/* Сетка кнопок 4xN */}
      <div className="grid grid-cols-4 gap-0 flex-shrink-0" style={{ gridTemplateRows: `repeat(${isExpanded ? totalRows : visibleRows}, ${ROW_HEIGHT}px)` }}>
        {/* Видимые кнопки (первый ряд) */}
        {visibleItems.map(renderNavButton)}
        
        {/* Скрытые кнопки (показываются при вытягивании) */}
        {isExpanded && hiddenItems.map(renderNavButton)}
      </div>
    </motion.nav>
  )
}

