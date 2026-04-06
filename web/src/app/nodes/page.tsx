'use client';

import { useEffect, useState, useCallback } from 'react';
import StatusBadge from '@/components/StatusBadge';

interface NodeData {
  metadata: { name: string; uid: string };
  status: { ready: boolean; phase: string };
  capacity: { cpu: string; memory: string };
  lastHeartbeat: string;
}

function formatHeartbeat(ts: string): string {
  if (!ts) return 'N/A';
  try {
    const date = new Date(ts);
    if (isNaN(date.getTime())) return ts;
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffSec = Math.floor(diffMs / 1000);
    if (diffSec < 60) return `${diffSec}s ago`;
    const diffMin = Math.floor(diffSec / 60);
    if (diffMin < 60) return `${diffMin}m ago`;
    const diffHr = Math.floor(diffMin / 60);
    if (diffHr < 24) return `${diffHr}h ago`;
    return date.toLocaleDateString();
  } catch {
    return ts;
  }
}

export default function NodesPage() {
  const [nodes, setNodes] = useState<NodeData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);

  const fetchNodes = useCallback(async () => {
    try {
      const res = await fetch('/api/v1/nodes');
      if (!res.ok) throw new Error(`API error: ${res.status}`);
      const data = await res.json();
      setNodes(Array.isArray(data) ? data : []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch nodes');
    } finally {
      setLoading(false);
      setLastRefresh(new Date());
    }
  }, []);

  useEffect(() => {
    fetchNodes();
    const interval = setInterval(fetchNodes, 5000);
    return () => clearInterval(interval);
  }, [fetchNodes]);

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-white">Nodes</h1>
          <p className="text-gray-400 mt-1 text-sm">
            Cluster worker and control plane nodes
          </p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 bg-gray-800 rounded-lg px-3 py-2 border border-gray-700">
            <div className="w-2 h-2 rounded-full bg-green-400 animate-pulse" />
            <span className="text-xs text-gray-400">Auto-refresh 5s</span>
          </div>
          <button
            onClick={fetchNodes}
            className="flex items-center gap-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            Refresh
          </button>
        </div>
      </div>

      {/* Error banner */}
      {error && (
        <div className="mb-6 bg-red-900/40 border border-red-700 rounded-lg px-4 py-3 flex items-center gap-3">
          <svg className="w-5 h-5 text-red-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
              d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span className="text-red-300 text-sm">{error}</span>
        </div>
      )}

      {/* Summary strip */}
      {!loading && nodes.length > 0 && (
        <div className="grid grid-cols-3 gap-4 mb-6">
          {[
            { label: 'Total', value: nodes.length, color: 'text-white' },
            { label: 'Ready', value: nodes.filter(n => n.status?.ready).length, color: 'text-green-400' },
            { label: 'Not Ready', value: nodes.filter(n => !n.status?.ready).length, color: 'text-red-400' },
          ].map(({ label, value, color }) => (
            <div key={label} className="bg-gray-800 rounded-lg border border-gray-700 px-4 py-3 text-center">
              <p className={`text-2xl font-bold ${color}`}>{value}</p>
              <p className="text-xs text-gray-400 mt-0.5">{label}</p>
            </div>
          ))}
        </div>
      )}

      {/* Table */}
      <div className="bg-gray-800 rounded-xl border border-gray-700 overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-700 bg-gray-900/50">
              <th className="text-left px-6 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Name</th>
              <th className="text-left px-6 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Status</th>
              <th className="text-left px-6 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Phase</th>
              <th className="text-left px-6 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">CPU</th>
              <th className="text-left px-6 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Memory</th>
              <th className="text-left px-6 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Last Heartbeat</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={6} className="text-center py-12">
                  <div className="flex flex-col items-center gap-3">
                    <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
                    <span className="text-gray-500 text-sm">Loading nodes...</span>
                  </div>
                </td>
              </tr>
            ) : nodes.length === 0 ? (
              <tr>
                <td colSpan={6} className="text-center py-12">
                  <div className="flex flex-col items-center gap-3">
                    <svg className="w-12 h-12 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
                        d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
                    </svg>
                    <p className="text-gray-500">No nodes found</p>
                  </div>
                </td>
              </tr>
            ) : (
              nodes.map((node, idx) => (
                <tr
                  key={node.metadata?.uid || node.metadata?.name}
                  className={`border-b border-gray-700 last:border-0 hover:bg-gray-700/40 transition-colors ${
                    idx % 2 === 1 ? 'bg-gray-900/20' : ''
                  }`}
                >
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-2">
                      <div className={`w-2 h-2 rounded-full flex-shrink-0 ${node.status?.ready ? 'bg-green-400' : 'bg-red-400'}`} />
                      <span className="font-mono text-gray-200 font-medium">{node.metadata?.name}</span>
                    </div>
                    {node.metadata?.uid && (
                      <p className="text-xs text-gray-600 font-mono mt-0.5 ml-4">{node.metadata.uid.slice(0, 8)}...</p>
                    )}
                  </td>
                  <td className="px-6 py-4">
                    <StatusBadge status={node.status?.ready ? 'Ready' : 'NotReady'} />
                  </td>
                  <td className="px-6 py-4">
                    <span className="text-gray-400 text-sm">{node.status?.phase || 'Unknown'}</span>
                  </td>
                  <td className="px-6 py-4">
                    <span className="font-mono text-gray-300 bg-gray-900/50 px-2 py-0.5 rounded text-xs">
                      {node.capacity?.cpu || 'N/A'}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <span className="font-mono text-gray-300 bg-gray-900/50 px-2 py-0.5 rounded text-xs">
                      {node.capacity?.memory || 'N/A'}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <span className="text-gray-400 text-sm" title={node.lastHeartbeat}>
                      {formatHeartbeat(node.lastHeartbeat)}
                    </span>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Footer */}
      {!loading && nodes.length > 0 && (
        <p className="text-xs text-gray-600 mt-3 text-right">
          {nodes.length} node{nodes.length !== 1 ? 's' : ''} &middot; Last refreshed {lastRefresh.toLocaleTimeString()}
        </p>
      )}
    </div>
  );
}
