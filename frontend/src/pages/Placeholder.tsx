import { useTranslation } from 'react-i18next';

export default function Placeholder({ titleKey }: { titleKey: string }) {
  const { t } = useTranslation();
  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">{t(titleKey)}</h1>
      <div className="bg-white rounded-xl p-12 shadow-sm border border-gray-100 text-center text-gray-400">
        Coming soon
      </div>
    </div>
  );
}
