import { useEffect, useRef, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'
import RegistrationRequired from '../common/RegistrationRequired'
import { useRegistration } from '../../contexts/RegistrationContext'
import { loadConfig } from '../../utils/api'
import { getInitData, hapticFeedback, showAlert } from '../../utils/telegram'
import { useUser } from '../../contexts/UserContext'
import type { Config } from '../../types'

type UploadSource = 'webapp_camera' | 'webapp_gallery'
type MediaKind = 'image' | 'video'

const MAX_IMAGE_SIZE = 10 * 1024 * 1024
const MAX_VIDEO_SIZE = 50 * 1024 * 1024

export default function PhotoTab() {
  const { isRegistered, isLoading } = useRegistration()
  const { userId, manualUsername } = useUser()
  const [config, setConfig] = useState<Config | null>(null)
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [selectedSource, setSelectedSource] = useState<UploadSource>('webapp_gallery')
  const [selectedKind, setSelectedKind] = useState<MediaKind>('image')
  const [successKind, setSuccessKind] = useState<MediaKind>('image')
  const [isUploading, setIsUploading] = useState(false)
  const [isSuccess, setIsSuccess] = useState(false)
  const cameraPhotoInputRef = useRef<HTMLInputElement>(null)
  const galleryPhotoInputRef = useRef<HTMLInputElement>(null)
  const cameraVideoInputRef = useRef<HTMLInputElement>(null)
  const galleryVideoInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    loadConfig().then(setConfig)
  }, [])

  useEffect(() => {
    return () => {
      if (previewUrl) {
        URL.revokeObjectURL(previewUrl)
      }
    }
  }, [previewUrl])

  const resetSelectedMedia = () => {
    if (previewUrl) {
      URL.revokeObjectURL(previewUrl)
    }
    setPreviewUrl(null)
    setSelectedFile(null)
    setSelectedKind('image')

    const refs = [
      cameraPhotoInputRef,
      galleryPhotoInputRef,
      cameraVideoInputRef,
      galleryVideoInputRef,
    ]
    refs.forEach((ref) => {
      if (ref.current) {
        ref.current.value = ''
      }
    })
  }

  const handleFileSelect = (
    event: React.ChangeEvent<HTMLInputElement>,
    source: UploadSource,
    expectedKind: MediaKind
  ) => {
    const file = event.target.files?.[0]
    if (!file) return

    if (!file.type.startsWith(`${expectedKind}/`)) {
      showAlert(expectedKind === 'image' ? 'Пожалуйста, выберите изображение' : 'Пожалуйста, выберите видео')
      hapticFeedback('medium')
      event.target.value = ''
      return
    }

    const maxSize = expectedKind === 'image' ? MAX_IMAGE_SIZE : MAX_VIDEO_SIZE
    if (file.size > maxSize) {
      showAlert(
        expectedKind === 'image'
          ? 'Размер фото не должен превышать 10 MB'
          : 'Размер видео не должен превышать 50 MB'
      )
      hapticFeedback('medium')
      event.target.value = ''
      return
    }

    if (previewUrl) {
      URL.revokeObjectURL(previewUrl)
    }

    setSelectedFile(file)
    setSelectedSource(source)
    setSelectedKind(expectedKind)
    setPreviewUrl(URL.createObjectURL(file))
    hapticFeedback('light')
  }

  const openPicker = (ref: React.RefObject<HTMLInputElement>) => {
    ref.current?.click()
    hapticFeedback('light')
  }

  const handleUpload = async () => {
    if (!selectedFile) return

    setIsUploading(true)
    hapticFeedback('medium')

    try {
      const initData = getInitData()
      if (!userId && !initData && !manualUsername) {
        showAlert('Не удалось определить пользователя для загрузки медиа.')
        hapticFeedback('heavy')
        return
      }

      const formData = new FormData()
      formData.append('media', selectedFile)
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
        setSuccessKind(selectedKind)
        setIsSuccess(true)
        resetSelectedMedia()
        hapticFeedback('heavy')
        showAlert(selectedKind === 'image' ? '📸 Фото успешно сохранено! 🙌' : '🎥 Видео успешно сохранено! 🙌')

        window.setTimeout(() => {
          setIsSuccess(false)
        }, 3000)
      } else {
        showAlert(data.message || data.error || 'Не удалось загрузить медиа. Попробуйте еще раз.')
        hapticFeedback('heavy')
      }
    } catch (error) {
      console.error('Error uploading media:', error)
      showAlert('Ошибка сети. Проверьте подключение и попробуйте еще раз.')
      hapticFeedback('heavy')
    } finally {
      setIsUploading(false)
    }
  }

  const handleRetake = () => {
    resetSelectedMedia()
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
        <SectionTitle>ФОТО И ВИДЕО</SectionTitle>

        <AnimatePresence mode="wait">
          {isSuccess ? (
            <motion.div
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.9 }}
              className="text-center py-8"
            >
              <div className="text-6xl mb-4">{successKind === 'image' ? '📸' : '🎥'}</div>
              <p className="text-[21.6px] text-gray-700 mb-2 leading-[1.2]">
                {successKind === 'image' ? 'Фото успешно сохранено!' : 'Видео успешно сохранено!'}
              </p>
              <p className="text-[19.2px] text-gray-600 leading-[1.2]">
                Спасибо за участие в создании свадебного альбома 💕
              </p>
            </motion.div>
          ) : selectedFile && previewUrl ? (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="space-y-4"
            >
              <div className="relative w-full rounded-lg overflow-hidden bg-black/5">
                {selectedKind === 'image' ? (
                  <img
                    src={previewUrl}
                    alt="Preview"
                    className="w-full h-auto max-h-[60vh] object-contain"
                  />
                ) : (
                  <video
                    src={previewUrl}
                    controls
                    playsInline
                    className="w-full h-auto max-h-[60vh] object-contain"
                  />
                )}
              </div>

              <p className="text-center text-sm text-gray-500">
                {selectedKind === 'image' ? 'Фото' : 'Видео'} будет сохранено в общий свадебный альбом.
              </p>

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
                Добавьте фото или видео из камеры либо галереи в наш свадебный альбом.
              </p>

              <input
                ref={cameraPhotoInputRef}
                type="file"
                accept="image/*"
                capture="environment"
                onChange={(event) => handleFileSelect(event, 'webapp_camera', 'image')}
                className="hidden"
              />

              <input
                ref={galleryPhotoInputRef}
                type="file"
                accept="image/*"
                onChange={(event) => handleFileSelect(event, 'webapp_gallery', 'image')}
                className="hidden"
              />

              <input
                ref={cameraVideoInputRef}
                type="file"
                accept="video/*"
                capture="environment"
                onChange={(event) => handleFileSelect(event, 'webapp_camera', 'video')}
                className="hidden"
              />

              <input
                ref={galleryVideoInputRef}
                type="file"
                accept="video/*"
                onChange={(event) => handleFileSelect(event, 'webapp_gallery', 'video')}
                className="hidden"
              />

              <div className="grid gap-3 sm:grid-cols-2">
                <motion.button
                  onClick={() => openPicker(cameraPhotoInputRef)}
                  whileTap={{ scale: 0.95 }}
                  className="w-full py-6 bg-[#FFE9AD] text-primary-dark rounded-lg font-bold text-xl shadow-lg hover:shadow-xl transition-all"
                >
                  📸 Сделать фото
                </motion.button>

                <motion.button
                  onClick={() => openPicker(galleryPhotoInputRef)}
                  whileTap={{ scale: 0.95 }}
                  className="w-full py-6 bg-white text-primary-dark border-2 border-[#FFE9AD] rounded-lg font-bold text-xl shadow-lg hover:shadow-xl transition-all"
                >
                  🖼️ Фото из галереи
                </motion.button>

                <motion.button
                  onClick={() => openPicker(cameraVideoInputRef)}
                  whileTap={{ scale: 0.95 }}
                  className="w-full py-6 bg-[#F7D6BE] text-primary-dark rounded-lg font-bold text-xl shadow-lg hover:shadow-xl transition-all"
                >
                  🎥 Снять видео
                </motion.button>

                <motion.button
                  onClick={() => openPicker(galleryVideoInputRef)}
                  whileTap={{ scale: 0.95 }}
                  className="w-full py-6 bg-white text-primary-dark border-2 border-[#F7D6BE] rounded-lg font-bold text-xl shadow-lg hover:shadow-xl transition-all"
                >
                  🎞️ Видео из галереи
                </motion.button>
              </div>

              <p className="text-center text-gray-500 text-sm">
                Фото до 10 MB, видео до 50 MB
              </p>
            </motion.div>
          )}
        </AnimatePresence>
      </SectionCard>
    </div>
  )
}
