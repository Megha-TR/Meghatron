export function formatCPU(value?: string): string {
  if (!value) return 'N/A';
  const cores = value.endsWith('m') ? Number(value.slice(0, -1)) / 1000 : Number(value);
  return Number.isFinite(cores) ? `${cores.toLocaleString()} ${cores === 1 ? 'core' : 'cores'}` : value;
}
export function formatMemory(value?: string): string {
  if (!value) return 'N/A';
  const match = value.match(/^(\d+(?:\.\d+)?)(Gi|Mi|Ki|G|M|K|B)?$/);
  if (!match) return value;
  const units: Record<string, number> = { Gi: 2 ** 30, Mi: 2 ** 20, Ki: 2 ** 10, G: 1e9, M: 1e6, K: 1e3, B: 1 };
  const bytes = Number(match[1]) * (units[match[2] || 'Mi']);
  return bytes >= 2 ** 30 ? `${(bytes / 2 ** 30).toFixed(1)} GiB` : `${Math.round(bytes / 2 ** 20)} MiB`;
}
