'use client';

import Link from 'next/link';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Users, Download, Printer } from 'lucide-react';
import { WamParticipantTable } from './WamParticipantTable';
import { useWamParticipants, useUpdateJoker } from '@/hooks/useWam';
import type { JokerValue } from '@/types/wam';
import { toast } from 'sonner';

interface WamParticipantsTabProps {
  tournamentId: string;
  canRegister?: boolean;
  canModify?: boolean;
}

export function WamParticipantsTab({
  tournamentId,
  canRegister = false,
  canModify = false,
}: WamParticipantsTabProps) {
  const { data: participants, isLoading } = useWamParticipants(tournamentId);
  const updateJokerMutation = useUpdateJoker(tournamentId);

  const handleJokerChange = async (participantId: string, jokerPoints: JokerValue) => {
    try {
      await updateJokerMutation.mutateAsync({ participantId, jokerPoints });
      toast.success('Joker aktualisiert');
    } catch (error: any) {
      toast.error(error.message || 'Fehler beim Aktualisieren');
    }
  };

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-48" />
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {[...Array(10)].map((_, i) => (
              <Skeleton key={i} className="h-10" />
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  const activeParticipants = participants || [];

  return (
    <Card>
      <CardHeader>
        <div className="flex justify-between items-center">
          <CardTitle>Teilnehmerliste ({activeParticipants.length})</CardTitle>
          <div className="flex gap-2">
            {activeParticipants.length > 0 && (
              <>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => window.print()}
                  className="no-print"
                >
                  <Printer className="mr-2 h-4 w-4" />
                  Drucken
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() =>
                    window.open(`/api/v1/tournaments/${tournamentId}/export/participants/pdf`)
                  }
                  className="no-print"
                >
                  <Download className="mr-2 h-4 w-4" />
                  PDF
                </Button>
              </>
            )}
            {canRegister && (
              <Link href={`/tournaments/${tournamentId}/register`}>
                <Button>
                  <Users className="mr-2 h-4 w-4" />
                  Spieler anmelden
                </Button>
              </Link>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {activeParticipants.length === 0 ? (
          <p className="text-muted-foreground text-center py-8">
            Noch keine Teilnehmer angemeldet
          </p>
        ) : (
          <WamParticipantTable
            participants={activeParticipants}
            onJokerChange={canModify ? handleJokerChange : undefined}
            isUpdating={updateJokerMutation.isPending ? updateJokerMutation.variables?.participantId : undefined}
            readOnly={!canModify}
          />
        )}
      </CardContent>
    </Card>
  );
}
