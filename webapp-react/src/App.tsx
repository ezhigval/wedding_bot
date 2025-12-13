import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import BottomNavbar from './components/BottomNavbar'
import HomeTab from './components/tabs/HomeTab'
import DresscodeTab from './components/tabs/DresscodeTab'
import TimelineTab from './components/tabs/TimelineTab'
import SeatingTab from './components/tabs/SeatingTab'
import MenuTab from './components/tabs/MenuTab'
import LoadingScreen from './components/LoadingScreen'
import type { TabName } from './types'

function App() {
  const [activeTab, setActiveTab] = useState<TabName>('home')
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    // Симуляция загрузки (можно добавить реальную загрузку данных)
    const timer = setTimeout(() => {
      setIsLoading(false)
    }, 1500)

    return () => clearTimeout(timer)
  }, [])

  if (isLoading) {
    return <LoadingScreen />
  }

  return (
    <div className="min-h-screen bg-[#F8F8F8] pb-20">
      <AnimatePresence mode="wait">
        <motion.div
          key={activeTab}
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -20 }}
          transition={{ duration: 0.3 }}
          className="min-h-screen"
        >
          {activeTab === 'home' && <HomeTab />}
          {activeTab === 'dresscode' && <DresscodeTab />}
          {activeTab === 'timeline' && <TimelineTab />}
          {activeTab === 'seating' && <SeatingTab />}
          {activeTab === 'menu' && <MenuTab />}
        </motion.div>
      </AnimatePresence>

      <BottomNavbar activeTab={activeTab} onTabChange={setActiveTab} />
    </div>
  )
}

export default App

