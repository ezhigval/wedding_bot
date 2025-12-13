import SectionCard from '../common/SectionCard'
import SectionTitle from '../common/SectionTitle'

export default function SeatingTab() {
  return (
    <div className="min-h-screen px-4 py-4">
      <SectionCard>
        <SectionTitle>РАССАДКА</SectionTitle>
        <p className="text-center text-gray-500 py-4 text-[19.2px]">
          Информация о рассадке будет доступна позже
        </p>
      </SectionCard>
    </div>
  )
}

