'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Crown, MoreVertical, Plus, Search, UserMinus, ArrowUpDown } from 'lucide-react';

export interface TeamMember {
  id: string;
  teamId: string;
  playerId: string;
  boardNumber?: number;
  isCaptain: boolean;
  isReserve: boolean;
  validFrom?: string;
  validUntil?: string;
  createdAt: string;
  playerName: string;
  playerRating: number;
  playerTitle?: string;
}

interface RosterManagerProps {
  teamId: string;
  members: TeamMember[];
  onAddMember?: () => void;
  onRemoveMember?: (memberId: string) => void;
  onSetCaptain?: (memberId: string) => void;
  onSetBoardNumber?: (memberId: string, boardNumber: number | null) => void;
  onToggleReserve?: (memberId: string) => void;
  readOnly?: boolean;
}

type SortField = 'boardNumber' | 'rating' | 'name';
type SortDirection = 'asc' | 'desc';

export function RosterManager({
  members,
  onAddMember,
  onRemoveMember,
  onSetCaptain,
  onSetBoardNumber,
  onToggleReserve,
  readOnly = false,
}: RosterManagerProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [sortField, setSortField] = useState<SortField>('boardNumber');
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc');

  const filteredMembers = members.filter((member) =>
    member.playerName.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const sortedMembers = [...filteredMembers].sort((a, b) => {
    let comparison = 0;
    switch (sortField) {
      case 'boardNumber':
        const aBoard = a.boardNumber ?? 999;
        const bBoard = b.boardNumber ?? 999;
        comparison = aBoard - bBoard;
        break;
      case 'rating':
        comparison = b.playerRating - a.playerRating;
        break;
      case 'name':
        comparison = a.playerName.localeCompare(b.playerName);
        break;
    }
    return sortDirection === 'asc' ? comparison : -comparison;
  });

  const toggleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('asc');
    }
  };

  const regularMembers = sortedMembers.filter((m) => !m.isReserve);
  const reserveMembers = sortedMembers.filter((m) => m.isReserve);

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle>Team Roster</CardTitle>
          {!readOnly && onAddMember && (
            <Button onClick={onAddMember} size="sm">
              <Plus className="h-4 w-4 mr-1" />
              Add Player
            </Button>
          )}
        </div>
        <div className="relative">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search players..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-8"
          />
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-6">
          <RosterTable
            title="Active Players"
            members={regularMembers}
            sortField={sortField}
            sortDirection={sortDirection}
            onSort={toggleSort}
            onRemoveMember={onRemoveMember}
            onSetCaptain={onSetCaptain}
            onSetBoardNumber={onSetBoardNumber}
            onToggleReserve={onToggleReserve}
            readOnly={readOnly}
          />
          {reserveMembers.length > 0 && (
            <RosterTable
              title="Reserve Players"
              members={reserveMembers}
              sortField={sortField}
              sortDirection={sortDirection}
              onSort={toggleSort}
              onRemoveMember={onRemoveMember}
              onSetCaptain={onSetCaptain}
              onSetBoardNumber={onSetBoardNumber}
              onToggleReserve={onToggleReserve}
              readOnly={readOnly}
            />
          )}
        </div>
      </CardContent>
    </Card>
  );
}

interface RosterTableProps {
  title: string;
  members: TeamMember[];
  sortField: SortField;
  sortDirection: SortDirection;
  onSort: (field: SortField) => void;
  onRemoveMember?: (memberId: string) => void;
  onSetCaptain?: (memberId: string) => void;
  onSetBoardNumber?: (memberId: string, boardNumber: number | null) => void;
  onToggleReserve?: (memberId: string) => void;
  readOnly: boolean;
}

function RosterTable({
  title,
  members,
  sortField,
  sortDirection,
  onSort,
  onRemoveMember,
  onSetCaptain,
  onSetBoardNumber,
  onToggleReserve,
  readOnly,
}: RosterTableProps) {
  const SortButton = ({ field, children }: { field: SortField; children: React.ReactNode }) => (
    <Button
      variant="ghost"
      size="sm"
      className="-ml-3 h-8"
      onClick={() => onSort(field)}
    >
      {children}
      <ArrowUpDown className="ml-1 h-3 w-3" />
    </Button>
  );

  return (
    <div>
      <h4 className="text-sm font-medium mb-2">{title} ({members.length})</h4>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-16">
              <SortButton field="boardNumber">Board</SortButton>
            </TableHead>
            <TableHead>
              <SortButton field="name">Player</SortButton>
            </TableHead>
            <TableHead className="w-24">
              <SortButton field="rating">Rating</SortButton>
            </TableHead>
            <TableHead className="w-24">Status</TableHead>
            {!readOnly && <TableHead className="w-12"></TableHead>}
          </TableRow>
        </TableHeader>
        <TableBody>
          {members.map((member) => (
            <TableRow key={member.id}>
              <TableCell className="font-medium">
                {member.boardNumber ?? '-'}
              </TableCell>
              <TableCell>
                <div className="flex items-center gap-2">
                  {member.playerTitle && (
                    <Badge variant="outline" className="text-xs">
                      {member.playerTitle}
                    </Badge>
                  )}
                  {member.playerName}
                  {member.isCaptain && (
                    <Crown className="h-4 w-4 text-yellow-500" />
                  )}
                </div>
              </TableCell>
              <TableCell>{member.playerRating}</TableCell>
              <TableCell>
                {member.isReserve ? (
                  <Badge variant="secondary">Reserve</Badge>
                ) : (
                  <Badge variant="default">Active</Badge>
                )}
              </TableCell>
              {!readOnly && (
                <TableCell>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="icon" className="h-8 w-8">
                        <MoreVertical className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      {onSetCaptain && !member.isCaptain && (
                        <DropdownMenuItem onClick={() => onSetCaptain(member.id)}>
                          <Crown className="h-4 w-4 mr-2" />
                          Set as Captain
                        </DropdownMenuItem>
                      )}
                      {onToggleReserve && (
                        <DropdownMenuItem onClick={() => onToggleReserve(member.id)}>
                          {member.isReserve ? 'Move to Active' : 'Move to Reserve'}
                        </DropdownMenuItem>
                      )}
                      {onSetBoardNumber && (
                        <DropdownMenuItem
                          onClick={() => {
                            const board = prompt('Enter board number (or leave empty to clear):');
                            if (board !== null) {
                              onSetBoardNumber(member.id, board ? parseInt(board, 10) : null);
                            }
                          }}
                        >
                          Set Board Number
                        </DropdownMenuItem>
                      )}
                      {onRemoveMember && (
                        <DropdownMenuItem
                          onClick={() => onRemoveMember(member.id)}
                          className="text-destructive"
                        >
                          <UserMinus className="h-4 w-4 mr-2" />
                          Remove from Team
                        </DropdownMenuItem>
                      )}
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              )}
            </TableRow>
          ))}
          {members.length === 0 && (
            <TableRow>
              <TableCell colSpan={readOnly ? 4 : 5} className="text-center text-muted-foreground">
                No players
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}
