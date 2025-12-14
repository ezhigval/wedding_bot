import { useState, useRef, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { RegistrationProvider } from './contexts/RegistrationContext'
import BottomNavbar from './components/BottomNavbar'
import HomeTab from './components/tabs/HomeTab'
import ChallengeTab from './components/tabs/ChallengeTab'
import MenuTab from './components/tabs/MenuTab'
import PhotoTab from './components/tabs/PhotoTab'
import TimelineTab from './components/tabs/TimelineTab'
import DresscodeTab from './components/tabs/DresscodeTab'
import SeatingTab from './components/tabs/SeatingTab'
import WishesTab from './components/tabs/WishesTab'
import LoadingScreen from './components/LoadingScreen'
import type { TabName } from './types'

function App() {
  const [activeTab, setActiveTab] = useState<TabName>('home')
  const [isLoading, setIsLoading] = useState(true)
  const scrollContainerRef = useRef<HTMLDivElement>(null)

  const handleLoadingComplete = () => {
    setIsLoading(false)
  }

  // Сбрасываем скролл наверх при смене таба
  useEffect(() => {
    if (scrollContainerRef.current) {
      scrollContainerRef.current.scrollTo({ top: 0, behavior: 'smooth' })
    }
  }, [activeTab])

  if (isLoading) {
    return <LoadingScreen onComplete={handleLoadingComplete} />
  }

  return (
    <RegistrationProvider>
      <div className="h-screen bg-[#F8F8F8] flex flex-col overflow-hidden">
        <div 
          ref={scrollContainerRef}
          className="flex-1 overflow-y-auto" 
          style={{ maxHeight: '100vh' }}
        >
          <AnimatePresence mode="wait">
            <motion.div
              key={activeTab}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              transition={{ duration: 0.3 }}
              className="min-h-full"
            >
            {activeTab === 'home' && <HomeTab />}
            {activeTab === 'challenge' && <ChallengeTab />}
            {activeTab === 'menu' && <MenuTab />}
            {activeTab === 'photo' && <PhotoTab />}
            {activeTab === 'timeline' && <TimelineTab />}
            {activeTab === 'dresscode' && <DresscodeTab />}
            {activeTab === 'seating' && <SeatingTab />}
            {activeTab === 'wishes' && <WishesTab />}
            </motion.div>
          </AnimatePresence>
        </div>

        <BottomNavbar activeTab={activeTab} onTabChange={setActiveTab} />
      </div>
    </RegistrationProvider>
  )
}

export default App

