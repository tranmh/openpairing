/**
 * Mock data for chess game testing.
 */

export interface MockGameState {
  id: string;
  whiteId: string;
  blackId: string;
  fen: string;
  pgn: string;
  status: GameStatus;
  result?: GameResult;
  whiteTime: number;
  blackTime: number;
  isWhiteTurn: boolean;
  moves: string[];
}

type GameStatus = 'waiting' | 'active' | 'paused' | 'completed' | 'aborted';
type GameResult = '1-0' | '0-1' | '1/2-1/2' | '*';

/**
 * Creates a mock game state.
 */
export function createMockGame(overrides?: Partial<MockGameState>): MockGameState {
  return {
    id: 'game-1',
    whiteId: 'player-1',
    blackId: 'player-2',
    fen: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
    pgn: '',
    status: 'active',
    whiteTime: 5400000, // 90 minutes in ms
    blackTime: 5400000,
    isWhiteTurn: true,
    moves: [],
    ...overrides,
  };
}

/**
 * Creates a game in the starting position.
 */
export function createNewGame(): MockGameState {
  return createMockGame();
}

/**
 * Creates a game in progress (after some moves).
 */
export function createGameInProgress(): MockGameState {
  return createMockGame({
    fen: 'r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3',
    pgn: '1. e4 e5 2. Nf3 Nc6',
    moves: ['e4', 'e5', 'Nf3', 'Nc6'],
    isWhiteTurn: true,
    whiteTime: 5340000,
    blackTime: 5320000,
  });
}

/**
 * Creates a completed game with white winning.
 */
export function createWhiteWinGame(): MockGameState {
  return createMockGame({
    status: 'completed',
    result: '1-0',
    fen: '4k3/8/8/8/8/8/4Q3/4K3 b - - 0 1',
    pgn: '1. e4 e5 2. Qh5 Nc6 3. Bc4 Nf6 4. Qxf7# 1-0',
    moves: ['e4', 'e5', 'Qh5', 'Nc6', 'Bc4', 'Nf6', 'Qxf7#'],
  });
}

/**
 * Creates a completed game with black winning.
 */
export function createBlackWinGame(): MockGameState {
  return createMockGame({
    status: 'completed',
    result: '0-1',
    isWhiteTurn: false,
  });
}

/**
 * Creates a drawn game.
 */
export function createDrawnGame(): MockGameState {
  return createMockGame({
    status: 'completed',
    result: '1/2-1/2',
  });
}

/**
 * Creates a game with low time for white.
 */
export function createLowTimeGame(): MockGameState {
  return createMockGame({
    whiteTime: 30000, // 30 seconds
    blackTime: 300000, // 5 minutes
    isWhiteTurn: true,
  });
}

/**
 * Standard starting FEN position.
 */
export const STARTING_FEN = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1';

/**
 * Some common test positions.
 */
export const testPositions = {
  starting: STARTING_FEN,
  sicilian: 'rnbqkbnr/pp1ppppp/8/2p5/4P3/8/PPPP1PPP/RNBQKBNR w KQkq c6 0 2',
  queensGambit: 'rnbqkbnr/ppp1pppp/8/3p4/2PP4/8/PP2PPPP/RNBQKBNR b KQkq c3 0 2',
  scholarsMate: 'r1bqkb1r/pppp1Qpp/2n2n2/4p3/2B1P3/8/PPPP1PPP/RNB1K1NR b KQkq - 0 4',
  endgameKingPawn: '8/8/8/8/8/4k3/4P3/4K3 w - - 0 1',
};
