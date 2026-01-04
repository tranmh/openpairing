import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

/**
 * Combines class names using clsx and tailwind-merge
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * Format a date in German locale
 */
export function formatDate(date: Date | string, options?: Intl.DateTimeFormatOptions): string {
  const d = typeof date === 'string' ? new Date(date) : date;
  return d.toLocaleDateString('de-DE', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    ...options,
  });
}

/**
 * Format a short date (DD.MM.YYYY)
 */
export function formatShortDate(date: Date | string): string {
  const d = typeof date === 'string' ? new Date(date) : date;
  return d.toLocaleDateString('de-DE', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  });
}

/**
 * Format time in HH:MM format
 */
export function formatTime(date: Date | string): string {
  const d = typeof date === 'string' ? new Date(date) : date;
  return d.toLocaleTimeString('de-DE', {
    hour: '2-digit',
    minute: '2-digit',
  });
}

/**
 * Format chess clock time (milliseconds to MM:SS or H:MM:SS)
 */
export function formatClockTime(ms: number): string {
  const totalSeconds = Math.floor(ms / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
  }
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
}

/**
 * Format a rating with proper display
 */
export function formatRating(rating: number | null | undefined): string {
  if (rating === null || rating === undefined || rating === 0) {
    return '-';
  }
  return rating.toString();
}

/**
 * Format tournament points (e.g., 5.5)
 */
export function formatPoints(points: number): string {
  if (Number.isInteger(points)) {
    return points.toString();
  }
  return points.toFixed(1);
}

/**
 * Get FIDE title display
 */
export function getTitleDisplay(title: string | null | undefined): string | null {
  if (!title) return null;

  const titles: Record<string, string> = {
    GM: 'GM',
    IM: 'IM',
    FM: 'FM',
    CM: 'CM',
    WGM: 'WGM',
    WIM: 'WIM',
    WFM: 'WFM',
    WCM: 'WCM',
  };

  return titles[title.toUpperCase()] || title;
}

/**
 * Get title CSS class for styling
 */
export function getTitleClass(title: string | null | undefined): string {
  if (!title) return '';

  const upper = title.toUpperCase();
  if (upper === 'GM' || upper === 'WGM') return 'title-gm';
  if (upper === 'IM' || upper === 'WIM') return 'title-im';
  if (upper === 'FM' || upper === 'WFM') return 'title-fm';
  if (upper === 'CM' || upper === 'WCM') return 'title-cm';
  return '';
}

/**
 * Get result display text
 */
export function getResultDisplay(result: string | null | undefined): string {
  if (!result) return '-';

  const results: Record<string, string> = {
    '1-0': '1-0',
    '0-1': '0-1',
    '1/2-1/2': '½-½',
    '1-0F': '1-0 F',
    '0-1F': '0-1 F',
    '0-0F': '0-0 F',
    BYE: 'BYE',
  };

  return results[result] || result;
}

/**
 * Get result CSS class
 */
export function getResultClass(result: string | null | undefined, forWhite: boolean): string {
  if (!result) return '';

  if (result === '1-0' || result === '1-0F') {
    return forWhite ? 'result-win' : 'result-loss';
  }
  if (result === '0-1' || result === '0-1F') {
    return forWhite ? 'result-loss' : 'result-win';
  }
  if (result === '1/2-1/2') {
    return 'result-draw';
  }
  if (result === 'BYE') {
    return 'result-draw';
  }

  return '';
}

/**
 * Generate a URL-safe slug from a string
 */
export function slugify(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replace(/[äÄ]/g, 'ae')
    .replace(/[öÖ]/g, 'oe')
    .replace(/[üÜ]/g, 'ue')
    .replace(/ß/g, 'ss')
    .replace(/[^\w\s-]/g, '')
    .replace(/[\s_-]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

/**
 * Truncate text with ellipsis
 */
export function truncate(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength - 1) + '…';
}

/**
 * Parse time control string (e.g., "90+30" -> { initial: 90, increment: 30 })
 */
export function parseTimeControl(tc: string): { initial: number; increment: number } | null {
  const match = tc.match(/^(\d+)\+(\d+)$/);
  if (!match) return null;
  return {
    initial: parseInt(match[1], 10),
    increment: parseInt(match[2], 10),
  };
}

/**
 * Debounce function
 */
export function debounce<T extends (...args: unknown[]) => unknown>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: NodeJS.Timeout | null = null;

  return function (...args: Parameters<T>) {
    if (timeout) clearTimeout(timeout);
    timeout = setTimeout(() => func(...args), wait);
  };
}

/**
 * Local storage helper with JSON parsing
 */
export const storage = {
  get<T>(key: string, defaultValue: T): T {
    if (typeof window === 'undefined') return defaultValue;
    try {
      const item = localStorage.getItem(key);
      return item ? JSON.parse(item) : defaultValue;
    } catch {
      return defaultValue;
    }
  },
  set<T>(key: string, value: T): void {
    if (typeof window === 'undefined') return;
    try {
      localStorage.setItem(key, JSON.stringify(value));
    } catch {
      // Storage full or disabled
    }
  },
  remove(key: string): void {
    if (typeof window === 'undefined') return;
    localStorage.removeItem(key);
  },
};
