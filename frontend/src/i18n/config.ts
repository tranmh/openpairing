import de from './de.json';
import en from './en.json';

export const locales = ['de', 'en'] as const;
export type Locale = (typeof locales)[number];

export const defaultLocale: Locale = 'de';

export const translations = {
  de,
  en,
} as const;

export type TranslationKey = keyof typeof de;

/**
 * Get a nested translation value by dot-notation key
 */
export function getTranslation(
  locale: Locale,
  key: string
): string {
  const keys = key.split('.');
  let value: unknown = translations[locale];

  for (const k of keys) {
    if (value && typeof value === 'object' && k in value) {
      value = (value as Record<string, unknown>)[k];
    } else {
      // Fallback to German if key not found
      value = translations.de;
      for (const fallbackKey of keys) {
        if (value && typeof value === 'object' && fallbackKey in value) {
          value = (value as Record<string, unknown>)[fallbackKey];
        } else {
          return key; // Return key if not found
        }
      }
      break;
    }
  }

  return typeof value === 'string' ? value : key;
}

/**
 * Format a date according to locale
 */
export function formatLocalizedDate(
  date: Date | string,
  locale: Locale,
  options?: Intl.DateTimeFormatOptions
): string {
  const d = typeof date === 'string' ? new Date(date) : date;
  const localeCode = locale === 'de' ? 'de-DE' : 'en-US';

  return d.toLocaleDateString(localeCode, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    ...options,
  });
}

/**
 * Format a number according to locale
 */
export function formatLocalizedNumber(
  num: number,
  locale: Locale,
  options?: Intl.NumberFormatOptions
): string {
  const localeCode = locale === 'de' ? 'de-DE' : 'en-US';
  return num.toLocaleString(localeCode, options);
}

/**
 * Format currency according to locale
 */
export function formatLocalizedCurrency(
  amount: number,
  locale: Locale,
  currency: string = 'EUR'
): string {
  const localeCode = locale === 'de' ? 'de-DE' : 'en-US';
  return new Intl.NumberFormat(localeCode, {
    style: 'currency',
    currency,
  }).format(amount);
}
