'use client';

interface RatingBadgeProps {
  type: 'fide' | 'dwz' | 'oesb' | 'rapid' | 'blitz';
  rating?: number;
  size?: 'sm' | 'md' | 'lg';
  showLabel?: boolean;
}

const typeConfig = {
  fide: {
    label: 'FIDE',
    bgColor: 'bg-blue-100 dark:bg-blue-900',
    textColor: 'text-blue-800 dark:text-blue-200',
    borderColor: 'border-blue-200 dark:border-blue-700',
  },
  dwz: {
    label: 'DWZ',
    bgColor: 'bg-red-100 dark:bg-red-900',
    textColor: 'text-red-800 dark:text-red-200',
    borderColor: 'border-red-200 dark:border-red-700',
  },
  oesb: {
    label: 'ÖSB',
    bgColor: 'bg-green-100 dark:bg-green-900',
    textColor: 'text-green-800 dark:text-green-200',
    borderColor: 'border-green-200 dark:border-green-700',
  },
  rapid: {
    label: 'Rapid',
    bgColor: 'bg-purple-100 dark:bg-purple-900',
    textColor: 'text-purple-800 dark:text-purple-200',
    borderColor: 'border-purple-200 dark:border-purple-700',
  },
  blitz: {
    label: 'Blitz',
    bgColor: 'bg-orange-100 dark:bg-orange-900',
    textColor: 'text-orange-800 dark:text-orange-200',
    borderColor: 'border-orange-200 dark:border-orange-700',
  },
};

const sizeClasses = {
  sm: 'text-xs px-1.5 py-0.5',
  md: 'text-sm px-2 py-1',
  lg: 'text-base px-3 py-1.5',
};

export function RatingBadge({
  type,
  rating,
  size = 'md',
  showLabel = true,
}: RatingBadgeProps) {
  const config = typeConfig[type];

  if (!rating || rating === 0) {
    return null;
  }

  return (
    <span
      className={`inline-flex items-center gap-1 font-medium rounded border ${config.bgColor} ${config.textColor} ${config.borderColor} ${sizeClasses[size]}`}
    >
      {showLabel && <span className="opacity-75">{config.label}</span>}
      <span className="font-bold">{rating}</span>
    </span>
  );
}

interface RatingDisplayProps {
  fideStandard?: number;
  fideRapid?: number;
  fideBlitz?: number;
  dwz?: number;
  oesbElo?: number;
  size?: 'sm' | 'md' | 'lg';
  layout?: 'horizontal' | 'vertical';
}

export function RatingDisplay({
  fideStandard,
  fideRapid,
  fideBlitz,
  dwz,
  oesbElo,
  size = 'md',
  layout = 'horizontal',
}: RatingDisplayProps) {
  const hasRatings =
    fideStandard || fideRapid || fideBlitz || dwz || oesbElo;

  if (!hasRatings) {
    return (
      <span className="text-gray-400 dark:text-gray-500 text-sm">
        Keine Wertung
      </span>
    );
  }

  const containerClass =
    layout === 'horizontal' ? 'flex flex-wrap gap-2' : 'flex flex-col gap-1';

  return (
    <div className={containerClass}>
      <RatingBadge type="fide" rating={fideStandard} size={size} />
      <RatingBadge type="rapid" rating={fideRapid} size={size} />
      <RatingBadge type="blitz" rating={fideBlitz} size={size} />
      <RatingBadge type="dwz" rating={dwz} size={size} />
      <RatingBadge type="oesb" rating={oesbElo} size={size} />
    </div>
  );
}
