import type { Tournament, TournamentRound } from '@/lib/api';

/**
 * Creates a mock tournament with optional overrides.
 */
export function createMockTournament(overrides?: Partial<Tournament>): Tournament {
  return {
    id: 'tournament-1',
    name: 'Test Tournament',
    slug: 'test-tournament',
    type: 'swiss',
    mode: 'otb',
    pairingEngine: 'javafo',
    pairingSystem: 'dutch',
    startDate: '2024-06-15',
    rounds: 7,
    status: 'draft',
    ...overrides,
  };
}

/**
 * Creates a mock Swiss tournament.
 */
export function createSwissTournament(overrides?: Partial<Tournament>): Tournament {
  return createMockTournament({
    name: 'Swiss Open',
    type: 'swiss',
    pairingEngine: 'javafo',
    pairingSystem: 'dutch',
    rounds: 9,
    maxPlayers: 100,
    ...overrides,
  });
}

/**
 * Creates a mock round robin tournament.
 */
export function createRoundRobinTournament(overrides?: Partial<Tournament>): Tournament {
  return createMockTournament({
    name: 'Round Robin',
    type: 'round_robin',
    pairingEngine: 'bbp',
    pairingSystem: 'dutch',
    rounds: 9,
    maxPlayers: 10,
    ...overrides,
  });
}

/**
 * Creates a mock ongoing tournament.
 */
export function createOngoingTournament(overrides?: Partial<Tournament>): Tournament {
  return createMockTournament({
    status: 'ongoing',
    currentRound: 3,
    ...overrides,
  });
}

/**
 * Creates a mock completed tournament.
 */
export function createCompletedTournament(overrides?: Partial<Tournament>): Tournament {
  return createMockTournament({
    status: 'completed',
    currentRound: 7,
    ...overrides,
  });
}

/**
 * Creates a mock tournament round.
 */
export function createMockRound(overrides?: Partial<TournamentRound>): TournamentRound {
  return {
    id: 'round-1',
    tournamentId: 'tournament-1',
    roundNumber: 1,
    status: 'pending',
    ...overrides,
  };
}

/**
 * Creates multiple rounds for a tournament.
 */
export function createTournamentRounds(
  count: number,
  completedRounds = 0
): TournamentRound[] {
  return Array.from({ length: count }, (_, i) => ({
    id: `round-${i + 1}`,
    tournamentId: 'tournament-1',
    roundNumber: i + 1,
    status: i < completedRounds ? 'completed' : 'pending',
    ...(i < completedRounds && {
      actualStart: `2024-06-${15 + i}T09:00:00Z`,
      completedAt: `2024-06-${15 + i}T18:00:00Z`,
    }),
  }));
}

/**
 * Tournament status options for testing UI states.
 */
export const tournamentStatuses = [
  'draft',
  'published',
  'registration',
  'ongoing',
  'pairing',
  'completed',
  'cancelled',
] as const;

/**
 * Tournament types for testing.
 */
export const tournamentTypes = [
  'swiss',
  'round_robin',
  'knockout',
  'team_swiss',
] as const;
