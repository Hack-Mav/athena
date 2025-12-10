import { render, screen } from "@testing-library/react";
import { TelemetryChart } from "../telemetry-chart";
import type { TelemetrySeries } from "@/lib/telemetry/types";

const mockSeries: TelemetrySeries[] = [
  {
    deviceId: "dev-1",
    metric: "temperature",
    unit: "°C",
    points: [
      { timestamp: "2025-01-01T12:00:00Z", value: 22.5 },
      { timestamp: "2025-01-01T13:00:00Z", value: 23.1 },
      { timestamp: "2025-01-01T14:00:00Z", value: 24.2 },
    ],
  },
  {
    deviceId: "dev-1",
    metric: "humidity",
    unit: "%",
    points: [
      { timestamp: "2025-01-01T12:00:00Z", value: 45 },
      { timestamp: "2025-01-01T13:00:00Z", value: 48 },
      { timestamp: "2025-01-01T14:00:00Z", value: 52 },
    ],
  },
];

describe("TelemetryChart", () => {
  it("renders SVG chart", () => {
    const { container } = render(<TelemetryChart series={mockSeries} />);
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });

  it("shows no data message when series is empty", () => {
    render(<TelemetryChart series={[]} />);
    expect(screen.getByText("No data")).toBeInTheDocument();
  });

  it("renders multiple series", () => {
    const { container } = render(<TelemetryChart series={mockSeries} />);
    const paths = container.querySelectorAll("svg path");
    // Should have paths for each series (excluding grid lines)
    expect(paths.length).toBeGreaterThan(0);
  });

  it("respects custom height", () => {
    const { container } = render(<TelemetryChart series={mockSeries} height={300} />);
    const svg = container.querySelector("svg");
    expect(svg).toHaveAttribute("height", "300");
  });

  it("applies custom className", () => {
    const { container } = render(
      <TelemetryChart series={mockSeries} className="custom-class" />
    );
    const div = container.firstChild;
    expect(div).toHaveClass("custom-class");
  });

  it("renders grid lines", () => {
    const { container } = render(<TelemetryChart series={mockSeries} />);
    const lines = container.querySelectorAll("svg line");
    expect(lines.length).toBeGreaterThan(0);
  });

  it("handles single point in series", () => {
    const singlePoint: TelemetrySeries[] = [
      {
        deviceId: "dev-1",
        metric: "temperature",
        unit: "°C",
        points: [{ timestamp: "2025-01-01T12:00:00Z", value: 22.5 }],
      },
    ];
    const { container } = render(<TelemetryChart series={singlePoint} />);
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });

  it("handles negative values", () => {
    const negativeSeries: TelemetrySeries[] = [
      {
        deviceId: "dev-1",
        metric: "temperature",
        unit: "°C",
        points: [
          { timestamp: "2025-01-01T12:00:00Z", value: -5 },
          { timestamp: "2025-01-01T13:00:00Z", value: 0 },
          { timestamp: "2025-01-01T14:00:00Z", value: 5 },
        ],
      },
    ];
    const { container } = render(<TelemetryChart series={negativeSeries} />);
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });
});
