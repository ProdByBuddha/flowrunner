"use client";
import { useEffect, useState } from 'react';
import Link from 'next/link';

interface ExecutionSummary {
  id: string;
  flow_id?: string;
  status?: string;
  started_at?: string;
}

export default function ExecutionsPage() {
  const [executions, setExecutions] = useState<ExecutionSummary[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    const server = process.env.NEXT_PUBLIC_FLOWRUNNER_SERVER || 'http://localhost:8080';
    fetch(`${server}/api/v1/executions`, { signal: controller.signal })
      .then(async (r) => {
        if (!r.ok) throw new Error(await r.text());
        return r.json();
      })
      .then(setExecutions)
      .catch((e) => setError(e.message));
    return () => controller.abort();
  }, []);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Executions</h1>
      </div>
      {error && (
        <div className="cyber-card p-4 text-red-300 border border-red-500/30">{error}</div>
      )}
      <div className="grid grid-cols-1 gap-4">
        {executions.map((e) => (
          <Link key={e.id} href={`/execution/${e.id}`} className="cyber-card p-4 block">
            <div className="flex items-center justify-between">
              <div>
                <div className="font-medium">{e.id}</div>
                <div className="text-white/50 text-xs mt-1">{e.status || 'unknown'} · {e.started_at || ''}</div>
              </div>
              <div className="text-white/50 text-xs">Flow: {e.flow_id || '—'}</div>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
