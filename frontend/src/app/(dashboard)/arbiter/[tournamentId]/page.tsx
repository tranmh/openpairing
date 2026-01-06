'use client';

import { useParams } from 'next/navigation';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Skeleton } from '@/components/ui/skeleton';
import {
  AlertTriangle,
  Clock,
  PlayCircle,
  RefreshCw,
  Settings,
  Shield,
  Shuffle,
  Users,
} from 'lucide-react';

import {
  useArbiterStatus,
  StatusCards,
  CreateAnnouncementDialog,
  ActiveGamesTab,
  ParticipantsTab,
  RoundControlTab,
  PairingsManagementTab,
  ActionsTab,
  SettingsTab,
} from '@/components/arbiter';

/**
 * Arbiter Console Page
 *
 * Main dashboard for tournament arbiters to manage:
 * - Active games and results
 * - Participants (withdraw, reactivate, rating override)
 * - Round control (generate pairings, start, complete)
 * - Pairing management (swap, color change, delete, manual create)
 * - Special actions (bye, result correction, late entry)
 * - Settings (clone, forbidden pairs, special rankings)
 */
export default function ArbiterConsolePage() {
  const params = useParams();
  const tournamentId = params.tournamentId as string;

  const { status, isLoading, error, refetch, isRefetching } = useArbiterStatus(tournamentId);

  if (isLoading) {
    return (
      <div className="container mx-auto py-6 space-y-6">
        <Skeleton className="h-10 w-1/3" />
        <div className="grid gap-4 md:grid-cols-4">
          {[...Array(4)].map((_, i) => (
            <Skeleton key={i} className="h-32" />
          ))}
        </div>
        <Skeleton className="h-96" />
      </div>
    );
  }

  if (error || !status) {
    return (
      <div className="container mx-auto py-6">
        <Card>
          <CardContent className="py-10 text-center">
            <AlertTriangle className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground">Fehler beim Laden der Turnierdaten</p>
            <Button variant="outline" className="mt-4" onClick={() => refetch()}>
              <RefreshCw className="mr-2 h-4 w-4" />
              Erneut versuchen
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-6 space-y-6">
      {/* Header */}
      <div className="flex justify-between items-start">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <Shield className="h-6 w-6" />
            <h1 className="text-3xl font-bold tracking-tight">Schiedsrichterkonsole</h1>
          </div>
          <p className="text-muted-foreground">{status.tournament.name}</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => refetch()} disabled={isRefetching}>
            <RefreshCw className={`mr-2 h-4 w-4 ${isRefetching ? 'animate-spin' : ''}`} />
            Aktualisieren
          </Button>
          <CreateAnnouncementDialog tournamentId={tournamentId} />
        </div>
      </div>

      {/* Status Cards */}
      <StatusCards
        currentRound={status.currentRound}
        totalRounds={status.tournament.rounds}
        statistics={status.statistics}
        activeGamesCount={status.activeGames?.length ?? 0}
        pendingResults={status.pendingResults}
      />

      {/* Main Tabs */}
      <Tabs defaultValue="games" className="space-y-4">
        <TabsList>
          <TabsTrigger value="games">
            <Clock className="mr-2 h-4 w-4" />
            Aktive Partien
          </TabsTrigger>
          <TabsTrigger value="participants">
            <Users className="mr-2 h-4 w-4" />
            Teilnehmer
          </TabsTrigger>
          <TabsTrigger value="rounds">
            <PlayCircle className="mr-2 h-4 w-4" />
            Runden
          </TabsTrigger>
          <TabsTrigger value="pairings">
            <Shuffle className="mr-2 h-4 w-4" />
            Paarungen
          </TabsTrigger>
          <TabsTrigger value="actions">
            <Settings className="mr-2 h-4 w-4" />
            Aktionen
          </TabsTrigger>
          <TabsTrigger value="settings">
            <Settings className="mr-2 h-4 w-4" />
            Einstellungen
          </TabsTrigger>
        </TabsList>

        <TabsContent value="games">
          <ActiveGamesTab
            tournamentId={tournamentId}
            games={status.activeGames ?? []}
            roundStatus={status.currentRound?.status || 'pending'}
          />
        </TabsContent>

        <TabsContent value="participants">
          <ParticipantsTab tournamentId={tournamentId} participants={status.participants ?? []} />
        </TabsContent>

        <TabsContent value="rounds">
          <RoundControlTab
            tournamentId={tournamentId}
            rounds={status.rounds ?? []}
            currentRound={status.currentRound}
            pendingResults={status.pendingResults}
          />
        </TabsContent>

        <TabsContent value="pairings">
          <PairingsManagementTab
            tournamentId={tournamentId}
            rounds={status.rounds ?? []}
            participants={status.participants ?? []}
          />
        </TabsContent>

        <TabsContent value="actions">
          <ActionsTab
            tournamentId={tournamentId}
            currentRound={status.currentRound}
            participants={status.participants ?? []}
            tournamentRounds={status.tournament.rounds}
          />
        </TabsContent>

        <TabsContent value="settings">
          <SettingsTab
            tournamentId={tournamentId}
            tournamentName={status.tournament.name}
            participants={status.participants ?? []}
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}
