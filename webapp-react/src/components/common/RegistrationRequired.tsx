import SectionCard from './SectionCard'
import SectionTitle from './SectionTitle'

export default function RegistrationRequired() {
  return (
    <div className="min-h-screen px-4 py-4">
      <SectionCard>
        <SectionTitle>РЕГИСТРАЦИЯ ТРЕБУЕТСЯ</SectionTitle>
        <p className="text-center text-gray-600 mb-2 leading-[1.2] text-[19.2px]">
          Для доступа к этому разделу необходимо подтвердить ваше присутствие на главной странице.
        </p>
      </SectionCard>
    </div>
  )
}

