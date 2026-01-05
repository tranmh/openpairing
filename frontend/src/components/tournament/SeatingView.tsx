'use client';

import { useState, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { tournamentsApi } from '@/lib/api';
import { SeatingChartCircular } from './SeatingChartCircular';
import { SeatingChartColumns } from './SeatingChartColumns';
import { SeatingChartList } from './SeatingChartList';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Loader2, Circle, Columns, List, Printer, Play } from 'lucide-react';
import { cn } from '@/lib/utils';

type ViewMode = 'circular' | 'columns' | 'list';

interface SeatingViewProps {
  tournamentId: string;
  totalRounds: number;
  initialRound?: number;
  mode?: 'arbiter' | 'public';
  className?: string;
}

export function SeatingView({
  tournamentId,
  totalRounds,
  initialRound,
  mode = 'public',
  className,
}: SeatingViewProps) {
  const [selectedRound, setSelectedRound] = useState(initialRound || 1);
  const [viewMode, setViewMode] = useState<ViewMode>('circular');
  const [isMobile, setIsMobile] = useState(false);

  // Detect mobile viewport
  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < 768);
    };
    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, []);

  // Auto-switch view mode on mobile
  useEffect(() => {
    if (isMobile && viewMode === 'circular') {
      setViewMode('columns');
    }
  }, [isMobile, viewMode]);

  // Fetch seating chart data
  const {
    data: chartResponse,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['seatingChart', tournamentId, selectedRound],
    queryFn: async () => {
      const response = await tournamentsApi.getSeatingChart(
        tournamentId,
        selectedRound
      );
      if (response.error) {
        throw new Error(response.error.message);
      }
      return response.data;
    },
  });

  const handlePrint = () => {
    window.print();
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error || !chartResponse) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        {error instanceof Error
          ? error.message
          : 'Sitzplan konnte nicht geladen werden.'}
      </div>
    );
  }

  const roundOptions = Array.from({ length: totalRounds }, (_, i) => i + 1);

  return (
    <div className={cn('space-y-4', className)}>
      {/* Controls */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        {/* Round selector */}
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">Runde:</span>
          <Select
            value={String(selectedRound)}
            onValueChange={(v) => setSelectedRound(Number(v))}
          >
            <SelectTrigger className="w-20">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {roundOptions.map((round) => (
                <SelectItem key={round} value={String(round)}>
                  {round}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* View mode toggle */}
        <div className="flex items-center gap-2">
          <Tabs
            value={viewMode}
            onValueChange={(v) => setViewMode(v as ViewMode)}
          >
            <TabsList className="grid grid-cols-3 w-auto">
              <TabsTrigger
                value="circular"
                disabled={isMobile}
                className="px-3"
                title="Kreisansicht"
              >
                <Circle className="h-4 w-4" />
              </TabsTrigger>
              <TabsTrigger value="columns" className="px-3" title="Spaltenansicht">
                <Columns className="h-4 w-4" />
              </TabsTrigger>
              <TabsTrigger value="list" className="px-3" title="Listenansicht">
                <List className="h-4 w-4" />
              </TabsTrigger>
            </TabsList>
          </Tabs>

          {/* Print button */}
          <Button
            variant="outline"
            size="icon"
            onClick={handlePrint}
            title="Drucken"
            className="print:hidden"
          >
            <Printer className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Info bar */}
      <div className="flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
        <span>
          {chartResponse.numPlayers} Spieler • {chartResponse.numSeats} Plätze
        </span>
        {chartResponse.byeSeat > 0 && (
          <span className="text-orange-600 dark:text-orange-400">
            Spielfrei: Platz {chartResponse.byeSeat}
          </span>
        )}
      </div>

      {/* Chart view */}
      <div className="seating-chart-print">
        {viewMode === 'circular' && !isMobile && (
          <SeatingChartCircular
            data={chartResponse}
            showPairings={true}
            size={500}
          />
        )}
        {viewMode === 'columns' && <SeatingChartColumns data={chartResponse} />}
        {viewMode === 'list' && <SeatingChartList data={chartResponse} />}
      </div>

      {/* Pairings summary */}
      <div className="border-t pt-4 print:border-t-0">
        <h3 className="font-medium mb-2">
          Paarungen Runde {selectedRound}
        </h3>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
          {chartResponse.pairings.map((pairing) => (
            <div
              key={pairing.board}
              className={cn(
                'flex items-center justify-between p-2 rounded border text-sm',
                pairing.isBye ? 'bg-muted/50' : 'bg-background'
              )}
            >
              <span className="font-medium text-muted-foreground w-8">
                {pairing.board}.
              </span>
              {pairing.isBye ? (
                <span className="flex-1 text-center text-muted-foreground italic">
                  Spielfrei: #{pairing.whiteStartNo > chartResponse.numPlayers ? pairing.blackStartNo : pairing.whiteStartNo}
                </span>
              ) : (
                <>
                  <span className="flex-1 truncate text-right pr-2">
                    <span className="inline-block w-2 h-2 rounded-full bg-white border border-gray-300 mr-1" />
                    {pairing.whiteName || `#${pairing.whiteStartNo}`}
                  </span>
                  <span className="text-muted-foreground">vs</span>
                  <span className="flex-1 truncate text-left pl-2">
                    <span className="inline-block w-2 h-2 rounded-full bg-gray-800 mr-1" />
                    {pairing.blackName || `#${pairing.blackStartNo}`}
                  </span>
                </>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
