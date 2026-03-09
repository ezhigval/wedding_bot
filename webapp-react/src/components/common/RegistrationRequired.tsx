import { useState } from 'react'
import SectionCard from './SectionCard'
import SectionTitle from './SectionTitle'
import { useUser } from '../../contexts/UserContext'
import { useRegistration } from '../../contexts/RegistrationContext'

export default function RegistrationRequired() {
  const { userId, manualUsername, setManualUsername } = useUser()
  const { refreshRegistration } = useRegistration()
  const [username, setUsername] = useState(manualUsername ? `@${manualUsername}` : '')

  return (
    <div className="min-h-screen px-4 py-4">
      <SectionCard>
        <SectionTitle>РЕГИСТРАЦИЯ ТРЕБУЕТСЯ</SectionTitle>
        <p className="text-center text-gray-600 mb-2 leading-[1.2] text-[19.2px]">
          Для доступа к этому разделу необходимо подтвердить ваше присутствие на главной странице.
        </p>

        {!userId && (
          <div className="mt-3">
            <p className="text-center text-gray-600 mb-2 leading-[1.2] text-[16.8px]">
              Если вы открыли ссылку вне Telegram, можно войти по вашему Telegram username.
            </p>
            <div className="flex gap-2 items-center justify-center">
              <input
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="@username"
                className="border border-gray-300 rounded-lg px-3 py-2 text-[16.8px] w-56"
              />
              <button
                className="bg-primary text-white rounded-lg px-4 py-2 font-semibold"
                onClick={async () => {
                  setManualUsername(username)
                  await refreshRegistration()
                }}
              >
                Войти
              </button>
            </div>
          </div>
        )}
      </SectionCard>
    </div>
  )
}

