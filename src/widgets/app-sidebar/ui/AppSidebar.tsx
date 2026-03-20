"use client";

import { useSidebarStore } from "@/shared/model/sidebar.store";
import { SidebarLogo } from "./SidebarLogo";
import { SidebarNavItem } from "./SidebarNavItem";
import { SidebarUserInfo } from "./SidebarUserInfo";

const NAV_ITEMS_TOP = [
  { href: "/dashboard", icon: "■", label: "Dashboard" },
  { href: "/search", icon: "◬", label: "Search" },
  { href: "/chart", icon: "╱", label: "Chart" },
  { href: "/pipeline", icon: "⚙", label: "Pipeline" },
  { href: "/cases", icon: "☰", label: "Cases" },
  { href: "/backtest", icon: "◔", label: "Backtest" },
  { href: "/portfolio", icon: "◆", label: "Portfolio" },
];

const NAV_ITEMS_BOTTOM = [
  { href: "/alerts", icon: "⚠", label: "Alerts", badge: true },
  { href: "/marketplace", icon: "★", label: "Marketplace" },
];

export function AppSidebar() {
  const { isPinned, isExpanded, togglePin, setExpanded } = useSidebarStore();
  const expanded = isPinned || isExpanded;

  return (
    <nav
      data-testid="sidebar-nav"
      className={`absolute inset-y-0 left-0 z-10 flex flex-col py-3 bg-nexus-sidebar border-r border-nexus-border transition-[width] duration-200 ${
        expanded ? "w-[200px]" : "w-16"
      } ${!isPinned && expanded ? "shadow-[4px_0_24px_rgba(0,0,0,0.5)]" : ""}`}
      onMouseEnter={() => setExpanded(true)}
      onMouseLeave={() => setExpanded(false)}
    >
      <SidebarLogo isExpanded={expanded} />

      <div className="flex flex-col gap-0.5 px-2">
        {NAV_ITEMS_TOP.map((item) => (
          <SidebarNavItem key={item.href} {...item} isExpanded={expanded} />
        ))}
      </div>

      <div className="flex-1" />

      <div className="flex flex-col gap-0.5 px-2 mb-2">
        {NAV_ITEMS_BOTTOM.map((item) => (
          <SidebarNavItem key={item.href} {...item} isExpanded={expanded} />
        ))}
      </div>

      <div className="border-t border-nexus-border pt-3 px-2">
        <SidebarUserInfo isExpanded={expanded} />
      </div>

      {expanded && (
        <button
          data-testid="pin-toggle"
          onClick={togglePin}
          className={`absolute top-3 right-2 w-6 h-6 rounded flex items-center justify-center text-xs transition-colors ${
            isPinned
              ? "text-nexus-accent bg-nexus-sidebar-active"
              : "text-nexus-text-muted hover:text-nexus-text-secondary"
          }`}
        >
          {isPinned ? "◉" : "○"}
        </button>
      )}
    </nav>
  );
}
