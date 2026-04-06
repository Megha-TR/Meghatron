interface StatusBadgeProps {
  status: string;
}

const statusStyles: Record<string, string> = {
  Running:   'bg-green-900/50 text-green-400 border border-green-700',
  Succeeded: 'bg-blue-900/50 text-blue-400 border border-blue-700',
  Pending:   'bg-yellow-900/50 text-yellow-400 border border-yellow-700',
  Failed:    'bg-red-900/50 text-red-400 border border-red-700',
  Unknown:   'bg-gray-700/50 text-gray-400 border border-gray-600',
  Ready:     'bg-green-900/50 text-green-400 border border-green-700',
  NotReady:  'bg-red-900/50 text-red-400 border border-red-700',
};

const dotStyles: Record<string, string> = {
  Running:   'bg-green-400',
  Succeeded: 'bg-blue-400',
  Pending:   'bg-yellow-400 animate-pulse',
  Failed:    'bg-red-400',
  Unknown:   'bg-gray-400',
  Ready:     'bg-green-400',
  NotReady:  'bg-red-400',
};

export default function StatusBadge({ status }: StatusBadgeProps) {
  const style = statusStyles[status] ?? statusStyles.Unknown;
  const dot   = dotStyles[status]   ?? dotStyles.Unknown;

  return (
    <span className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium ${style}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${dot}`} />
      {status}
    </span>
  );
}
