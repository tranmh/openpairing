'use client';

import { useState, useEffect, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { AlertCircle, CheckCircle2, Clock, GripVertical, Save } from 'lucide-react';
import { TeamMember } from './RosterManager';

export interface LineupEntry {
  board: number;
  playerId: string;
  rating: number;
}

export interface TeamLineup {
  id: string;
  tournamentId: string;
  roundId: string;
  teamParticipantId: string;
  lineup: LineupEntry[];
  isValid: boolean;
  validationErrors?: string[];
  submittedAt?: string;
  submittedBy?: string;
}

interface LineupEditorProps {
  roundId: string;
  teamParticipantId: string;
  boardsPerMatch: number;
  roster: TeamMember[];
  existingLineup?: TeamLineup;
  deadline?: Date;
  enforceRatingOrder?: boolean;
  ratingOrderTolerance?: number;
  onSubmit: (lineup: LineupEntry[]) => Promise<void>;
  readOnly?: boolean;
}

export function LineupEditor({
  boardsPerMatch,
  roster,
  existingLineup,
  deadline,
  enforceRatingOrder = true,
  ratingOrderTolerance = 0,
  onSubmit,
  readOnly = false,
}: LineupEditorProps) {
  const [lineup, setLineup] = useState<(LineupEntry | null)[]>(
    Array(boardsPerMatch).fill(null)
  );
  const [validationErrors, setValidationErrors] = useState<string[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [timeRemaining, setTimeRemaining] = useState<string | null>(null);
  const [isOverdue, setIsOverdue] = useState(false);

  // Initialize lineup from existing data
  useEffect(() => {
    if (existingLineup) {
      const newLineup: (LineupEntry | null)[] = Array(boardsPerMatch).fill(null);
      existingLineup.lineup.forEach((entry) => {
        if (entry.board >= 1 && entry.board <= boardsPerMatch) {
          newLineup[entry.board - 1] = entry;
        }
      });
      setLineup(newLineup);
    }
  }, [existingLineup, boardsPerMatch]);

  // Deadline countdown
  useEffect(() => {
    if (!deadline) return;

    const updateTime = () => {
      const now = new Date();
      const diff = deadline.getTime() - now.getTime();

      if (diff <= 0) {
        setTimeRemaining('Overdue');
        setIsOverdue(true);
        return;
      }

      setIsOverdue(false);
      const hours = Math.floor(diff / (1000 * 60 * 60));
      const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
      const seconds = Math.floor((diff % (1000 * 60)) / 1000);

      if (hours > 0) {
        setTimeRemaining(`${hours}h ${minutes}m`);
      } else if (minutes > 0) {
        setTimeRemaining(`${minutes}m ${seconds}s`);
      } else {
        setTimeRemaining(`${seconds}s`);
      }
    };

    updateTime();
    const interval = setInterval(updateTime, 1000);
    return () => clearInterval(interval);
  }, [deadline]);

  // Validate lineup
  const validateLineup = useCallback(() => {
    const errors: string[] = [];
    const filledBoards = lineup.filter((e) => e !== null) as LineupEntry[];

    // Check if all boards are filled
    if (filledBoards.length < boardsPerMatch) {
      errors.push(`All ${boardsPerMatch} boards must be filled`);
    }

    // Check for duplicate players
    const playerIds = filledBoards.map((e) => e.playerId);
    const uniqueIds = new Set(playerIds);
    if (playerIds.length !== uniqueIds.size) {
      errors.push('Each player can only play on one board');
    }

    // Check rating order
    if (enforceRatingOrder && filledBoards.length >= 2) {
      for (let i = 1; i < filledBoards.length; i++) {
        const prev = lineup[i - 1];
        const curr = lineup[i];
        if (prev && curr) {
          const ratingDiff = curr.rating - prev.rating;
          if (ratingDiff > ratingOrderTolerance) {
            errors.push(
              `Board ${i + 1} rating (${curr.rating}) exceeds board ${i} rating (${prev.rating}) by more than ${ratingOrderTolerance} points`
            );
          }
        }
      }
    }

    setValidationErrors(errors);
    return errors.length === 0;
  }, [lineup, boardsPerMatch, enforceRatingOrder, ratingOrderTolerance]);

  useEffect(() => {
    validateLineup();
  }, [validateLineup]);

  const handlePlayerSelect = (boardIndex: number, playerId: string) => {
    const player = roster.find((m) => m.playerId === playerId);
    if (!player) return;

    const newLineup = [...lineup];
    newLineup[boardIndex] = {
      board: boardIndex + 1,
      playerId: player.playerId,
      rating: player.playerRating,
    };
    setLineup(newLineup);
  };

  const handleClearBoard = (boardIndex: number) => {
    const newLineup = [...lineup];
    newLineup[boardIndex] = null;
    setLineup(newLineup);
  };

  const handleSubmit = async () => {
    if (!validateLineup()) return;

    setIsSubmitting(true);
    try {
      const filledLineup = lineup.filter((e) => e !== null) as LineupEntry[];
      await onSubmit(filledLineup);
    } finally {
      setIsSubmitting(false);
    }
  };

  const getAvailablePlayers = (boardIndex: number) => {
    const usedPlayerIds = new Set(
      lineup
        .filter((e, i) => e !== null && i !== boardIndex)
        .map((e) => e!.playerId)
    );
    return roster
      .filter((m) => !m.isReserve && !usedPlayerIds.has(m.playerId))
      .sort((a, b) => b.playerRating - a.playerRating);
  };

  const isValid = validationErrors.length === 0;
  const hasSubmitted = !!existingLineup?.submittedAt;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between">
          <div>
            <CardTitle>Lineup Submission</CardTitle>
            <CardDescription>
              Select {boardsPerMatch} players for the upcoming match
            </CardDescription>
          </div>
          {deadline && (
            <div className={`flex items-center gap-2 ${isOverdue ? 'text-destructive' : ''}`}>
              <Clock className="h-4 w-4" />
              <span className="text-sm font-medium">{timeRemaining}</span>
            </div>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Validation status */}
        {validationErrors.length > 0 ? (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertTitle>Validation Errors</AlertTitle>
            <AlertDescription>
              <ul className="list-disc list-inside mt-2">
                {validationErrors.map((error, i) => (
                  <li key={i}>{error}</li>
                ))}
              </ul>
            </AlertDescription>
          </Alert>
        ) : (
          <Alert>
            <CheckCircle2 className="h-4 w-4" />
            <AlertTitle>Lineup Valid</AlertTitle>
            <AlertDescription>
              {hasSubmitted
                ? `Submitted at ${new Date(existingLineup!.submittedAt!).toLocaleString()}`
                : 'Ready to submit'}
            </AlertDescription>
          </Alert>
        )}

        {/* Board assignments */}
        <div className="space-y-2">
          {lineup.map((entry, index) => (
            <div
              key={index}
              className="flex items-center gap-3 p-3 border rounded-lg bg-muted/50"
            >
              <div className="flex items-center gap-2 w-20">
                <GripVertical className="h-4 w-4 text-muted-foreground" />
                <Badge variant="outline">Board {index + 1}</Badge>
              </div>
              {!readOnly ? (
                <Select
                  value={entry?.playerId ?? ''}
                  onValueChange={(value) => {
                    if (value === 'clear') {
                      handleClearBoard(index);
                    } else {
                      handlePlayerSelect(index, value);
                    }
                  }}
                >
                  <SelectTrigger className="flex-1">
                    <SelectValue placeholder="Select player...">
                      {entry && (
                        <span>
                          {roster.find((m) => m.playerId === entry.playerId)?.playerName} ({entry.rating})
                        </span>
                      )}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {entry && (
                      <SelectItem value="clear" className="text-muted-foreground">
                        Clear selection
                      </SelectItem>
                    )}
                    {getAvailablePlayers(index).map((member) => (
                      <SelectItem key={member.playerId} value={member.playerId}>
                        <div className="flex items-center gap-2">
                          {member.playerTitle && (
                            <Badge variant="outline" className="text-xs">
                              {member.playerTitle}
                            </Badge>
                          )}
                          {member.playerName}
                          <span className="text-muted-foreground">
                            ({member.playerRating})
                          </span>
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <div className="flex-1 text-sm">
                  {entry ? (
                    <span>
                      {roster.find((m) => m.playerId === entry.playerId)?.playerName ?? 'Unknown'} ({entry.rating})
                    </span>
                  ) : (
                    <span className="text-muted-foreground">Not assigned</span>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>

        {/* Submit button */}
        {!readOnly && (
          <div className="flex justify-end">
            <Button
              onClick={handleSubmit}
              disabled={!isValid || isSubmitting || isOverdue}
            >
              <Save className="h-4 w-4 mr-2" />
              {isSubmitting ? 'Submitting...' : hasSubmitted ? 'Update Lineup' : 'Submit Lineup'}
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
