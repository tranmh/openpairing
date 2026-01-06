import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  cn,
  formatDate,
  formatShortDate,
  formatTime,
  formatClockTime,
  formatRating,
  formatPoints,
  getTitleDisplay,
  getTitleClass,
  getResultDisplay,
  getResultClass,
  slugify,
  truncate,
  parseTimeControl,
  debounce,
  storage,
} from './utils';

describe('cn (className merge)', () => {
  it('merges class names', () => {
    expect(cn('foo', 'bar')).toBe('foo bar');
  });

  it('handles conditional classes', () => {
    expect(cn('foo', false && 'bar', 'baz')).toBe('foo baz');
  });

  it('merges tailwind classes correctly', () => {
    expect(cn('px-2 py-1', 'px-4')).toBe('py-1 px-4');
  });
});

describe('formatDate', () => {
  it('formats Date object', () => {
    const date = new Date('2024-06-15T10:00:00Z');
    const result = formatDate(date);
    expect(result).toContain('15');
    expect(result).toContain('2024');
  });

  it('formats date string', () => {
    const result = formatDate('2024-06-15');
    expect(result).toContain('15');
    expect(result).toContain('2024');
  });

  it('applies custom options', () => {
    const date = new Date('2024-06-15');
    const result = formatDate(date, { month: 'short' });
    expect(result).toContain('15');
  });
});

describe('formatShortDate', () => {
  it('formats date as DD.MM.YYYY', () => {
    const date = new Date('2024-06-15');
    const result = formatShortDate(date);
    expect(result).toBe('15.06.2024');
  });

  it('handles date string input', () => {
    const result = formatShortDate('2024-01-01');
    expect(result).toBe('01.01.2024');
  });
});

describe('formatTime', () => {
  it('formats time in HH:MM', () => {
    const date = new Date('2024-06-15T14:30:00');
    const result = formatTime(date);
    expect(result).toBe('14:30');
  });

  it('handles date string input', () => {
    const result = formatTime('2024-06-15T09:05:00');
    expect(result).toBe('09:05');
  });
});

describe('formatClockTime', () => {
  it('formats milliseconds to MM:SS', () => {
    expect(formatClockTime(90000)).toBe('1:30');
    expect(formatClockTime(60000)).toBe('1:00');
    expect(formatClockTime(30000)).toBe('0:30');
    expect(formatClockTime(5000)).toBe('0:05');
  });

  it('formats with hours when needed', () => {
    expect(formatClockTime(3600000)).toBe('1:00:00');
    expect(formatClockTime(5400000)).toBe('1:30:00');
    expect(formatClockTime(7265000)).toBe('2:01:05');
  });

  it('handles zero', () => {
    expect(formatClockTime(0)).toBe('0:00');
  });

  it('pads seconds correctly', () => {
    expect(formatClockTime(65000)).toBe('1:05');
  });
});

describe('formatRating', () => {
  it('formats valid rating', () => {
    expect(formatRating(2000)).toBe('2000');
    expect(formatRating(1850)).toBe('1850');
  });

  it('returns dash for null/undefined', () => {
    expect(formatRating(null)).toBe('-');
    expect(formatRating(undefined)).toBe('-');
  });

  it('returns dash for zero', () => {
    expect(formatRating(0)).toBe('-');
  });
});

describe('formatPoints', () => {
  it('formats integer points without decimal', () => {
    expect(formatPoints(5)).toBe('5');
    expect(formatPoints(0)).toBe('0');
  });

  it('formats half points with one decimal', () => {
    expect(formatPoints(5.5)).toBe('5.5');
    expect(formatPoints(3.5)).toBe('3.5');
  });
});

describe('getTitleDisplay', () => {
  it('returns standard titles', () => {
    expect(getTitleDisplay('GM')).toBe('GM');
    expect(getTitleDisplay('IM')).toBe('IM');
    expect(getTitleDisplay('FM')).toBe('FM');
    expect(getTitleDisplay('CM')).toBe('CM');
  });

  it('returns women titles', () => {
    expect(getTitleDisplay('WGM')).toBe('WGM');
    expect(getTitleDisplay('WIM')).toBe('WIM');
    expect(getTitleDisplay('WFM')).toBe('WFM');
    expect(getTitleDisplay('WCM')).toBe('WCM');
  });

  it('handles lowercase input', () => {
    expect(getTitleDisplay('gm')).toBe('GM');
    expect(getTitleDisplay('wgm')).toBe('WGM');
  });

  it('returns null for empty/null input', () => {
    expect(getTitleDisplay(null)).toBeNull();
    expect(getTitleDisplay(undefined)).toBeNull();
    expect(getTitleDisplay('')).toBeNull();
  });

  it('returns unknown title as-is', () => {
    expect(getTitleDisplay('NM')).toBe('NM');
  });
});

describe('getTitleClass', () => {
  it('returns correct class for grandmaster', () => {
    expect(getTitleClass('GM')).toBe('title-gm');
    expect(getTitleClass('WGM')).toBe('title-gm');
  });

  it('returns correct class for international master', () => {
    expect(getTitleClass('IM')).toBe('title-im');
    expect(getTitleClass('WIM')).toBe('title-im');
  });

  it('returns correct class for fide master', () => {
    expect(getTitleClass('FM')).toBe('title-fm');
    expect(getTitleClass('WFM')).toBe('title-fm');
  });

  it('returns correct class for candidate master', () => {
    expect(getTitleClass('CM')).toBe('title-cm');
    expect(getTitleClass('WCM')).toBe('title-cm');
  });

  it('returns empty string for null/undefined', () => {
    expect(getTitleClass(null)).toBe('');
    expect(getTitleClass(undefined)).toBe('');
  });

  it('handles case insensitivity', () => {
    expect(getTitleClass('gm')).toBe('title-gm');
  });
});

describe('getResultDisplay', () => {
  it('displays standard results', () => {
    expect(getResultDisplay('1-0')).toBe('1-0');
    expect(getResultDisplay('0-1')).toBe('0-1');
    expect(getResultDisplay('1/2-1/2')).toBe('½-½');
  });

  it('displays forfeit results', () => {
    expect(getResultDisplay('1-0F')).toBe('1-0 F');
    expect(getResultDisplay('0-1F')).toBe('0-1 F');
    expect(getResultDisplay('0-0F')).toBe('0-0 F');
  });

  it('displays bye', () => {
    expect(getResultDisplay('BYE')).toBe('BYE');
  });

  it('returns dash for null/undefined', () => {
    expect(getResultDisplay(null)).toBe('-');
    expect(getResultDisplay(undefined)).toBe('-');
  });

  it('returns unknown result as-is', () => {
    expect(getResultDisplay('unknown')).toBe('unknown');
  });
});

describe('getResultClass', () => {
  it('returns win class for white on 1-0', () => {
    expect(getResultClass('1-0', true)).toBe('result-win');
    expect(getResultClass('1-0F', true)).toBe('result-win');
  });

  it('returns loss class for black on 1-0', () => {
    expect(getResultClass('1-0', false)).toBe('result-loss');
    expect(getResultClass('1-0F', false)).toBe('result-loss');
  });

  it('returns loss class for white on 0-1', () => {
    expect(getResultClass('0-1', true)).toBe('result-loss');
    expect(getResultClass('0-1F', true)).toBe('result-loss');
  });

  it('returns win class for black on 0-1', () => {
    expect(getResultClass('0-1', false)).toBe('result-win');
    expect(getResultClass('0-1F', false)).toBe('result-win');
  });

  it('returns draw class for draw', () => {
    expect(getResultClass('1/2-1/2', true)).toBe('result-draw');
    expect(getResultClass('1/2-1/2', false)).toBe('result-draw');
  });

  it('returns draw class for bye', () => {
    expect(getResultClass('BYE', true)).toBe('result-draw');
  });

  it('returns empty string for null/undefined', () => {
    expect(getResultClass(null, true)).toBe('');
    expect(getResultClass(undefined, false)).toBe('');
  });
});

describe('slugify', () => {
  it('converts to lowercase', () => {
    expect(slugify('Test Tournament')).toBe('test-tournament');
  });

  it('replaces spaces with hyphens', () => {
    expect(slugify('my tournament name')).toBe('my-tournament-name');
  });

  it('handles German umlauts', () => {
    expect(slugify('Müller')).toBe('mueller');
    expect(slugify('Größe')).toBe('groesse');
    expect(slugify('Übung')).toBe('uebung');
    expect(slugify('Straße')).toBe('strasse');
  });

  it('removes special characters', () => {
    expect(slugify('Test! @#$% Tournament')).toBe('test-tournament');
  });

  it('handles multiple consecutive spaces/hyphens', () => {
    expect(slugify('test   tournament')).toBe('test-tournament');
    expect(slugify('test---tournament')).toBe('test-tournament');
  });

  it('trims leading/trailing hyphens', () => {
    expect(slugify('-test-')).toBe('test');
    expect(slugify('  test  ')).toBe('test');
  });
});

describe('truncate', () => {
  it('returns original string if shorter than max', () => {
    expect(truncate('hello', 10)).toBe('hello');
  });

  it('truncates long strings with ellipsis', () => {
    expect(truncate('hello world', 8)).toBe('hello w…');
  });

  it('handles exact length', () => {
    expect(truncate('hello', 5)).toBe('hello');
  });

  it('handles very short max length', () => {
    expect(truncate('hello', 2)).toBe('h…');
  });
});

describe('parseTimeControl', () => {
  it('parses standard time control', () => {
    expect(parseTimeControl('90+30')).toEqual({ initial: 90, increment: 30 });
    expect(parseTimeControl('15+10')).toEqual({ initial: 15, increment: 10 });
  });

  it('parses no increment', () => {
    expect(parseTimeControl('60+0')).toEqual({ initial: 60, increment: 0 });
  });

  it('returns null for invalid format', () => {
    expect(parseTimeControl('invalid')).toBeNull();
    expect(parseTimeControl('90')).toBeNull();
    expect(parseTimeControl('90-30')).toBeNull();
  });
});

describe('debounce', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('delays function execution', () => {
    const fn = vi.fn();
    const debounced = debounce(fn, 100);

    debounced();
    expect(fn).not.toHaveBeenCalled();

    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it('only calls once for rapid successive calls', () => {
    const fn = vi.fn();
    const debounced = debounce(fn, 100);

    debounced();
    debounced();
    debounced();

    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it('passes arguments to debounced function', () => {
    const fn = vi.fn();
    const debounced = debounce(fn, 100);

    debounced('arg1', 'arg2');
    vi.advanceTimersByTime(100);

    expect(fn).toHaveBeenCalledWith('arg1', 'arg2');
  });
});

describe('storage', () => {
  const mockLocalStorage = (() => {
    let store: Record<string, string> = {};
    return {
      getItem: vi.fn((key: string) => store[key] ?? null),
      setItem: vi.fn((key: string, value: string) => {
        store[key] = value;
      }),
      removeItem: vi.fn((key: string) => {
        delete store[key];
      }),
      clear: () => {
        store = {};
      },
    };
  })();

  beforeEach(() => {
    Object.defineProperty(global, 'localStorage', {
      value: mockLocalStorage,
      writable: true,
    });
    mockLocalStorage.clear();
    vi.clearAllMocks();
  });

  describe('get', () => {
    it('returns parsed value from localStorage', () => {
      mockLocalStorage.getItem.mockReturnValue('{"foo":"bar"}');
      expect(storage.get('test', {})).toEqual({ foo: 'bar' });
    });

    it('returns default value when key not found', () => {
      mockLocalStorage.getItem.mockReturnValue(null);
      expect(storage.get('missing', 'default')).toBe('default');
    });

    it('returns default value on parse error', () => {
      mockLocalStorage.getItem.mockReturnValue('invalid json');
      expect(storage.get('test', 'default')).toBe('default');
    });
  });

  describe('set', () => {
    it('stores stringified value', () => {
      storage.set('test', { foo: 'bar' });
      expect(mockLocalStorage.setItem).toHaveBeenCalledWith(
        'test',
        '{"foo":"bar"}'
      );
    });
  });

  describe('remove', () => {
    it('removes item from localStorage', () => {
      storage.remove('test');
      expect(mockLocalStorage.removeItem).toHaveBeenCalledWith('test');
    });
  });
});
