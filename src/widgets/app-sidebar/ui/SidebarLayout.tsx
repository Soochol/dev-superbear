"use client";

import { useEffect } from "react";
import { useSidebarStore } from "@/shared/model/sidebar.store";
import { AppSidebar } from "./AppSidebar";

export function SidebarLayout({ children }: { children: React.ReactNode }) {
  const isPinned = useSidebarStore((s) => s.isPinned);

  useEffect(() => {
    useSidebarStore.getState().hydrate();
  }, []);

  return (
    <div className="flex h-screen">
      <div
        className={`relative flex-shrink-0 transition-[width] duration-200 ${
          isPinned ? "w-[200px]" : "w-16"
        }`}
      >
        <AppSidebar />
      </div>
      <main className="flex-1 overflow-y-auto min-w-0 bg-nexus-bg">
        {children}
      </main>
    </div>
  );
}
