'use client';

import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { U12Badge } from './U12Badge';
import { cn } from '@/lib/utils';
import type {
  GroupProposal,
  ProposedGroup,
  U12GroupProposal,
  U12Override,
  WamParticipant,
} from '@/types/wam';
import { AlertTriangle, Users } from 'lucide-react';

interface GroupProposalDialogProps {
  open: boolean;
  onOpenChange: (_open: boolean) => void;
  proposal: GroupProposal | null;
  onConfirm: (_u12Overrides: U12Override[]) => void;
  isConfirming?: boolean;
}

/**
 * Dialog for reviewing and confirming WAM group proposals
 * Shows proposed groups, warnings, and allows U12 player placement overrides
 */
export function GroupProposalDialog({
  open,
  onOpenChange,
  proposal,
  onConfirm,
  isConfirming,
}: GroupProposalDialogProps) {
  const [u12Overrides, setU12Overrides] = useState<Map<string, number>>(new Map());

  if (!proposal) return null;

  const handleU12Override = (participantId: string, groupIndex: number) => {
    setU12Overrides((prev) => {
      const next = new Map(prev);
      // Find the original group index for this participant
      const u12Proposal = proposal.u12Proposals?.find(
        (p) => p.player.participantId === participantId
      );
      // If selecting the proposed group, remove the override
      if (u12Proposal && groupIndex === u12Proposal.proposedGroupIndex) {
        next.delete(participantId);
      } else {
        next.set(participantId, groupIndex);
      }
      return next;
    });
  };

  const handleConfirm = () => {
    const overrides: U12Override[] = Array.from(u12Overrides.entries()).map(
      ([participantId, targetGroupIndex]) => ({
        participantId,
        targetGroupIndex,
      })
    );
    onConfirm(overrides);
  };

  const getGroupForU12 = (participantId: string, defaultIndex: number): number => {
    return u12Overrides.get(participantId) ?? defaultIndex;
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Users className="h-5 w-5" />
            Gruppenvorschlag
          </DialogTitle>
          <DialogDescription>
            {proposal.totalPlayers} Spieler werden in {proposal.groups.length} Gruppen
            eingeteilt
          </DialogDescription>
        </DialogHeader>

        {/* Warnings */}
        {proposal.warnings && proposal.warnings.length > 0 && (
          <Alert variant="destructive" className="my-4">
            <AlertTriangle className="h-4 w-4" />
            <AlertTitle>Hinweise</AlertTitle>
            <AlertDescription>
              <ul className="list-disc pl-4 mt-2">
                {proposal.warnings.map((warning, index) => (
                  <li key={index}>{warning}</li>
                ))}
              </ul>
            </AlertDescription>
          </Alert>
        )}

        {/* U12 Proposals */}
        {proposal.u12Proposals && proposal.u12Proposals.length > 0 && (
          <div className="my-4 p-4 bg-yellow-50 dark:bg-yellow-950/20 rounded-lg border border-yellow-200 dark:border-yellow-900">
            <h4 className="font-semibold flex items-center gap-2 mb-3">
              <U12Badge />
              U12-Spieler Platzierung
            </h4>
            <p className="text-sm text-muted-foreground mb-3">
              Folgende U12-Spieler können in eine stärkere Gruppe verschoben werden.
              Sie können die Vorschläge übernehmen oder die ursprüngliche Gruppe
              beibehalten.
            </p>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Spieler</TableHead>
                  <TableHead>DWZ</TableHead>
                  <TableHead>Urspr. Gruppe</TableHead>
                  <TableHead>Vorschlag</TableHead>
                  <TableHead>Zielgruppe</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {proposal.u12Proposals.map((u12) => {
                  const currentGroup = getGroupForU12(
                    u12.player.participantId,
                    u12.proposedGroupIndex
                  );
                  return (
                    <TableRow key={u12.player.participantId}>
                      <TableCell className="font-medium">
                        {u12.player.name}
                      </TableCell>
                      <TableCell className="font-mono">
                        {u12.player.effectiveRating}
                      </TableCell>
                      <TableCell>{proposal.groups[u12.originalGroupIndex]?.name}</TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {u12.reason}
                      </TableCell>
                      <TableCell>
                        <Select
                          value={String(currentGroup)}
                          onValueChange={(v) =>
                            handleU12Override(u12.player.participantId, Number(v))
                          }
                        >
                          <SelectTrigger className="w-[140px]">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {proposal.groups.map((group, index) => (
                              <SelectItem key={index} value={String(index)}>
                                {group.name}
                                {index === u12.proposedGroupIndex && ' (Vorschlag)'}
                                {index === u12.originalGroupIndex && ' (Urspr.)'}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        )}

        {/* Proposed Groups */}
        <div className="grid gap-4 md:grid-cols-2 my-4">
          {proposal.groups.map((group, groupIndex) => (
            <GroupCard
              key={groupIndex}
              group={group}
              groupIndex={groupIndex}
              u12Proposals={proposal.u12Proposals}
              u12Overrides={u12Overrides}
            />
          ))}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button onClick={handleConfirm} disabled={isConfirming}>
            {isConfirming ? 'Wird erstellt...' : 'Gruppen erstellen'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface GroupCardProps {
  group: ProposedGroup;
  groupIndex: number;
  u12Proposals?: U12GroupProposal[];
  u12Overrides: Map<string, number>;
}

function GroupCard({ group, groupIndex, u12Proposals, u12Overrides }: GroupCardProps) {
  // Calculate which players will be in this group considering U12 overrides
  const getEffectivePlayers = (): WamParticipant[] => {
    const basePlayers = group.players.filter((p) => {
      // Check if this player is a U12 with an override
      const override = u12Overrides.get(p.participantId);
      if (override !== undefined) {
        return override === groupIndex;
      }
      // Check if this player is a U12 with a proposal
      const proposal = u12Proposals?.find((u) => u.player.participantId === p.participantId);
      if (proposal) {
        return proposal.proposedGroupIndex === groupIndex;
      }
      return true;
    });

    // Add U12 players that were moved to this group
    const movedIn: WamParticipant[] = [];
    u12Proposals?.forEach((proposal) => {
      const override = u12Overrides.get(proposal.player.participantId);
      const targetGroup = override ?? proposal.proposedGroupIndex;
      if (targetGroup === groupIndex && proposal.originalGroupIndex !== groupIndex) {
        // Check if not already in base players
        if (!basePlayers.some((p) => p.participantId === proposal.player.participantId)) {
          movedIn.push(proposal.player);
        }
      }
    });

    return [...basePlayers, ...movedIn].sort(
      (a, b) => b.effectiveRating - a.effectiveRating
    );
  };

  const effectivePlayers = getEffectivePlayers();

  return (
    <div className="border rounded-lg p-4">
      <div className="flex items-center justify-between mb-3">
        <h4 className="font-semibold">{group.name}</h4>
        <span className="text-sm text-muted-foreground">
          {effectivePlayers.length}/{group.targetSize} Spieler
        </span>
      </div>
      {group.warnings && group.warnings.length > 0 && (
        <div className="text-sm text-yellow-600 dark:text-yellow-500 mb-2">
          {group.warnings.map((w, i) => (
            <div key={i} className="flex items-center gap-1">
              <AlertTriangle className="h-3 w-3" />
              {w}
            </div>
          ))}
        </div>
      )}
      <div className="space-y-1">
        {effectivePlayers.map((player, index) => (
          <div
            key={player.participantId}
            className={cn(
              'flex items-center justify-between text-sm py-1 px-2 rounded',
              player.isU12 && 'bg-yellow-50 dark:bg-yellow-950/30'
            )}
          >
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground w-5">{index + 1}.</span>
              <span>{player.name}</span>
              {player.isU12 && <U12Badge className="text-[10px] px-1.5 py-0" />}
            </div>
            <span className="font-mono text-muted-foreground">
              {player.effectiveRating}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
