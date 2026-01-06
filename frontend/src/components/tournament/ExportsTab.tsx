'use client';

import { useState, useEffect } from 'react';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Download, FileText, Table, List, Award, Gamepad2 } from 'lucide-react';

interface ExportsTabProps {
  tournamentId: string;
  currentRound?: number;
  totalRounds?: number;
}

export function ExportsTab({ tournamentId, currentRound = 0, totalRounds = 0 }: ExportsTabProps) {
  // Check if any rounds exist
  const hasRounds = totalRounds > 0;
  const validRounds = Math.max(0, totalRounds);
  const clampedRound = hasRounds ? Math.max(1, Math.min(currentRound, validRounds)) : 1;

  const [pgnRound, setPgnRound] = useState(clampedRound.toString());
  const [pairingsRound, setPairingsRound] = useState(clampedRound.toString());
  const [certificateCount, setCertificateCount] = useState('3');

  // Update round selections when currentRound or totalRounds prop changes
  useEffect(() => {
    if (hasRounds) {
      const newRound = Math.max(1, Math.min(currentRound, validRounds)).toString();
      setPgnRound(newRound);
      setPairingsRound(newRound);
    }
  }, [currentRound, validRounds, hasRounds]);

  const handleExport = (path: string) => {
    window.open(`/api/v1${path}`);
  };

  const roundOptions = hasRounds
    ? Array.from({ length: validRounds }, (_, i) => ({
        value: (i + 1).toString(),
        label: `Runde ${i + 1}`,
      }))
    : [];

  return (
    <div className="grid gap-4 md:grid-cols-2">
      {/* Games (PGN) */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Gamepad2 className="h-5 w-5" />
            Spiele
          </CardTitle>
          <CardDescription>PGN-Dateien für Schachpartien</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <Button
              variant="outline"
              className="flex-1"
              onClick={() => handleExport(`/tournaments/${tournamentId}/export/pgn`)}
            >
              <Download className="mr-2 h-4 w-4" />
              Turnier PGN
            </Button>
          </div>
          {hasRounds ? (
            <div className="space-y-2">
              <Label>Runde exportieren</Label>
              <div className="flex gap-2">
                <Select value={pgnRound} onValueChange={setPgnRound}>
                  <SelectTrigger className="flex-1">
                    <SelectValue placeholder="Runde wählen" />
                  </SelectTrigger>
                  <SelectContent>
                    {roundOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Button
                  variant="outline"
                  onClick={() => handleExport(`/tournaments/${tournamentId}/rounds/${pgnRound}/export/pgn`)}
                >
                  <Download className="mr-2 h-4 w-4" />
                  PGN
                </Button>
              </div>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              Rundenbasierte Exporte verfügbar, sobald Runden erstellt wurden.
            </p>
          )}
        </CardContent>
      </Card>

      {/* Tournament Data */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="h-5 w-5" />
            Turnierdaten
          </CardTitle>
          <CardDescription>Strukturierte Datenformate</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <Button
            variant="outline"
            className="w-full justify-start"
            onClick={() => handleExport(`/tournaments/${tournamentId}/export/json`)}
          >
            <Download className="mr-2 h-4 w-4" />
            JSON (vollständige Daten)
          </Button>
          <Button
            variant="outline"
            className="w-full justify-start"
            onClick={() => handleExport(`/tournaments/${tournamentId}/export/trf`)}
          >
            <Download className="mr-2 h-4 w-4" />
            TRF (FIDE-Format)
          </Button>
        </CardContent>
      </Card>

      {/* Tables */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Table className="h-5 w-5" />
            Tabellen
          </CardTitle>
          <CardDescription>Ranglisten und Kreuztabellen</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="space-y-2">
            <Label>Rangliste</Label>
            <div className="flex gap-2">
              <Button
                variant="outline"
                className="flex-1"
                onClick={() => handleExport(`/tournaments/${tournamentId}/export/standings`)}
              >
                <Download className="mr-2 h-4 w-4" />
                CSV
              </Button>
              <Button
                variant="outline"
                className="flex-1"
                onClick={() => handleExport(`/tournaments/${tournamentId}/export/standings/pdf`)}
              >
                <Download className="mr-2 h-4 w-4" />
                PDF
              </Button>
            </div>
          </div>
          <div className="space-y-2">
            <Label>Kreuztabelle</Label>
            <div className="flex gap-2">
              <Button
                variant="outline"
                className="flex-1"
                onClick={() => handleExport(`/tournaments/${tournamentId}/export/crosstable`)}
              >
                <Download className="mr-2 h-4 w-4" />
                CSV
              </Button>
              <Button
                variant="outline"
                className="flex-1"
                onClick={() => handleExport(`/tournaments/${tournamentId}/export/crosstable/pdf`)}
              >
                <Download className="mr-2 h-4 w-4" />
                PDF
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Lists */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <List className="h-5 w-5" />
            Listen
          </CardTitle>
          <CardDescription>Teilnehmer- und Paarungslisten</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Button
            variant="outline"
            className="w-full justify-start"
            onClick={() => handleExport(`/tournaments/${tournamentId}/export/participants/pdf`)}
          >
            <Download className="mr-2 h-4 w-4" />
            Teilnehmerliste (PDF)
          </Button>
          {hasRounds ? (
            <div className="space-y-2">
              <Label>Paarungsliste</Label>
              <div className="flex gap-2">
                <Select value={pairingsRound} onValueChange={setPairingsRound}>
                  <SelectTrigger className="flex-1">
                    <SelectValue placeholder="Runde wählen" />
                  </SelectTrigger>
                  <SelectContent>
                    {roundOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Button
                  variant="outline"
                  onClick={() => handleExport(`/tournaments/${tournamentId}/rounds/${pairingsRound}/export/pairings/pdf`)}
                >
                  <Download className="mr-2 h-4 w-4" />
                  PDF
                </Button>
              </div>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              Keine Runden vorhanden.
            </p>
          )}
        </CardContent>
      </Card>

      {/* Certificates */}
      <Card className="md:col-span-2">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Award className="h-5 w-5" />
            Urkunden
          </CardTitle>
          <CardDescription>Teilnahmeurkunden für Spieler</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-end gap-4">
            <div className="space-y-2">
              <Label htmlFor="certificateCount">Anzahl Top-Spieler</Label>
              <Input
                id="certificateCount"
                type="number"
                min="1"
                max="100"
                value={certificateCount}
                onChange={(e) => {
                  const value = e.target.value;
                  // Allow empty string for typing, but clamp on blur
                  if (value === '' || /^\d+$/.test(value)) {
                    setCertificateCount(value);
                  }
                }}
                onBlur={() => {
                  // Validate and clamp on blur
                  const num = parseInt(certificateCount);
                  if (isNaN(num) || num < 1) {
                    setCertificateCount('1');
                  } else if (num > 100) {
                    setCertificateCount('100');
                  }
                }}
                className="w-24"
              />
            </div>
            <Button
              variant="outline"
              onClick={() => {
                const num = parseInt(certificateCount);
                const count = isNaN(num) || num < 1 ? 1 : Math.min(num, 100);
                handleExport(`/tournaments/${tournamentId}/certificates?top=${count}`);
              }}
            >
              <Download className="mr-2 h-4 w-4" />
              Urkunden herunterladen (ZIP)
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default ExportsTab;
