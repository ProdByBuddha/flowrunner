import Link from 'next/link';

export default function HomePage() {
  return (
    <div className="space-y-10">
      <div>
        <h1 className="text-3xl font-semibold neon-text">Operational Command</h1>
        <p className="text-white/70 mt-2">Orchestrate and observe flows in real-time.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Link href="/flows" className="cyber-card p-6 hover:shadow-glow transition-shadow">
          <h2 className="text-xl font-medium">Flows</h2>
          <p className="text-white/60 mt-1 text-sm">Create, update, and manage YAML-defined flows.</p>
        </Link>

        <Link href="/executions" className="cyber-card p-6 hover:shadow-glow transition-shadow">
          <h2 className="text-xl font-medium">Executions</h2>
          <p className="text-white/60 mt-1 text-sm">Monitor executions and stream live logs.</p>
        </Link>
      </div>
    </div>
  );
}
