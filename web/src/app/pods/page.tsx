'use client';

import { useEffect, useState, useCallback } from 'react';
import StatusBadge from '@/components/StatusBadge';

interface PodData {
  metadata: { name: string; namespace: string; uid: string };
  spec: { nodeName: string; containers: { name: string; image: string }[] };
  status: { phase: string };
}

interface CreateForm {
  name: string;
  image: string;
}

export default function PodsPage() {
  const [pods, setPods] = useState<PodData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState<CreateForm>({ name: '', image: '' });
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [deletingPod, setDeletingPod] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);

  const fetchPods = useCallback(async () => {
    try {
      const res = await fetch('/api/v1/pods/default');
      if (!res.ok) throw new Error(`API error: ${res.status}`);
      const data = await res.json();
      setPods(Array.isArray(data) ? data : []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch pods');
    } finally {
      setLoading(false);
      setLastRefresh(new Date());
    }
  }, []);

  useEffect(() => {
    fetchPods();
    const interval = setInterval(fetchPods, 5000);
    return () => clearInterval(interval);
  }, [fetchPods]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.name.trim() || !form.image.trim()) return;
    setCreating(true);
    setCreateError(null);
    try {
      const body = {
        metadata: { name: form.name.trim(), namespace: 'default' },
        spec: {
          containers: [{ name: form.name.trim(), image: form.image.trim() }],
        },
      };
      const res = await fetch('/api/v1/pods', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || `Server error: ${res.status}`);
      }
      setForm({ name: '', image: '' });
      setShowCreate(false);
      fetchPods();
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : 'Failed to create pod');
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (name: string) => {
    setDeletingPod(name);
    setConfirmDelete(null);
    try {
      const res = await fetch(`/api/v1/pods/default/${name}`, { method: 'DELETE' });
      if (!res.ok) throw new Error(`Delete failed: ${res.status}`);
      fetchPods();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete pod');
    } finally {
      setDeletingPod(null);
    }
  };

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-white">Pods</h1>
          <p className="text-gray-400 mt-1 text-sm">Running containers in the default namespace</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 bg-gray-800 rounded-lg px-3 py-2 border border-gray-700">
            <div className="w-2 h-2 rounded-full bg-green-400 animate-pulse" />
            <span className="text-xs text-gray-400">Auto-refresh 5s</span>
          </div>
          <button
            onClick={fetchPods}
            className="flex items-center gap-2 bg-gray-700 hover:bg-gray-600 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            Refresh
          </button>
          <button
            onClick={() => { setShowCreate(true); setCreateError(null); }}
            className="flex items-center gap-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Create Pod
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
      {!loading && pods.length > 0 && (
        <div className="grid grid-cols-4 gap-4 mb-6">
          {[
            { label: 'Total',     value: pods.length,                                             color: 'text-white' },
            { label: 'Running',   value: pods.filter(p => p.status?.phase === 'Running').length,  color: 'text-green-400' },
            { label: 'Pending',   value: pods.filter(p => p.status?.phase === 'Pending').length,  color: 'text-yellow-400' },
            { label: 'Failed',    value: pods.filter(p => p.status?.phase === 'Failed').length,   color: 'text-red-400' },
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
              <th className="text-left px-6 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Image</th>
              <th className="text-left px-6 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Phase</th>
              <th className="text-left px-6 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Node</th>
              <th className="text-right px-6 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={5} className="text-center py-12">
                  <div className="flex flex-col items-center gap-3">
                    <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
                    <span className="text-gray-500 text-sm">Loading pods...</span>
                  </div>
                </td>
              </tr>
            ) : pods.length === 0 ? (
              <tr>
                <td colSpan={5} className="text-center py-12">
                  <div className="flex flex-col items-center gap-3">
                    <svg className="w-12 h-12 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
                        d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                    </svg>
                    <p className="text-gray-500">No pods found</p>
                    <button
                      onClick={() => setShowCreate(true)}
                      className="text-indigo-400 hover:text-indigo-300 text-sm font-medium transition-colors"
                    >
                      Create your first pod →
                    </button>
                  </div>
                </td>
              </tr>
            ) : (
              pods.map((pod, idx) => (
                <tr
                  key={pod.metadata?.uid || pod.metadata?.name}
                  className={`border-b border-gray-700 last:border-0 hover:bg-gray-700/40 transition-colors ${idx % 2 === 1 ? 'bg-gray-900/20' : ''}`}
                >
                  <td className="px-6 py-4">
                    <span className="font-mono text-gray-200 font-medium">{pod.metadata?.name}</span>
                    <p className="text-xs text-gray-500 mt-0.5">{pod.metadata?.namespace}</p>
                  </td>
                  <td className="px-6 py-4">
                    <span className="font-mono text-gray-300 bg-gray-900/50 px-2 py-0.5 rounded text-xs">
                      {pod.spec?.containers?.[0]?.image || 'N/A'}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <StatusBadge status={pod.status?.phase || 'Unknown'} />
                  </td>
                  <td className="px-6 py-4">
                    <span className="text-gray-400 text-sm font-mono">{pod.spec?.nodeName || '—'}</span>
                  </td>
                  <td className="px-6 py-4 text-right">
                    {confirmDelete === pod.metadata?.name ? (
                      <div className="flex items-center justify-end gap-2">
                        <span className="text-xs text-gray-400">Delete?</span>
                        <button
                          onClick={() => handleDelete(pod.metadata.name)}
                          disabled={deletingPod === pod.metadata?.name}
                          className="text-xs bg-red-600 hover:bg-red-700 text-white px-3 py-1 rounded transition-colors disabled:opacity-50"
                        >
                          Yes
                        </button>
                        <button
                          onClick={() => setConfirmDelete(null)}
                          className="text-xs bg-gray-600 hover:bg-gray-500 text-white px-3 py-1 rounded transition-colors"
                        >
                          No
                        </button>
                      </div>
                    ) : (
                      <button
                        onClick={() => setConfirmDelete(pod.metadata?.name)}
                        disabled={deletingPod === pod.metadata?.name}
                        className="text-xs text-red-400 hover:text-red-300 hover:bg-red-900/30 px-3 py-1 rounded transition-colors disabled:opacity-50"
                      >
                        {deletingPod === pod.metadata?.name ? 'Deleting...' : 'Delete'}
                      </button>
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {!loading && pods.length > 0 && (
        <p className="text-xs text-gray-600 mt-3 text-right">
          {pods.length} pod{pods.length !== 1 ? 's' : ''} &middot; Last refreshed {lastRefresh.toLocaleTimeString()}
        </p>
      )}

      {/* Create Pod Modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="bg-gray-800 rounded-xl border border-gray-700 w-full max-w-md shadow-2xl">
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-700">
              <h2 className="text-lg font-semibold text-white">Create Pod</h2>
              <button onClick={() => { setShowCreate(false); setCreateError(null); }}
                className="text-gray-400 hover:text-white transition-colors">
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <form onSubmit={handleCreate} className="px-6 py-5 space-y-4">
              {createError && (
                <div className="bg-red-900/40 border border-red-700 rounded-lg px-3 py-2 text-red-300 text-sm">
                  {createError}
                </div>
              )}
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1.5">Pod Name</label>
                <input
                  type="text"
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                  placeholder="e.g. my-nginx"
                  className="w-full bg-gray-900 border border-gray-600 rounded-lg px-3 py-2 text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1.5">Container Image</label>
                <input
                  type="text"
                  value={form.image}
                  onChange={e => setForm(f => ({ ...f, image: e.target.value }))}
                  placeholder="e.g. nginx:latest"
                  className="w-full bg-gray-900 border border-gray-600 rounded-lg px-3 py-2 text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                  required
                />
              </div>
              <div className="pt-2 flex gap-3">
                <button type="submit" disabled={creating}
                  className="flex-1 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white text-sm font-medium py-2 rounded-lg transition-colors">
                  {creating ? 'Creating...' : 'Create Pod'}
                </button>
                <button type="button" onClick={() => { setShowCreate(false); setCreateError(null); }}
                  className="flex-1 bg-gray-700 hover:bg-gray-600 text-white text-sm font-medium py-2 rounded-lg transition-colors">
                  Cancel
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
