import { describe, it, expect } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import { StandingsTable } from './StandingsTable';

// Test fixtures
function createStanding(overrides?: Partial<Standing>) {
  return {
    rank: 1,
    participantId: 'participant-1',
    playerId: 'player-1',
    name: 'Test Player',
    rating: 2000,
    points: 5.5,
    buchholz: 28.0,
    buchholzCut1: 24.5,
    sonnebornBerger: 22.5,
    wins: 5,
    draws: 1,
    losses: 1,
    ...overrides,
  };
}

interface Standing {
  rank: number;
  participantId: string;
  playerId: string;
  name: string;
  title?: string;
  rating: number;
  federation?: string;
  points: number;
  buchholz: number;
  buchholzCut1: number;
  sonnebornBerger: number;
  wins: number;
  draws: number;
  losses: number;
}

function createStandingsField(count: number): Standing[] {
  return Array.from({ length: count }, (_, i) => ({
    rank: i + 1,
    participantId: `participant-${i + 1}`,
    playerId: `player-${i + 1}`,
    name: `Player ${i + 1}`,
    rating: 2200 - i * 50,
    points: (14 - i) / 2,
    buchholz: 28.0 - i * 0.5,
    buchholzCut1: 24.0 - i * 0.5,
    sonnebornBerger: 22.0 - i * 0.5,
    wins: 7 - Math.floor(i / 2),
    draws: i % 2,
    losses: i,
  }));
}

describe('StandingsTable', () => {
  describe('basic rendering', () => {
    it('renders table with standings data', () => {
      const standings = createStandingsField(5);
      render(<StandingsTable standings={standings} />);

      // Check headers
      expect(screen.getByText('Platz')).toBeInTheDocument();
      expect(screen.getByText('Name')).toBeInTheDocument();
      expect(screen.getByText('ELO')).toBeInTheDocument();
      expect(screen.getByText('Punkte')).toBeInTheDocument();
    });

    it('renders player names', () => {
      const standings = [createStanding({ name: 'Magnus Carlsen' })];
      render(<StandingsTable standings={standings} />);

      expect(screen.getByText('Magnus Carlsen')).toBeInTheDocument();
    });

    it('renders player ranks', () => {
      const standings = createStandingsField(3);
      render(<StandingsTable standings={standings} />);

      // Ranks appear twice - once in the rank badge, once in the Nr. column
      expect(screen.getAllByText('1').length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText('2').length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText('3').length).toBeGreaterThanOrEqual(1);
    });

    it('renders player ratings', () => {
      const standings = [createStanding({ rating: 2830 })];
      render(<StandingsTable standings={standings} />);

      expect(screen.getByText('2830')).toBeInTheDocument();
    });

    it('renders dash for zero rating', () => {
      const standings = [createStanding({ rating: 0 })];
      render(<StandingsTable standings={standings} />);

      expect(screen.getByText('-')).toBeInTheDocument();
    });

    it('renders points with decimal', () => {
      const standings = [createStanding({ points: 5.5 })];
      render(<StandingsTable standings={standings} />);

      expect(screen.getByText('5.5')).toBeInTheDocument();
    });
  });

  describe('results display', () => {
    it('shows win/draw/loss record when showResults is true', () => {
      const standings = [createStanding({ wins: 5, draws: 2, losses: 0 })];
      render(<StandingsTable standings={standings} showResults={true} />);

      expect(screen.getByText('5/2/0')).toBeInTheDocument();
    });

    it('shows results header when showResults is true', () => {
      const standings = createStandingsField(1);
      render(<StandingsTable standings={standings} showResults={true} />);

      expect(screen.getByText('+/=/−')).toBeInTheDocument();
    });

    it('hides results when showResults is false', () => {
      const standings = [createStanding({ wins: 5, draws: 2, losses: 0 })];
      render(<StandingsTable standings={standings} showResults={false} />);

      expect(screen.queryByText('5/2/0')).not.toBeInTheDocument();
      expect(screen.queryByText('+/=/−')).not.toBeInTheDocument();
    });

    it('shows results by default', () => {
      const standings = [createStanding({ wins: 4, draws: 3, losses: 0 })];
      render(<StandingsTable standings={standings} />);

      expect(screen.getByText('4/3/0')).toBeInTheDocument();
    });
  });

  describe('player titles', () => {
    it('renders GM title badge', () => {
      const standings = [createStanding({ title: 'GM' })];
      render(<StandingsTable standings={standings} />);

      expect(screen.getByText('GM')).toBeInTheDocument();
    });

    it('renders IM title badge', () => {
      const standings = [createStanding({ title: 'IM' })];
      render(<StandingsTable standings={standings} />);

      expect(screen.getByText('IM')).toBeInTheDocument();
    });

    it('renders FM title badge', () => {
      const standings = [createStanding({ title: 'FM' })];
      render(<StandingsTable standings={standings} />);

      expect(screen.getByText('FM')).toBeInTheDocument();
    });

    it('renders women titles', () => {
      const standings = [createStanding({ title: 'WGM' })];
      render(<StandingsTable standings={standings} />);

      expect(screen.getByText('WGM')).toBeInTheDocument();
    });

    it('does not render badge when no title', () => {
      const standings = [createStanding({ title: undefined })];
      render(<StandingsTable standings={standings} />);

      // Badge should not be present
      expect(screen.queryByText('GM')).not.toBeInTheDocument();
      expect(screen.queryByText('IM')).not.toBeInTheDocument();
    });
  });

  describe('federation display', () => {
    it('shows federation code', () => {
      const standings = [createStanding({ federation: 'GER' })];
      render(<StandingsTable standings={standings} />);

      expect(screen.getByText('(GER)')).toBeInTheDocument();
    });

    it('does not show federation when not provided', () => {
      const standings = [createStanding({ federation: undefined })];
      render(<StandingsTable standings={standings} />);

      expect(screen.queryByText('(GER)')).not.toBeInTheDocument();
    });
  });

  describe('tiebreaks', () => {
    it('renders default tiebreaks', () => {
      const standings = createStandingsField(1);
      render(<StandingsTable standings={standings} />);

      // Default tiebreaks: buchholz, buchholz_cut1, sonneborn_berger
      expect(screen.getByText('BH')).toBeInTheDocument();
      expect(screen.getByText('BH-C1')).toBeInTheDocument();
      expect(screen.getByText('SB')).toBeInTheDocument();
    });

    it('renders custom tiebreaks', () => {
      const standings = createStandingsField(1);
      render(
        <StandingsTable standings={standings} tiebreaks={['wins', 'rating']} />
      );

      expect(screen.getByText('S')).toBeInTheDocument(); // S for wins (Siege)
      expect(screen.getAllByText('ELO').length).toBeGreaterThan(0);
    });

    it('limits tiebreaks to 4', () => {
      const standings = createStandingsField(1);
      render(
        <StandingsTable
          standings={standings}
          tiebreaks={['buchholz', 'buchholz_cut1', 'sonneborn_berger', 'wins', 'rating']}
        />
      );

      // Should show first 4 only
      expect(screen.getByText('BH')).toBeInTheDocument();
      expect(screen.getByText('BH-C1')).toBeInTheDocument();
      expect(screen.getByText('SB')).toBeInTheDocument();
      expect(screen.getByText('S')).toBeInTheDocument();
      // 5th one (rating as tiebreak) shouldn't appear as separate header beyond ELO column
    });

    it('renders tiebreak values', () => {
      const standings = [
        createStanding({
          buchholz: 28.5,
          buchholzCut1: 24.0,
          sonnebornBerger: 22.5,
        }),
      ];
      render(<StandingsTable standings={standings} />);

      expect(screen.getByText('28.5')).toBeInTheDocument();
      expect(screen.getByText('24')).toBeInTheDocument();
      expect(screen.getByText('22.5')).toBeInTheDocument();
    });
  });

  describe('highlighting', () => {
    it('highlights player when highlightPlayerId matches', () => {
      const standings = [
        createStanding({ playerId: 'player-1' }),
        createStanding({ playerId: 'player-2', rank: 2, participantId: 'p2' }),
      ];
      const { container } = render(
        <StandingsTable standings={standings} highlightPlayerId="player-1" />
      );

      // The highlighted row should have the bg-primary/5 class
      const rows = container.querySelectorAll('tbody tr');
      expect(rows[0]).toHaveClass('bg-primary/5');
      expect(rows[1]).not.toHaveClass('bg-primary/5');
    });

    it('does not highlight any player when highlightPlayerId is undefined', () => {
      const standings = createStandingsField(3);
      const { container } = render(<StandingsTable standings={standings} />);

      const rows = container.querySelectorAll('tbody tr');
      rows.forEach((row) => {
        expect(row).not.toHaveClass('bg-primary/5');
      });
    });
  });

  describe('ranking styles', () => {
    it('applies gold style to 1st place', () => {
      const standings = createStandingsField(3);
      const { container } = render(<StandingsTable standings={standings} />);

      // Find the rank cell for first place
      const firstRow = container.querySelector('tbody tr:first-child');
      const rankBadge = firstRow?.querySelector('.bg-yellow-100');
      expect(rankBadge).toBeInTheDocument();
    });

    it('applies silver style to 2nd place', () => {
      const standings = createStandingsField(3);
      const { container } = render(<StandingsTable standings={standings} />);

      // Find the rank cell for second place
      const secondRow = container.querySelector('tbody tr:nth-child(2)');
      const rankBadge = secondRow?.querySelector('.bg-gray-100');
      expect(rankBadge).toBeInTheDocument();
    });

    it('applies bronze style to 3rd place', () => {
      const standings = createStandingsField(3);
      const { container } = render(<StandingsTable standings={standings} />);

      // Find the rank cell for third place
      const thirdRow = container.querySelector('tbody tr:nth-child(3)');
      const rankBadge = thirdRow?.querySelector('.bg-orange-100');
      expect(rankBadge).toBeInTheDocument();
    });

    it('applies font-medium to top 3 players', () => {
      const standings = createStandingsField(5);
      const { container } = render(<StandingsTable standings={standings} />);

      const rows = container.querySelectorAll('tbody tr');
      expect(rows[0]).toHaveClass('font-medium');
      expect(rows[1]).toHaveClass('font-medium');
      expect(rows[2]).toHaveClass('font-medium');
      expect(rows[3]).not.toHaveClass('font-medium');
      expect(rows[4]).not.toHaveClass('font-medium');
    });
  });

  describe('empty state', () => {
    it('renders empty table when no standings', () => {
      const { container } = render(<StandingsTable standings={[]} />);

      // Table should exist but have no body rows
      expect(container.querySelector('table')).toBeInTheDocument();
      expect(container.querySelector('tbody tr')).not.toBeInTheDocument();
    });
  });

  describe('multiple players', () => {
    it('renders all players in order', () => {
      const standings = [
        createStanding({ rank: 1, name: 'First Player', participantId: 'p1' }),
        createStanding({ rank: 2, name: 'Second Player', participantId: 'p2' }),
        createStanding({ rank: 3, name: 'Third Player', participantId: 'p3' }),
      ];
      render(<StandingsTable standings={standings} />);

      expect(screen.getByText('First Player')).toBeInTheDocument();
      expect(screen.getByText('Second Player')).toBeInTheDocument();
      expect(screen.getByText('Third Player')).toBeInTheDocument();
    });

    it('uses participantId as row key', () => {
      const standings = [
        createStanding({ participantId: 'unique-id-1', name: 'Player A' }),
        createStanding({ participantId: 'unique-id-2', name: 'Player B' }),
      ];
      const { container } = render(<StandingsTable standings={standings} />);

      const rows = container.querySelectorAll('tbody tr');
      expect(rows.length).toBe(2);
    });
  });

  describe('complete standings scenario', () => {
    it('renders a realistic tournament standings table', () => {
      const standings = [
        {
          rank: 1,
          participantId: 'p1',
          playerId: 'magnus',
          name: 'Magnus Carlsen',
          title: 'GM',
          rating: 2830,
          federation: 'NOR',
          points: 7.5,
          buchholz: 42.0,
          buchholzCut1: 38.5,
          sonnebornBerger: 35.0,
          wins: 7,
          draws: 1,
          losses: 0,
        },
        {
          rank: 2,
          participantId: 'p2',
          playerId: 'fabiano',
          name: 'Fabiano Caruana',
          title: 'GM',
          rating: 2805,
          federation: 'USA',
          points: 7.0,
          buchholz: 41.5,
          buchholzCut1: 37.0,
          sonnebornBerger: 33.5,
          wins: 6,
          draws: 2,
          losses: 0,
        },
        {
          rank: 3,
          participantId: 'p3',
          playerId: 'hikaru',
          name: 'Hikaru Nakamura',
          title: 'GM',
          rating: 2780,
          federation: 'USA',
          points: 6.5,
          buchholz: 40.0,
          buchholzCut1: 36.0,
          sonnebornBerger: 31.0,
          wins: 6,
          draws: 1,
          losses: 1,
        },
      ];

      render(<StandingsTable standings={standings} highlightPlayerId="fabiano" />);

      // All players rendered
      expect(screen.getByText('Magnus Carlsen')).toBeInTheDocument();
      expect(screen.getByText('Fabiano Caruana')).toBeInTheDocument();
      expect(screen.getByText('Hikaru Nakamura')).toBeInTheDocument();

      // Titles
      expect(screen.getAllByText('GM').length).toBe(3);

      // Ratings
      expect(screen.getByText('2830')).toBeInTheDocument();
      expect(screen.getByText('2805')).toBeInTheDocument();
      expect(screen.getByText('2780')).toBeInTheDocument();

      // Points
      expect(screen.getByText('7.5')).toBeInTheDocument();
      expect(screen.getByText('7.0')).toBeInTheDocument();
      expect(screen.getByText('6.5')).toBeInTheDocument();

      // Results
      expect(screen.getByText('7/1/0')).toBeInTheDocument();
      expect(screen.getByText('6/2/0')).toBeInTheDocument();
      expect(screen.getByText('6/1/1')).toBeInTheDocument();

      // Federations
      expect(screen.getByText('(NOR)')).toBeInTheDocument();
      expect(screen.getAllByText('(USA)').length).toBe(2);
    });
  });
});
