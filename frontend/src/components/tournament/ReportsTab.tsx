'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { AlertTriangle } from 'lucide-react';
import { reportsApi } from '@/lib/api';
import { DWZReportViewer } from '@/components/reports/DWZReportViewer';
import { TRFReportViewer } from '@/components/reports/TRFReportViewer';
import { NormReportViewer } from '@/components/reports/NormReportViewer';
import { useAuthStore } from '@/store/auth';

interface ReportsTabProps {
  tournamentId: string;
}

type ReportType = 'dwz' | 'trf' | 'norm';

export function ReportsTab({ tournamentId }: ReportsTabProps) {
  const [activeReport, setActiveReport] = useState<ReportType>('dwz');
  const { isHydrated, isAuthenticated } = useAuthStore();

  // Wait for auth hydration before making authenticated API calls
  const canFetch = isHydrated && isAuthenticated;

  const { data: dwzData, isLoading: dwzLoading, error: dwzError } = useQuery({
    queryKey: ['report', 'dwz', tournamentId],
    queryFn: async () => {
      const response = await reportsApi.getDWZReport(tournamentId);
      if (response.error) {
        throw new Error(response.error.message);
      }
      return response.data;
    },
    enabled: canFetch && activeReport === 'dwz',
  });

  const { data: trfData, isLoading: trfLoading, error: trfError } = useQuery({
    queryKey: ['report', 'trf', tournamentId],
    queryFn: async () => {
      const response = await reportsApi.getTRFReport(tournamentId);
      if (response.error) {
        throw new Error(response.error.message);
      }
      return response.data;
    },
    enabled: canFetch && activeReport === 'trf',
  });

  const { data: normData, isLoading: normLoading, error: normError } = useQuery({
    queryKey: ['report', 'norm', tournamentId],
    queryFn: async () => {
      const response = await reportsApi.getNormReport(tournamentId);
      if (response.error) {
        throw new Error(response.error.message);
      }
      return response.data;
    },
    enabled: canFetch && activeReport === 'norm',
  });

  const handleDownloadDWZCSV = () => {
    window.open(`/api/v1/reports/dwz/${tournamentId}/csv`);
  };

  const handleDownloadDWZXML = () => {
    window.open(`/api/v1/reports/dwz/${tournamentId}/xml`);
  };

  const handleDownloadTRF = () => {
    window.open(`/api/v1/reports/trf/${tournamentId}/download`);
  };

  return (
    <div className="space-y-4">
      {/* Report Type Tabs */}
      <div className="flex gap-2">
        <Button
          variant={activeReport === 'dwz' ? 'default' : 'outline'}
          onClick={() => setActiveReport('dwz')}
        >
          DWZ-Bericht
        </Button>
        <Button
          variant={activeReport === 'trf' ? 'default' : 'outline'}
          onClick={() => setActiveReport('trf')}
        >
          TRF (FIDE)
        </Button>
        <Button
          variant={activeReport === 'norm' ? 'default' : 'outline'}
          onClick={() => setActiveReport('norm')}
        >
          Normen
        </Button>
      </div>

      {/* Report Content */}
      {!canFetch && <LoadingSkeleton />}
      {canFetch && activeReport === 'dwz' && (
        dwzLoading ? (
          <LoadingSkeleton />
        ) : dwzError ? (
          <ErrorAlert error={dwzError} />
        ) : (
          <DWZReportViewer
            tournamentId={tournamentId}
            report={dwzData?.report}
            validationErrors={dwzData?.validationErrors}
            isValid={dwzData?.isValid}
            onDownloadCSV={handleDownloadDWZCSV}
            onDownloadXML={handleDownloadDWZXML}
          />
        )
      )}

      {canFetch && activeReport === 'trf' && (
        trfLoading ? (
          <LoadingSkeleton />
        ) : trfError ? (
          <ErrorAlert error={trfError} />
        ) : (
          <TRFReportViewer
            tournamentId={tournamentId}
            report={trfData?.report}
            validationErrors={trfData?.validationErrors}
            isValid={trfData?.isValid}
            onDownload={handleDownloadTRF}
          />
        )
      )}

      {canFetch && activeReport === 'norm' && (
        normLoading ? (
          <LoadingSkeleton />
        ) : normError ? (
          <ErrorAlert error={normError} />
        ) : (
          <NormReportViewer
            report={normData?.report}
            summary={normData?.summary}
          />
        )
      )}
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-10 w-full" />
      <Skeleton className="h-32 w-full" />
      <Skeleton className="h-48 w-full" />
    </div>
  );
}

function ErrorAlert({ error }: { error: Error }) {
  return (
    <Alert variant="destructive">
      <AlertTriangle className="h-4 w-4" />
      <AlertTitle>Fehler beim Laden des Berichts</AlertTitle>
      <AlertDescription>
        {error.message || 'Ein unbekannter Fehler ist aufgetreten.'}
      </AlertDescription>
    </Alert>
  );
}

export default ReportsTab;
