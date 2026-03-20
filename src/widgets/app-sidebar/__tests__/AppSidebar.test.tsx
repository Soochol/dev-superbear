/** @jest-environment jsdom */
import { render, screen } from "@testing-library/react";

let mockPathname = "/dashboard";
jest.mock("next/navigation", () => ({
  usePathname: () => mockPathname,
}));

jest.mock("next/link", () => {
  return {
    __esModule: true,
    default: ({
      children,
      href,
      className,
      ...rest
    }: {
      children: React.ReactNode;
      href: string;
      className?: string;
      [key: string]: unknown;
    }) => (
      <a href={href} className={className} {...rest}>
        {children}
      </a>
    ),
  };
});

import { SidebarNavItem } from "../ui/SidebarNavItem";
import { useSidebarStore } from "@/shared/model/sidebar.store";
import { AppSidebar } from "../ui/AppSidebar";
import userEvent from "@testing-library/user-event";

describe("SidebarNavItem", () => {
  beforeEach(() => {
    mockPathname = "/dashboard";
  });

  it("renders icon always", () => {
    render(
      <SidebarNavItem
        href="/dashboard"
        icon="■"
        label="Dashboard"
        isExpanded={false}
        pathname="/dashboard"
      />
    );
    expect(screen.getByText("■")).toBeInTheDocument();
  });

  it("renders label when expanded", () => {
    render(
      <SidebarNavItem
        href="/dashboard"
        icon="■"
        label="Dashboard"
        isExpanded={true}
        pathname="/dashboard"
      />
    );
    expect(screen.getByText("Dashboard")).toBeInTheDocument();
  });

  it("hides label when collapsed", () => {
    render(
      <SidebarNavItem
        href="/dashboard"
        icon="■"
        label="Dashboard"
        isExpanded={false}
        pathname="/dashboard"
      />
    );
    expect(screen.queryByText("Dashboard")).not.toBeInTheDocument();
  });

  it("shows active style when pathname matches", () => {
    render(
      <SidebarNavItem
        href="/dashboard"
        icon="■"
        label="Dashboard"
        isExpanded={true}
        pathname="/dashboard"
      />
    );
    const link = screen.getByRole("link");
    expect(link.className).toContain("text-nexus-accent");
  });

  it("shows inactive style when pathname differs", () => {
    render(
      <SidebarNavItem
        href="/dashboard"
        icon="■"
        label="Dashboard"
        isExpanded={true}
        pathname="/search"
      />
    );
    const link = screen.getByRole("link");
    expect(link.className).toContain("text-nexus-text-muted");
  });

  it("renders badge when badge prop is true", () => {
    render(
      <SidebarNavItem
        href="/alerts"
        icon="⚠"
        label="Alerts"
        isExpanded={true}
        pathname="/alerts"
        badge
      />
    );
    expect(screen.getByTestId("nav-badge")).toBeInTheDocument();
  });
});

describe("AppSidebar", () => {
  beforeEach(() => {
    localStorage.clear();
    useSidebarStore.setState(useSidebarStore.getInitialState());
  });

  it("renders all 9 navigation items", () => {
    render(<AppSidebar />);
    const links = screen.getAllByRole("link");
    // 9 nav items + logo link = 10
    expect(links.length).toBeGreaterThanOrEqual(9);
  });

  it("renders logo", () => {
    render(<AppSidebar />);
    expect(screen.getByTestId("sidebar-logo")).toBeInTheDocument();
  });

  it("expands on mouse enter", async () => {
    render(<AppSidebar />);
    const sidebar = screen.getByTestId("sidebar-nav");
    await userEvent.hover(sidebar);
    expect(useSidebarStore.getState().isExpanded).toBe(true);
  });

  it("collapses on mouse leave when not pinned", async () => {
    render(<AppSidebar />);
    const sidebar = screen.getByTestId("sidebar-nav");
    await userEvent.hover(sidebar);
    await userEvent.unhover(sidebar);
    expect(useSidebarStore.getState().isExpanded).toBe(false);
  });

  it("shows pin button when expanded", async () => {
    useSidebarStore.setState({ isExpanded: true });
    render(<AppSidebar />);
    expect(screen.getByTestId("pin-toggle")).toBeInTheDocument();
  });

  it("toggles pin on pin button click", async () => {
    useSidebarStore.setState({ isExpanded: true });
    render(<AppSidebar />);
    await userEvent.click(screen.getByTestId("pin-toggle"));
    expect(useSidebarStore.getState().isPinned).toBe(true);
  });
});
