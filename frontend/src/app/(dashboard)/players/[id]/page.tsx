'use client';

import { useParams } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import { TitleBadge } from '@/components/player/TitleBadge';
import { RatingDisplay } from '@/components/player/RatingBadge';
import { Player } from '@/lib/api';

interface RatingHistory {
  id: string;
  ratingType: string;
  oldRating?: number;
  newRating: number;
  changeDate: string;
  source: string;
}

async function getPlayer(id: string): Promise<Player> {
  const response = await fetch(`/api/v1/players/${id}`);
  if (!response.ok) {
    if (response.status === 404) {
      throw new Error('Player not found');
    }
    throw new Error('Failed to fetch player');
  }
  return response.json();
}

async function getRatingHistory(id: string): Promise<{ history: RatingHistory[] }> {
  const response = await fetch(`/api/v1/players/${id}/history?limit=20`);
  if (!response.ok) {
    throw new Error('Failed to fetch rating history');
  }
  return response.json();
}

export default function PlayerDetailPage() {
  const params = useParams();
  const playerId = params.id as string;

  const {
    data: player,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['player', playerId],
    queryFn: () => getPlayer(playerId),
  });

  const { data: historyData } = useQuery({
    queryKey: ['player-history', playerId],
    queryFn: () => getRatingHistory(playerId),
    enabled: !!player,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (error || !player) {
    return (
      <div className="text-center py-12">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
          Spieler nicht gefunden
        </h2>
        <p className="mt-2 text-gray-500 dark:text-gray-400">
          Der gesuchte Spieler existiert nicht oder wurde entfernt.
        </p>
        <Link
          href="/players"
          className="mt-4 inline-block text-blue-600 hover:text-blue-700"
        >
          ← Zurück zur Spielerliste
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Back link */}
      <Link
        href="/players"
        className="inline-flex items-center text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
      >
        <svg
          className="w-4 h-4 mr-1"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M15 19l-7-7 7-7"
          />
        </svg>
        Zurück zur Spielerliste
      </Link>

      {/* Player Header */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <div className="flex items-start justify-between">
          <div>
            <div className="flex items-center gap-3">
              <TitleBadge title={player.fideTitle} size="lg" />
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                {player.firstName} {player.lastName}
              </h1>
            </div>
            {player.clubName && (
              <p className="mt-1 text-gray-500 dark:text-gray-400">
                {player.clubName}
              </p>
            )}
            {player.federation && (
              <p className="mt-1 text-sm text-gray-400 dark:text-gray-500">
                {player.federation}
              </p>
            )}
          </div>

          <div className="text-right">
            <RatingDisplay
              fideStandard={player.fideStandard}
              fideRapid={player.fideRapid}
              fideBlitz={player.fideBlitz}
              dwz={player.dwz}
              oesbElo={player.oesbElo}
              layout="vertical"
            />
          </div>
        </div>
      </div>

      {/* Federation IDs */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Verbands-IDs
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {player.fideId && (
            <div>
              <dt className="text-sm text-gray-500 dark:text-gray-400">
                FIDE-ID
              </dt>
              <dd className="mt-1 text-gray-900 dark:text-white font-mono">
                <a
                  href={`https://ratings.fide.com/profile/${player.fideId}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-600 hover:text-blue-700 hover:underline"
                >
                  {player.fideId}
                </a>
              </dd>
            </div>
          )}
          {player.dsbId && (
            <div>
              <dt className="text-sm text-gray-500 dark:text-gray-400">
                DSB PKZ
              </dt>
              <dd className="mt-1 text-gray-900 dark:text-white font-mono">
                <a
                  href={`https://www.schachbund.de/spieler/${player.dsbId}.html`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-600 hover:text-blue-700 hover:underline"
                >
                  {player.dsbId}
                </a>
              </dd>
            </div>
          )}
          {player.oesbId && (
            <div>
              <dt className="text-sm text-gray-500 dark:text-gray-400">
                ÖSB-ID
              </dt>
              <dd className="mt-1 text-gray-900 dark:text-white font-mono">
                {player.oesbId}
              </dd>
            </div>
          )}
          {!player.fideId && !player.dsbId && !player.oesbId && (
            <p className="text-gray-500 dark:text-gray-400 col-span-3">
              Keine Verbands-IDs hinterlegt
            </p>
          )}
        </div>
      </div>

      {/* Rating History */}
      {historyData && historyData.history && historyData.history.length > 0 && (
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
            Wertungsverlauf
          </h2>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead>
                <tr>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Datum
                  </th>
                  <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Typ
                  </th>
                  <th className="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Alt
                  </th>
                  <th className="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Neu
                  </th>
                  <th className="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Änderung
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {historyData.history.map((entry) => {
                  const change = entry.oldRating
                    ? entry.newRating - entry.oldRating
                    : null;
                  return (
                    <tr key={entry.id}>
                      <td className="px-4 py-2 text-sm text-gray-900 dark:text-white">
                        {new Date(entry.changeDate).toLocaleDateString('de-DE')}
                      </td>
                      <td className="px-4 py-2 text-sm text-gray-500 dark:text-gray-400">
                        {entry.ratingType.replace('_', ' ').toUpperCase()}
                      </td>
                      <td className="px-4 py-2 text-sm text-gray-500 dark:text-gray-400 text-right">
                        {entry.oldRating ?? '-'}
                      </td>
                      <td className="px-4 py-2 text-sm font-medium text-gray-900 dark:text-white text-right">
                        {entry.newRating}
                      </td>
                      <td
                        className={`px-4 py-2 text-sm font-medium text-right ${
                          change && change > 0
                            ? 'text-green-600'
                            : change && change < 0
                              ? 'text-red-600'
                              : 'text-gray-500'
                        }`}
                      >
                        {change !== null
                          ? change > 0
                            ? `+${change}`
                            : change
                          : '-'}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
