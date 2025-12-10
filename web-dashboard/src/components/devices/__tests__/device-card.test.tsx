import { render, screen } from "@testing-library/react";
import { DeviceCard } from "../device-card";
import type { Device } from "@/lib/devices/types";

const mockDevice: Device = {
  id: "dev-1",
  name: "Living Room Sensor",
  description: "Temperature/humidity sensor",
  templateId: "tpl-1",
  templateName: "Blinky",
  status: "online",
  firmwareVersion: "1.2.3",
  lastSeen: "2025-01-01T12:00:00Z",
  createdAt: "2025-01-01T00:00:00Z",
  updatedAt: "2025-01-01T00:00:00Z",
  config: { interval: 30 },
  metadata: { location: "living-room" },
  serialPort: "COM3",
  board: "esp32dev",
  mcu: "esp32",
};

describe("DeviceCard", () => {
  it("renders device information", () => {
    render(<DeviceCard device={mockDevice} />);
    expect(screen.getByText("Living Room Sensor")).toBeInTheDocument();
    expect(screen.getByText((content) => content.includes("Blinky"))).toBeInTheDocument();
    expect(screen.getByText((content) => content.includes("COM3"))).toBeInTheDocument();
    expect(screen.getByText((content) => content.includes("esp32dev"))).toBeInTheDocument();
    expect(screen.getByText((content) => content.includes("1.2.3"))).toBeInTheDocument();
  });

  it("shows correct status badge", () => {
    render(<DeviceCard device={mockDevice} />);
    const badge = screen.getByText("online");
    expect(badge).toHaveClass("bg-green-100", "text-green-800");
  });

  it("links to device detail page", () => {
    render(<DeviceCard device={mockDevice} />);
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/provisioning/devices/dev-1");
  });

  it("handles missing optional fields gracefully", () => {
    const minimalDevice = {
      ...mockDevice,
      description: undefined,
      firmwareVersion: undefined,
      serialPort: undefined,
      board: undefined,
    };
    render(<DeviceCard device={minimalDevice} />);
    expect(screen.queryByText("Temperature/humidity sensor")).not.toBeInTheDocument();
    expect(screen.queryByText("1.2.3")).not.toBeInTheDocument();
    expect(screen.queryByText("COM3")).not.toBeInTheDocument();
    expect(screen.queryByText("esp32dev")).not.toBeInTheDocument();
  });

  it("displays formatted timestamps", () => {
    render(<DeviceCard device={mockDevice} />);
    expect(screen.getByText(/2025/)).toBeInTheDocument();
  });
});
