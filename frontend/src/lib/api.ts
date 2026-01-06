/**
 * API client for OpenPairing backend
 */

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || '/api/v1';

interface ApiError {
  message: string;
  code?: string;
  details?: Record<string, string[]>;
}

interface ApiResponse<T> {
  data?: T;
  error?: ApiError;
  status: number;
}

class ApiClient {
  private baseUrl: string;
  private accessToken: string | null = null;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  setAccessToken(token: string | null) {
    this.accessToken = token;
  }

  /**
   * Get the access token, falling back to localStorage if not set.
   * This ensures the token is available even if setAccessToken wasn't called
   * due to timing issues during hydration.
   */
  private getAccessToken(): string | null {
    if (this.accessToken) {
      return this.accessToken;
    }
    // Fallback: try to read from localStorage if available
    if (typeof window !== 'undefined') {
      try {
        const stored = localStorage.getItem('openpairing-auth');
        if (stored) {
          const parsed = JSON.parse(stored);
          if (parsed?.state?.accessToken) {
            // Also update the local cache
            this.accessToken = parsed.state.accessToken;
            return this.accessToken;
          }
        }
      } catch {
        // Ignore localStorage errors
      }
    }
    return null;
  }

  private async request<T>(
    method: string,
    path: string,
    options?: {
      body?: unknown;
      params?: Record<string, string>;
      headers?: Record<string, string>;
    }
  ): Promise<ApiResponse<T>> {
    let url = `${this.baseUrl}${path}`;

    if (options?.params) {
      const searchParams = new URLSearchParams(options.params);
      url += `?${searchParams.toString()}`;
    }

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...options?.headers,
    };

    const token = this.getAccessToken();
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    try {
      const response = await fetch(url, {
        method,
        headers,
        body: options?.body ? JSON.stringify(options.body) : undefined,
        credentials: 'include',
      });

      const status = response.status;

      if (status === 204) {
        return { status };
      }

      // Check Content-Type before parsing JSON
      const contentType = response.headers.get('content-type');
      let data: unknown;

      if (contentType?.includes('application/json')) {
        try {
          data = await response.json();
        } catch {
          return {
            status,
            error: {
              message: 'Invalid JSON response from server',
              code: 'PARSE_ERROR',
            },
          };
        }
      } else {
        // Non-JSON response
        const text = await response.text();
        if (!response.ok) {
          return {
            status,
            error: {
              message: text || `HTTP ${status} error`,
              code: 'NON_JSON_ERROR',
            },
          };
        }
        return { data: text as T, status };
      }

      if (!response.ok) {
        const errorData = data as Record<string, unknown>;
        // Backend returns { error: { message, code } } format
        const errorObj = errorData.error as { message?: string; code?: string } | string | undefined;
        let message: string;
        let code: string | undefined;

        if (typeof errorObj === 'object' && errorObj !== null) {
          message = errorObj.message || 'An error occurred';
          code = errorObj.code;
        } else if (typeof errorObj === 'string') {
          message = errorObj;
        } else {
          message = (errorData.message as string) || 'An error occurred';
          code = errorData.code as string | undefined;
        }

        return {
          status,
          error: {
            message,
            code,
            details: errorData.details as Record<string, string[]> | undefined,
          },
        };
      }

      return { data: data as T, status };
    } catch (error) {
      return {
        status: 0,
        error: {
          message: error instanceof Error ? error.message : 'Network error',
          code: 'NETWORK_ERROR',
        },
      };
    }
  }

  // Generic HTTP methods
  get<T>(path: string, params?: Record<string, string>) {
    return this.request<T>('GET', path, { params });
  }

  post<T>(path: string, body?: unknown) {
    return this.request<T>('POST', path, { body });
  }

  put<T>(path: string, body?: unknown) {
    return this.request<T>('PUT', path, { body });
  }

  patch<T>(path: string, body?: unknown) {
    return this.request<T>('PATCH', path, { body });
  }

  delete<T>(path: string) {
    return this.request<T>('DELETE', path);
  }
}

// Singleton instance
export const api = new ApiClient(API_BASE_URL);

// Type definitions for API responses
export interface User {
  id: string;
  email: string;
  role: 'player' | 'arbiter' | 'organizer' | 'admin';
  preferredLanguage: string;
  createdAt: string;
}

export interface Player {
  id: string;
  userId?: string;
  firstName: string;
  lastName: string;
  birthDate?: string;
  gender?: string;
  fideId?: string;
  dsbId?: string;
  oesbId?: string;
  fideStandard?: number;
  fideRapid?: number;
  fideBlitz?: number;
  dwz?: number;
  dwzIndex?: number;
  oesbElo?: number;
  fideTitle?: string;
  nationalTitle?: string;
  clubId?: string;
  clubName?: string;
  federation?: string;
}

export interface Tournament {
  id: string;
  organizationId?: string;
  name: string;
  description?: string;
  slug: string;
  type: 'swiss' | 'round_robin' | 'knockout' | 'team_swiss';
  mode: 'otb' | 'online' | 'hybrid';
  timeControl?: string;
  timeControlType?: 'standard' | 'rapid' | 'blitz';
  pairingEngine: 'javafo' | 'bbp' | 'wam';
  pairingSystem: 'dutch' | 'burstein' | 'dubov';
  startDate: string;
  endDate?: string;
  rounds: number;
  maxPlayers?: number;
  status: 'draft' | 'published' | 'registration' | 'ongoing' | 'completed' | 'cancelled';
  fideRated: boolean;
  dwzRated: boolean;
  oesbRated: boolean;
  tiebreaks: string[];
}

export interface TournamentParticipant {
  id: string;
  tournamentId: string;
  playerId: string;
  player?: Player;
  registrationStatus: 'registered' | 'confirmed' | 'withdrawn' | 'expelled';
  startRank?: number;
  startingRating?: number;
  manualRatingOverride?: number | null;
  points: number;
  buchholz: number;
  buchholzCut1: number;
  sonnebornBerger: number;
  wins: number;
  draws: number;
  losses: number;
  isActive: boolean;
  // Player info from join
  playerFirstName?: string;
  playerLastName?: string;
  playerFideId?: string;
  playerClubName?: string;
}

export interface Game {
  id: string;
  tournamentId: string;
  roundId: string;
  boardNumber: number;
  whiteParticipantId?: string;
  blackParticipantId?: string;
  whiteParticipant?: TournamentParticipant;
  blackParticipant?: TournamentParticipant;
  result?: string;
  whiteScore?: number;
  blackScore?: number;
  status: 'scheduled' | 'ongoing' | 'completed' | 'adjourned';
  pgn?: string;
}

export interface TournamentRound {
  id: string;
  tournamentId: string;
  roundNumber: number;
  scheduledStart?: string;
  actualStart?: string;
  completedAt?: string;
  status: 'pending' | 'pairing' | 'paired' | 'ongoing' | 'completed';
}

// Tournament Operations types
export interface CloneTournamentRequest {
  name: string;
  description?: string;
  startDate?: string;
  endDate?: string;
}

export interface ForbiddenPair {
  id: string;
  tournamentId: string;
  player1Id: string;
  player2Id: string;
  player1Name?: string;
  player2Name?: string;
  reason: string;
  createdAt: string;
}

export interface AddForbiddenPairRequest {
  player1Id: string;
  player2Id: string;
  reason?: string;
}

export interface SpecialRanking {
  id: string;
  tournamentId: string;
  name: string;
  displayName: string;
  minBirthYear?: number;
  maxBirthYear?: number;
  minRating?: number;
  maxRating?: number;
  gender?: string;
  sortOrder: number;
  isActive: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface CreateSpecialRankingRequest {
  name: string;
  displayName: string;
  minBirthYear?: number;
  maxBirthYear?: number;
  minRating?: number;
  maxRating?: number;
  gender?: string;
  sortOrder: number;
}

export interface UpdateSpecialRankingRequest {
  name?: string;
  displayName?: string;
  minBirthYear?: number;
  maxBirthYear?: number;
  minRating?: number;
  maxRating?: number;
  gender?: string;
  sortOrder?: number;
  isActive?: boolean;
}

export interface LateEntryRequest {
  playerId?: string;  // UUID of existing player (optional if dsbId provided)
  dsbId?: string;     // PKZ for importing from DSB/portal64
  sectionId?: string;
  startingRating?: number;
  entryRound: number;
}

// Auth API
export const authApi = {
  register: (data: { email: string; password: string; firstName: string; lastName: string }) =>
    api.post<{ user: User }>('/auth/register', data),

  login: (email: string, password: string) =>
    api.post<{ user: User; accessToken: string; refreshToken: string }>('/auth/login', {
      email,
      password,
    }),

  logout: () => api.post('/auth/logout'),

  refreshToken: () => api.post<{ accessToken: string }>('/auth/refresh'),

  forgotPassword: (email: string) => api.post('/auth/forgot-password', { email }),

  resetPassword: (token: string, password: string) =>
    api.post('/auth/reset-password', { token, password }),
};

// Players API
export const playersApi = {
  search: (query: string, params?: { federation?: string; minRating?: number; maxRating?: number }) => {
    const queryParams: Record<string, string> = { q: query };
    if (params?.federation) queryParams.federation = params.federation;
    if (params?.minRating !== undefined) queryParams.minRating = String(params.minRating);
    if (params?.maxRating !== undefined) queryParams.maxRating = String(params.maxRating);
    return api.get<{ players: Player[]; total: number }>('/players', queryParams);
  },

  get: (id: string) => api.get<Player>(`/players/${id}`),

  getHistory: (id: string) => api.get<{ history: { date: string; rating: number; type: string }[] }>(`/players/${id}/history`),

  searchFederation: (query: string, federation: 'fide' | 'dsb' | 'oesb') =>
    api.post<{ players: Player[] }>('/players/search-federation', { query, federation }),
};

// Tournaments API
export const tournamentsApi = {
  list: (params?: { status?: string; page?: number; limit?: number }) =>
    api.get<{ tournaments: Tournament[]; total: number }>('/tournaments', params as Record<string, string>),

  get: (id: string) => api.get<{ tournament: Tournament }>(`/tournaments/${id}`),

  create: (data: Partial<Tournament>) => api.post<{ tournament: Tournament }>('/tournaments', data),

  update: (id: string, data: Partial<Tournament>) => api.put<{ tournament: Tournament }>(`/tournaments/${id}`, data),

  delete: (id: string) => api.delete(`/tournaments/${id}`),

  getParticipants: (id: string) =>
    api.get<{ participants: TournamentParticipant[] }>(`/tournaments/${id}/participants`),

  register: (id: string, playerId: string) =>
    api.post<TournamentParticipant>(`/tournaments/${id}/participants`, { playerId }),

  getRounds: (id: string) => api.get<{ rounds: TournamentRound[] }>(`/tournaments/${id}/rounds`),

  getPairings: (id: string, roundNumber: number) =>
    api.get<{ games: Game[] }>(`/tournaments/${id}/rounds/${roundNumber}/pairings`),

  generatePairings: (id: string, roundNumber: number) =>
    api.post<{ games: Game[] }>(`/tournaments/${id}/rounds/${roundNumber}/pair`),

  getStandings: (id: string) =>
    api.get<{ standings: TournamentParticipant[] }>(`/tournaments/${id}/standings`),

  enterResult: (_tournamentId: string, gameId: string, result: string) =>
    api.post<{ game: Game }>(`/results`, { gameId, result }),

  // Round Robin Seating Chart
  getSeatingChart: (id: string, round?: number) =>
    api.get<SeatingChartData>(
      `/tournaments/${id}/seating-chart`,
      round ? { round: String(round) } : undefined
    ),

  getRotationData: (id: string, fromRound: number, toRound: number) =>
    api.get<RotationData>(`/tournaments/${id}/rotation/${fromRound}/${toRound}`),

  getRoundRobinSettings: (id: string) =>
    api.get<RoundRobinSettingsResponse>(`/tournaments/${id}/round-robin-settings`),

  updateRoundRobinSettings: (id: string, settings: UpdateRoundRobinSettingsRequest) =>
    api.put<RoundRobinSettingsResponse>(`/tournaments/${id}/round-robin-settings`, settings),

  // Clone tournament
  cloneTournament: (id: string, data: CloneTournamentRequest) =>
    api.post<{ tournament: Tournament }>(`/tournaments/${id}/clone`, data),

  // Forbidden pairs
  getForbiddenPairs: (id: string) =>
    api.get<{ forbiddenPairs: ForbiddenPair[] }>(`/tournaments/${id}/forbidden-pairs`),
  addForbiddenPair: (id: string, data: AddForbiddenPairRequest) =>
    api.post<void>(`/tournaments/${id}/forbidden-pairs`, data), // Returns 201 with no body
  removeForbiddenPair: (id: string, pairId: string) =>
    api.delete(`/tournaments/${id}/forbidden-pairs/${pairId}`),

  // Special rankings
  getSpecialRankings: (id: string) =>
    api.get<{ rankings: SpecialRanking[] }>(`/tournaments/${id}/special-rankings`),
  createSpecialRanking: (id: string, data: CreateSpecialRankingRequest) =>
    api.post<{ ranking: SpecialRanking }>(`/tournaments/${id}/special-rankings`, data),
  updateSpecialRanking: (id: string, rankingId: string, data: UpdateSpecialRankingRequest) =>
    api.put<{ ranking: SpecialRanking }>(`/tournaments/${id}/special-rankings/${rankingId}`, data),
  deleteSpecialRanking: (id: string, rankingId: string) =>
    api.delete(`/tournaments/${id}/special-rankings/${rankingId}`),

  // Rating override
  setRatingOverride: (id: string, participantId: string, rating: number) =>
    api.put(`/tournaments/${id}/participants/${participantId}/rating-override`, { rating }),
  clearRatingOverride: (id: string, participantId: string) =>
    api.delete(`/tournaments/${id}/participants/${participantId}/rating-override`),

  // Late entry
  addLateEntry: (id: string, data: LateEntryRequest) =>
    api.post<{ participant: TournamentParticipant }>(`/tournaments/${id}/late-entry`, data),
};

// Round Robin types
import type {
  SeatingChartData,
  RotationData,
  RoundRobinSettingsResponse,
  UpdateRoundRobinSettingsRequest,
} from '@/types/roundrobin';

// WAM (Wiener Arbeiter-Mannschaftsmeisterschaft) API
import type {
  WamParticipant,
  GroupProposal,
  ConfirmGroupsRequest,
  ConfirmGroupsResponse,
  WamTournamentStatus,
  UpdateJokerRequest,
  UpdateJokerResponse,
  JokerValue,
} from '@/types/wam';

// Federation API types
export interface FederationPlayer {
  id: string; // VKZ-Spielernummer (e.g., "56024-173")
  name: string; // Last name
  firstName?: string;
  lastName?: string;
  federation: string; // "DSB", "FIDE", etc.
  rating?: number; // DWZ for DSB
  fideId?: string;
  fideRating?: number;
  title?: string;
  club?: string;
}

export interface FederationSearchResponse {
  results: FederationPlayer[];
  source: string;
}

// Federation API
export const federationApi = {
  search: (query: string, federation: 'dsb' | 'fide' | 'oesb' | 'ssb', limit?: number) => {
    const params: Record<string, string> = { q: query, federation };
    if (limit) params.limit = String(limit);
    return api.get<FederationSearchResponse>('/federation/search', params);
  },
};

// Reports API types
export interface DWZPlayer {
  startNumber: number;
  pkz: string;
  name: string;
  firstName: string;
  lastName: string;
  clubId: string;
  clubName: string;
  dwz: number;
  dwzIndex: number;
  fideId: string;
  fideRating: number;
  birthYear: number;
  gender: string;
  score: number;
  rank: number;
}

export interface DWZTournament {
  turnierCode: string;
  name: string;
  startDate: string;
  endDate: string;
  city: string;
  organizer: string;
  chiefArbiter: string;
  timeControl: string;
  rounds: number;
  pairingSystem: string;
}

export interface DWZReportData {
  tournament: DWZTournament;
  players: DWZPlayer[];
  generatedAt: string;
}

export interface DWZReportResponse {
  report: DWZReportData;
  validationErrors: string[];
  isValid: boolean;
}

export interface TRFPlayer {
  startNo: number;
  gender: string;
  title: string;
  name: string;
  fideRating: number;
  federation: string;
  fideId: string;
  birthDate: string;
  points: number;
  rank: number;
  roundResults: {
    opponentNo: number;
    color: string;
    result: string;
  }[];
}

export interface TRFReportData {
  tournamentId: string;
  tournamentName: string;
  tournamentCode: string;
  city: string;
  federation: string;
  startDate: string;
  endDate: string;
  numPlayers: number;
  numRatedPlayers: number;
  numRounds: number;
  ratingType: string;
  chiefArbiter: string;
  deputyArbiter: string;
  timeControl: string;
  players: TRFPlayer[];
}

export interface TRFReportResponse {
  report: TRFReportData;
  validationErrors: string[];
  isValid: boolean;
}

export interface NormGameDetail {
  round: number;
  opponentName: string;
  opponentFideId: string;
  opponentRating: number;
  opponentTitle: string;
  opponentFed: string;
  color: string;
  result: number;
  ratingDiff: number;
  expectedScore: number;
}

export interface NormResult {
  playerId: string;
  playerName: string;
  playerFideId: string;
  playerRating: number;
  playerFederation: string;
  normType: string;
  achieved: boolean;
  performance: number;
  gamesPlayed: number;
  score: number;
  opponentCount: number;
  federationCount: number;
  titledOppCount: number;
  foreignOppCount: number;
  averageOppRating: number;
  details: NormGameDetail[];
  missingReasons: string[];
}

export interface NormReportData {
  tournamentId: string;
  tournamentName: string;
  generatedAt: string;
  normResults: NormResult[];
}

export interface NormReportResponse {
  report: NormReportData;
  summary: {
    achieved: number;
    nearMiss: number;
    total: number;
  };
}

// Reports API
export const reportsApi = {
  getDWZReport: (tournamentId: string) =>
    api.get<DWZReportResponse>(`/reports/dwz/${tournamentId}`),
  getTRFReport: (tournamentId: string) =>
    api.get<TRFReportResponse>(`/reports/trf/${tournamentId}`),
  getNormReport: (tournamentId: string) =>
    api.get<NormReportResponse>(`/reports/norms/${tournamentId}`),
};

export const wamApi = {
  // Get WAM tournament status and groups
  getStatus: (tournamentId: string) =>
    api.get<WamTournamentStatus>(`/tournaments/${tournamentId}/wam/status`),

  // Get WAM participants with joker and U12 information
  getParticipants: (tournamentId: string) =>
    api.get<{ participants: WamParticipant[] }>(`/tournaments/${tournamentId}/wam/participants`),

  // Set joker points for a participant
  setJokerPoints: (tournamentId: string, participantId: string, jokerPoints: JokerValue) =>
    api.put<UpdateJokerResponse>(
      `/tournaments/${tournamentId}/wam/participants/${participantId}/joker`,
      { jokerPoints } as UpdateJokerRequest
    ),

  // Get group proposal (calculates optimal groups)
  proposeGroups: (tournamentId: string) =>
    api.get<GroupProposal>(`/tournaments/${tournamentId}/wam/groups/propose`),

  // Confirm group proposal (creates groups)
  confirmGroups: (tournamentId: string, request: ConfirmGroupsRequest) =>
    api.post<ConfirmGroupsResponse>(`/tournaments/${tournamentId}/wam/groups/confirm`, request),

  // Delete all groups (reset)
  deleteGroups: (tournamentId: string) =>
    api.delete(`/tournaments/${tournamentId}/wam/groups`),
};

export default api;
