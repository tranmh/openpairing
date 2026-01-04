import { useCallback, useEffect, useRef, useState } from 'react';
import { io, Socket } from 'socket.io-client';
import { useAuthStore } from '@/store/auth';

export interface GameState {
  id: string;
  whiteUserId: string;
  blackUserId: string;
  fen: string;
  moves: string[];
  moveCount: number;
  whiteClockMs: number;
  blackClockMs: number;
  status: 'waiting' | 'ongoing' | 'completed' | 'aborted';
  result: string;
  termination: string;
  turn: 'white' | 'black';
  inCheck: boolean;
  drawOfferedBy?: string;
}

export interface GameMessage {
  type: string;
  gameId: string;
  userId?: string;
  move?: string;
  state?: GameState;
  error?: string;
  count?: number;
}

interface UseGameOptions {
  gameId: string;
  onMove?: (move: string, state: GameState) => void;
  onGameOver?: (result: string, termination: string) => void;
  onDrawOffered?: (byUserId: string) => void;
  onError?: (error: string) => void;
}

export function useGame(options: UseGameOptions) {
  const { gameId, onMove, onGameOver, onDrawOffered, onError } = options;
  const user = useAuthStore((state) => state.user);
  const token = useAuthStore((state) => state.accessToken);

  const socketRef = useRef<Socket | null>(null);
  const gameIdRef = useRef(gameId);
  const [gameState, setGameState] = useState<GameState | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [spectatorCount, setSpectatorCount] = useState(0);
  const [opponentConnected, setOpponentConnected] = useState(false);

  // Keep gameIdRef in sync
  useEffect(() => {
    gameIdRef.current = gameId;
  }, [gameId]);

  // Initialize WebSocket connection
  useEffect(() => {
    const wsUrl = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8081';
    const currentGameId = gameId;

    const socket = io(wsUrl, {
      auth: {
        token,
      },
      transports: ['websocket'],
      reconnection: true,
      reconnectionAttempts: 5,
      reconnectionDelay: 1000,
    });

    socketRef.current = socket;

    socket.on('connect', () => {
      setIsConnected(true);
      // Join the game room
      socket.emit('message', {
        type: 'join_game',
        gameId: currentGameId,
      });
    });

    socket.on('disconnect', () => {
      setIsConnected(false);
    });

    socket.on('message', (message: GameMessage) => {
      handleMessage(message);
    });

    socket.on('connect_error', (error) => {
      console.error('Connection error:', error);
      onError?.('Failed to connect to game server');
    });

    return () => {
      socket.emit('message', {
        type: 'leave_game',
        gameId: currentGameId,
      });
      socket.disconnect();
    };
  }, [gameId, token]);

  const handleMessage = useCallback((message: GameMessage) => {
    switch (message.type) {
      case 'game_state':
        if (message.state) {
          setGameState(message.state);
        }
        break;

      case 'move_made':
        if (message.state) {
          setGameState(message.state);
          if (message.move) {
            onMove?.(message.move, message.state);
          }
        }
        break;

      case 'game_over':
        if (message.state) {
          setGameState(message.state);
          onGameOver?.(message.state.result, message.state.termination);
        }
        break;

      case 'draw_offered':
        if (message.userId) {
          onDrawOffered?.(message.userId);
        }
        break;

      case 'draw_declined':
        // Clear draw offer in state
        setGameState(prev => prev ? { ...prev, drawOfferedBy: undefined } : null);
        break;

      case 'clock_update':
        if (message.state) {
          setGameState(prev => prev ? {
            ...prev,
            whiteClockMs: message.state!.whiteClockMs,
            blackClockMs: message.state!.blackClockMs,
          } : null);
        }
        break;

      case 'spectator_count':
        if (message.count !== undefined) {
          setSpectatorCount(message.count);
        }
        break;

      case 'opponent_connected':
        setOpponentConnected(true);
        break;

      case 'opponent_disconnected':
        setOpponentConnected(false);
        break;

      case 'error':
        onError?.(message.error || 'Unknown error');
        break;
    }
  }, [onMove, onGameOver, onDrawOffered, onError]);

  const makeMove = useCallback((move: string) => {
    if (!socketRef.current || !isConnected) {
      onError?.('Not connected to game server');
      return;
    }

    socketRef.current.emit('message', {
      type: 'make_move',
      gameId,
      move,
    });
  }, [gameId, isConnected, onError]);

  const resign = useCallback(() => {
    if (!socketRef.current || !isConnected) {
      onError?.('Not connected to game server');
      return;
    }

    socketRef.current.emit('message', {
      type: 'resign',
      gameId,
    });
  }, [gameId, isConnected, onError]);

  const offerDraw = useCallback(() => {
    if (!socketRef.current || !isConnected) {
      onError?.('Not connected to game server');
      return;
    }

    socketRef.current.emit('message', {
      type: 'offer_draw',
      gameId,
    });
  }, [gameId, isConnected, onError]);

  const acceptDraw = useCallback(() => {
    if (!socketRef.current || !isConnected) {
      onError?.('Not connected to game server');
      return;
    }

    socketRef.current.emit('message', {
      type: 'accept_draw',
      gameId,
    });
  }, [gameId, isConnected, onError]);

  const declineDraw = useCallback(() => {
    if (!socketRef.current || !isConnected) {
      onError?.('Not connected to game server');
      return;
    }

    socketRef.current.emit('message', {
      type: 'decline_draw',
      gameId,
    });
  }, [gameId, isConnected, onError]);

  const abort = useCallback(() => {
    if (!socketRef.current || !isConnected) {
      onError?.('Not connected to game server');
      return;
    }

    socketRef.current.emit('message', {
      type: 'abort',
      gameId,
    });
  }, [gameId, isConnected, onError]);

  // Determine player's color
  const playerColor = user?.id === gameState?.whiteUserId
    ? 'white'
    : user?.id === gameState?.blackUserId
      ? 'black'
      : null;

  const isMyTurn = playerColor === gameState?.turn;
  const isSpectator = playerColor === null;
  const canAbort = gameState?.status === 'ongoing' && gameState?.moveCount < 2;
  const hasDrawOffer = !!gameState?.drawOfferedBy;
  const isDrawOfferMine = gameState?.drawOfferedBy === user?.id;

  return {
    gameState,
    isConnected,
    spectatorCount,
    opponentConnected,
    playerColor,
    isMyTurn,
    isSpectator,
    canAbort,
    hasDrawOffer,
    isDrawOfferMine,
    makeMove,
    resign,
    offerDraw,
    acceptDraw,
    declineDraw,
    abort,
  };
}
