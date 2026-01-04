import React, { useState } from 'react';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible';
import { Progress } from '@/components/ui/progress';
import { Award, ChevronDown, CheckCircle, XCircle } from 'lucide-react';

interface NormGameDetail {
  round: number;
  opponentName: string;
  opponentFideId: string;
  opponentRating: number;
  opponentTitle: string;
  opponentFed: string;
  color: string;
  result: number;
  ratingDiff: number;
  expectedScore: number;
}

interface NormResult {
  playerId: string;
  playerName: string;
  playerFideId: string;
  playerRating: number;
  playerFederation: string;
  normType: string;
  achieved: boolean;
  performance: number;
  gamesPlayed: number;
  score: number;
  opponentCount: number;
  federationCount: number;
  titledOppCount: number;
  foreignOppCount: number;
  averageOppRating: number;
  details: NormGameDetail[];
  missingReasons: string[];
}

interface NormReportData {
  tournamentId: string;
  tournamentName: string;
  generatedAt: string;
  normResults: NormResult[];
}

interface NormReportViewerProps {
  report?: NormReportData;
  summary?: {
    achieved: number;
    nearMiss: number;
    total: number;
  };
  isLoading?: boolean;
}

const normRequirements: Record<string, { performance: number; minGames: number; minFederations: number; minTitled: number }> = {
  GM: { performance: 2600, minGames: 9, minFederations: 3, minTitled: 3 },
  IM: { performance: 2450, minGames: 9, minFederations: 3, minTitled: 3 },
  WGM: { performance: 2400, minGames: 9, minFederations: 3, minTitled: 3 },
  WIM: { performance: 2250, minGames: 9, minFederations: 3, minTitled: 3 },
  FM: { performance: 2300, minGames: 9, minFederations: 2, minTitled: 0 },
  CM: { performance: 2200, minGames: 9, minFederations: 2, minTitled: 0 },
};

function NormBadge({ normType }: { normType: string }) {
  const colors: Record<string, string> = {
    GM: 'bg-amber-500 text-white',
    IM: 'bg-amber-400 text-black',
    WGM: 'bg-pink-500 text-white',
    WIM: 'bg-pink-400 text-black',
    FM: 'bg-gray-400 text-black',
    WFM: 'bg-rose-300 text-black',
    CM: 'bg-gray-300 text-black',
    WCM: 'bg-rose-200 text-black',
  };

  return (
    <Badge className={colors[normType] || 'bg-gray-200'}>
      {normType}
    </Badge>
  );
}

function NormProgressCard({ result }: { result: NormResult }) {
  const [isOpen, setIsOpen] = useState(false);
  const req = normRequirements[result.normType];

  const performanceProgress = req ? Math.min(100, (result.performance / req.performance) * 100) : 0;
  const gamesProgress = req ? Math.min(100, (result.gamesPlayed / req.minGames) * 100) : 0;
  const fedProgress = req ? Math.min(100, (result.federationCount / req.minFederations) * 100) : 0;

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <Card className={result.achieved ? 'border-green-300 bg-green-50' : ''}>
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <NormBadge normType={result.normType} />
              <div>
                <CardTitle className="text-base">{result.playerName}</CardTitle>
                <CardDescription className="text-xs">
                  {result.playerFideId} | Rating: {result.playerRating}
                </CardDescription>
              </div>
            </div>
            <div className="flex items-center gap-2">
              {result.achieved ? (
                <Badge variant="default" className="bg-green-600">
                  <CheckCircle className="h-3 w-3 mr-1" />
                  Achieved
                </Badge>
              ) : (
                <Badge variant="secondary">
                  <XCircle className="h-3 w-3 mr-1" />
                  Near Miss
                </Badge>
              )}
              <CollapsibleTrigger asChild>
                <button className="p-1 hover:bg-muted rounded">
                  <ChevronDown className={`h-4 w-4 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
                </button>
              </CollapsibleTrigger>
            </div>
          </div>
        </CardHeader>
        <CardContent className="pt-0">
          <div className="grid grid-cols-3 gap-4 text-sm">
            <div>
              <div className="flex justify-between mb-1">
                <span className="text-muted-foreground">Performance</span>
                <span className="font-mono">{result.performance}</span>
              </div>
              <Progress value={performanceProgress} className="h-2" />
              <div className="text-xs text-muted-foreground text-right mt-1">
                Required: {req?.performance || '-'}
              </div>
            </div>
            <div>
              <div className="flex justify-between mb-1">
                <span className="text-muted-foreground">Games</span>
                <span className="font-mono">{result.gamesPlayed}</span>
              </div>
              <Progress value={gamesProgress} className="h-2" />
              <div className="text-xs text-muted-foreground text-right mt-1">
                Required: {req?.minGames || 9}
              </div>
            </div>
            <div>
              <div className="flex justify-between mb-1">
                <span className="text-muted-foreground">Federations</span>
                <span className="font-mono">{result.federationCount}</span>
              </div>
              <Progress value={fedProgress} className="h-2" />
              <div className="text-xs text-muted-foreground text-right mt-1">
                Required: {req?.minFederations || 3}
              </div>
            </div>
          </div>

          <CollapsibleContent>
            <div className="mt-4 pt-4 border-t">
              {result.missingReasons.length > 0 && (
                <div className="mb-4 p-3 bg-amber-50 border border-amber-200 rounded text-sm">
                  <strong className="text-amber-800">Missing Requirements:</strong>
                  <ul className="list-disc list-inside mt-1 text-amber-700">
                    {result.missingReasons.map((reason, idx) => (
                      <li key={idx}>{reason}</li>
                    ))}
                  </ul>
                </div>
              )}

              <div className="grid grid-cols-4 gap-4 text-sm mb-4">
                <div className="text-center p-2 bg-muted rounded">
                  <div className="font-mono text-lg">{result.score.toFixed(1)}</div>
                  <div className="text-xs text-muted-foreground">Score</div>
                </div>
                <div className="text-center p-2 bg-muted rounded">
                  <div className="font-mono text-lg">{result.averageOppRating}</div>
                  <div className="text-xs text-muted-foreground">Avg Opp Rating</div>
                </div>
                <div className="text-center p-2 bg-muted rounded">
                  <div className="font-mono text-lg">{result.titledOppCount}</div>
                  <div className="text-xs text-muted-foreground">Titled Opponents</div>
                </div>
                <div className="text-center p-2 bg-muted rounded">
                  <div className="font-mono text-lg">{result.foreignOppCount}</div>
                  <div className="text-xs text-muted-foreground">Foreign Opponents</div>
                </div>
              </div>

              {result.details && result.details.length > 0 && (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-12">Rd</TableHead>
                      <TableHead>Opponent</TableHead>
                      <TableHead>Title</TableHead>
                      <TableHead>Fed</TableHead>
                      <TableHead className="text-right">Rating</TableHead>
                      <TableHead className="text-center">Color</TableHead>
                      <TableHead className="text-center">Result</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {result.details.map((game, idx) => (
                      <TableRow key={idx}>
                        <TableCell className="font-mono">{game.round}</TableCell>
                        <TableCell className="font-medium">{game.opponentName}</TableCell>
                        <TableCell>
                          {game.opponentTitle && (
                            <Badge variant="outline" className="text-xs">
                              {game.opponentTitle}
                            </Badge>
                          )}
                        </TableCell>
                        <TableCell className="font-mono text-xs">{game.opponentFed}</TableCell>
                        <TableCell className="text-right font-mono">{game.opponentRating}</TableCell>
                        <TableCell className="text-center">
                          <span className={game.color === 'W' ? 'text-white bg-gray-800 px-2 py-0.5 rounded' : 'text-black bg-gray-200 px-2 py-0.5 rounded'}>
                            {game.color}
                          </span>
                        </TableCell>
                        <TableCell className="text-center font-mono">
                          {game.result === 1 ? '1' : game.result === 0.5 ? '½' : '0'}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </div>
          </CollapsibleContent>
        </CardContent>
      </Card>
    </Collapsible>
  );
}

export function NormReportViewer({
  report,
  summary,
  isLoading = false,
}: NormReportViewerProps) {
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
          <p className="text-muted-foreground text-center">No norm data available</p>
        </CardContent>
      </Card>
    );
  }

  const achievedNorms = report.normResults?.filter(r => r.achieved) || [];
  const nearMissNorms = report.normResults?.filter(r => !r.achieved) || [];

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Award className="h-5 w-5" />
                FIDE Norm Report
              </CardTitle>
              <CardDescription>
                Title norm calculations for {report.tournamentName}
              </CardDescription>
            </div>
            {summary && (
              <div className="flex gap-4">
                <div className="text-center">
                  <div className="text-2xl font-bold text-green-600">{summary.achieved}</div>
                  <div className="text-xs text-muted-foreground">Achieved</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-amber-600">{summary.nearMiss}</div>
                  <div className="text-xs text-muted-foreground">Near Miss</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold">{summary.total}</div>
                  <div className="text-xs text-muted-foreground">Total</div>
                </div>
              </div>
            )}
          </div>
        </CardHeader>
      </Card>

      {achievedNorms.length > 0 && (
        <div className="space-y-4">
          <h3 className="text-lg font-semibold flex items-center gap-2">
            <CheckCircle className="h-5 w-5 text-green-600" />
            Achieved Norms ({achievedNorms.length})
          </h3>
          {achievedNorms.map((result, idx) => (
            <NormProgressCard key={idx} result={result} />
          ))}
        </div>
      )}

      {nearMissNorms.length > 0 && (
        <div className="space-y-4">
          <h3 className="text-lg font-semibold flex items-center gap-2">
            <XCircle className="h-5 w-5 text-amber-600" />
            Near Misses ({nearMissNorms.length})
          </h3>
          {nearMissNorms.map((result, idx) => (
            <NormProgressCard key={idx} result={result} />
          ))}
        </div>
      )}

      {report.normResults?.length === 0 && (
        <Card>
          <CardContent className="p-6">
            <p className="text-muted-foreground text-center">
              No players are close to achieving a norm in this tournament.
            </p>
          </CardContent>
        </Card>
      )}

      <div className="text-xs text-muted-foreground text-right">
        Generated: {new Date(report.generatedAt).toLocaleString()}
      </div>
    </div>
  );
}

export default NormReportViewer;
