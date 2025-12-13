import { motion } from 'framer-motion'
import { useEffect, useState } from 'react'

export default function LoadingScreen() {
  const [animationData, setAnimationData] = useState<any>(null)

  useEffect(() => {
    // Загружаем Lottie анимацию
    fetch('/ring_animation.json')
      .then((res) => res.json())
      .then((data) => setAnimationData(data))
      .catch(() => {
        // Fallback если анимация не загрузилась
        setAnimationData(null)
      })
  }, [])

  return (
    <motion.div
      initial={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="fixed inset-0 z-50 flex items-center justify-center bg-[#F8F8F8]"
    >
      {animationData ? (
        <motion.div
          initial={{ scale: 0.8, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={{ duration: 0.5 }}
          className="w-32 h-32"
        >
          {/* Здесь можно добавить Lottie player если нужно */}
          <div className="w-full h-full bg-primary/10 rounded-full flex items-center justify-center">
            <motion.div
              animate={{ rotate: 360 }}
              transition={{ duration: 2, repeat: Infinity, ease: 'linear' }}
              className="w-16 h-16 border-4 border-primary border-t-transparent rounded-full"
            />
          </div>
        </motion.div>
      ) : (
        <motion.div
          initial={{ scale: 0.8, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={{ duration: 0.5 }}
          className="w-16 h-16 border-4 border-primary border-t-transparent rounded-full"
        >
          <motion.div
            animate={{ rotate: 360 }}
            transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
            className="w-full h-full"
          />
        </motion.div>
      )}
    </motion.div>
  )
}

