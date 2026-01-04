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
import { Textarea } from '@/components/ui/textarea';
import { Download, FileText, AlertTriangle, CheckCircle, Copy, Eye } from 'lucide-react';

interface TRFPlayer {
  startNo: number;
  gender: string;
  title: string;
  name: string;
  fideRating: number;
  federation: string;
  fideId: string;
  birthDate: string;
  points: number;
  rank: number;
  roundResults: {
    opponentNo: number;
    color: string;
    result: string;
  }[];
}

interface TRFReportData {
  tournamentId: string;
  tournamentName: string;
  tournamentCode: string;
  city: string;
  federation: string;
  startDate: string;
  endDate: string;
  numPlayers: number;
  numRatedPlayers: number;
  numRounds: number;
  ratingType: string;
  chiefArbiter: string;
  deputyArbiter: string;
  timeControl: string;
  players: TRFPlayer[];
}

interface TRFReportViewerProps {
  tournamentId: string;
  report?: TRFReportData;
  validationErrors?: string[];
  isValid?: boolean;
  isLoading?: boolean;
  onDownload?: () => void;
}

export function TRFReportViewer({
  tournamentId,
  report,
  validationErrors = [],
  isValid = true,
  isLoading = false,
  onDownload,
}: TRFReportViewerProps) {
  const [activeTab, setActiveTab] = useState<'overview' | 'players' | 'preview'>('overview');
  const [copied, setCopied] = useState(false);

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

  // Generate preview TRF content
  const generatePreview = () => {
    const lines: string[] = [];
    lines.push(`012 ${report.tournamentName}`);
    lines.push(`022 ${report.city}`);
    lines.push(`032 ${report.federation || ''}`);
    lines.push(`042 ${report.startDate}`);
    lines.push(`052 ${report.endDate}`);
    lines.push(`062 ${report.numPlayers}`);
    lines.push(`072 ${report.numRatedPlayers}`);
    lines.push(`092 ${report.numRounds}`);
    lines.push(`102 ${report.chiefArbiter}`);
    lines.push(`122 ${report.timeControl}`);
    lines.push('');

    report.players?.forEach((player) => {
      let line = `001 ${String(player.startNo).padStart(4)} ${player.gender} ${(player.title || '').padStart(3)} `;
      line += `${player.name.padEnd(33)} ${String(player.fideRating || 0).padStart(4)} `;
      line += `${(player.federation || '').padStart(3)} ${(player.fideId || '').padStart(11)} `;
      line += `${(player.birthDate || '').padStart(10)} ${player.points.toFixed(1).padStart(5)} ${String(player.rank).padStart(4)}`;

      player.roundResults?.forEach((round) => {
        line += `  ${String(round.opponentNo).padStart(4)} ${round.color} ${round.result}`;
      });

      lines.push(line);
    });

    return lines.join('\n');
  };

  const handleCopy = async () => {
    const content = generatePreview();
    await navigator.clipboard.writeText(content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <FileText className="h-5 w-5" />
              TRF Report
            </CardTitle>
            <CardDescription>
              FIDE Tournament Report Format for rating submission
            </CardDescription>
          </div>
          <Button variant="default" size="sm" onClick={onDownload}>
            <Download className="h-4 w-4 mr-2" />
            Download .trf
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {/* Validation Status */}
        {validationErrors.length > 0 ? (
          <Alert variant="destructive" className="mb-4">
            <AlertTriangle className="h-4 w-4" />
            <AlertTitle>Validation Warnings</AlertTitle>
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
              The TRF file passes all validation checks.
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
          <Button
            variant={activeTab === 'preview' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setActiveTab('preview')}
          >
            <Eye className="h-4 w-4 mr-1" />
            Preview
          </Button>
        </div>

        {activeTab === 'overview' && (
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Tournament</span>
                <span className="font-medium">{report.tournamentName}</span>
              </div>
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">City</span>
                <span className="font-medium">{report.city}</span>
              </div>
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Federation</span>
                <span className="font-medium">{report.federation || 'Not set'}</span>
              </div>
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Start Date</span>
                <span className="font-medium">{report.startDate}</span>
              </div>
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">End Date</span>
                <span className="font-medium">{report.endDate}</span>
              </div>
            </div>
            <div className="space-y-2">
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Players</span>
                <span className="font-medium">{report.numPlayers}</span>
              </div>
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Rated Players</span>
                <span className="font-medium">{report.numRatedPlayers}</span>
              </div>
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Rounds</span>
                <span className="font-medium">{report.numRounds}</span>
              </div>
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Chief Arbiter</span>
                <span className="font-medium">{report.chiefArbiter || 'Not set'}</span>
              </div>
              <div className="flex justify-between py-2 border-b">
                <span className="text-muted-foreground">Time Control</span>
                <span className="font-medium">{report.timeControl || 'Not set'}</span>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'players' && (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-12">No</TableHead>
                  <TableHead>Title</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Fed</TableHead>
                  <TableHead>FIDE ID</TableHead>
                  <TableHead className="text-right">Rating</TableHead>
                  <TableHead className="text-right">Points</TableHead>
                  <TableHead className="text-right">Rank</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {report.players?.map((player) => (
                  <TableRow key={player.startNo}>
                    <TableCell className="font-mono">{player.startNo}</TableCell>
                    <TableCell>
                      {player.title && (
                        <Badge variant="outline" className="text-xs">
                          {player.title}
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className="font-medium">{player.name}</TableCell>
                    <TableCell className="font-mono text-xs">{player.federation}</TableCell>
                    <TableCell className="font-mono text-xs">
                      {player.fideId || (
                        <span className="text-muted-foreground">-</span>
                      )}
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      {player.fideRating > 0 ? player.fideRating : '-'}
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      {player.points.toFixed(1)}
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

        {activeTab === 'preview' && (
          <div className="space-y-2">
            <div className="flex justify-end">
              <Button variant="outline" size="sm" onClick={handleCopy}>
                <Copy className="h-4 w-4 mr-2" />
                {copied ? 'Copied!' : 'Copy'}
              </Button>
            </div>
            <Textarea
              readOnly
              value={generatePreview()}
              className="font-mono text-xs h-96"
            />
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export default TRFReportViewer;
