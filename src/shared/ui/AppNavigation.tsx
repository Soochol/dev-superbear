"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const NAV_ITEMS = [
  { href: "/search", label: "Search" },
  { href: "/chart", label: "Chart" },
  { href: "/pipeline", label: "Pipeline" },
  { href: "/cases", label: "Cases" },
];

export function AppNavigation() {
  const pathname = usePathname();

  return (
    <nav className="flex items-center gap-6 px-6 py-3 bg-nexus-surface border-b border-nexus-border">
      <Link href="/" className="text-lg font-bold text-nexus-accent">
        NEXUS
      </Link>
      <div className="flex gap-1">
        {NAV_ITEMS.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            className={`px-3 py-1.5 text-sm rounded-md transition-colors ${
              pathname.startsWith(item.href)
                ? "bg-nexus-accent/10 text-nexus-accent"
                : "text-nexus-text-secondary hover:text-nexus-text-primary"
            }`}
          >
            {item.label}
          </Link>
        ))}
      </div>
    </nav>
  );
}
