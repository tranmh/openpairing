'use client';

import { useState } from 'react';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Users, Trash2, RefreshCw } from 'lucide-react';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog';
import { U12Badge } from './U12Badge';
import { GroupProposalDialog } from './GroupProposalDialog';
import {
  useWamStatus,
  useGroupProposal,
  useConfirmGroups,
  useDeleteGroups,
} from '@/hooks/useWam';
import type { WamGroup, U12Override } from '@/types/wam';
import { toast } from 'sonner';
import { cn } from '@/lib/utils';

interface WamGroupsTabProps {
  tournamentId: string;
  canModify?: boolean;
}

export function WamGroupsTab({ tournamentId, canModify = false }: WamGroupsTabProps) {
  const [showProposalDialog, setShowProposalDialog] = useState(false);

  const { data: status, isLoading: statusLoading, refetch: refetchStatus } = useWamStatus(tournamentId);
  const {
    data: proposal,
    isLoading: proposalLoading,
    refetch: refetchProposal,
  } = useGroupProposal(tournamentId, showProposalDialog);
  const confirmGroupsMutation = useConfirmGroups(tournamentId);
  const deleteGroupsMutation = useDeleteGroups(tournamentId);

  const handleOpenProposal = async () => {
    setShowProposalDialog(true);
    await refetchProposal();
  };

  const handleConfirmGroups = async (u12Overrides: U12Override[]) => {
    try {
      await confirmGroupsMutation.mutateAsync(u12Overrides);
      setShowProposalDialog(false);
      toast.success('Gruppen wurden erstellt');
    } catch (error: any) {
      toast.error(error.message || 'Fehler beim Erstellen der Gruppen');
    }
  };

  const handleDeleteGroups = async () => {
    try {
      await deleteGroupsMutation.mutateAsync();
      toast.success('Gruppen wurden gelöscht');
    } catch (error: any) {
      toast.error(error.message || 'Fehler beim Löschen der Gruppen');
    }
  };

  if (statusLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-32" />
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {[...Array(3)].map((_, i) => (
              <Skeleton key={i} className="h-32" />
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!status) {
    return (
      <Card>
        <CardContent className="py-10 text-center">
          <p className="text-muted-foreground">WAM-Daten nicht verfügbar</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex justify-between items-center">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Users className="h-5 w-5" />
                Gruppen
              </CardTitle>
              <CardDescription>
                {status.totalPlayers} Spieler, {status.u12Count} U12-Spieler
              </CardDescription>
            </div>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={() => refetchStatus()}>
                <RefreshCw className="h-4 w-4" />
              </Button>
              {canModify && !status.hasGroups && (
                <Button onClick={handleOpenProposal} disabled={proposalLoading}>
                  {proposalLoading ? 'Wird berechnet...' : 'Gruppen einteilen'}
                </Button>
              )}
              {canModify && status.hasGroups && (
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button variant="destructive" size="sm">
                      <Trash2 className="mr-2 h-4 w-4" />
                      Gruppen löschen
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>Gruppen löschen?</AlertDialogTitle>
                      <AlertDialogDescription>
                        Alle {status.groupCount} Gruppen werden gelöscht. Diese Aktion kann
                        nicht rückgängig gemacht werden.
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>Abbrechen</AlertDialogCancel>
                      <AlertDialogAction
                        onClick={handleDeleteGroups}
                        className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                      >
                        Löschen
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {/* Joker Statistics */}
          <div className="mb-6 p-4 bg-muted rounded-lg">
            <h4 className="font-medium mb-2">Joker-Verteilung</h4>
            <div className="grid grid-cols-4 gap-4 text-center">
              <div>
                <div className="text-2xl font-bold">{status.jokerStats.joker0}</div>
                <div className="text-sm text-muted-foreground">0</div>
              </div>
              <div>
                <div className="text-2xl font-bold">{status.jokerStats.joker50}</div>
                <div className="text-sm text-muted-foreground">+50</div>
              </div>
              <div>
                <div className="text-2xl font-bold">{status.jokerStats.joker100}</div>
                <div className="text-sm text-muted-foreground">+100</div>
              </div>
              <div>
                <div className="text-2xl font-bold">{status.jokerStats.joker150}</div>
                <div className="text-sm text-muted-foreground">+150</div>
              </div>
            </div>
          </div>

          {/* Groups Display */}
          {!status.hasGroups ? (
            <div className="text-center py-10">
              <Users className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <p className="text-muted-foreground mb-4">
                Noch keine Gruppen eingeteilt
              </p>
              {canModify && (
                <Button onClick={handleOpenProposal} disabled={proposalLoading}>
                  {proposalLoading ? 'Wird berechnet...' : 'Gruppen einteilen'}
                </Button>
              )}
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {status.groups?.map((group) => (
                <GroupCard key={group.id} group={group} />
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <GroupProposalDialog
        open={showProposalDialog}
        onOpenChange={setShowProposalDialog}
        proposal={proposal ?? null}
        onConfirm={handleConfirmGroups}
        isConfirming={confirmGroupsMutation.isPending}
      />
    </>
  );
}

interface GroupCardProps {
  group: WamGroup;
}

function GroupCard({ group }: GroupCardProps) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex justify-between items-center">
          <CardTitle className="text-lg">{group.name}</CardTitle>
          <span className="text-sm text-muted-foreground">
            {group.players.length}/{group.targetSize}
          </span>
        </div>
        <CardDescription>
          Runde {group.roundsPlayed}/{group.roundsTotal}
          {group.isComplete && ' (Abgeschlossen)'}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-8">#</TableHead>
              <TableHead>Name</TableHead>
              <TableHead className="text-right w-16">DWZ</TableHead>
              <TableHead className="text-right w-16">Pkt</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {group.players
              .sort((a, b) => a.rank - b.rank)
              .map((player) => (
                <TableRow
                  key={player.participantId}
                  className={cn(player.isU12 && 'bg-yellow-50 dark:bg-yellow-950/20')}
                >
                  <TableCell className="font-medium">{player.rank}</TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <span className="truncate max-w-[120px]">{player.name}</span>
                      {player.isU12 && <U12Badge className="text-[10px] px-1 py-0" />}
                    </div>
                  </TableCell>
                  <TableCell className="text-right font-mono text-sm">
                    {player.effectiveRating}
                  </TableCell>
                  <TableCell className="text-right font-mono font-medium">
                    {player.points}
                  </TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
