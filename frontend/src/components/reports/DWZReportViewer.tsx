import React, { useState } from 'react';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Download, FileText, AlertTriangle, CheckCircle } from 'lucide-react';

interface DWZPlayer {
  startNumber: number;
  pkz: string;
  name: string;
  firstName: string;
  lastName: string;
  clubId: string;
  clubName: string;
  dwz: number;
  dwzIndex: number;
  fideId: string;
  fideRating: number;
  birthYear: number;
  gender: string;
  score: number;
  rank: number;
}

interface DWZTournament {
  turnierCode: string;
  name: string;
  startDate: string;
  endDate: string;
  city: string;
  organizer: string;
  chiefArbiter: string;
  timeControl: string;
  rounds: number;
  pairingSystem: string;
}

interface DWZReportData {
  tournament: DWZTournament;
  players: DWZPlayer[];
  generatedAt: string;
}

interface DWZReportViewerProps {
  tournamentId: string;
  report?: DWZReportData;
  validationErrors?: string[];
  isValid?: boolean;
  isLoading?: boolean;
  onDownloadCSV?: () => void;
  onDownloadXML?: () => void;
}

export function DWZReportViewer({
  tournamentId,
  report,
  validationErrors = [],
  isValid = true,
  isLoading = false,
  onDownloadCSV,
  onDownloadXML,
}: DWZReportViewerProps) {
  const [activeTab, setActiveTab] = useState<'overview' | 'players'>('overview');

  if (isLoading) {
    return (
      <Card>
        <CardContent className="p-6">
          <div className="flex items-center justify-center h-32">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!report) {
    return (
      <Card>
        <CardContent className="p-6">
          <p className="text-muted-foreground text-center">No report data available</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <FileText className="h-5 w-5" />
              DWZ Report
            </CardTitle>
            <CardDescription>
              German Chess Federation rating report format
            </CardDescription>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={onDownloadCSV}>
              <Download className="h-4 w-4 mr-2" />
              CSV
            </Button>
            <Button variant="outline" size="sm" onClick={onDownloadXML}>
              <Download className="h-4 w-4 mr-2" />
              XML
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {/* Validation Status */}
        {validationErrors.length > 0 ? (
          <Alert variant="destructive" className="mb-4">
            <AlertTriangle className="h-4 w-4" />
            <AlertTitle>Validation Errors</AlertTitle>
            <AlertDescription>
              <ul className="list-disc list-inside mt-2">
                {validationErrors.map((error, idx) => (
                  <li key={idx}>{error}</li>
                ))}
              </ul>
            </AlertDescription>
          </Alert>
        ) : (
          <Alert className="mb-4 border-green-200 bg-green-50">
            <CheckCircle className="h-4 w-4 text-green-600" />
            <AlertTitle className="text-green-800">Valid Report</AlertTitle>
            <AlertDescription className="text-green-700">
              The report passes all validation checks and is ready for submission.
            </AlertDescription>
          </Alert>
        )}

        {/* Tabs */}
        <div className="flex gap-2 mb-4">
          <Button
            variant={activeTab === 'overview' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setActiveTab('overview')}
          >
            Overview
          </Button>
          <Button
            variant={activeTab === 'players' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setActiveTab('players')}
          >
            Players ({report.players?.length || 0})
          </Button>
        </div>

        {activeTab === 'overview' && (
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Tournament</span>
                <span className="font-medium">{report.tournament.name}</span>
              </div>
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">City</span>
                <span className="font-medium">{report.tournament.city}</span>
              </div>
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Start Date</span>
                <span className="font-medium">
                  {new Date(report.tournament.startDate).toLocaleDateString('de-DE')}
                </span>
              </div>
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">End Date</span>
                <span className="font-medium">
                  {new Date(report.tournament.endDate).toLocaleDateString('de-DE')}
                </span>
              </div>
            </div>
            <div className="space-y-2">
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Rounds</span>
                <span className="font-medium">{report.tournament.rounds}</span>
              </div>
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Pairing System</span>
                <span className="font-medium">{report.tournament.pairingSystem}</span>
              </div>
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Chief Arbiter</span>
                <span className="font-medium">{report.tournament.chiefArbiter}</span>
              </div>
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Time Control</span>
                <span className="font-medium">{report.tournament.timeControl}</span>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'players' && (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-12">Nr</TableHead>
                  <TableHead>PKZ</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Club</TableHead>
                  <TableHead className="text-right">DWZ</TableHead>
                  <TableHead className="text-right">ELO</TableHead>
                  <TableHead className="text-right">Score</TableHead>
                  <TableHead className="text-right">Rank</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {report.players?.map((player) => (
                  <TableRow key={player.startNumber}>
                    <TableCell className="font-mono">{player.startNumber}</TableCell>
                    <TableCell className="font-mono text-xs">
                      {player.pkz || (
                        <Badge variant="destructive" className="text-xs">Missing</Badge>
                      )}
                    </TableCell>
                    <TableCell className="font-medium">
                      {player.lastName}, {player.firstName}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {player.clubName || player.clubId}
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      {player.dwz > 0 ? player.dwz : '-'}
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      {player.fideRating > 0 ? player.fideRating : '-'}
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      {player.score.toFixed(1)}
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      {player.rank}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}

        <div className="mt-4 text-xs text-muted-foreground text-right">
          Generated: {new Date(report.generatedAt).toLocaleString('de-DE')}
        </div>
      </CardContent>
    </Card>
  );
}

export default DWZReportViewer;
