'use client';

import { useState, useEffect, useMemo, useCallback } from 'react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Slider } from '@/components/ui/slider';
import { Play, Pause, SkipBack, SkipForward } from 'lucide-react';
import type { RotationData, SeatingPosition } from '@/types/roundrobin';

interface RotationAnimationProps {
  data: RotationData;
  size?: number;
  className?: string;
  onComplete?: () => void;
}

type AnimationSpeed = 'slow' | 'normal' | 'fast';

const SPEED_MS: Record<AnimationSpeed, number> = {
  slow: 2000,
  normal: 1000,
  fast: 500,
};

export function RotationAnimation({
  data,
  size = 400,
  className,
  onComplete,
}: RotationAnimationProps) {
  const [isPlaying, setIsPlaying] = useState(false);
  const [progress, setProgress] = useState(0); // 0 = before, 100 = after
  const [speed, setSpeed] = useState<AnimationSpeed>('normal');

  const numSeats = data.beforeSeats.length;
  const centerX = size / 2;
  const centerY = size / 2;
  const radius = (size - 80) / 2;
  const seatRadius = Math.max(16, Math.min(24, 300 / numSeats));

  // Check for reduced motion preference
  const prefersReducedMotion = useMemo(() => {
    if (typeof window === 'undefined') return false;
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  }, []);

  // Calculate position for a seat number
  const getSeatPosition = useCallback(
    (seatNumber: number) => {
      const index = seatNumber - 1;
      const angle = (index / numSeats) * 2 * Math.PI - Math.PI / 2;
      return {
        x: centerX + radius * Math.cos(angle),
        y: centerY + radius * Math.sin(angle),
      };
    },
    [numSeats, centerX, centerY, radius]
  );

  // Interpolate position based on progress
  const getInterpolatedPosition = useCallback(
    (fromSeat: number, toSeat: number, progress: number) => {
      const from = getSeatPosition(fromSeat);
      const to = getSeatPosition(toSeat);
      const t = progress / 100;

      // Use easing for smoother animation
      const eased = t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;

      return {
        x: from.x + (to.x - from.x) * eased,
        y: from.y + (to.y - from.y) * eased,
      };
    },
    [getSeatPosition]
  );

  // Animation effect
  useEffect(() => {
    if (!isPlaying || prefersReducedMotion) return;

    const interval = setInterval(() => {
      setProgress((prev) => {
        if (prev >= 100) {
          setIsPlaying(false);
          onComplete?.();
          return 100;
        }
        return Math.min(100, prev + 2);
      });
    }, SPEED_MS[speed] / 50);

    return () => clearInterval(interval);
  }, [isPlaying, speed, prefersReducedMotion, onComplete]);

  const handlePlay = () => {
    if (progress >= 100) {
      setProgress(0);
    }
    setIsPlaying(true);
  };

  const handlePause = () => {
    setIsPlaying(false);
  };

  const handleReset = () => {
    setIsPlaying(false);
    setProgress(0);
  };

  const handleEnd = () => {
    setIsPlaying(false);
    setProgress(100);
  };

  // Build player data with interpolated positions
  const playerPositions = useMemo(() => {
    return data.movements.map((movement) => {
      const pos = getInterpolatedPosition(
        movement.fromSeat,
        movement.toSeat,
        progress
      );

      // Find player info from beforeSeats
      const beforeSeat = data.beforeSeats.find(
        (s) => s.startNumber === movement.startNumber
      );

      return {
        ...movement,
        ...pos,
        playerName: beforeSeat?.playerName || movement.playerName,
        isBye: beforeSeat?.isBye || false,
      };
    });
  }, [data.movements, data.beforeSeats, progress, getInterpolatedPosition]);

  return (
    <div className={cn('flex flex-col items-center gap-4', className)}>
      {/* Animation canvas */}
      <div className="relative">
        <svg
          width={size}
          height={size}
          viewBox={`0 0 ${size} ${size}`}
          className="overflow-visible"
        >
          {/* Static seat positions (background) */}
          {Array.from({ length: numSeats }, (_, i) => {
            const pos = getSeatPosition(i + 1);
            return (
              <g key={`seat-${i}`}>
                <circle
                  cx={pos.x}
                  cy={pos.y}
                  r={seatRadius + 4}
                  className="fill-muted/30 stroke-none"
                />
                <text
                  x={pos.x}
                  y={pos.y - seatRadius - 8}
                  textAnchor="middle"
                  className="fill-muted-foreground text-[10px]"
                >
                  {i + 1}
                </text>
              </g>
            );
          })}

          {/* Center label */}
          <text
            x={centerX}
            y={centerY}
            textAnchor="middle"
            dominantBaseline="middle"
            className="fill-muted-foreground text-xs"
          >
            {progress < 50
              ? `Runde ${data.fromRound}`
              : `Runde ${data.toRound}`}
          </text>

          {/* Moving players */}
          {playerPositions.map((player) => (
            <g
              key={player.startNumber}
              style={{
                transition: prefersReducedMotion ? 'none' : undefined,
              }}
            >
              <circle
                cx={player.x}
                cy={player.y}
                r={seatRadius}
                className={cn(
                  'stroke-2',
                  player.isBye
                    ? 'fill-muted stroke-muted-foreground stroke-dashed'
                    : player.isFixed
                      ? 'fill-yellow-100 stroke-yellow-500 dark:fill-yellow-900/30'
                      : 'fill-background stroke-primary'
                )}
                strokeDasharray={player.isBye ? '3 3' : undefined}
              />
              <text
                x={player.x}
                y={player.y}
                textAnchor="middle"
                dominantBaseline="middle"
                className={cn(
                  'text-xs font-bold',
                  player.isBye ? 'fill-muted-foreground' : 'fill-foreground'
                )}
              >
                {player.isBye ? '—' : player.startNumber}
              </text>
              {player.isFixed && (
                <text
                  x={player.x + seatRadius - 4}
                  y={player.y - seatRadius + 4}
                  className="fill-yellow-500 text-[10px]"
                >
                  ★
                </text>
              )}
            </g>
          ))}
        </svg>
      </div>

      {/* Controls */}
      <div className="flex flex-col items-center gap-3 w-full max-w-xs">
        {/* Progress slider */}
        <div className="w-full">
          <Slider
            value={[progress]}
            onValueChange={([v]) => {
              setIsPlaying(false);
              setProgress(v);
            }}
            max={100}
            step={1}
            className="w-full"
          />
        </div>

        {/* Playback controls */}
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="icon"
            onClick={handleReset}
            title="Zurück zum Start"
          >
            <SkipBack className="h-4 w-4" />
          </Button>

          {isPlaying ? (
            <Button variant="default" size="icon" onClick={handlePause}>
              <Pause className="h-4 w-4" />
            </Button>
          ) : (
            <Button variant="default" size="icon" onClick={handlePlay}>
              <Play className="h-4 w-4" />
            </Button>
          )}

          <Button
            variant="outline"
            size="icon"
            onClick={handleEnd}
            title="Zum Ende"
          >
            <SkipForward className="h-4 w-4" />
          </Button>
        </div>

        {/* Speed selector */}
        <div className="flex items-center gap-2 text-sm">
          <span className="text-muted-foreground">Geschwindigkeit:</span>
          <div className="flex gap-1">
            {(['slow', 'normal', 'fast'] as const).map((s) => (
              <Button
                key={s}
                variant={speed === s ? 'default' : 'outline'}
                size="sm"
                onClick={() => setSpeed(s)}
                className="px-2 py-1 h-7 text-xs"
              >
                {s === 'slow' ? 'Langsam' : s === 'normal' ? 'Normal' : 'Schnell'}
              </Button>
            ))}
          </div>
        </div>

        {/* Info text */}
        <p className="text-xs text-muted-foreground text-center">
          Runde {data.fromRound} → Runde {data.toRound}
          <br />
          Alle Spieler außer #{data.movements.find((m) => m.isFixed)?.startNumber || 1} rutschen im Uhrzeigersinn
        </p>
      </div>
    </div>
  );
}
