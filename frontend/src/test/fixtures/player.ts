import type { Player, TournamentParticipant } from '@/lib/api';

/**
 * Creates a mock player with optional overrides.
 */
export function createMockPlayer(overrides?: Partial<Player>): Player {
  return {
    id: 'player-1',
    firstName: 'Test',
    lastName: 'Player',
    fideStandard: 1800,
    ...overrides,
  };
}

/**
 * Creates a mock grandmaster player.
 */
export function createGrandmasterPlayer(overrides?: Partial<Player>): Player {
  return createMockPlayer({
    id: 'gm-player',
    firstName: 'Magnus',
    lastName: 'Carlsen',
    fideId: '1503014',
    fideStandard: 2830,
    fideRapid: 2820,
    fideBlitz: 2885,
    fideTitle: 'GM',
    federation: 'NOR',
    ...overrides,
  });
}

/**
 * Creates a mock club player.
 */
export function createClubPlayer(overrides?: Partial<Player>): Player {
  return createMockPlayer({
    id: 'club-player',
    firstName: 'Club',
    lastName: 'Player',
    fideStandard: 1650,
    dwz: 1720,
    federation: 'GER',
    ...overrides,
  });
}

/**
 * Creates a mock unrated player.
 */
export function createUnratedPlayer(overrides?: Partial<Player>): Player {
  return createMockPlayer({
    id: 'unrated-player',
    firstName: 'New',
    lastName: 'Player',
    fideStandard: undefined,
    dwz: undefined,
    ...overrides,
  });
}

/**
 * Creates multiple mock players for a tournament field.
 */
export function createPlayerField(count: number): Player[] {
  return Array.from({ length: count }, (_, i) => ({
    id: `player-${i + 1}`,
    firstName: `Player`,
    lastName: `${i + 1}`,
    fideStandard: 2200 - i * 50,
    federation: 'GER',
  }));
}

/**
 * Creates a mock tournament participant.
 */
export function createMockParticipant(
  overrides?: Partial<TournamentParticipant>
): TournamentParticipant {
  return {
    id: 'participant-1',
    tournamentId: 'tournament-1',
    playerId: 'player-1',
    registrationStatus: 'confirmed',
    points: 0,
    buchholz: 0,
    buchholzCut1: 0,
    sonnebornBerger: 0,
    wins: 0,
    draws: 0,
    losses: 0,
    gamesPlayed: 0,
    ...overrides,
  };
}

/**
 * Creates a mock participant with a player attached.
 */
export function createParticipantWithPlayer(
  player?: Partial<Player>,
  participant?: Partial<TournamentParticipant>
): TournamentParticipant {
  const mockPlayer = createMockPlayer(player);
  return createMockParticipant({
    playerId: mockPlayer.id,
    player: mockPlayer,
    ...participant,
  });
}
