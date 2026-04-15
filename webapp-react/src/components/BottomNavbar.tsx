import { motion } from 'framer-motion'
import type { TabName } from '../types'
import { hapticFeedback } from '../utils/telegram'
import NavIcon from './NavIcon'

interface BottomNavbarProps {
  activeTab: TabName
  onTabChange: (tab: TabName) => void
}

// Все кнопки в одном массиве для сетки 4xN
// Первый ряд: Главная, План-сетка, Дресс-код, Рассадка
// Второй ряд: Пожелания, Фотоаппарат, Испытание, Меню
const allNavItems: Array<{ id: TabName; label: string; isSpecial?: boolean }> = [
  { id: 'home', label: 'Главная' },
  { id: 'timeline', label: 'План-сетка' },
  { id: 'dresscode', label: 'Дресс-Код' },
  { id: 'seating', label: 'Рассадка' },
  { id: 'wishes', label: 'Пожелания' },
  { id: 'photo', label: 'Фотоаппарат' },
  { id: 'challenge', label: 'Испытание' },
  { id: 'menu', label: 'Меню' },
]

export default function BottomNavbar({ activeTab, onTabChange }: BottomNavbarProps) {
  const handleTabClick = (tab: TabName, e?: React.MouseEvent) => {
    e?.preventDefault()
    hapticFeedback('light')
    onTabChange(tab)
    // Скролл будет сброшен в App.tsx через useEffect при изменении activeTab
  }

  const renderNavButton = (item: { id: TabName; label: string; isSpecial?: boolean }) => {
    const isActive = activeTab === item.id
    
    // Обычная кнопка для всех элементов
    return (
      <motion.button
        key={item.id}
        onClick={(e) => handleTabClick(item.id, e)}
        className="flex flex-col items-center justify-center gap-0.5 px-1 py-1 h-12 min-w-0 transition-colors"
        whileTap={{ scale: 0.95 }}
      >
        <motion.div
          animate={{ scale: isActive ? 1.1 : 1 }}
          transition={{ duration: 0.2 }}
          className="w-5 h-5"
        >
          <NavIcon
            name={item.id as 'home' | 'challenge' | 'menu' | 'photo' | 'timeline' | 'dresscode' | 'seating' | 'wishes'}
            isActive={isActive}
            className="w-full h-full"
          />
        </motion.div>
        <span
          className={`text-[10px] font-main transition-colors ${
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

  return (
    <nav
      className="fixed bottom-0 left-0 right-0 z-50 bg-white/95 border-t-2 border-primary/30 shadow-lg backdrop-blur-sm"
      style={{
        paddingBottom: 'calc(env(safe-area-inset-bottom, 0px) + 6px)',
      }}
    >
      {/* Сетка кнопок 4x2 всегда видимая */}
      <div className="grid grid-cols-4 gap-0 px-1 pt-1">
        {allNavItems.map(renderNavButton)}
      </div>
    </nav>
  )
}
