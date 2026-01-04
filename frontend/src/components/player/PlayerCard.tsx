'use client';

import Link from 'next/link';

interface Player {
  id: string;
  firstName: string;
  lastName: string;
  fideId?: string;
  dsbId?: string;
  oesbId?: string;
  fideStandard?: number;
  fideRapid?: number;
  fideBlitz?: number;
  dwz?: number;
  oesbElo?: number;
  fideTitle?: string;
  clubName?: string;
  federation?: string;
}

interface PlayerCardProps {
  player: Player;
  showDetails?: boolean;
}

const titleColors: Record<string, string> = {
  GM: 'bg-yellow-500 text-black',
  IM: 'bg-gray-400 text-black',
  FM: 'bg-amber-600 text-white',
  CM: 'bg-amber-800 text-white',
  WGM: 'bg-yellow-500 text-black',
  WIM: 'bg-gray-400 text-black',
  WFM: 'bg-amber-600 text-white',
  WCM: 'bg-amber-800 text-white',
};

export function PlayerCard({ player, showDetails = false }: PlayerCardProps) {
  const highestRating = Math.max(
    player.fideStandard ?? 0,
    player.dwz ?? 0,
    player.oesbElo ?? 0
  );

  return (
    <Link href={`/players/${player.id}`}>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-4 hover:shadow-md transition-shadow cursor-pointer">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <div className="flex items-center gap-2">
              {player.fideTitle && (
                <span
                  className={`px-1.5 py-0.5 text-xs font-bold rounded ${
                    titleColors[player.fideTitle] ?? 'bg-gray-200 text-gray-800'
                  }`}
                >
                  {player.fideTitle}
                </span>
              )}
              <h3 className="font-semibold text-gray-900 dark:text-white">
                {player.lastName}, {player.firstName}
              </h3>
            </div>

            {player.clubName && (
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                {player.clubName}
              </p>
            )}

            {showDetails && (
              <div className="mt-2 flex flex-wrap gap-2 text-xs text-gray-500">
                {player.fideId && (
                  <span className="bg-gray-100 dark:bg-gray-700 px-2 py-0.5 rounded">
                    FIDE: {player.fideId}
                  </span>
                )}
                {player.dsbId && (
                  <span className="bg-gray-100 dark:bg-gray-700 px-2 py-0.5 rounded">
                    DSB: {player.dsbId}
                  </span>
                )}
                {player.oesbId && (
                  <span className="bg-gray-100 dark:bg-gray-700 px-2 py-0.5 rounded">
                    ÖSB: {player.oesbId}
                  </span>
                )}
              </div>
            )}
          </div>

          <div className="text-right">
            {highestRating > 0 && (
              <div className="text-xl font-bold text-gray-900 dark:text-white">
                {highestRating}
              </div>
            )}
            {showDetails && (
              <div className="mt-1 space-y-0.5 text-xs text-gray-500">
                {player.fideStandard && (
                  <div>FIDE: {player.fideStandard}</div>
                )}
                {player.dwz && <div>DWZ: {player.dwz}</div>}
                {player.oesbElo && <div>ÖSB: {player.oesbElo}</div>}
              </div>
            )}
          </div>
        </div>
      </div>
    </Link>
  );
}
