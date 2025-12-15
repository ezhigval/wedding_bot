import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'
import RegistrationRequired from '../common/RegistrationRequired'
import { useRegistration } from '../../contexts/RegistrationContext'

export default function SeatingTab() {
  const { isRegistered, isLoading } = useRegistration()

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
    <div className="min-h-screen px-4 py-4 pb-[120px]">
      <SectionCard>
        <SectionTitle>РАССАДКА</SectionTitle>
        <p className="text-center text-gray-500 py-4 text-[19.2px]">
          Информация о рассадке будет доступна позже
        </p>
      </SectionCard>
    </div>
  )
}

