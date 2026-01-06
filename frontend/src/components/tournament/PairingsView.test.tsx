import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { PairingsView } from './PairingsView';

// Mock the api module
vi.mock('@/lib/api', () => ({
  api: {
    put: vi.fn(),
  },
}));

// Mock sonner
vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

// Test wrapper with QueryClientProvider
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
  };
}

// Test fixtures
interface Game {
  id: string;
  boardNumber: number;
  whiteParticipantId?: string;
  blackParticipantId?: string;
  whiteName?: string;
  blackName?: string;
  whiteRating?: number;
  blackRating?: number;
  result?: string;
  status: string;
}

function createGame(overrides?: Partial<Game>): Game {
  return {
    id: 'game-1',
    boardNumber: 1,
    whiteParticipantId: 'p1',
    blackParticipantId: 'p2',
    whiteName: 'White Player',
    blackName: 'Black Player',
    whiteRating: 2000,
    blackRating: 1900,
    status: 'pending',
    ...overrides,
  };
}

function createPairings(count: number): Game[] {
  return Array.from({ length: count }, (_, i) => ({
    id: `game-${i + 1}`,
    boardNumber: i + 1,
    whiteParticipantId: `wp-${i + 1}`,
    blackParticipantId: `bp-${i + 1}`,
    whiteName: `White ${i + 1}`,
    blackName: `Black ${i + 1}`,
    whiteRating: 2100 - i * 50,
    blackRating: 2050 - i * 50,
    status: 'pending',
  }));
}

describe('PairingsView', () => {
  describe('basic rendering', () => {
    it('renders table with headers', () => {
      render(
        <PairingsView
          tournamentId="t1"
          games={[]}
          roundNumber={1}
          roundStatus="pending"
        />,
        { wrapper: createWrapper() }
      );

      expect(screen.getByText('Brett')).toBeInTheDocument();
      expect(screen.getByText('Weiß')).toBeInTheDocument();
      expect(screen.getByText('Ergebnis')).toBeInTheDocument();
      expect(screen.getByText('Schwarz')).toBeInTheDocument();
    });

    it('renders games with player names', () => {
      const games = [createGame()];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
        />,
        { wrapper: createWrapper() }
      );

      expect(screen.getByText('White Player')).toBeInTheDocument();
      expect(screen.getByText('Black Player')).toBeInTheDocument();
    });

    it('renders board numbers', () => {
      const games = createPairings(3);
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
        />,
        { wrapper: createWrapper() }
      );

      expect(screen.getByText('1')).toBeInTheDocument();
      expect(screen.getByText('2')).toBeInTheDocument();
      expect(screen.getByText('3')).toBeInTheDocument();
    });

    it('renders player ratings', () => {
      const games = [createGame({ whiteRating: 2200, blackRating: 2150 })];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
        />,
        { wrapper: createWrapper() }
      );

      expect(screen.getByText('(2200)')).toBeInTheDocument();
      expect(screen.getByText('(2150)')).toBeInTheDocument();
    });

    it('renders empty table when no games', () => {
      const { container } = render(
        <PairingsView
          tournamentId="t1"
          games={[]}
          roundNumber={1}
          roundStatus="pending"
        />,
        { wrapper: createWrapper() }
      );

      expect(container.querySelector('table')).toBeInTheDocument();
      expect(container.querySelector('tbody tr')).not.toBeInTheDocument();
    });
  });

  describe('result display', () => {
    it('displays 1-0 result with badge', () => {
      const games = [createGame({ result: '1-0' })];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="completed"
        />,
        { wrapper: createWrapper() }
      );

      expect(screen.getByText('1 - 0')).toBeInTheDocument();
    });

    it('displays 0-1 result with badge', () => {
      const games = [createGame({ result: '0-1' })];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="completed"
        />,
        { wrapper: createWrapper() }
      );

      expect(screen.getByText('0 - 1')).toBeInTheDocument();
    });

    it('displays draw result', () => {
      const games = [createGame({ result: '1/2-1/2' })];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="completed"
        />,
        { wrapper: createWrapper() }
      );

      expect(screen.getByText('½ - ½')).toBeInTheDocument();
    });

    it('displays forfeit result', () => {
      const games = [createGame({ result: '1-0F' })];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="completed"
        />,
        { wrapper: createWrapper() }
      );

      expect(screen.getByText('1 - 0 (k)')).toBeInTheDocument();
    });

    it('displays dash when no result', () => {
      const games = [createGame({ result: undefined })];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
        />,
        { wrapper: createWrapper() }
      );

      expect(screen.getByText('—')).toBeInTheDocument();
    });

    it('highlights winner (white wins)', () => {
      const games = [createGame({ result: '1-0' })];
      const { container } = render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="completed"
        />,
        { wrapper: createWrapper() }
      );

      // The winner's name should have font-semibold class
      const whiteName = screen.getByText('White Player');
      expect(whiteName).toHaveClass('text-green-700');
    });

    it('highlights winner (black wins)', () => {
      const games = [createGame({ result: '0-1' })];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="completed"
        />,
        { wrapper: createWrapper() }
      );

      const blackName = screen.getByText('Black Player');
      expect(blackName).toHaveClass('text-green-700');
    });
  });

  describe('bye games', () => {
    it('displays "spielfrei" for bye games', () => {
      const games = [
        createGame({
          blackParticipantId: undefined,
          blackName: undefined,
          result: 'BYE',
        }),
      ];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
        />,
        { wrapper: createWrapper() }
      );

      expect(screen.getByText('spielfrei')).toBeInTheDocument();
    });

    it('displays BYE result', () => {
      const games = [
        createGame({
          blackParticipantId: undefined,
          result: 'BYE',
        }),
      ];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="completed"
        />,
        { wrapper: createWrapper() }
      );

      expect(screen.getByText('BYE')).toBeInTheDocument();
    });
  });

  describe('edit functionality', () => {
    it('shows edit button when canEdit is true and round is ongoing', () => {
      const games = [createGame()];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
          canEdit={true}
        />,
        { wrapper: createWrapper() }
      );

      // Edit button should be visible (Edit2 icon button)
      expect(screen.getByRole('button')).toBeInTheDocument();
    });

    it('hides edit button when canEdit is false', () => {
      const games = [createGame()];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
          canEdit={false}
        />,
        { wrapper: createWrapper() }
      );

      // No buttons should be present
      expect(screen.queryByRole('button')).not.toBeInTheDocument();
    });

    it('hides edit button when round is not ongoing', () => {
      const games = [createGame()];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="completed"
          canEdit={true}
        />,
        { wrapper: createWrapper() }
      );

      // No buttons should be present
      expect(screen.queryByRole('button')).not.toBeInTheDocument();
    });

    it('does not show edit button for bye games', () => {
      const games = [
        createGame({
          blackParticipantId: undefined,
          result: 'BYE',
        }),
      ];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
          canEdit={true}
        />,
        { wrapper: createWrapper() }
      );

      // No edit button for bye games
      expect(screen.queryByRole('button')).not.toBeInTheDocument();
    });

    it('opens dialog when edit button is clicked', async () => {
      const user = userEvent.setup();
      const games = [createGame()];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
          canEdit={true}
        />,
        { wrapper: createWrapper() }
      );

      const editButton = screen.getByRole('button');
      await user.click(editButton);

      expect(screen.getByText('Ergebnis eintragen')).toBeInTheDocument();
      expect(screen.getByText(/Brett 1:/)).toBeInTheDocument();
    });

    it('shows result select in dialog', async () => {
      const user = userEvent.setup();
      const games = [createGame()];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
          canEdit={true}
        />,
        { wrapper: createWrapper() }
      );

      await user.click(screen.getByRole('button'));

      // Check that the select trigger is present with placeholder
      const selectTrigger = screen.getByRole('combobox');
      expect(selectTrigger).toBeInTheDocument();
      expect(screen.getByText('Ergebnis wählen')).toBeInTheDocument();
    });

    it('can close dialog with cancel button', async () => {
      const user = userEvent.setup();
      const games = [createGame()];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
          canEdit={true}
        />,
        { wrapper: createWrapper() }
      );

      await user.click(screen.getByRole('button'));
      expect(screen.getByText('Ergebnis eintragen')).toBeInTheDocument();

      await user.click(screen.getByText('Abbrechen'));
      expect(screen.queryByText('Ergebnis eintragen')).not.toBeInTheDocument();
    });
  });

  describe('multiple games', () => {
    it('renders all games in order', () => {
      const games = createPairings(5);
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
        />,
        { wrapper: createWrapper() }
      );

      expect(screen.getByText('White 1')).toBeInTheDocument();
      expect(screen.getByText('White 2')).toBeInTheDocument();
      expect(screen.getByText('White 3')).toBeInTheDocument();
      expect(screen.getByText('White 4')).toBeInTheDocument();
      expect(screen.getByText('White 5')).toBeInTheDocument();
    });

    it('renders mixed results correctly', () => {
      const games = [
        createGame({ id: 'g1', boardNumber: 1, result: '1-0' }),
        createGame({ id: 'g2', boardNumber: 2, result: '0-1', whiteName: 'Player A', blackName: 'Player B' }),
        createGame({ id: 'g3', boardNumber: 3, result: '1/2-1/2', whiteName: 'Player C', blackName: 'Player D' }),
        createGame({ id: 'g4', boardNumber: 4, whiteName: 'Player E', blackName: 'Player F' }),
      ];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
        />,
        { wrapper: createWrapper() }
      );

      expect(screen.getByText('1 - 0')).toBeInTheDocument();
      expect(screen.getByText('0 - 1')).toBeInTheDocument();
      expect(screen.getByText('½ - ½')).toBeInTheDocument();
      expect(screen.getByText('—')).toBeInTheDocument();
    });
  });

  describe('row styling', () => {
    it('applies background to completed games', () => {
      const games = [createGame({ result: '1-0' })];
      const { container } = render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
        />,
        { wrapper: createWrapper() }
      );

      const row = container.querySelector('tbody tr');
      expect(row).toHaveClass('bg-muted/30');
    });

    it('does not apply background to pending games', () => {
      const games = [createGame({ result: undefined })];
      const { container } = render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
        />,
        { wrapper: createWrapper() }
      );

      const row = container.querySelector('tbody tr');
      expect(row).not.toHaveClass('bg-muted/30');
    });
  });

  describe('fallback player names', () => {
    it('shows fallback name when whiteName is missing', () => {
      const games = [
        createGame({
          whiteName: undefined,
          whiteParticipantId: 'p123',
        }),
      ];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
        />,
        { wrapper: createWrapper() }
      );

      expect(screen.getByText('Spieler p123')).toBeInTheDocument();
    });

    it('shows fallback name when blackName is missing', () => {
      const games = [
        createGame({
          blackName: undefined,
          blackParticipantId: 'p456',
        }),
      ];
      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={1}
          roundStatus="ongoing"
        />,
        { wrapper: createWrapper() }
      );

      expect(screen.getByText('Spieler p456')).toBeInTheDocument();
    });
  });

  describe('complete scenario', () => {
    it('renders a realistic round of pairings', () => {
      const games: Game[] = [
        {
          id: 'g1',
          boardNumber: 1,
          whiteParticipantId: 'p1',
          blackParticipantId: 'p2',
          whiteName: 'Magnus Carlsen',
          blackName: 'Fabiano Caruana',
          whiteRating: 2830,
          blackRating: 2805,
          result: '1-0',
          status: 'completed',
        },
        {
          id: 'g2',
          boardNumber: 2,
          whiteParticipantId: 'p3',
          blackParticipantId: 'p4',
          whiteName: 'Ding Liren',
          blackName: 'Ian Nepomniachtchi',
          whiteRating: 2790,
          blackRating: 2775,
          result: '1/2-1/2',
          status: 'completed',
        },
        {
          id: 'g3',
          boardNumber: 3,
          whiteParticipantId: 'p5',
          blackParticipantId: 'p6',
          whiteName: 'Hikaru Nakamura',
          blackName: 'Wesley So',
          whiteRating: 2780,
          blackRating: 2765,
          status: 'ongoing',
        },
      ];

      render(
        <PairingsView
          tournamentId="t1"
          games={games}
          roundNumber={5}
          roundStatus="ongoing"
          canEdit={true}
        />,
        { wrapper: createWrapper() }
      );

      // All players present
      expect(screen.getByText('Magnus Carlsen')).toBeInTheDocument();
      expect(screen.getByText('Fabiano Caruana')).toBeInTheDocument();
      expect(screen.getByText('Ding Liren')).toBeInTheDocument();
      expect(screen.getByText('Ian Nepomniachtchi')).toBeInTheDocument();
      expect(screen.getByText('Hikaru Nakamura')).toBeInTheDocument();
      expect(screen.getByText('Wesley So')).toBeInTheDocument();

      // Results
      expect(screen.getByText('1 - 0')).toBeInTheDocument();
      expect(screen.getByText('½ - ½')).toBeInTheDocument();
      expect(screen.getByText('—')).toBeInTheDocument();

      // Ratings
      expect(screen.getByText('(2830)')).toBeInTheDocument();
      expect(screen.getByText('(2805)')).toBeInTheDocument();

      // Edit buttons for non-bye games when canEdit is true
      // All 3 games should have edit buttons
      expect(screen.getAllByRole('button').length).toBe(3);
    });
  });
});
