import { SidebarLayout } from "@/widgets/app-sidebar";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  return <SidebarLayout>{children}</SidebarLayout>;
}
