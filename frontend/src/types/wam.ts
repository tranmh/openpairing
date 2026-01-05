/**
 * WAM (Wiener Arbeiter-Mannschaftsmeisterschaft) specific types
 */

// Joker values: 0, 50, 100, or 150 DWZ points
export type JokerValue = 0 | 50 | 100 | 150;

// WAM participant with joker and U12 information
export interface WamParticipant {
  participantId: string;
  playerId: string;
  name: string;
  rating: number; // Original DWZ
  jokerPoints: JokerValue;
  effectiveRating: number; // rating + jokerPoints
  isU12: boolean;
  age?: number;
  clubId?: string;
  clubName?: string;
  birthDate?: string;
}

// Player within a WAM group
export interface WamGroupPlayer {
  participantId: string;
  playerId: string;
  name: string;
  rating: number;
  jokerPoints: JokerValue;
  effectiveRating: number;
  isU12: boolean;
  rank: number;
  points: number;
  gamesPlayed: number;
}

// WAM group (Gruppe)
export interface WamGroup {
  id: string;
  name: string;
  targetSize: number;
  players: WamGroupPlayer[];
  roundsTotal: number;
  roundsPlayed: number;
  isComplete: boolean;
}

// Proposed group during group formation
export interface ProposedGroup {
  name: string;
  players: WamParticipant[];
  targetSize: number;
  warnings?: string[];
}

// U12 group proposal with target group suggestion
export interface U12GroupProposal {
  player: WamParticipant;
  originalGroupIndex: number;
  proposedGroupIndex: number;
  reason: string;
}

// Complete group proposal response from API
export interface GroupProposal {
  groups: ProposedGroup[];
  totalPlayers: number;
  warnings?: string[];
  u12Proposals?: U12GroupProposal[];
}

// Request to confirm groups with optional U12 overrides
export interface ConfirmGroupsRequest {
  acceptProposal: boolean;
  u12Overrides?: U12Override[];
}

// Override for U12 player placement
export interface U12Override {
  participantId: string;
  targetGroupIndex: number;
}

// WAM tournament status response
export interface WamTournamentStatus {
  tournamentId: string;
  hasGroups: boolean;
  groupCount: number;
  groups?: WamGroup[];
  totalPlayers: number;
  u12Count: number;
  jokerStats: {
    joker0: number;
    joker50: number;
    joker100: number;
    joker150: number;
  };
}

// Update joker request
export interface UpdateJokerRequest {
  jokerPoints: JokerValue;
}

// Update joker response
export interface UpdateJokerResponse {
  participant: WamParticipant;
  message: string;
}

// Confirm groups response
export interface ConfirmGroupsResponse {
  groups: WamGroup[];
  warnings?: string[];
}
