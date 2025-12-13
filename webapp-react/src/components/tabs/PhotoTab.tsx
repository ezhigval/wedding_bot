import { useState, useRef, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'
import RegistrationRequired from '../common/RegistrationRequired'
import { useRegistration } from '../../contexts/RegistrationContext'
import { loadConfig } from '../../utils/api'
import { showAlert, hapticFeedback, getInitData } from '../../utils/telegram'
import type { Config } from '../../types'

export default function PhotoTab() {
  const { isRegistered, isLoading } = useRegistration()
  const [config, setConfig] = useState<Config | null>(null)
  const [photoPreview, setPhotoPreview] = useState<string | null>(null)
  const [isUploading, setIsUploading] = useState(false)
  const [isSuccess, setIsSuccess] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    loadConfig().then(setConfig)
  }, [])

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    // Проверяем тип файла
    if (!file.type.startsWith('image/')) {
      showAlert('Пожалуйста, выберите изображение')
      hapticFeedback('medium')
      return
    }

    // Проверяем размер (макс 10MB)
    if (file.size > 10 * 1024 * 1024) {
      showAlert('Размер файла не должен превышать 10MB')
      hapticFeedback('medium')
      return
    }

    // Создаем превью
    const reader = new FileReader()
    reader.onload = (event) => {
      setPhotoPreview(event.target?.result as string)
      hapticFeedback('light')
    }
    reader.readAsDataURL(file)
  }

  const handleCapture = () => {
    fileInputRef.current?.click()
    hapticFeedback('light')
  }

  const handleUpload = async () => {
    if (!photoPreview) return

    setIsUploading(true)
    hapticFeedback('medium')

    try {
      const initData = getInitData()
      const response = await fetch(`${config?.apiUrl || '/api'}/upload-photo`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          photo: photoPreview, // base64 строка
          initData,
        }),
      })

      const data = await response.json()

      if (response.ok && data.success) {
        setIsSuccess(true)
        setPhotoPreview(null)
        hapticFeedback('heavy')
        showAlert('📸 Фото успешно сохранено в свадебный альбом! 🙌')
        
        // Сбрасываем input
        if (fileInputRef.current) {
          fileInputRef.current.value = ''
        }
        
        // Через 3 секунды скрываем сообщение об успехе
        setTimeout(() => {
          setIsSuccess(false)
        }, 3000)
      } else {
        showAlert(data.error || 'Не удалось загрузить фото. Попробуйте еще раз.')
        hapticFeedback('heavy')
      }
    } catch (error) {
      console.error('Error uploading photo:', error)
      showAlert('Ошибка сети. Проверьте подключение и попробуйте еще раз.')
      hapticFeedback('heavy')
    } finally {
      setIsUploading(false)
    }
  }

  const handleRetake = () => {
    setPhotoPreview(null)
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
    hapticFeedback('light')
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

  return (
    <div className="min-h-screen px-4 py-4">
      <SectionCard>
        <SectionTitle>СДЕЛАТЬ ФОТО</SectionTitle>
        
        <AnimatePresence mode="wait">
          {isSuccess ? (
            <motion.div
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.9 }}
              className="text-center py-8"
            >
              <div className="text-6xl mb-4">📸</div>
              <p className="text-[21.6px] text-gray-700 mb-2 leading-[1.2]">
                Фото успешно сохранено!
              </p>
              <p className="text-[19.2px] text-gray-600 leading-[1.2]">
                Спасибо за участие в создании свадебного альбома 💕
              </p>
            </motion.div>
          ) : photoPreview ? (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="space-y-4"
            >
              <div className="relative w-full rounded-lg overflow-hidden">
                <img
                  src={photoPreview}
                  alt="Preview"
                  className="w-full h-auto max-h-[60vh] object-contain"
                />
              </div>
              
              <div className="flex gap-3">
                <button
                  onClick={handleRetake}
                  disabled={isUploading}
                  className="flex-1 px-4 py-3 bg-gray-200 text-gray-700 rounded-lg font-semibold hover:bg-gray-300 transition-colors disabled:opacity-50"
                >
                  Переснять
                </button>
                <button
                  onClick={handleUpload}
                  disabled={isUploading}
                  className="flex-1 px-4 py-3 bg-primary text-white rounded-lg font-semibold hover:bg-primary-dark transition-colors disabled:opacity-50"
                >
                  {isUploading ? 'Загрузка...' : 'Сохранить'}
                </button>
              </div>
            </motion.div>
          ) : (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="space-y-6 py-8"
            >
              <p className="text-center text-gray-600 mb-6 leading-[1.2] text-[19.2px]">
                Сделайте фото и добавьте его в наш свадебный альбом!
              </p>
              
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                capture="environment"
                onChange={handleFileSelect}
                className="hidden"
              />
              
              <motion.button
                onClick={handleCapture}
                whileTap={{ scale: 0.95 }}
                className="w-full py-6 bg-[#FFE9AD] text-primary-dark rounded-lg font-bold text-xl shadow-lg hover:shadow-xl transition-all"
              >
                📸 Сделать фото
              </motion.button>
              
              <p className="text-center text-gray-500 text-sm">
                Или выберите фото из галереи
              </p>
            </motion.div>
          )}
        </AnimatePresence>
      </SectionCard>
    </div>
  )
}
