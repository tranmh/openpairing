'use client';

import { useState, useCallback, useEffect } from 'react';
import { useRouter, useSearchParams, usePathname } from 'next/navigation';
import { useDebouncedCallback } from 'use-debounce';
import { useLanguageStore } from '@/store/language';

interface PlayerSearchProps {
  onSearch?: (query: string) => void;
  placeholder?: string;
  autoFocus?: boolean;
}

export function PlayerSearch({
  onSearch,
  placeholder,
  autoFocus = false,
}: PlayerSearchProps) {
  const { t } = useLanguageStore();
  const searchPlaceholder = placeholder || t('player.searchPlayers');
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [query, setQuery] = useState(searchParams.get('q') ?? '');

  const debouncedSearch = useDebouncedCallback((value: string) => {
    const params = new URLSearchParams(searchParams.toString());
    if (value) {
      params.set('q', value);
    } else {
      params.delete('q');
    }
    params.set('page', '1'); // Reset to first page on new search
    router.push(`${pathname}?${params.toString()}`);
    onSearch?.(value);
  }, 300);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setQuery(value);
    debouncedSearch(value);
  };

  const handleClear = () => {
    setQuery('');
    debouncedSearch('');
  };

  return (
    <div className="relative">
      <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
        <svg
          className="h-5 w-5 text-gray-400"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
          />
        </svg>
      </div>
      <input
        type="text"
        value={query}
        onChange={handleChange}
        placeholder={searchPlaceholder}
        autoFocus={autoFocus}
        className="block w-full pl-10 pr-10 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
      />
      {query && (
        <button
          onClick={handleClear}
          className="absolute inset-y-0 right-0 pr-3 flex items-center text-gray-400 hover:text-gray-600"
        >
          <svg
            className="h-5 w-5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </button>
      )}
    </div>
  );
}

interface FederationFilterProps {
  selected?: string;
  onChange: (federation?: string) => void;
}

export function FederationFilter({ selected, onChange }: FederationFilterProps) {
  const { t } = useLanguageStore();
  const federations = [
    { value: '', label: t('common.all') },
    { value: 'fide', label: 'FIDE' },
    { value: 'dsb', label: 'DSB' },
    { value: 'oesb', label: 'ÖSB' },
  ];

  return (
    <div className="flex gap-2">
      {federations.map((fed) => (
        <button
          key={fed.value}
          onClick={() => onChange(fed.value || undefined)}
          className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
            (selected ?? '') === fed.value
              ? 'bg-blue-600 text-white'
              : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
          }`}
        >
          {fed.label}
        </button>
      ))}
    </div>
  );
}

interface RatingRangeFilterProps {
  minRating?: number;
  maxRating?: number;
  onChange: (min?: number, max?: number) => void;
}

export function RatingRangeFilter({
  minRating,
  maxRating,
  onChange,
}: RatingRangeFilterProps) {
  const { t } = useLanguageStore();
  const [min, setMin] = useState(minRating?.toString() ?? '');
  const [max, setMax] = useState(maxRating?.toString() ?? '');

  const handleApply = () => {
    onChange(
      min ? parseInt(min, 10) : undefined,
      max ? parseInt(max, 10) : undefined
    );
  };

  return (
    <div className="flex items-center gap-2">
      <input
        type="number"
        value={min}
        onChange={(e) => setMin(e.target.value)}
        placeholder={t('common.min')}
        className="w-20 px-2 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
      />
      <span className="text-gray-500">-</span>
      <input
        type="number"
        value={max}
        onChange={(e) => setMax(e.target.value)}
        placeholder={t('common.max')}
        className="w-20 px-2 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
      />
      <button
        onClick={handleApply}
        className="px-3 py-1.5 text-sm font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700"
      >
        {t('common.apply')}
      </button>
    </div>
  );
}
