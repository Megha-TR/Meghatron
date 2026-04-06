'use client';

import { useEffect, useState, useCallback } from 'react';
import Link from 'next/link';

interface NodeData {
  metadata: { name: string; uid: string };
  status: { ready: boolean; phase: string };
  capacity: { cpu: string; memory: string };
  lastHeartbeat: string;
}

interface PodData {
  metadata: { name: string; namespace: string };
  spec: { nodeName: string; containers: { name: string; image: string }[] };
  status: { phase: string };
}

interface DeploymentData {
  metadata: { name: string; namespace: string };
  spec: { replicas: number };
  status: { replicas: number; availableReplicas: number };
}

interface StatCardProps {
  title: string;
  value: number | string;
  subtitle: string;
  href: string;
  color: string;
  icon: React.ReactNode;
}

function StatCard({ title, value, subtitle, href, color, icon }: StatCardProps) {
  return (
    <Link href={href}>
      <div className="bg-gray-800 rounded-xl p-6 border border-gray-700 hover:border-indigo-500 transition-colors duration-200 cursor-pointer group">
        <div className="flex items-center justify-between mb-4">
          <div className={`p-3 rounded-lg ${color}`}>
            {icon}
          </div>
          <svg className="w-4 h-4 text-gray-600 group-hover:text-indigo-400 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        </div>
        <div>
          <p className="text-3xl font-bold text-white mb-1">{value}</p>
          <p className="text-sm font-medium text-gray-300">{title}</p>
          <p className="text-xs text-gray-500 mt-1">{subtitle}</p>
        </div>
      </div>
    </Link>
  );
}

export default function OverviewPage() {
  const [nodes, setNodes] = useState<NodeData[]>([]);
  const [pods, setPods] = useState<PodData[]>([]);
  const [deployments, setDeployments] = useState<DeploymentData[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);

  const fetchAll = useCallback(async () => {
    try {
      const [nodesRes, podsRes, deploymentsRes] = await Promise.all([
        fetch('/api/v1/nodes'),
        fetch('/api/v1/pods/default'),
        fetch('/api/v1/deployments/default'),
      ]);

      const [nodesData, podsData, deploymentsData] = await Promise.all([
        nodesRes.ok ? nodesRes.json() : [],
        podsRes.ok ? podsRes.json() : [],
        deploymentsRes.ok ? deploymentsRes.json() : [],
      ]);

      setNodes(Array.isArray(nodesData) ? nodesData : []);
      setPods(Array.isArray(podsData) ? podsData : []);
      setDeployments(Array.isArray(deploymentsData) ? deploymentsData : []);
      setError(null);
      setLastRefresh(new Date());
    } catch (err) {
      setError('Failed to connect to API at localhost:8080');
    }
  }, []);

  useEffect(() => {
    fetchAll();
    const interval = setInterval(fetchAll, 5000);
    return () => clearInterval(interval);
  }, [fetchAll]);

  const readyNodes = nodes.filter((n) => n.status?.ready).length;
  const runningPods = pods.filter((p) => p.status?.phase === 'Running').length;
  const pendingPods = pods.filter((p) => p.status?.phase === 'Pending').length;
  const failedPods = pods.filter((p) => p.status?.phase === 'Failed').length;
  const totalReplicas = deployments.reduce((sum, d) => sum + (d.spec?.replicas || 0), 0);
  const availableReplicas = deployments.reduce((sum, d) => sum + (d.status?.availableReplicas || 0), 0);

  const podPhases: Record<string, number> = {};
  pods.forEach((p) => {
    const phase = p.status?.phase || 'Unknown';
    podPhases[phase] = (podPhases[phase] || 0) + 1;
  });

  return (
    <div>
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-white">Overview</h1>
        <p className="text-gray-400 mt-1 text-sm">
          Cluster status at a glance &mdash; auto-refreshing every 5 seconds
        </p>
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

      {/* Stat cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4 mb-8">
        <StatCard
          title="Nodes"
          value={nodes.length}
          subtitle={`${readyNodes} ready / ${nodes.length - readyNodes} not ready`}
          href="/nodes"
          color="bg-blue-600/20"
          icon={
            <svg className="w-6 h-6 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
            </svg>
          }
        />
        <StatCard
          title="Pods"
          value={pods.length}
          subtitle={`${runningPods} running / ${pendingPods} pending / ${failedPods} failed`}
          href="/pods"
          color="bg-purple-600/20"
          icon={
            <svg className="w-6 h-6 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
          }
        />
        <StatCard
          title="Deployments"
          value={deployments.length}
          subtitle={`${availableReplicas}/${totalReplicas} replicas available`}
          href="/deployments"
          color="bg-green-600/20"
          icon={
            <svg className="w-6 h-6 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
            </svg>
          }
        />
        <StatCard
          title="Cluster Health"
          value={error ? 'Offline' : nodes.length === 0 ? 'Empty' : readyNodes === nodes.length ? 'Healthy' : 'Degraded'}
          subtitle={error ? 'Cannot reach API' : `${readyNodes}/${nodes.length} nodes ready`}
          href="/nodes"
          color={error ? 'bg-red-600/20' : readyNodes === nodes.length && nodes.length > 0 ? 'bg-green-600/20' : 'bg-yellow-600/20'}
          icon={
            <svg className={`w-6 h-6 ${error ? 'text-red-400' : readyNodes === nodes.length && nodes.length > 0 ? 'text-green-400' : 'text-yellow-400'}`}
              fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          }
        />
      </div>

      {/* Secondary info panels */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        {/* Pod phase breakdown */}
        <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
          <h2 className="text-base font-semibold text-white mb-4">Pod Phase Distribution</h2>
          {pods.length === 0 ? (
            <p className="text-gray-500 text-sm">No pods found.</p>
          ) : (
            <div className="space-y-3">
              {Object.entries(podPhases).map(([phase, count]) => {
                const pct = Math.round((count / pods.length) * 100);
                const barColor =
                  phase === 'Running' ? 'bg-green-500' :
                  phase === 'Pending' ? 'bg-yellow-500' :
                  phase === 'Failed' ? 'bg-red-500' :
                  phase === 'Succeeded' ? 'bg-blue-500' : 'bg-gray-500';
                return (
                  <div key={phase}>
                    <div className="flex justify-between text-sm mb-1">
                      <span className="text-gray-300">{phase}</span>
                      <span className="text-gray-400">{count} ({pct}%)</span>
                    </div>
                    <div className="w-full bg-gray-700 rounded-full h-2">
                      <div className={`h-2 rounded-full ${barColor} transition-all duration-500`}
                        style={{ width: `${pct}%` }} />
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Recent nodes */}
        <div className="bg-gray-800 rounded-xl border border-gray-700 p-6">
          <h2 className="text-base font-semibold text-white mb-4">Node Status</h2>
          {nodes.length === 0 ? (
            <p className="text-gray-500 text-sm">No nodes found.</p>
          ) : (
            <div className="space-y-2">
              {nodes.slice(0, 6).map((node) => (
                <div key={node.metadata?.uid || node.metadata?.name}
                  className="flex items-center justify-between py-2 border-b border-gray-700 last:border-0">
                  <div className="flex items-center gap-2">
                    <div className={`w-2 h-2 rounded-full ${node.status?.ready ? 'bg-green-400' : 'bg-red-400'}`} />
                    <span className="text-sm text-gray-300 font-mono">{node.metadata?.name}</span>
                  </div>
                  <div className="flex items-center gap-4 text-xs text-gray-500">
                    <span>CPU: {node.capacity?.cpu || 'N/A'}</span>
                    <span>Mem: {node.capacity?.memory || 'N/A'}</span>
                    <span className={node.status?.ready ? 'text-green-400' : 'text-red-400'}>
                      {node.status?.ready ? 'Ready' : 'NotReady'}
                    </span>
                  </div>
                </div>
              ))}
              {nodes.length > 6 && (
                <p className="text-xs text-gray-500 pt-1">+{nodes.length - 6} more nodes</p>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Last refresh */}
      <div className="text-xs text-gray-600 text-right">
        Last refreshed: {lastRefresh?.toLocaleTimeString() ?? ''}
      </div>
    </div>
  );
}
