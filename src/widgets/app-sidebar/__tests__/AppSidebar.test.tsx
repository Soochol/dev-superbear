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
      />
    );
    expect(screen.queryByText("Dashboard")).not.toBeInTheDocument();
  });

  it("shows active style when pathname matches", () => {
    mockPathname = "/dashboard";
    render(
      <SidebarNavItem
        href="/dashboard"
        icon="■"
        label="Dashboard"
        isExpanded={true}
      />
    );
    const link = screen.getByRole("link");
    expect(link.className).toContain("text-nexus-accent");
  });

  it("shows inactive style when pathname differs", () => {
    mockPathname = "/search";
    render(
      <SidebarNavItem
        href="/dashboard"
        icon="■"
        label="Dashboard"
        isExpanded={true}
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
        badge
      />
    );
    expect(screen.getByTestId("nav-badge")).toBeInTheDocument();
  });
});
