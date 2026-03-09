import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'
import RegistrationRequired from '../common/RegistrationRequired'
import RankIcon from '../common/RankIcon'
import DragonGame from '../games/DragonGame'
import FlappyBirdGame from '../games/FlappyBirdGame'
import CrosswordGame from '../games/CrosswordGame'
import WordleGame from '../games/WordleGame'
import { useRegistration } from '../../contexts/RegistrationContext'
import { useUser } from '../../contexts/UserContext'
import { loadConfig, getGameStats, updateGameScore, type GameStats } from '../../utils/api'
import { hapticFeedback } from '../../utils/telegram'
import type { Config } from '../../types'

type GameType = 'dragon' | 'flappy' | 'crossword' | 'wordle'

interface Game {
  id: GameType
  name: string
  emoji: string
}

const ALL_GAMES: Game[] = [
  { id: 'dragon', name: 'Дракончик', emoji: '🐉' },
  { id: 'flappy', name: 'ФлэппиБёрд', emoji: '🐦' },
  { id: 'crossword', name: 'Кроссворд', emoji: '📝' },
  { id: 'wordle', name: 'ВОРДЛИ', emoji: '🔤' },
]

export default function ChallengeTab() {
  const { isRegistered, isLoading } = useRegistration()
  const { userId } = useUser()
  const [config, setConfig] = useState<Config | null>(null)
  const [stats, setStats] = useState<GameStats | null>(null)
  const [loadingStats, setLoadingStats] = useState(true)
  const [activeGame, setActiveGame] = useState<GameType | null>(null)
  const [favoriteGames, setFavoriteGames] = useState<GameType[]>([])

  useEffect(() => {
    loadConfig().then(setConfig)
    
    // Загружаем избранные игры из localStorage
    const savedFavorites = localStorage.getItem('challenge_favorite_games')
    if (savedFavorites) {
      try {
        const favorites = JSON.parse(savedFavorites) as GameType[]
        setFavoriteGames(favorites)
      } catch (e) {
        console.error('Error loading favorite games:', e)
      }
    }
  }, [])

  useEffect(() => {
    if (isRegistered && config) {
      loadStats()
    }
  }, [isRegistered, config, userId])

  const loadStats = async () => {
    if (!config) return
    
    setLoadingStats(true)
    try {
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

  const handleGameClick = (gameType: GameType) => {
    hapticFeedback('light')
    setActiveGame(gameType)
  }

  const handleFavoriteToggle = (gameType: GameType, e: React.MouseEvent) => {
    e.stopPropagation()
    hapticFeedback('light')
    
    setFavoriteGames(prev => {
      const isFavorite = prev.includes(gameType)
      let newFavorites: GameType[]
      
      if (isFavorite) {
        // Убираем из избранного
        newFavorites = prev.filter(id => id !== gameType)
      } else {
        // Добавляем в избранное (максимум 5)
        if (prev.length >= 5) {
          hapticFeedback('heavy')
          alert('Можно добавить максимум 5 игр в избранное')
          return prev
        }
        newFavorites = [...prev, gameType]
      }
      
      // Сохраняем в localStorage
      localStorage.setItem('challenge_favorite_games', JSON.stringify(newFavorites))
      return newFavorites
    })
  }

  // Сортируем игры: сначала избранные, потом остальные
  const sortedGames = [...ALL_GAMES].sort((a, b) => {
    const aIsFavorite = favoriteGames.includes(a.id)
    const bIsFavorite = favoriteGames.includes(b.id)
    
    if (aIsFavorite && !bIsFavorite) return -1
    if (!aIsFavorite && bIsFavorite) return 1
    
    // Если оба избранные или оба не избранные, сохраняем исходный порядок
    return ALL_GAMES.indexOf(a) - ALL_GAMES.indexOf(b)
  })

  const handleGameScore = async (score: number, gameType: 'dragon' | 'flappy' | 'crossword' | 'wordle') => {
    if (!config) return

    if (!userId) {
      console.error('Cannot update score: userId not found')
      return
    }

    // БАЛАНС ОЧКОВ ПО ИГРАМ:
    // - Дракончик (простая): 200 очков в игре = 1 очко в рейтинге
    //   Пример: 200 очков в игре = 1 очко в статистике, 1000 очков = 5 очков
    // - ФлэппиБёрд (средняя): 2 очка в игре = 1 очко в рейтинге
    //   Пример: 2 очка в игре = 1 очко в статистике, 100 очков = 50 очков
    // - Кроссворд (сложная): 1 игровое очко = 25 рейтинговых очков
    //   Пример: 1 очко в игре = 25 очков в статистике
    // - ВОРДЛИ: 1 отгаданное слово = 5 рейтинговых очков
    //   Пример: 1 слово = 5 очков в статистике
    
    let gamePoints = 0
    if (gameType === 'dragon') {
      gamePoints = Math.floor(score / 200)
    } else if (gameType === 'flappy') {
      gamePoints = Math.floor(score / 2)
    } else if (gameType === 'crossword') {
      gamePoints = score * 25
    } else if (gameType === 'wordle') {
      // Wordle: 1 отгаданное слово = 5 рейтинговых очков
      gamePoints = score * 5
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

  // Функция для определения прогресса до следующего звания
  const getRankProgress = (score: number) => {
    const rankThresholds = [
      { rank: 'Незнакомец', min: 0, max: 50 },
      { rank: 'Ты хто?', min: 50, max: 100 },
      { rank: 'Люся', min: 100, max: 150 },
      { rank: 'Бедный родственник', min: 150, max: 200 },
      { rank: 'Братуха', min: 200, max: 300 },
      { rank: 'Батя в здании', min: 300, max: 400 },
      { rank: 'Монстр', min: 400, max: Infinity },
    ]

    const currentThreshold = rankThresholds.find(t => 
      score >= t.min && (t.max === Infinity || score < t.max)
    ) || rankThresholds[0]

    const nextThreshold = rankThresholds[rankThresholds.indexOf(currentThreshold) + 1]

    if (!nextThreshold) {
      // Достигнуто максимальное звание
      return {
        current: currentThreshold.rank,
        next: null,
        progress: 100,
        currentScore: score,
        nextScore: null,
        remaining: 0,
      }
    }

    const progressInRange = score - currentThreshold.min
    const rangeSize = currentThreshold.max - currentThreshold.min
    const progress = (progressInRange / rangeSize) * 100

    return {
      current: currentThreshold.rank,
      next: nextThreshold.rank,
      progress: Math.min(100, Math.max(0, progress)),
      currentScore: score,
      nextScore: currentThreshold.max,
      remaining: currentThreshold.max - score,
    }
  }

  const rankProgress = getRankProgress(currentScore)

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
  
  if (activeGame === 'wordle') {
    return <WordleGame onScore={(score) => handleGameScore(score, 'wordle')} onClose={handleGameClose} />
  }

  return (
    <div className="h-screen flex flex-col overflow-hidden pb-[calc(env(safe-area-inset-bottom,0px)+112px)]">
      {/* Прокручиваемая область с играми */}
      <div
        className="flex-1 overflow-y-auto px-4 py-4"
        style={{ paddingBottom: 'calc(env(safe-area-inset-bottom, 0px) + 140px)' }}
      >
        <SectionCard>
          <SectionTitle>ИСПЫТАНИЕ</SectionTitle>
          <p className="text-center text-gray-600 mb-6 leading-[1.2] text-[19.2px]">
            Выберите игру и зарабатывайте очки!
          </p>

          <div className="space-y-4">
            {sortedGames.map((game) => {
              const isFavorite = favoriteGames.includes(game.id)
              return (
                <motion.button
                  key={game.id}
                  onClick={() => handleGameClick(game.id)}
                  whileTap={{ scale: 0.95 }}
                  className="w-full py-4 bg-primary text-white rounded-lg font-semibold text-lg shadow-md hover:shadow-lg transition-all relative pr-12"
                >
                  <span>{game.emoji} {game.name}</span>
                  
                  {/* Кнопка избранного */}
                  <button
                    onClick={(e) => handleFavoriteToggle(game.id, e)}
                    className="absolute right-3 top-1/2 transform -translate-y-1/2 p-2 hover:bg-white/20 rounded-full transition-colors"
                    aria-label={isFavorite ? 'Убрать из избранного' : 'Добавить в избранное'}
                  >
                    <svg
                      className={`w-6 h-6 transition-all ${isFavorite ? 'fill-yellow-400 stroke-yellow-400' : 'fill-none stroke-white'}`}
                      viewBox="0 0 24 24"
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
                    </svg>
                  </button>
                </motion.button>
              )
            })}
          </div>
        </SectionCard>
      </div>

      {/* Статистика игрока - фиксированная внизу, опущена на 30% */}
      <div className="fixed bottom-28 left-1/2 transform -translate-x-1/2 px-4 pb-2 w-full max-w-md z-10">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="bg-white/90 backdrop-blur-sm rounded-lg p-4 shadow-lg border-2 border-primary/30"
        >
          {loadingStats ? (
            <div className="text-gray-500 text-sm text-center">Загрузка статистики...</div>
          ) : (
            <div className="space-y-3">
              {/* Верхняя часть - Звание и рейтинг */}
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

              {/* Прогресс-бар до следующего звания */}
              {rankProgress.next ? (
                <div className="space-y-1">
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-gray-600">
                      До звания <span className="font-semibold text-primary">{rankProgress.next}</span>
                    </span>
                    <span className="text-gray-600 font-semibold">
                      {rankProgress.remaining} очков
                    </span>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-3 overflow-hidden">
                    <motion.div
                      initial={{ width: 0 }}
                      animate={{ width: `${rankProgress.progress}%` }}
                      transition={{ duration: 0.5, ease: 'easeOut' }}
                      className="h-full bg-gradient-to-r from-primary to-[#F5D98A] rounded-full shadow-sm"
                    />
                  </div>
                  <div className="text-xs text-gray-500 text-center">
                    {currentScore} / {rankProgress.nextScore} очков
                  </div>
                </div>
              ) : (
                <div className="text-center py-2">
                  <div className="text-sm font-semibold text-primary">
                    🏆 Вы достигли максимального звания!
                  </div>
                </div>
              )}
            </div>
          )}
        </motion.div>
      </div>
    </div>
  )
}
