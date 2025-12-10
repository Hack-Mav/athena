import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { DashboardShell } from "../dashboard-shell";

// Mock next/navigation
jest.mock("next/navigation", () => ({
  useRouter: () => ({
    push: jest.fn(),
    replace: jest.fn(),
  }),
  usePathname: () => "/dashboard",
}));

// Mock auth client
jest.mock("@/lib/auth/client", () => ({
  logout: jest.fn(),
}));

// Mock useAuth hook
jest.mock("@/hooks/useAuth", () => ({
  useAuth: () => ({
    refreshToken: "test-token",
    isAuthenticated: true,
    initialized: true,
  }),
}));

describe("DashboardShell", () => {
  it("renders dashboard title", () => {
    render(
      <DashboardShell title="Test Page">
        <div>Test content</div>
      </DashboardShell>
    );
    expect(screen.getByText("Test Page")).toBeInTheDocument();
  });

  it("renders children content", () => {
    render(
      <DashboardShell title="Test Page">
        <div>Test content</div>
      </DashboardShell>
    );
    expect(screen.getByText("Test content")).toBeInTheDocument();
  });

  it("displays navigation menu", () => {
    render(
      <DashboardShell title="Test Page">
        <div>Test content</div>
      </DashboardShell>
    );
    expect(screen.getByText("Overview")).toBeInTheDocument();
    expect(screen.getByText("Templates")).toBeInTheDocument();
    expect(screen.getByText("Provisioning")).toBeInTheDocument();
    expect(screen.getByText("Monitoring")).toBeInTheDocument();
    expect(screen.getByText("OTA")).toBeInTheDocument();
  });

  it("shows ATHENA branding", () => {
    render(
      <DashboardShell title="Test Page">
        <div>Test content</div>
      </DashboardShell>
    );
    expect(screen.getByText("ATHENA")).toBeInTheDocument();
  });

  it("displays logout button", () => {
    render(
      <DashboardShell title="Test Page">
        <div>Test content</div>
      </DashboardShell>
    );
    expect(screen.getByText("Log out")).toBeInTheDocument();
  });

  it("has responsive layout", () => {
    const { container } = render(
      <DashboardShell title="Test Page">
        <div>Test content</div>
      </DashboardShell>
    );
    const aside = container.querySelector("aside");
    expect(aside).toHaveClass("hidden", "md:flex");
  });

  it("renders header with title", () => {
    const { container } = render(
      <DashboardShell title="Test Page">
        <div>Test content</div>
      </DashboardShell>
    );
    const header = container.querySelector("header");
    expect(header).toBeInTheDocument();
  });
});
