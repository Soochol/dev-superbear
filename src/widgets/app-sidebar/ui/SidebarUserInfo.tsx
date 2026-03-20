interface SidebarUserInfoProps {
  isExpanded: boolean;
}

export function SidebarUserInfo({ isExpanded }: SidebarUserInfoProps) {
  return (
    <div
      className={`flex items-center gap-3 ${isExpanded ? "px-3" : "justify-center"}`}
    >
      <div className="w-7 h-7 rounded-full bg-gradient-to-br from-nexus-accent to-blue-400 flex items-center justify-center text-[11px] font-semibold text-white flex-shrink-0">
        U
      </div>
      {isExpanded && (
        <span className="text-xs text-nexus-text-secondary whitespace-nowrap">
          User
        </span>
      )}
    </div>
  );
}
