import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { Locale, defaultLocale, getTranslation } from '@/i18n/config';

interface LanguageState {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: string) => string;
}

export const useLanguageStore = create<LanguageState>()(
  persist(
    (set, get) => ({
      locale: defaultLocale,

      setLocale: (locale: Locale) => {
        set({ locale });
        // Update HTML lang attribute
        if (typeof document !== 'undefined') {
          document.documentElement.lang = locale;
        }
      },

      t: (key: string) => {
        return getTranslation(get().locale, key);
      },
    }),
    {
      name: 'openpairing-language',
      partialize: (state) => ({ locale: state.locale }),
      onRehydrateStorage: () => (state) => {
        // Set HTML lang attribute after rehydration
        if (state && typeof document !== 'undefined') {
          document.documentElement.lang = state.locale;
        }
      },
    }
  )
);

/**
 * Hook to get translation function
 */
export function useTranslation() {
  const { locale, t, setLocale } = useLanguageStore();

  return {
    locale,
    t,
    setLocale,
  };
}
