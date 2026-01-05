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

    if (this.accessToken) {
      headers['Authorization'] = `Bearer ${this.accessToken}`;
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
        return {
          status,
          error: {
            message: (errorData.error || errorData.message || 'An error occurred') as string,
            code: errorData.code as string | undefined,
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
  pairingEngine: 'javafo' | 'bbp';
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
  points: number;
  buchholz: number;
  buchholzCut1: number;
  sonnebornBerger: number;
  wins: number;
  draws: number;
  losses: number;
  isActive: boolean;
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

  enterResult: (tournamentId: string, gameId: string, result: string) =>
    api.put<Game>(`/tournaments/${tournamentId}/games/${gameId}/result`, { result }),
};

export default api;
