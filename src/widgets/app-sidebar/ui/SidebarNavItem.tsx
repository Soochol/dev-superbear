import Link from "next/link";

interface SidebarNavItemProps {
  href: string;
  icon: string;
  label: string;
  isExpanded: boolean;
  pathname: string;
  badge?: boolean;
}

export function SidebarNavItem({
  href,
  icon,
  label,
  isExpanded,
  pathname,
  badge,
}: SidebarNavItemProps) {
  const isActive = pathname === href || pathname.startsWith(href + "/");

  return (
    <Link
      href={href}
      className={`relative flex items-center gap-3 rounded-lg h-10 transition-colors ${
        isExpanded ? "px-3" : "justify-center"
      } ${
        isActive
          ? "bg-nexus-sidebar-active text-nexus-accent"
          : "text-nexus-text-muted hover:bg-nexus-sidebar-hover hover:text-nexus-text-secondary"
      }`}
    >
      {isActive && (
        <span className="absolute left-0 w-[3px] h-5 bg-nexus-accent rounded-r" />
      )}
      <span className="text-lg flex-shrink-0 w-5 text-center">{icon}</span>
      {isExpanded && (
        <span className="text-sm font-medium whitespace-nowrap overflow-hidden">
          {label}
        </span>
      )}
      {badge && (
        <span
          data-testid="nav-badge"
          className="absolute top-1.5 right-1.5 w-2 h-2 bg-nexus-failure rounded-full border-2 border-nexus-sidebar"
        />
      )}
    </Link>
  );
}
