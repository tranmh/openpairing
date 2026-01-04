'use client';

import { useState, useEffect } from 'react';
import { useSearchParams } from 'next/navigation';
import { PlayerCard } from '@/components/player/PlayerCard';
import {
  PlayerSearch,
  FederationFilter,
  RatingRangeFilter,
} from '@/components/player/PlayerSearch';
import { useQuery } from '@tanstack/react-query';
import { apiClient, Player } from '@/lib/api';

interface PlayerListResponse {
  players: Player[];
  total: number;
  page: number;
  limit: number;
}

async function searchPlayers(params: {
  q?: string;
  federation?: string;
  minRating?: number;
  maxRating?: number;
  page?: number;
  limit?: number;
}): Promise<PlayerListResponse> {
  const searchParams = new URLSearchParams();
  if (params.q) searchParams.set('q', params.q);
  if (params.federation) searchParams.set('federation', params.federation);
  if (params.minRating) searchParams.set('minRating', params.minRating.toString());
  if (params.maxRating) searchParams.set('maxRating', params.maxRating.toString());
  searchParams.set('page', (params.page ?? 1).toString());
  searchParams.set('limit', (params.limit ?? 20).toString());

  const response = await fetch(`/api/v1/players?${searchParams.toString()}`);
  if (!response.ok) {
    throw new Error('Failed to fetch players');
  }
  return response.json();
}

export default function PlayersPage() {
  const searchParams = useSearchParams();
  const [federation, setFederation] = useState<string | undefined>(
    searchParams.get('federation') ?? undefined
  );
  const [minRating, setMinRating] = useState<number | undefined>();
  const [maxRating, setMaxRating] = useState<number | undefined>();
  const [page, setPage] = useState(Number(searchParams.get('page')) || 1);

  const query = searchParams.get('q') ?? '';

  const { data, isLoading, error } = useQuery({
    queryKey: ['players', query, federation, minRating, maxRating, page],
    queryFn: () =>
      searchPlayers({
        q: query,
        federation,
        minRating,
        maxRating,
        page,
        limit: 20,
      }),
  });

  const totalPages = data ? Math.ceil(data.total / data.limit) : 0;

  const handleRatingChange = (min?: number, max?: number) => {
    setMinRating(min);
    setMaxRating(max);
    setPage(1);
  };

  const handleFederationChange = (fed?: string) => {
    setFederation(fed);
    setPage(1);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
          Spielerdatenbank
        </h1>
        <p className="text-gray-500 dark:text-gray-400 mt-1">
          Suchen Sie nach Spielern aus FIDE, DSB und ÖSB
        </p>
      </div>

      {/* Search and Filters */}
      <div className="space-y-4">
        <PlayerSearch placeholder="Name, FIDE-ID, PKZ oder ÖSB-ID eingeben..." />

        <div className="flex flex-wrap items-center gap-4">
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-500 dark:text-gray-400">
              Verband:
            </span>
            <FederationFilter
              selected={federation}
              onChange={handleFederationChange}
            />
          </div>

          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-500 dark:text-gray-400">
              Wertungszahl:
            </span>
            <RatingRangeFilter
              minRating={minRating}
              maxRating={maxRating}
              onChange={handleRatingChange}
            />
          </div>
        </div>
      </div>

      {/* Results */}
      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        </div>
      ) : error ? (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
          <p className="text-red-800 dark:text-red-200">
            Fehler beim Laden der Spieler. Bitte versuchen Sie es erneut.
          </p>
        </div>
      ) : data && data.players.length > 0 ? (
        <>
          {/* Result count */}
          <div className="text-sm text-gray-500 dark:text-gray-400">
            {data.total} Spieler gefunden
          </div>

          {/* Player grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {data.players.map((player) => (
              <PlayerCard key={player.id} player={player} showDetails />
            ))}
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                className="px-4 py-2 text-sm font-medium bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Zurück
              </button>

              <span className="text-sm text-gray-500 dark:text-gray-400">
                Seite {page} von {totalPages}
              </span>

              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="px-4 py-2 text-sm font-medium bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Weiter
              </button>
            </div>
          )}
        </>
      ) : (
        <div className="text-center py-12">
          <svg
            className="mx-auto h-12 w-12 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
            />
          </svg>
          <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-white">
            Keine Spieler gefunden
          </h3>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Versuchen Sie eine andere Suche oder passen Sie die Filter an.
          </p>
        </div>
      )}
    </div>
  );
}
