"use client";
import { useEffect, useMemo, useRef, useState } from 'react';

export default function ExecutionDetail({ params }: { params: { id: string } }) {
  const { id } = params;
  const [status, setStatus] = useState<any>({});
  const [logMessages, setLogMessages] = useState<string[]>([]);
  const logRef = useRef<HTMLDivElement>(null);
  const server = process.env.NEXT_PUBLIC_FLOWRUNNER_SERVER || 'http://localhost:8080';

  // Fetch status
  useEffect(() => {
    let cancelled = false;
    fetch(`${server}/api/v1/executions/${id}`)
      .then((r) => r.json())
      .then((s) => !cancelled && setStatus(s));
    return () => { cancelled = true; };
  }, [id, server]);

  // Stream logs via WebSocket
  useEffect(() => {
    const socketUrl = `${server.replace('http', 'ws')}/ws/executions/${id}`;
    const ws = new WebSocket(socketUrl);
    ws.onmessage = (ev) => {
      setLogMessages((prev) => [...prev, ev.data]);
      // Auto-scroll
      if (logRef.current) {
        logRef.current.scrollTop = logRef.current.scrollHeight;
      }
    };
    return () => ws.close();
  }, [id, server]);

  const statusBadges = useMemo(() => {
    const s = String(status?.status || 'unknown');
    const color = s === 'completed' ? 'bg-emerald-400/20 text-emerald-300 border-emerald-400/30' : s === 'failed' ? 'bg-red-400/20 text-red-300 border-red-400/30' : 'bg-white/10 text-white/80 border-white/10';
    return <span className={`px-2 py-0.5 rounded text-xs border ${color}`}>{s}</span>;
  }, [status]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Execution {id}</h1>
        {statusBadges}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="cyber-card p-4">
          <h2 className="text-sm font-medium text-white/70">Details</h2>
          <pre className="mt-2 text-xs text-white/70 overflow-auto">{JSON.stringify(status, null, 2)}</pre>
        </div>
        <div className="cyber-card p-4">
          <h2 className="text-sm font-medium text-white/70">Live Logs</h2>
          <div ref={logRef} className="mt-2 h-80 overflow-auto bg-black/40 rounded border border-white/5 p-2 text-xs">
            {logMessages.map((m, i) => (
              <div key={i} className="font-mono text-white/80">{m}</div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
