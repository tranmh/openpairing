import { bench, describe, vi } from 'vitest';
import { act, renderHook } from '@testing-library/react';

type Handler = (...args: unknown[]) => void;

// Fake socket: records the handlers useGame registers so the bench can
// dispatch server messages synchronously without any network.
const sockets: { handlers: Record<string, Handler> }[] = [];
vi.mock('socket.io-client', () => ({
  io: () => {
    const handlers: Record<string, Handler> = {};
    const socket = {
      on: (event: string, fn: Handler) => {
        handlers[event] = fn;
      },
      emit: () => {},
      disconnect: () => {},
    };
    sockets.push({ handlers });
    return socket;
  },
}));

import { useGame, GameState, GameMessage } from './useGame';

function makeState(moveCount: number): GameState {
  const moves = Array.from({ length: moveCount }, (_, i) => (i % 2 ? 'e7e5' : 'e2e4'));
  return {
    id: 'game-1',
    whiteUserId: 'w',
    blackUserId: 'b',
    fen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1',
    moves,
    moveCount,
    whiteClockMs: 300000,
    blackClockMs: 300000,
    status: 'ongoing',
    result: '',
    termination: '',
    turn: moveCount % 2 ? 'black' : 'white',
    inCheck: false,
  };
}

// A mixed stream as a live game produces it: moves, clock ticks, spectators.
function makeStream(count: number): GameMessage[] {
  const out: GameMessage[] = [];
  for (let i = 0; i < count; i++) {
    const kind = i % 4;
    const state = makeState(i + 1);
    if (kind === 0) out.push({ type: 'move_made', gameId: 'game-1', move: 'e2e4', state });
    else if (kind === 1) out.push({ type: 'clock_update', gameId: 'game-1', state });
    else if (kind === 2) out.push({ type: 'spectator_count', gameId: 'game-1', count: i });
    else out.push({ type: 'game_state', gameId: 'game-1', state });
  }
  return out;
}

function mount() {
  const onMove = vi.fn();
  const hook = renderHook(() => useGame({ gameId: 'game-1', onMove }));
  const socket = sockets[sockets.length - 1];
  act(() => socket.handlers.connect());
  return { hook, socket };
}

let sink: unknown;

describe('useGame message handling', () => {
  for (const n of [10, 100, 1000]) {
    const stream = makeStream(n);
    const { hook, socket } = mount();
    bench(`${n} messages, one act each`, () => {
      for (const msg of stream) {
        act(() => socket.handlers.message(msg));
      }
      sink = hook.result.current.gameState;
    });
  }
});

describe('useGame mount + connect + unmount', () => {
  for (const n of [1, 10, 50]) {
    bench(`${n} hook instances`, () => {
      for (let i = 0; i < n; i++) {
        const { hook } = mount();
        sink = hook.result.current.isConnected;
        hook.unmount();
      }
      sockets.length = 0;
    });
  }
});

export { sink };
