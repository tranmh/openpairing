'use client';

import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';

interface U12BadgeProps {
  age?: number;
  className?: string;
}

/**
 * Badge component for U12 players (under 12 years old)
 * Displays a yellow warning badge to highlight young players
 */
export function U12Badge({ age, className }: U12BadgeProps) {
  return (
    <Badge
      variant="warning"
      className={cn('text-xs', className)}
      title={age ? `${age} Jahre alt` : 'Unter 12 Jahre'}
    >
      U12
    </Badge>
  );
}
