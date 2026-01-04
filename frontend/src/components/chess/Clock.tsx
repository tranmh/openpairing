'use client';

import { useEffect, useState } from 'react';
import { cn } from '@/lib/utils';

interface ClockProps {
  timeMs: number;
  isRunning: boolean;
  isPlayerTurn: boolean;
  className?: string;
}

export function Clock({ timeMs, isRunning, isPlayerTurn, className }: ClockProps) {
  const [displayTime, setDisplayTime] = useState(timeMs);

  // Update display time when prop changes (server sync)
  useEffect(() => {
    setDisplayTime(timeMs);
  }, [timeMs]);

  // Local countdown when running
  useEffect(() => {
    if (!isRunning || !isPlayerTurn) return;

    const interval = setInterval(() => {
      setDisplayTime((prev) => Math.max(0, prev - 100));
    }, 100);

    return () => clearInterval(interval);
  }, [isRunning, isPlayerTurn]);

  const formatTime = (ms: number) => {
    const totalSeconds = Math.floor(ms / 1000);
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;
    const tenths = Math.floor((ms % 1000) / 100);

    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }

    if (minutes >= 1) {
      return `${minutes}:${seconds.toString().padStart(2, '0')}`;
    }

    // Show tenths of second when under 1 minute
    return `${seconds}.${tenths}`;
  };

  const isLowTime = displayTime < 30000; // Less than 30 seconds
  const isCriticalTime = displayTime < 10000; // Less than 10 seconds
  const isExpired = displayTime <= 0;

  return (
    <div
      className={cn(
        'flex items-center justify-center rounded-lg px-4 py-2 font-mono text-2xl font-bold transition-colors',
        isPlayerTurn
          ? 'bg-green-600 text-white'
          : 'bg-gray-700 text-gray-300',
        isLowTime && isPlayerTurn && 'bg-yellow-600',
        isCriticalTime && isPlayerTurn && 'animate-pulse bg-red-600',
        isExpired && 'bg-red-900 text-red-300',
        className
      )}
    >
      {formatTime(displayTime)}
    </div>
  );
}

interface GameClocksProps {
  whiteClockMs: number;
  blackClockMs: number;
  isWhiteTurn: boolean;
  isRunning: boolean;
  orientation: 'white' | 'black';
  whitePlayerName?: string;
  blackPlayerName?: string;
  className?: string;
}

export function GameClocks({
  whiteClockMs,
  blackClockMs,
  isWhiteTurn,
  isRunning,
  orientation,
  whitePlayerName = 'White',
  blackPlayerName = 'Black',
  className,
}: GameClocksProps) {
  // Determine which clock is on top based on board orientation
  const topClock = orientation === 'white' ? 'black' : 'white';
  const bottomClock = orientation === 'white' ? 'white' : 'black';

  const renderClock = (color: 'white' | 'black') => {
    const timeMs = color === 'white' ? whiteClockMs : blackClockMs;
    const isPlayerTurn = color === 'white' ? isWhiteTurn : !isWhiteTurn;
    const playerName = color === 'white' ? whitePlayerName : blackPlayerName;

    return (
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-2">
          <div
            className={cn(
              'h-4 w-4 rounded-full',
              color === 'white' ? 'bg-white border border-gray-300' : 'bg-gray-900'
            )}
          />
          <span className="font-medium text-gray-200">{playerName}</span>
        </div>
        <Clock
          timeMs={timeMs}
          isRunning={isRunning}
          isPlayerTurn={isPlayerTurn}
        />
      </div>
    );
  };

  return (
    <div className={cn('flex flex-col gap-4', className)}>
      {renderClock(topClock)}
      {renderClock(bottomClock)}
    </div>
  );
}
