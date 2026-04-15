import { useState, useRef, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'
import RegistrationRequired from '../common/RegistrationRequired'
import { useRegistration } from '../../contexts/RegistrationContext'
import { loadConfig } from '../../utils/api'
import { showAlert, hapticFeedback, getInitData } from '../../utils/telegram'
import { useUser } from '../../contexts/UserContext'
import type { Config } from '../../types'

type PhotoSource = 'webapp_camera' | 'webapp_gallery'

export default function PhotoTab() {
  const { isRegistered, isLoading } = useRegistration()
  const { userId, manualUsername } = useUser()
  const [config, setConfig] = useState<Config | null>(null)
  const [photoPreview, setPhotoPreview] = useState<string | null>(null)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [selectedSource, setSelectedSource] = useState<PhotoSource>('webapp_gallery')
  const [isUploading, setIsUploading] = useState(false)
  const [isSuccess, setIsSuccess] = useState(false)
  const cameraInputRef = useRef<HTMLInputElement>(null)
  const galleryInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    loadConfig().then(setConfig)
  }, [])

  const resetSelectedPhoto = () => {
    setPhotoPreview(null)
    setSelectedFile(null)
    if (cameraInputRef.current) {
      cameraInputRef.current.value = ''
    }
    if (galleryInputRef.current) {
      galleryInputRef.current.value = ''
    }
  }

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>, source: PhotoSource) => {
    const file = event.target.files?.[0]
    if (!file) return

    if (!file.type.startsWith('image/')) {
      showAlert('Пожалуйста, выберите изображение')
      hapticFeedback('medium')
      event.target.value = ''
      return
    }

    if (file.size > 10 * 1024 * 1024) {
      showAlert('Размер файла не должен превышать 10MB')
      hapticFeedback('medium')
      event.target.value = ''
      return
    }

    const reader = new FileReader()
    reader.onload = (loadEvent) => {
      setSelectedFile(file)
      setSelectedSource(source)
      setPhotoPreview(loadEvent.target?.result as string)
      hapticFeedback('light')
    }
    reader.readAsDataURL(file)
  }

  const openCamera = () => {
    cameraInputRef.current?.click()
    hapticFeedback('light')
  }

  const openGallery = () => {
    galleryInputRef.current?.click()
    hapticFeedback('light')
  }

  const handleUpload = async () => {
    if (!selectedFile) return

    setIsUploading(true)
    hapticFeedback('medium')

    try {
      const initData = getInitData()
      if (!userId && !initData && !manualUsername) {
        showAlert('Не удалось определить пользователя для загрузки фото.')
        hapticFeedback('heavy')
        return
      }

      const formData = new FormData()
      formData.append('photo', selectedFile)
      formData.append('source', selectedSource)
      if (selectedFile.name) {
        formData.append('fileName', selectedFile.name)
      }
      if (userId) {
        formData.append('userId', String(userId))
      }
      if (manualUsername) {
        formData.append('username', manualUsername)
      }
      if (initData) {
        formData.append('initData', initData)
      }

      const response = await fetch(`${config?.apiUrl || '/api'}/upload-photo`, {
        method: 'POST',
        body: formData,
      })

      const data = await response.json()

      if (response.ok && data.success) {
        setIsSuccess(true)
        resetSelectedPhoto()
        hapticFeedback('heavy')
        showAlert('📸 Фото успешно сохранено в свадебный альбом! 🙌')

        setTimeout(() => {
          setIsSuccess(false)
        }, 3000)
      } else {
        showAlert(data.message || data.error || 'Не удалось загрузить фото. Попробуйте еще раз.')
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
    resetSelectedPhoto()
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
    <div className="min-h-screen px-4 py-4 pb-[calc(env(safe-area-inset-bottom,0px)+112px)]">
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
                  Выбрать другое
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
                Сделайте фото или выберите его из галереи и добавьте в наш свадебный альбом.
              </p>

              <input
                ref={cameraInputRef}
                type="file"
                accept="image/*"
                capture="environment"
                onChange={(event) => handleFileSelect(event, 'webapp_camera')}
                className="hidden"
              />

              <input
                ref={galleryInputRef}
                type="file"
                accept="image/*"
                onChange={(event) => handleFileSelect(event, 'webapp_gallery')}
                className="hidden"
              />

              <div className="grid gap-3 sm:grid-cols-2">
                <motion.button
                  onClick={openCamera}
                  whileTap={{ scale: 0.95 }}
                  className="w-full py-6 bg-[#FFE9AD] text-primary-dark rounded-lg font-bold text-xl shadow-lg hover:shadow-xl transition-all"
                >
                  📸 Сделать фото
                </motion.button>

                <motion.button
                  onClick={openGallery}
                  whileTap={{ scale: 0.95 }}
                  className="w-full py-6 bg-white text-primary-dark border-2 border-[#FFE9AD] rounded-lg font-bold text-xl shadow-lg hover:shadow-xl transition-all"
                >
                  🖼️ Выбрать из галереи
                </motion.button>
              </div>

              <p className="text-center text-gray-500 text-sm">
                Максимальный размер файла — 10 MB
              </p>
            </motion.div>
          )}
        </AnimatePresence>
      </SectionCard>
    </div>
  )
}
