import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import { commonEn } from '@/shared/i18n/locales/en/common';

void i18n.use(initReactI18next).init({
  lng: 'en',
  fallbackLng: 'en',
  defaultNS: 'common',
  showSupportNotice: false,
  interpolation: { escapeValue: false },
  resources: {
    en: { common: commonEn },
  },
});

export { i18n };
