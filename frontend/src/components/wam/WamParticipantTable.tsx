'use client';

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { U12Badge } from './U12Badge';
import { JokerSelect } from './JokerSelect';
import type { WamParticipant, JokerValue } from '@/types/wam';
import { cn } from '@/lib/utils';

interface WamParticipantTableProps {
  participants: WamParticipant[];
  onJokerChange?: (_participantId: string, _jokerPoints: JokerValue) => void;
  isUpdating?: string; // participantId being updated
  readOnly?: boolean;
  className?: string;
}

/**
 * Table displaying WAM participants with joker points and U12 status
 * Sorted by effective rating (rating + jokerPoints)
 */
export function WamParticipantTable({
  participants,
  onJokerChange,
  isUpdating,
  readOnly = false,
  className,
}: WamParticipantTableProps) {
  // Sort by effective rating descending
  const sortedParticipants = [...participants].sort(
    (a, b) => b.effectiveRating - a.effectiveRating
  );

  return (
    <Table className={className}>
      <TableHeader>
        <TableRow>
          <TableHead className="w-12">#</TableHead>
          <TableHead>Name</TableHead>
          <TableHead className="text-right w-20">DWZ</TableHead>
          <TableHead className="text-center w-24">Joker</TableHead>
          <TableHead className="text-right w-24">Eff. DWZ</TableHead>
          <TableHead className="w-16"></TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {sortedParticipants.map((participant, index) => (
          <TableRow
            key={participant.participantId}
            className={cn(participant.isU12 && 'bg-yellow-50 dark:bg-yellow-950/20')}
          >
            <TableCell className="font-medium">{index + 1}</TableCell>
            <TableCell>
              <div className="flex items-center gap-2">
                <span>{participant.name}</span>
                {participant.clubName && (
                  <span className="text-muted-foreground text-xs">
                    ({participant.clubName})
                  </span>
                )}
              </div>
            </TableCell>
            <TableCell className="text-right font-mono">
              {participant.rating}
            </TableCell>
            <TableCell className="text-center">
              {readOnly ? (
                <span className="font-mono">
                  {participant.jokerPoints > 0 ? `+${participant.jokerPoints}` : '0'}
                </span>
              ) : (
                <JokerSelect
                  value={participant.jokerPoints}
                  onChange={(value) =>
                    onJokerChange?.(participant.participantId, value)
                  }
                  disabled={isUpdating === participant.participantId}
                  className="w-[80px] mx-auto"
                />
              )}
            </TableCell>
            <TableCell className="text-right font-mono font-semibold">
              {participant.effectiveRating}
            </TableCell>
            <TableCell>
              {participant.isU12 && <U12Badge age={participant.age} />}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
