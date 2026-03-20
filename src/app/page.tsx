export default function Home() {
  return (
    <main className="flex flex-1 items-center justify-center">
      <div className="text-center space-y-4">
        <h1 className="text-4xl font-bold tracking-tight">
          NEXUS
        </h1>
        <p className="text-nexus-text-secondary text-lg">
          AI-Native Investment Intelligence Platform
        </p>
        <div className="flex gap-2 justify-center mt-6">
          <span className="inline-block w-2 h-2 rounded-full bg-nexus-success animate-pulse" />
          <span className="text-nexus-text-muted text-sm">System Online</span>
        </div>
      </div>
    </main>
  );
}
