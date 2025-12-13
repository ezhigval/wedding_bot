import { motion } from 'framer-motion'
import type { TabName } from '../types'
import { hapticFeedback } from '../utils/telegram'

interface BottomNavbarProps {
  activeTab: TabName
  onTabChange: (tab: TabName) => void
}

const navItems: Array<{ id: TabName; icon: string; label: string }> = [
  { id: 'home', icon: '🏠', label: 'Главная' },
  { id: 'dresscode', icon: '👔', label: 'Дресс-Код' },
  { id: 'timeline', icon: '📅', label: 'План-сетка' },
  { id: 'seating', icon: '🪑', label: 'Рассадка' },
  { id: 'menu', icon: '📋', label: 'Меню' },
]

export default function BottomNavbar({ activeTab, onTabChange }: BottomNavbarProps) {
  const handleTabClick = (tab: TabName) => {
    hapticFeedback('light')
    onTabChange(tab)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  return (
    <motion.nav
      initial={{ y: 100 }}
      animate={{ y: 0 }}
      className="fixed bottom-0 left-0 right-0 z-50 bg-white border-t-2 border-primary/30 shadow-lg backdrop-blur-sm"
    >
      <div className="flex justify-around items-center px-2 py-2">
        {navItems.map((item) => (
          <motion.button
            key={item.id}
            onClick={() => handleTabClick(item.id)}
            className="flex flex-col items-center justify-center gap-0.5 px-1 py-0.5 flex-1 min-w-0 transition-colors"
            whileTap={{ scale: 0.95 }}
          >
            <motion.span
              className="text-2xl leading-none"
              animate={{ scale: activeTab === item.id ? 1.1 : 1 }}
              transition={{ duration: 0.2 }}
            >
              {item.icon}
            </motion.span>
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
        ))}
      </div>
    </motion.nav>
  )
}

