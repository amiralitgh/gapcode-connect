import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import en from './locales/en.json';
import fa from './locales/fa.json';

const savedLanguage = localStorage.getItem('cc_lang') || navigator.language.split('-')[0] || 'en';
const saved = savedLanguage === 'fa' ? 'fa' : 'en';

i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    fa: { translation: fa },
  },
  lng: saved,
  fallbackLng: 'en',
  interpolation: { escapeValue: false },
});

export default i18n;
