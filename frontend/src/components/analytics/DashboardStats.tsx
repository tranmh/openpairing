import React from 'react';
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
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  LineChart,
  Line,
} from 'recharts';
import {
  Trophy,
  Users,
  Calendar,
  Building,
  TrendingUp,
  Gamepad2,
} from 'lucide-react';

interface TournamentSummary {
  id: string;
  name: string;
  startDate: string;
  endDate: string;
  participants: number;
  rounds: number;
  status: string;
}

interface PlayerSummary {
  id: string;
  name: string;
  rating: number;
  title?: string;
  club?: string;
  gamesPlayed: number;
}

interface MonthlyActivity {
  month: string;
  tournaments: number;
  games: number;
  players: number;
}

interface DashboardStatsData {
  totalTournaments: number;
  activeTournaments: number;
  totalPlayers: number;
  totalGames: number;
  totalClubs: number;
  totalTeams: number;
  recentTournaments: TournamentSummary[];
  upcomingTournaments: TournamentSummary[];
  topPlayers: PlayerSummary[];
  activityByMonth: MonthlyActivity[];
}

interface DashboardStatsProps {
  stats?: DashboardStatsData;
  isLoading?: boolean;
}

function StatCard({
  title,
  value,
  icon: Icon,
  description,
}: {
  title: string;
  value: number | string;
  icon: React.ElementType;
  description?: string;
}) {
  const formattedValue = typeof value === 'number' ? value.toLocaleString() : value;

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{formattedValue}</div>
        {description && (
          <p className="text-xs text-muted-foreground">{description}</p>
        )}
      </CardContent>
    </Card>
  );
}

function TitleBadge({ title }: { title?: string }) {
  if (!title) return null;

  const colors: Record<string, string> = {
    GM: 'bg-amber-500 text-white',
    IM: 'bg-amber-400 text-black',
    FM: 'bg-gray-400 text-black',
    CM: 'bg-gray-300 text-black',
    WGM: 'bg-pink-500 text-white',
    WIM: 'bg-pink-400 text-black',
    WFM: 'bg-rose-300 text-black',
    WCM: 'bg-rose-200 text-black',
  };

  return (
    <Badge className={`text-xs ${colors[title] || 'bg-gray-200'}`}>
      {title}
    </Badge>
  );
}

export function DashboardStats({ stats, isLoading = false }: DashboardStatsProps) {
  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {[...Array(4)].map((_, i) => (
            <Card key={i}>
              <CardContent className="p-6">
                <div className="h-16 animate-pulse bg-muted rounded" />
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    );
  }

  if (!stats) {
    return (
      <Card>
        <CardContent className="p-6">
          <p className="text-muted-foreground text-center">No statistics available</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Overview Stats */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Total Tournaments"
          value={stats.totalTournaments}
          icon={Trophy}
          description={`${stats.activeTournaments} active`}
        />
        <StatCard
          title="Total Players"
          value={stats.totalPlayers}
          icon={Users}
        />
        <StatCard
          title="Total Games"
          value={stats.totalGames}
          icon={Gamepad2}
        />
        <StatCard
          title="Clubs"
          value={stats.totalClubs}
          icon={Building}
        />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Activity Chart */}
        {stats.activityByMonth && stats.activityByMonth.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <TrendingUp className="h-5 w-5" />
                Activity Over Time
              </CardTitle>
              <CardDescription>Tournaments and games per month</CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={stats.activityByMonth}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis
                    dataKey="month"
                    tickFormatter={(value) => {
                      const [year, month] = value.split('-');
                      return `${month}/${year.slice(2)}`;
                    }}
                  />
                  <YAxis />
                  <Tooltip />
                  <Line
                    type="monotone"
                    dataKey="tournaments"
                    stroke="#8884d8"
                    name="Tournaments"
                  />
                  <Line
                    type="monotone"
                    dataKey="games"
                    stroke="#82ca9d"
                    name="Games"
                  />
                </LineChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        )}

        {/* Top Players */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Trophy className="h-5 w-5" />
              Top Rated Players
            </CardTitle>
            <CardDescription>Highest rated active players</CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-12">#</TableHead>
                  <TableHead>Player</TableHead>
                  <TableHead className="text-right">Rating</TableHead>
                  <TableHead className="text-right">Games</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {stats.topPlayers?.slice(0, 10).map((player, idx) => (
                  <TableRow key={player.id}>
                    <TableCell className="font-mono">{idx + 1}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <TitleBadge title={player.title} />
                        <span className="font-medium">{player.name}</span>
                      </div>
                      {player.club && (
                        <div className="text-xs text-muted-foreground">
                          {player.club}
                        </div>
                      )}
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      {player.rating}
                    </TableCell>
                    <TableCell className="text-right font-mono text-muted-foreground">
                      {player.gamesPlayed}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Upcoming Tournaments */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Calendar className="h-5 w-5" />
              Upcoming Tournaments
            </CardTitle>
            <CardDescription>Tournaments starting soon</CardDescription>
          </CardHeader>
          <CardContent>
            {stats.upcomingTournaments?.length > 0 ? (
              <div className="space-y-4">
                {stats.upcomingTournaments.map((tournament) => (
                  <div
                    key={tournament.id}
                    className="flex items-center justify-between p-3 bg-muted rounded-lg"
                  >
                    <div>
                      <div className="font-medium">{tournament.name}</div>
                      <div className="text-sm text-muted-foreground">
                        {new Date(tournament.startDate).toLocaleDateString()} |{' '}
                        {tournament.rounds} rounds
                      </div>
                    </div>
                    <div className="text-right">
                      <Badge variant="outline">{tournament.participants} players</Badge>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-muted-foreground text-center py-4">
                No upcoming tournaments
              </p>
            )}
          </CardContent>
        </Card>

        {/* Recent Tournaments */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Trophy className="h-5 w-5" />
              Recently Completed
            </CardTitle>
            <CardDescription>Recent tournament results</CardDescription>
          </CardHeader>
          <CardContent>
            {stats.recentTournaments?.length > 0 ? (
              <div className="space-y-4">
                {stats.recentTournaments.map((tournament) => (
                  <div
                    key={tournament.id}
                    className="flex items-center justify-between p-3 bg-muted rounded-lg"
                  >
                    <div>
                      <div className="font-medium">{tournament.name}</div>
                      <div className="text-sm text-muted-foreground">
                        {new Date(tournament.endDate).toLocaleDateString()} |{' '}
                        {tournament.rounds} rounds
                      </div>
                    </div>
                    <div className="text-right">
                      <Badge variant="secondary">{tournament.participants} players</Badge>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-muted-foreground text-center py-4">
                No recent tournaments
              </p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

export default DashboardStats;
