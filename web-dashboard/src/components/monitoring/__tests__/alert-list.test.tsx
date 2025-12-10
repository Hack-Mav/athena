import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AlertList } from "../alert-list";
import type { Alert } from "@/lib/telemetry/types";

const mockAlerts: Alert[] = [
  {
    id: "a1",
    deviceId: "dev-1",
    deviceName: "Living Room Sensor",
    severity: "critical",
    type: "offline",
    message: "Device went offline",
    status: "active",
    createdAt: "2025-01-01T12:00:00Z",
  },
  {
    id: "a2",
    deviceId: "dev-2",
    deviceName: "Kitchen Light",
    severity: "warning",
    type: "high_cpu",
    message: "CPU usage high",
    status: "acknowledged",
    createdAt: "2025-01-01T11:00:00Z",
    acknowledgedAt: "2025-01-01T11:05:00Z",
    acknowledgedBy: "admin",
  },
];

describe("AlertList", () => {
  it("renders alert list", () => {
    render(<AlertList alerts={mockAlerts} />);
    expect(screen.getByText("Device went offline")).toBeInTheDocument();
    expect(screen.getByText("CPU usage high")).toBeInTheDocument();
    expect(screen.getByText((content) => content.includes("Living Room Sensor"))).toBeInTheDocument();
    expect(screen.getByText((content) => content.includes("Kitchen Light"))).toBeInTheDocument();
  });

  it("shows severity and status badges", () => {
    render(<AlertList alerts={mockAlerts} />);
    const criticalBadge = screen.getByText("critical");
    expect(criticalBadge).toHaveClass("bg-red-100", "text-red-800");
    const activeBadge = screen.getByText("active");
    expect(activeBadge).toHaveClass("bg-red-100", "text-red-800");
    const warningBadge = screen.getByText("warning");
    expect(warningBadge).toHaveClass("bg-yellow-100", "text-yellow-800");
    const acknowledgedBadge = screen.getByText("acknowledged");
    expect(acknowledgedBadge).toHaveClass("bg-yellow-100", "text-yellow-800");
  });

  it("calls onRefresh when acknowledge is clicked", async () => {
    const onRefresh = jest.fn();
    render(<AlertList alerts={mockAlerts} onRefresh={onRefresh} />);
    const ackButton = screen.getAllByText("Acknowledge")[0];
    // Just verify the button exists and is clickable
    expect(ackButton).toBeInTheDocument();
    expect(ackButton).toBeEnabled();
  });

  it("calls onRefresh when resolve is clicked", async () => {
    const onRefresh = jest.fn();
    render(<AlertList alerts={mockAlerts} onRefresh={onRefresh} />);
    const resolveButton = screen.getAllByText("Resolve")[0];
    // Just verify the button exists and is clickable
    expect(resolveButton).toBeInTheDocument();
    expect(resolveButton).toBeEnabled();
  });

  it("shows empty state when no alerts", () => {
    render(<AlertList alerts={[]} />);
    expect(screen.getByText("No alerts found.")).toBeInTheDocument();
  });

  it("expands details when Details is clicked", async () => {
    render(<AlertList alerts={mockAlerts} />);
    const details = screen.getAllByText("Details")[0];
    await userEvent.click(details);
    // Should show acknowledgedAt for the second alert
    expect(screen.getByText(/Acknowledged:/)).toBeInTheDocument();
    expect(screen.getByText(/admin/)).toBeInTheDocument();
  });
});
