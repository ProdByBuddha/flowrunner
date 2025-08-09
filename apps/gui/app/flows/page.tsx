"use client";
import { useEffect, useState } from 'react';

interface FlowSummary {
  id: string;
  name?: string;
  version?: string;
  created_at?: string;
}

export default function FlowsPage() {
  const [flows, setFlows] = useState<FlowSummary[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    const server = process.env.NEXT_PUBLIC_FLOWRUNNER_SERVER || 'http://localhost:8080';
    fetch(`${server}/api/v1/flows`, { signal: controller.signal })
      .then(async (r) => {
        if (!r.ok) throw new Error(await r.text());
        return r.json();
      })
      .then(setFlows)
      .catch((e) => setError(e.message));
    return () => controller.abort();
  }, []);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Flows</h1>
      </div>
      {error && (
        <div className="cyber-card p-4 text-red-300 border border-red-500/30">{error}</div>
      )}
      <div className="grid grid-cols-1 gap-4">
        {flows.map((f) => (
          <div key={f.id} className="cyber-card p-4">
            <div className="flex items-center justify-between">
              <div>
                <div className="font-medium">{f.name || f.id}</div>
                <div className="text-white/50 text-xs mt-1">v{f.version || '—'} · {f.created_at || ''}</div>
              </div>
              <button className="px-3 py-1.5 rounded bg-neon-cyan/20 text-neon-cyan text-sm border border-neon-cyan/30 hover:bg-neon-cyan/30">Run</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
