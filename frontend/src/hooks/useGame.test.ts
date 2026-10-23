import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import type { GameState } from './useGame';

// ---------------------------------------------------------------------------
// Fake socket.io client and auth store
// ---------------------------------------------------------------------------
const { sockets, FakeSocket, authState, ioMock } = vi.hoisted(() => {
  type Handler = (...args: unknown[]) => void;

  class FakeSocket {
    handlers: Record<string, Handler[]> = {};
    emit = vi.fn();
    disconnect = vi.fn();
    on = vi.fn((event: string, cb: Handler) => {
      (this.handlers[event] ??= []).push(cb);
      return this;
    });
    trigger(event: string, ...args: unknown[]) {
      (this.handlers[event] ?? []).forEach((h) => h(...args));
    }
  }

  const sockets: FakeSocket[] = [];
  const ioMock = vi.fn(() => {
    const s = new FakeSocket();
    sockets.push(s);
    return s;
  });

  const authState: {
    current: { user: { id: string } | null; accessToken: string | null };
  } = { current: { user: { id: 'u-white' }, accessToken: 'tok-123' } };

  return { sockets, FakeSocket, authState, ioMock };
});

vi.mock('socket.io-client', () => ({ io: ioMock }));

vi.mock('@/store/auth', () => ({
  useAuthStore: (selector: (s: typeof authState.current) => unknown) =>
    selector(authState.current),
}));

import { useGame } from './useGame';

type Socket = InstanceType<typeof FakeSocket>;

function makeState(overrides: Partial<GameState> = {}): GameState {
  return {
    id: 'game-1',
    whiteUserId: 'u-white',
    blackUserId: 'u-black',
    fen: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
    moves: [],
    moveCount: 0,
    whiteClockMs: 300000,
    blackClockMs: 300000,
    status: 'ongoing',
    result: '*',
    termination: '',
    turn: 'white',
    inCheck: false,
    ...overrides,
  };
}

function lastSocket(): Socket {
  return sockets[sockets.length - 1];
}

function send(socket: Socket, message: Record<string, unknown>) {
  act(() => {
    socket.trigger('message', { gameId: 'game-1', ...message });
  });
}

function connect(socket: Socket) {
  act(() => {
    socket.trigger('connect');
  });
}

function setup(extra: Record<string, unknown> = {}) {
  const callbacks = {
    onMove: vi.fn(),
    onGameOver: vi.fn(),
    onDrawOffered: vi.fn(),
    onError: vi.fn(),
  };
  const hook = renderHook((props: { gameId: string }) =>
    useGame({ ...callbacks, ...extra, gameId: props.gameId }),
  { initialProps: { gameId: 'game-1' } });
  return { ...hook, callbacks, socket: lastSocket() };
}

describe('useGame', () => {
  beforeEach(() => {
    sockets.length = 0;
    ioMock.mockClear();
    authState.current = { user: { id: 'u-white' }, accessToken: 'tok-123' };
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('connection lifecycle', () => {
    it('opens a websocket connection with the access token', () => {
      setup();
      expect(ioMock).toHaveBeenCalledTimes(1);
      const [url, opts] = ioMock.mock.calls[0] as unknown as [string, Record<string, unknown>];
      expect(typeof url).toBe('string');
      expect(url.length).toBeGreaterThan(0);
      expect(opts).toMatchObject({
        auth: { token: 'tok-123' },
        transports: ['websocket'],
        reconnection: true,
      });
    });

    it('starts disconnected with empty state', () => {
      const { result } = setup();
      expect(result.current.isConnected).toBe(false);
      expect(result.current.gameState).toBeNull();
      expect(result.current.spectatorCount).toBe(0);
      expect(result.current.opponentConnected).toBe(false);
      expect(result.current.playerColor).toBeNull();
      expect(result.current.isSpectator).toBe(true);
      expect(result.current.canAbort).toBe(false);
      expect(result.current.hasDrawOffer).toBe(false);
    });

    it('marks itself connected and joins the game room on connect', () => {
      const { result, socket } = setup();
      connect(socket);
      expect(result.current.isConnected).toBe(true);
      expect(socket.emit).toHaveBeenCalledWith('message', {
        type: 'join_game',
        gameId: 'game-1',
      });
    });

    it('marks itself disconnected on disconnect', () => {
      const { result, socket } = setup();
      connect(socket);
      act(() => socket.trigger('disconnect'));
      expect(result.current.isConnected).toBe(false);
    });

    it('reports a connection error through onError', () => {
      const { callbacks, socket } = setup();
      act(() => socket.trigger('connect_error', new Error('refused')));
      expect(callbacks.onError).toHaveBeenCalledWith('Failed to connect to game server');
    });

    it('leaves the game and disconnects on unmount', () => {
      const { unmount, socket } = setup();
      connect(socket);
      unmount();
      expect(socket.emit).toHaveBeenLastCalledWith('message', {
        type: 'leave_game',
        gameId: 'game-1',
      });
      expect(socket.disconnect).toHaveBeenCalledTimes(1);
    });

    it('reconnects to the new room when the gameId changes', () => {
      const { rerender, socket: first } = setup();
      rerender({ gameId: 'game-2' });

      expect(first.emit).toHaveBeenCalledWith('message', {
        type: 'leave_game',
        gameId: 'game-1',
      });
      expect(first.disconnect).toHaveBeenCalled();
      expect(sockets).toHaveLength(2);

      const second = lastSocket();
      connect(second);
      expect(second.emit).toHaveBeenCalledWith('message', {
        type: 'join_game',
        gameId: 'game-2',
      });
    });
  });

  describe('incoming messages', () => {
    it('stores the game state from game_state', () => {
      const { result, socket } = setup();
      const state = makeState();
      send(socket, { type: 'game_state', state });
      expect(result.current.gameState).toEqual(state);
    });

    it('ignores game_state without a state payload', () => {
      const { result, socket } = setup();
      send(socket, { type: 'game_state' });
      expect(result.current.gameState).toBeNull();
    });

    it('updates the state and calls onMove for move_made', () => {
      const { result, socket, callbacks } = setup();
      const state = makeState({ moves: ['e4'], moveCount: 1, turn: 'black' });
      send(socket, { type: 'move_made', move: 'e4', state });
      expect(result.current.gameState).toEqual(state);
      expect(callbacks.onMove).toHaveBeenCalledWith('e4', state);
    });

    it('updates the state but does not call onMove when move_made has no move', () => {
      const { result, socket, callbacks } = setup();
      const state = makeState({ moveCount: 1 });
      send(socket, { type: 'move_made', state });
      expect(result.current.gameState).toEqual(state);
      expect(callbacks.onMove).not.toHaveBeenCalled();
    });

    it('calls onGameOver with result and termination', () => {
      const { result, socket, callbacks } = setup();
      const state = makeState({ status: 'completed', result: '1-0', termination: 'checkmate' });
      send(socket, { type: 'game_over', state });
      expect(result.current.gameState?.status).toBe('completed');
      expect(callbacks.onGameOver).toHaveBeenCalledWith('1-0', 'checkmate');
    });

    it('does not call onGameOver when game_over has no state', () => {
      const { socket, callbacks } = setup();
      send(socket, { type: 'game_over' });
      expect(callbacks.onGameOver).not.toHaveBeenCalled();
    });

    it('calls onDrawOffered with the offering user', () => {
      const { socket, callbacks } = setup();
      send(socket, { type: 'draw_offered', userId: 'u-black' });
      expect(callbacks.onDrawOffered).toHaveBeenCalledWith('u-black');
    });

    it('ignores draw_offered without a userId', () => {
      const { socket, callbacks } = setup();
      send(socket, { type: 'draw_offered' });
      expect(callbacks.onDrawOffered).not.toHaveBeenCalled();
    });

    it('clears a pending draw offer on draw_declined', () => {
      const { result, socket } = setup();
      send(socket, { type: 'game_state', state: makeState({ drawOfferedBy: 'u-black' }) });
      expect(result.current.hasDrawOffer).toBe(true);
      expect(result.current.isDrawOfferMine).toBe(false);

      send(socket, { type: 'draw_declined' });
      expect(result.current.hasDrawOffer).toBe(false);
      expect(result.current.gameState?.drawOfferedBy).toBeUndefined();
    });

    it('keeps the state null on draw_declined before any state', () => {
      const { result, socket } = setup();
      send(socket, { type: 'draw_declined' });
      expect(result.current.gameState).toBeNull();
    });

    it('updates only the clocks on clock_update', () => {
      const { result, socket } = setup();
      const state = makeState({ moves: ['e4', 'e5'], moveCount: 2 });
      send(socket, { type: 'game_state', state });
      send(socket, {
        type: 'clock_update',
        state: makeState({ whiteClockMs: 1000, blackClockMs: 2000, moves: [], moveCount: 0 }),
      });
      expect(result.current.gameState).toEqual({
        ...state,
        whiteClockMs: 1000,
        blackClockMs: 2000,
      });
    });

    it('ignores clock_update before any game state arrived', () => {
      const { result, socket } = setup();
      send(socket, { type: 'clock_update', state: makeState() });
      expect(result.current.gameState).toBeNull();
    });

    it('updates the spectator count, including zero', () => {
      const { result, socket } = setup();
      send(socket, { type: 'spectator_count', count: 7 });
      expect(result.current.spectatorCount).toBe(7);
      send(socket, { type: 'spectator_count', count: 0 });
      expect(result.current.spectatorCount).toBe(0);
    });

    it('ignores spectator_count without a count', () => {
      const { result, socket } = setup();
      send(socket, { type: 'spectator_count', count: 3 });
      send(socket, { type: 'spectator_count' });
      expect(result.current.spectatorCount).toBe(3);
    });

    it('tracks opponent presence', () => {
      const { result, socket } = setup();
      send(socket, { type: 'opponent_connected' });
      expect(result.current.opponentConnected).toBe(true);
      send(socket, { type: 'opponent_disconnected' });
      expect(result.current.opponentConnected).toBe(false);
    });

    it('forwards server errors to onError', () => {
      const { socket, callbacks } = setup();
      send(socket, { type: 'error', error: 'Illegal move' });
      expect(callbacks.onError).toHaveBeenCalledWith('Illegal move');
    });

    it('uses a fallback text for errors without a message', () => {
      const { socket, callbacks } = setup();
      send(socket, { type: 'error' });
      expect(callbacks.onError).toHaveBeenCalledWith('Unknown error');
    });

    it('ignores unknown message types', () => {
      const { result, socket, callbacks } = setup();
      send(socket, { type: 'something_else', state: makeState() });
      expect(result.current.gameState).toBeNull();
      expect(callbacks.onError).not.toHaveBeenCalled();
    });
  });

  describe('derived state', () => {
    it('identifies the white player and their turn', () => {
      const { result, socket } = setup();
      send(socket, { type: 'game_state', state: makeState({ turn: 'white' }) });
      expect(result.current.playerColor).toBe('white');
      expect(result.current.isSpectator).toBe(false);
      expect(result.current.isMyTurn).toBe(true);
    });

    it('identifies the black player who is not on move', () => {
      authState.current = { user: { id: 'u-black' }, accessToken: 'tok' };
      const { result, socket } = setup();
      send(socket, { type: 'game_state', state: makeState({ turn: 'white' }) });
      expect(result.current.playerColor).toBe('black');
      expect(result.current.isMyTurn).toBe(false);
    });

    it('treats other users as spectators', () => {
      authState.current = { user: { id: 'someone-else' }, accessToken: 'tok' };
      const { result, socket } = setup();
      send(socket, { type: 'game_state', state: makeState() });
      expect(result.current.playerColor).toBeNull();
      expect(result.current.isSpectator).toBe(true);
      expect(result.current.isMyTurn).toBe(false);
    });

    it('treats an anonymous user as a spectator', () => {
      authState.current = { user: null, accessToken: null };
      const { result, socket } = setup();
      send(socket, { type: 'game_state', state: makeState() });
      expect(result.current.isSpectator).toBe(true);
    });

    it.each([
      [0, 'ongoing', true],
      [1, 'ongoing', true],
      [2, 'ongoing', false],
      [0, 'completed', false],
      [0, 'waiting', false],
    ] as const)('canAbort with moveCount=%s status=%s is %s', (moveCount, status, expected) => {
      const { result, socket } = setup();
      send(socket, { type: 'game_state', state: makeState({ moveCount, status }) });
      expect(result.current.canAbort).toBe(expected);
    });

    it('recognises the own draw offer', () => {
      const { result, socket } = setup();
      send(socket, { type: 'game_state', state: makeState({ drawOfferedBy: 'u-white' }) });
      expect(result.current.hasDrawOffer).toBe(true);
      expect(result.current.isDrawOfferMine).toBe(true);
    });
  });

  describe('actions', () => {
    const actions = [
      ['makeMove', 'make_move'],
      ['resign', 'resign'],
      ['offerDraw', 'offer_draw'],
      ['acceptDraw', 'accept_draw'],
      ['declineDraw', 'decline_draw'],
      ['abort', 'abort'],
    ] as const;

    it.each(actions)('%s reports an error and sends nothing while disconnected', (name) => {
      const { result, socket, callbacks } = setup();
      act(() => {
        if (name === 'makeMove') result.current.makeMove('e4');
        else result.current[name]();
      });
      expect(callbacks.onError).toHaveBeenCalledWith('Not connected to game server');
      expect(socket.emit).not.toHaveBeenCalled();
    });

    it.each(actions)('%s sends a %s message when connected', (name, type) => {
      const { result, socket, callbacks } = setup();
      connect(socket);
      socket.emit.mockClear();
      act(() => {
        if (name === 'makeMove') result.current.makeMove('e2e4');
        else result.current[name]();
      });
      const expected: Record<string, unknown> = { type, gameId: 'game-1' };
      if (name === 'makeMove') expected.move = 'e2e4';
      expect(socket.emit).toHaveBeenCalledWith('message', expected);
      expect(callbacks.onError).not.toHaveBeenCalled();
    });

    it('refuses moves again after the connection dropped', () => {
      const { result, socket, callbacks } = setup();
      connect(socket);
      act(() => socket.trigger('disconnect'));
      socket.emit.mockClear();
      act(() => result.current.makeMove('e4'));
      expect(socket.emit).not.toHaveBeenCalled();
      expect(callbacks.onError).toHaveBeenCalledWith('Not connected to game server');
    });

    it('does not throw when no onError callback is given', () => {
      const { result } = renderHook(() => useGame({ gameId: 'game-1' }));
      expect(() => act(() => result.current.resign())).not.toThrow();
    });
  });
});
