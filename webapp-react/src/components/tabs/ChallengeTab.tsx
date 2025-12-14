import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'
import RegistrationRequired from '../common/RegistrationRequired'
import RankIcon from '../common/RankIcon'
import DragonGame from '../games/DragonGame'
import FlappyBirdGame from '../games/FlappyBirdGame'
import CrosswordGame from '../games/CrosswordGame'
import { useRegistration } from '../../contexts/RegistrationContext'
import { loadConfig, getGameStats, updateGameScore, type GameStats } from '../../utils/api'
import { hapticFeedback } from '../../utils/telegram'
import type { Config } from '../../types'

export default function ChallengeTab() {
  const { isRegistered, isLoading } = useRegistration()
  const [config, setConfig] = useState<Config | null>(null)
  const [stats, setStats] = useState<GameStats | null>(null)
  const [loadingStats, setLoadingStats] = useState(true)
  const [activeGame, setActiveGame] = useState<'dragon' | 'flappy' | 'crossword' | null>(null)

  useEffect(() => {
    loadConfig().then(setConfig)
  }, [])

  useEffect(() => {
    if (isRegistered && config) {
      loadStats()
    }
  }, [isRegistered, config])

  const loadStats = async () => {
    if (!config) return
    
    setLoadingStats(true)
    try {
      // Получаем user_id
      const tg = window.Telegram?.WebApp
      const initData = (tg as any)?.initData || ''
      let userId: number | null = null
      
      if (initData) {
        try {
          const response = await fetch(`${config.apiUrl}/parse-init-data`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ initData }),
          })
          if (response.ok) {
            const parsed = await response.json()
            userId = parsed.userId
          }
        } catch (e) {
          console.error('Error parsing initData:', e)
        }
      }
      
      if (!userId) {
        const savedUserId = localStorage.getItem('telegram_user_id')
        if (savedUserId) {
          userId = parseInt(savedUserId)
        }
      }
      
      if (userId) {
        const statsData = await getGameStats(userId)
        setStats(statsData)
      } else {
        // Если нет userId, создаем дефолтную статистику
        setStats({
          user_id: 0,
          first_name: '',
          last_name: '',
          total_score: 0,
          dragon_score: 0,
          flappy_score: 0,
          crossword_score: 0,
          rank: 'Незнакомец',
        })
      }
    } catch (error) {
      console.error('Error loading stats:', error)
    } finally {
      setLoadingStats(false)
    }
  }

  const handleGameClick = (gameType: 'dragon' | 'flappy' | 'crossword') => {
    hapticFeedback('light')
    if (gameType === 'dragon') {
      setActiveGame('dragon')
    } else if (gameType === 'flappy') {
      setActiveGame('flappy')
    } else if (gameType === 'crossword') {
      setActiveGame('crossword')
    }
  }

  const handleGameScore = async (score: number, gameType: 'dragon' | 'flappy' | 'crossword') => {
    if (!config) return

    // Получаем userId
    const tg = window.Telegram?.WebApp
    const initData = (tg as any)?.initData || ''
    let userId: number | null = null
    
    if (initData) {
      try {
        const response = await fetch(`${config.apiUrl}/parse-init-data`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ initData }),
        })
        if (response.ok) {
          const parsed = await response.json()
          userId = parsed.userId
        }
      } catch (e) {
        console.error('Error parsing initData:', e)
      }
    }
    
    if (!userId) {
      const savedUserId = localStorage.getItem('telegram_user_id')
      if (savedUserId) {
        userId = parseInt(savedUserId)
      }
    }

    if (!userId) {
      console.error('Cannot update score: userId not found')
      return
    }

    // БАЛАНС ОЧКОВ ПО ИГРАМ:
    // - Дракончик (простая): 200 очков в игре = 1 очко в рейтинге
    //   Пример: 200 очков в игре = 1 очко в статистике, 1000 очков = 5 очков
    // - ФлэппиБёрд (средняя): 2 очка в игре = 1 очко в рейтинге
    //   Пример: 2 очка в игре = 1 очко в статистике, 100 очков = 50 очков
    // - Кроссвод (сложная): счет / 5
    //   Пример: 100 очков в игре = 20 очков в статистике
    
    let gamePoints = 0
    if (gameType === 'dragon') {
      gamePoints = Math.floor(score / 200)
    } else if (gameType === 'flappy') {
      gamePoints = Math.floor(score / 2)
    } else if (gameType === 'crossword') {
      gamePoints = Math.floor(score / 5)
    }

    try {
      const result = await updateGameScore(userId, gameType, gamePoints)
      if (result.success && result.stats) {
        setStats(result.stats)
        hapticFeedback('heavy')
        // Перезагружаем статистику из сервера для актуальности
        await loadStats()
      }
    } catch (error) {
      console.error('Error updating game score:', error)
    }
  }

  const handleGameClose = () => {
    setActiveGame(null)
    hapticFeedback('light')
    // Перезагружаем статистику
    if (isRegistered && config) {
      loadStats()
    }
  }

  if (isLoading) {
    return (
      <div className="min-h-screen px-4 py-4 flex items-center justify-center">
        <div className="text-center text-gray-500">Загрузка...</div>
      </div>
    )
  }

  if (!isRegistered) {
    return <RegistrationRequired />
  }

  const currentRank = stats?.rank || 'Незнакомец'
  const currentScore = stats?.total_score || 0

  // Если игра активна, показываем её
  if (activeGame === 'dragon') {
    return <DragonGame onScore={(score) => handleGameScore(score, 'dragon')} onClose={handleGameClose} />
  }
  
  if (activeGame === 'flappy') {
    return <FlappyBirdGame onScore={(score) => handleGameScore(score, 'flappy')} onClose={handleGameClose} />
  }
  
  if (activeGame === 'crossword') {
    return <CrosswordGame onClose={handleGameClose} />
  }

  return (
    <div className="min-h-screen px-4 py-4 pb-24">
      <SectionCard>
        <SectionTitle>ИСПЫТАНИЕ</SectionTitle>
        <p className="text-center text-gray-600 mb-6 leading-[1.2] text-[19.2px]">
          Выберите игру и зарабатывайте очки!
        </p>

        <div className="space-y-4">
          <motion.button
            onClick={() => handleGameClick('dragon')}
            whileTap={{ scale: 0.95 }}
            className="w-full py-4 bg-primary text-white rounded-lg font-semibold text-lg shadow-md hover:shadow-lg transition-all"
          >
            🐉 Дракончик
          </motion.button>

          <motion.button
            onClick={() => handleGameClick('flappy')}
            whileTap={{ scale: 0.95 }}
            className="w-full py-4 bg-primary text-white rounded-lg font-semibold text-lg shadow-md hover:shadow-lg transition-all"
          >
            🐦 ФлэппиБёрд
          </motion.button>

          <motion.button
            onClick={() => handleGameClick('crossword')}
            whileTap={{ scale: 0.95 }}
            className="w-full py-4 bg-primary text-white rounded-lg font-semibold text-lg shadow-md hover:shadow-lg transition-all"
          >
            📝 Кроссворд
          </motion.button>
        </div>
      </SectionCard>

      {/* Статистика игрока - внизу по центру */}
      <div className="fixed bottom-40 left-1/2 transform -translate-x-1/2 px-4 pb-2 w-full max-w-md">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="bg-white/90 backdrop-blur-sm rounded-lg p-4 shadow-lg border-2 border-primary/30"
        >
          {loadingStats ? (
            <div className="text-gray-500 text-sm text-center">Загрузка статистики...</div>
          ) : (
            <div className="flex items-center justify-between gap-4">
              {/* Левая часть - Звание с иконкой */}
              <div className="flex items-center gap-3 flex-1">
                <RankIcon 
                  rank={currentRank as 'Незнакомец' | 'Ты хто?' | 'Люся' | 'Бедный родственник' | 'Братуха' | 'Батя в здании' | 'Монстр'} 
                  className="flex-shrink-0"
                />
                <div>
                  <div className="text-xs text-gray-600 mb-1">Ваше звание</div>
                  <div className="text-2xl font-bold text-primary capitalize">
                    {currentRank}
                  </div>
                </div>
              </div>
              
              {/* Правая часть - Рейтинг */}
              <div className="flex-1 text-right">
                <div className="text-xs text-gray-600 mb-1">Ваш рейтинг</div>
                <div className="text-2xl font-bold text-primary">
                  {currentScore}
                </div>
              </div>
            </div>
          )}
        </motion.div>
      </div>
    </div>
  )
}
