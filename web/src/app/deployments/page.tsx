'use client';

import { useEffect, useState, useCallback } from 'react';
import { fetchJSON } from '@/lib/api';
import Modal from '@/components/Modal';

interface DeploymentData {
  metadata: { name: string; namespace: string; uid: string; deletionTimestamp?: string };
  spec: { replicas: number; selector: Record<string, string> };
  status: { replicas: number; availableReplicas: number; updatedReplicas: number };
}

interface CreateForm {
  name: string;
  image: string;
  replicas: number;
}

export default function DeploymentsPage() {
  const [deployments, setDeployments] = useState<DeploymentData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState<CreateForm>({ name: '', image: '', replicas: 1 });
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [deletingDep, setDeletingDep] = useState<string | null>(null);
  const [scaling, setScaling] = useState<string | null>(null);
  const [replicaEdits, setReplicaEdits] = useState<Record<string, number>>({});
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);

  const fetchDeployments = useCallback(() => {
    return fetchJSON<DeploymentData[]>('/api/v1/deployments/default')
      .then(data => {
        setDeployments(Array.isArray(data) ? data : []);
        setError(null);
        setLastRefresh(new Date());
      })
      .catch(err => setError(err instanceof Error ? err.message : 'Unable to refresh deployments'))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    fetchDeployments();
    const interval = setInterval(fetchDeployments, 5000);
    return () => clearInterval(interval);
  }, [fetchDeployments]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.name.trim() || !form.image.trim()) return;
    setCreating(true);
    setCreateError(null);
    try {
      const body = {
        metadata: { name: form.name.trim(), namespace: 'default' },
        spec: {
          replicas: form.replicas,
          selector: { app: form.name.trim() },
          template: {
            metadata: { labels: { app: form.name.trim() } },
            spec: {
              containers: [{ name: form.name.trim(), image: form.image.trim() }],
            },
          },
        },
      };
      const res = await fetch('/api/v1/deployments', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || `Server error: ${res.status}`);
      }
      setForm({ name: '', image: '', replicas: 1 });
      setShowCreate(false);
      fetchDeployments();
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : 'Failed to create deployment');
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (name: string) => {
    setDeletingDep(name);
    setConfirmDelete(null);
    try {
      const res = await fetch(`/api/v1/deployments/default/${name}`, { method: 'DELETE' });
      if (!res.ok) throw new Error(`Delete failed: ${res.status}`);
      fetchDeployments();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete deployment');
    } finally {
      setDeletingDep(null);
    }
  };

  const handleScale = async (name: string, replicas: number) => {
    setScaling(name);
    try {
      const res = await fetch(`/api/v1/deployments/default/${encodeURIComponent(name)}/scale`, {
        method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ replicas }),
      });
      if (!res.ok) { const body = await res.json(); throw new Error(body.error || 'Scaling failed'); }
      await fetchDeployments();
    } catch (err) { setError(err instanceof Error ? err.message : 'Scaling failed'); }
    finally { setScaling(null); }
  };

  const replicaHealth = (dep: DeploymentData) => {
    if (dep.metadata.deletionTimestamp) return { label: 'Terminating', cls: 'text-amber-400' };
    const desired = dep.spec?.replicas ?? 0;
    const available = dep.status?.availableReplicas ?? 0;
    if (desired === 0) return { label: 'Scaled to zero', cls: 'text-gray-400' };
    if (available === desired && desired > 0) return { label: 'Healthy', cls: 'text-green-400' };
    if (available === 0) return { label: 'Unavailable', cls: 'text-red-400' };
    return { label: 'Degraded', cls: 'text-yellow-400' };
  };

  return (
    <div>
      {/* Header */}
      <div className="flex flex-wrap gap-4 items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-white">Deployments</h1>
          <p className="text-gray-400 mt-1 text-sm">Manage owned replicas, scaling and container cleanup</p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-2 bg-gray-800 rounded-lg px-3 py-2 border border-gray-700">
            <div className="w-2 h-2 rounded-full bg-green-400 animate-pulse" />
            <span className="text-xs text-gray-400">Auto-refresh 5s</span>
          </div>
          <button
            onClick={fetchDeployments}
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
            Create Deployment
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
      {!loading && deployments.length > 0 && (
        <div className="grid grid-cols-3 gap-4 mb-6">
          {[
            { label: 'Total',      value: deployments.length, color: 'text-white' },
            { label: 'Desired Replicas',
              value: deployments.reduce((s, d) => s + (d.spec?.replicas || 0), 0), color: 'text-indigo-400' },
            { label: 'Available Replicas',
              value: deployments.reduce((s, d) => s + (d.status?.availableReplicas || 0), 0), color: 'text-green-400' },
          ].map(({ label, value, color }) => (
            <div key={label} className="bg-gray-800 rounded-lg border border-gray-700 px-4 py-3 text-center">
              <p className={`text-2xl font-bold ${color}`}>{value}</p>
              <p className="text-xs text-gray-400 mt-0.5">{label}</p>
            </div>
          ))}
        </div>
      )}

      {/* Table */}
      <div className="bg-gray-800 rounded-xl border border-gray-700 overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-700 bg-gray-900/50">
              <th className="text-left px-6 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Name</th>
              <th className="text-left px-6 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Replicas</th>
              <th className="text-left px-6 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Progress</th>
              <th className="text-left px-6 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Health</th>
              <th className="text-right px-6 py-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={5} className="text-center py-12">
                  <div className="flex flex-col items-center gap-3">
                    <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
                    <span className="text-gray-500 text-sm">Loading deployments...</span>
                  </div>
                </td>
              </tr>
            ) : deployments.length === 0 ? (
              <tr>
                <td colSpan={5} className="text-center py-12">
                  <div className="flex flex-col items-center gap-3">
                    <svg className="w-12 h-12 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
                        d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                    </svg>
                    <p className="text-gray-500">No deployments found</p>
                    <button
                      onClick={() => setShowCreate(true)}
                      className="text-indigo-400 hover:text-indigo-300 text-sm font-medium transition-colors"
                    >
                      Create your first deployment →
                    </button>
                  </div>
                </td>
              </tr>
            ) : (
              deployments.map((dep, idx) => {
                const desired = dep.spec?.replicas ?? 0;
                const available = dep.status?.availableReplicas ?? 0;
                const pct = desired > 0 ? Math.round((available / desired) * 100) : 0;
                const health = replicaHealth(dep);
                return (
                  <tr
                    key={dep.metadata?.uid || dep.metadata?.name}
                    className={`border-b border-gray-700 last:border-0 hover:bg-gray-700/40 transition-colors ${idx % 2 === 1 ? 'bg-gray-900/20' : ''}`}
                  >
                    <td className="px-6 py-4">
                      <span className="font-mono text-gray-200 font-medium">{dep.metadata?.name}</span>
                        {!dep.metadata.deletionTimestamp && <div className="flex items-center gap-2 mt-2">
                          <input aria-label={`Desired replicas for ${dep.metadata.name}`} type="number" min={0} max={100}
                            value={replicaEdits[dep.metadata.name] ?? dep.spec.replicas}
                            onChange={e => setReplicaEdits(v => ({ ...v, [dep.metadata.name]: Number(e.target.value) }))}
                            className="w-16 rounded bg-gray-900 border border-gray-600 px-2 py-1 text-xs" />
                          <button onClick={() => handleScale(dep.metadata.name, replicaEdits[dep.metadata.name] ?? dep.spec.replicas)} disabled={scaling === dep.metadata.name}
                            className="text-xs text-indigo-300 disabled:opacity-50">{scaling === dep.metadata.name ? 'Saving…' : 'Scale'}</button>
                        </div>}

                      <p className="text-xs text-gray-500 mt-0.5">{dep.metadata?.namespace}</p>
                    </td>
                    <td className="px-6 py-4">
                      <span className="text-gray-300 font-mono text-sm">
                        {available}<span className="text-gray-500">/{desired}</span>
                      </span>
                    </td>
                    <td className="px-6 py-4 w-40">
                      <div className="w-full bg-gray-700 rounded-full h-2">
                        <div
                          className={`h-2 rounded-full transition-all duration-500 ${pct === 100 ? 'bg-green-500' : pct === 0 ? 'bg-red-500' : 'bg-yellow-500'}`}
                          style={{ width: `${pct}%` }}
                        />
                      </div>
                      <p className="text-xs text-gray-500 mt-1">{pct}% available</p>
                    </td>
                    <td className="px-6 py-4">
                      <span className={`text-sm font-medium ${health.cls}`}>{health.label}</span>
                    </td>
                    <td className="px-6 py-4 text-right">
                      {confirmDelete === dep.metadata?.name ? (
                        <div className="flex items-center justify-end gap-2">
                          <span className="text-xs text-gray-400">Delete?</span>
                          <button
                            onClick={() => handleDelete(dep.metadata.name)}
                            disabled={deletingDep === dep.metadata?.name}
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
                          onClick={() => setConfirmDelete(dep.metadata?.name)}
                          disabled={deletingDep === dep.metadata?.name}
                          className="text-xs text-red-400 hover:text-red-300 hover:bg-red-900/30 px-3 py-1 rounded transition-colors disabled:opacity-50"
                        >
                          {deletingDep === dep.metadata?.name ? 'Deleting...' : 'Delete'}
                        </button>
                      )}
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      {!loading && deployments.length > 0 && (
        <p className="text-xs text-gray-600 mt-3 text-right">
          {deployments.length} deployment{deployments.length !== 1 ? 's' : ''} &middot; Last refreshed {lastRefresh?.toLocaleTimeString() ?? 'Never'}
        </p>
      )}

      {/* Create Deployment Modal */}
      {showCreate && (
        <Modal title="Create Deployment" onClose={() => { setShowCreate(false); setCreateError(null); }}>
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-700">
              <h2 className="text-lg font-semibold text-white">Create Deployment</h2>
              <button onClick={() => { setShowCreate(false); setCreateError(null); }}
                aria-label="Close dialog" className="text-gray-400 hover:text-white transition-colors">
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
                <label htmlFor="deployments-deployment-name" className="block text-sm font-medium text-gray-300 mb-1.5">Deployment Name</label>
                <input id="deployments-deployment-name"
                  type="text"
                  pattern="[a-z0-9]([-a-z0-9]*[a-z0-9])?" maxLength={63}
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                  placeholder="e.g. web-app"
                  className="w-full bg-gray-900 border border-gray-600 rounded-lg px-3 py-2 text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                  required
                />
              </div>
              <div>
                <label htmlFor="deployments-container-image" className="block text-sm font-medium text-gray-300 mb-1.5">Container Image</label>
                <input id="deployments-container-image"
                  type="text"
                  value={form.image}
                  onChange={e => setForm(f => ({ ...f, image: e.target.value }))}
                  placeholder="e.g. nginx:latest"
                  className="w-full bg-gray-900 border border-gray-600 rounded-lg px-3 py-2 text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                  required
                />
              </div>
              <div>
                <label htmlFor="deployment-replicas" className="block text-sm font-medium text-gray-300 mb-1.5">
                  Replicas: <span className="text-indigo-400 font-bold">{form.replicas}</span>
                </label>
                <input
                  id="deployment-replicas" type="range"
                  min={0}
                  max={10}
                  value={form.replicas}
                  onChange={e => setForm(f => ({ ...f, replicas: Number(e.target.value) }))}
                  className="w-full accent-indigo-500"
                />
                <div className="flex justify-between text-xs text-gray-500 mt-1">
                  <span>0</span><span>10</span>
                </div>
              </div>
              <div className="pt-2 flex gap-3">
                <button type="submit" disabled={creating}
                  className="flex-1 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white text-sm font-medium py-2 rounded-lg transition-colors">
                  {creating ? 'Creating...' : 'Create Deployment'}
                </button>
                <button type="button" onClick={() => { setShowCreate(false); setCreateError(null); }}
                  className="flex-1 bg-gray-700 hover:bg-gray-600 text-white text-sm font-medium py-2 rounded-lg transition-colors">
                  Cancel
                </button>
              </div>
            </form>
        </Modal>
      )}
    </div>
  );
}
