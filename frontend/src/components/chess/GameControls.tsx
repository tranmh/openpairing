'use client';

import { useState } from 'react';
import {
  Flag,
  Handshake,
  XCircle,
  Eye,
  Wifi,
  WifiOff,
  Check,
  X,
} from 'lucide-react';
import { cn } from '@/lib/utils';

interface GameControlsProps {
  isSpectator: boolean;
  isMyTurn: boolean;
  canAbort: boolean;
  canResign: boolean;
  canOfferDraw: boolean;
  hasDrawOffer: boolean;
  isDrawOfferMine: boolean;
  isConnected: boolean;
  opponentConnected: boolean;
  spectatorCount: number;
  onResign: () => void;
  onOfferDraw: () => void;
  onAcceptDraw: () => void;
  onDeclineDraw: () => void;
  onAbort: () => void;
  className?: string;
}

export function GameControls({
  isSpectator,
  isMyTurn,
  canAbort,
  canResign,
  canOfferDraw,
  hasDrawOffer,
  isDrawOfferMine,
  isConnected,
  opponentConnected,
  spectatorCount,
  onResign,
  onOfferDraw,
  onAcceptDraw,
  onDeclineDraw,
  onAbort,
  className,
}: GameControlsProps) {
  const [showResignConfirm, setShowResignConfirm] = useState(false);
  const [showAbortConfirm, setShowAbortConfirm] = useState(false);

  const handleResignClick = () => {
    if (showResignConfirm) {
      onResign();
      setShowResignConfirm(false);
    } else {
      setShowResignConfirm(true);
      setShowAbortConfirm(false);
      // Auto-hide after 5 seconds
      setTimeout(() => setShowResignConfirm(false), 5000);
    }
  };

  const handleAbortClick = () => {
    if (showAbortConfirm) {
      onAbort();
      setShowAbortConfirm(false);
    } else {
      setShowAbortConfirm(true);
      setShowResignConfirm(false);
      // Auto-hide after 5 seconds
      setTimeout(() => setShowAbortConfirm(false), 5000);
    }
  };

  return (
    <div className={cn('space-y-4', className)}>
      {/* Connection status */}
      <div className="flex items-center justify-between text-sm">
        <div className="flex items-center gap-2">
          {isConnected ? (
            <Wifi className="h-4 w-4 text-green-500" />
          ) : (
            <WifiOff className="h-4 w-4 text-red-500" />
          )}
          <span className={isConnected ? 'text-green-500' : 'text-red-500'}>
            {isConnected ? 'Connected' : 'Disconnected'}
          </span>
        </div>

        <div className="flex items-center gap-4">
          {!isSpectator && (
            <span
              className={cn(
                'text-xs',
                opponentConnected ? 'text-green-500' : 'text-yellow-500'
              )}
            >
              Opponent: {opponentConnected ? 'online' : 'offline'}
            </span>
          )}

          <div className="flex items-center gap-1 text-gray-400">
            <Eye className="h-4 w-4" />
            <span>{spectatorCount}</span>
          </div>
        </div>
      </div>

      {/* Turn indicator */}
      {!isSpectator && (
        <div
          className={cn(
            'rounded-lg px-4 py-2 text-center font-medium',
            isMyTurn
              ? 'bg-green-600 text-white'
              : 'bg-gray-700 text-gray-300'
          )}
        >
          {isMyTurn ? 'Your turn' : "Opponent's turn"}
        </div>
      )}

      {/* Draw offer banner */}
      {hasDrawOffer && !isDrawOfferMine && (
        <div className="rounded-lg bg-yellow-900 p-4">
          <p className="mb-3 text-center text-yellow-200">
            Your opponent offers a draw
          </p>
          <div className="flex gap-2">
            <button
              onClick={onAcceptDraw}
              className="flex flex-1 items-center justify-center gap-2 rounded-lg bg-green-600 px-4 py-2 font-medium text-white hover:bg-green-700"
            >
              <Check className="h-4 w-4" />
              Accept
            </button>
            <button
              onClick={onDeclineDraw}
              className="flex flex-1 items-center justify-center gap-2 rounded-lg bg-red-600 px-4 py-2 font-medium text-white hover:bg-red-700"
            >
              <X className="h-4 w-4" />
              Decline
            </button>
          </div>
        </div>
      )}

      {isDrawOfferMine && (
        <div className="rounded-lg bg-yellow-900/50 px-4 py-2 text-center text-yellow-200">
          Draw offer sent...
        </div>
      )}

      {/* Game actions */}
      {!isSpectator && (
        <div className="flex flex-wrap gap-2">
          {/* Abort button (only before move 2) */}
          {canAbort && (
            <button
              onClick={handleAbortClick}
              className={cn(
                'flex flex-1 items-center justify-center gap-2 rounded-lg px-4 py-2 font-medium transition-colors',
                showAbortConfirm
                  ? 'bg-red-600 text-white hover:bg-red-700'
                  : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
              )}
            >
              <XCircle className="h-4 w-4" />
              {showAbortConfirm ? 'Click to confirm abort' : 'Abort'}
            </button>
          )}

          {/* Draw button */}
          {canOfferDraw && !hasDrawOffer && (
            <button
              onClick={onOfferDraw}
              className="flex flex-1 items-center justify-center gap-2 rounded-lg bg-gray-700 px-4 py-2 font-medium text-gray-300 hover:bg-gray-600"
            >
              <Handshake className="h-4 w-4" />
              Offer Draw
            </button>
          )}

          {/* Resign button */}
          {canResign && (
            <button
              onClick={handleResignClick}
              className={cn(
                'flex flex-1 items-center justify-center gap-2 rounded-lg px-4 py-2 font-medium transition-colors',
                showResignConfirm
                  ? 'bg-red-600 text-white hover:bg-red-700'
                  : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
              )}
            >
              <Flag className="h-4 w-4" />
              {showResignConfirm ? 'Click to confirm resign' : 'Resign'}
            </button>
          )}
        </div>
      )}

      {/* Spectator mode indicator */}
      {isSpectator && (
        <div className="flex items-center justify-center gap-2 rounded-lg bg-blue-900/50 px-4 py-2 text-blue-200">
          <Eye className="h-4 w-4" />
          Watching as spectator
        </div>
      )}
    </div>
  );
}
