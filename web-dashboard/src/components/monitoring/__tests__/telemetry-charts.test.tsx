import { render, screen } from "@testing-library/react";
import { TelemetryCharts, MetricCard } from "../telemetry-charts";
import type { TelemetrySeries } from "@/lib/telemetry/types";

const mockTelemetryData: TelemetrySeries[] = [
  {
    deviceId: "device-1",
    metric: "temperature",
    points: [
      { timestamp: "2025-01-01T12:00:00Z", value: 25.5 },
      { timestamp: "2025-01-01T12:01:00Z", value: 26.0 },
      { timestamp: "2025-01-01T12:02:00Z", value: 25.8 },
    ],
  },
  {
    deviceId: "device-1",
    metric: "humidity",
    points: [
      { timestamp: "2025-01-01T12:00:00Z", value: 60 },
      { timestamp: "2025-01-01T12:01:00Z", value: 62 },
      { timestamp: "2025-01-01T12:02:00Z", value: 61 },
    ],
  },
];

describe("TelemetryCharts", () => {
  it("renders no data message when no series provided", () => {
    render(<TelemetryCharts series={[]} />);
    expect(screen.getByText("No telemetry data available")).toBeInTheDocument();
  });

  it("renders chart title when data is provided", () => {
    render(<TelemetryCharts series={mockTelemetryData} />);
    expect(screen.getByText("Telemetry Visualization")).toBeInTheDocument();
  });

  it("renders chart type buttons", () => {
    render(<TelemetryCharts series={mockTelemetryData} />);
    expect(screen.getByText("Line")).toBeInTheDocument();
    expect(screen.getByText("Area")).toBeInTheDocument();
    expect(screen.getByText("Bar")).toBeInTheDocument();
  });

  it("highlights active chart type", () => {
    render(<TelemetryCharts series={mockTelemetryData} chartType="area" />);
    const areaButton = screen.getByText("Area");
    expect(areaButton).toHaveClass("bg-zinc-900", "text-white");
  });

  it("renders responsive chart container", () => {
    const { container } = render(<TelemetryCharts series={mockTelemetryData} />);
    // Check if the chart component renders at all
    expect(container.querySelector(".rounded-lg")).toBeInTheDocument();
  });

  it("applies custom height", () => {
    const { container } = render(<TelemetryCharts series={mockTelemetryData} height={400} />);
    const chartDiv = container.querySelector(".rounded-lg");
    expect(chartDiv).toHaveStyle({ height: "400px" });
  });

  it("applies custom className", () => {
    const { container } = render(<TelemetryCharts series={mockTelemetryData} className="custom-class" />);
    const chartDiv = container.querySelector(".rounded-lg");
    expect(chartDiv).toHaveClass("custom-class");
  });
});

describe("MetricCard", () => {
  it("renders metric title and value", () => {
    render(<MetricCard title="Temperature" value={25.5} unit="°C" />);
    expect(screen.getByText("Temperature")).toBeInTheDocument();
    expect(screen.getByText("25.5")).toBeInTheDocument();
    expect(screen.getByText("°C")).toBeInTheDocument();
  });

  it("displays upward trend", () => {
    render(<MetricCard title="CPU" value={75} trend="up" trendValue={5} />);
    expect(screen.getByText("↑")).toBeInTheDocument();
    expect(screen.getByText("5%")).toBeInTheDocument();
    const trendContainer = screen.getByText("↑").parentElement;
    expect(trendContainer).toHaveClass("text-green-600");
  });

  it("displays downward trend", () => {
    render(<MetricCard title="Memory" value={45} trend="down" trendValue={10} />);
    expect(screen.getByText("↓")).toBeInTheDocument();
    expect(screen.getByText("10%")).toBeInTheDocument();
    const trendContainer = screen.getByText("↓").parentElement;
    expect(trendContainer).toHaveClass("text-red-600");
  });

  it("displays stable trend", () => {
    render(<MetricCard title="Disk" value={50} trend="stable" trendValue={0} />);
    expect(screen.getByText("→")).toBeInTheDocument();
    expect(screen.getByText("0%")).toBeInTheDocument();
    const trendContainer = screen.getByText("→").parentElement;
    expect(trendContainer).toHaveClass("text-zinc-600");
  });

  it("applies custom color", () => {
    render(<MetricCard title="Custom" value={100} color="#ff0000" />);
    const valueElement = screen.getByText("100");
    expect(valueElement).toHaveStyle({ color: "#ff0000" });
  });

  it("renders without trend information", () => {
    render(<MetricCard title="Basic" value={42} />);
    expect(screen.getByText("Basic")).toBeInTheDocument();
    expect(screen.getByText("42")).toBeInTheDocument();
    expect(screen.queryByText("↑")).not.toBeInTheDocument();
    expect(screen.queryByText("↓")).not.toBeInTheDocument();
    expect(screen.queryByText("→")).not.toBeInTheDocument();
  });

  it("renders without unit", () => {
    render(<MetricCard title="Count" value={1000} />);
    expect(screen.getByText("Count")).toBeInTheDocument();
    expect(screen.getByText("1000")).toBeInTheDocument();
    expect(screen.queryByText("%")).not.toBeInTheDocument();
  });
});
