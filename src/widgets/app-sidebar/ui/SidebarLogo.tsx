import Link from "next/link";

interface SidebarLogoProps {
  isExpanded: boolean;
}

export function SidebarLogo({ isExpanded }: SidebarLogoProps) {
  return (
    <Link
      href="/dashboard"
      data-testid="sidebar-logo"
      className="flex items-center gap-3 px-3 mb-4"
    >
      <div className="w-9 h-9 rounded-[10px] bg-gradient-to-br from-nexus-accent to-purple-400 flex items-center justify-center font-extrabold text-sm text-white flex-shrink-0 shadow-[0_0_20px_rgba(99,102,241,0.25)]">
        N
      </div>
      {isExpanded && (
        <span className="text-sm font-bold text-nexus-text-primary whitespace-nowrap">
          NEXUS
        </span>
      )}
    </Link>
  );
}
