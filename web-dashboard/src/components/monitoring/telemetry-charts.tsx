"use client";

import { useMemo } from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  AreaChart,
  Area,
  BarChart,
  Bar,
} from "recharts";
import type { TelemetrySeries, TelemetryPoint } from "@/lib/telemetry/types";

interface TelemetryChartsProps {
  series: TelemetrySeries[];
  chartType?: "line" | "area" | "bar";
  height?: number;
  className?: string;
}

export function TelemetryCharts({ 
  series, 
  chartType = "line", 
  height = 300, 
  className = "" 
}: TelemetryChartsProps) {
  const { chartData, domains } = useMemo(() => {
    if (series.length === 0) {
      return { chartData: [], domains: { x: [], y: [] } };
    }

    // Combine all data points and normalize timestamps
    const allPoints: Array<TelemetryPoint & { deviceId: string; metric: string }> = [];
    
    series.forEach((s) => {
      s.points.forEach((point) => {
        allPoints.push({
          ...point,
          deviceId: s.deviceId,
          metric: s.metric,
        });
      });
    });

    // Sort by timestamp
    allPoints.sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());

    // Group by timestamp for chart data
    const timeGroups = new Map<string, any>();
    
    allPoints.forEach((point) => {
      const timeKey = new Date(point.timestamp).toISOString();
      if (!timeGroups.has(timeKey)) {
        timeGroups.set(timeKey, { timestamp: timeKey, time: new Date(point.timestamp) });
      }
      const group = timeGroups.get(timeKey);
      const key = `${point.deviceId}-${point.metric}`;
      group[key] = point.value;
    });

    const chartData = Array.from(timeGroups.values());

    // Calculate domains
    const values = allPoints.map(p => p.value);
    const minValue = Math.min(...values);
    const maxValue = Math.max(...values);
    const padding = (maxValue - minValue) * 0.1 || 1;

    return {
      chartData,
      domains: {
        y: [minValue - padding, maxValue + padding] as [number, number],
      },
    };
  }, [series]);

  const colors = ["#3b82f6", "#ef4444", "#10b981", "#f59e0b", "#8b5cf6", "#ec4899", "#14b8a6", "#f97316"];

  if (series.length === 0) {
    return (
      <div className={`rounded-lg border border-zinc-200 bg-white p-4 text-center ${className}`} style={{ height }}>
        <p className="text-sm text-zinc-600">No telemetry data available</p>
      </div>
    );
  }

  const renderChart = () => {
    const commonProps = {
      data: chartData,
      margin: { top: 5, right: 30, left: 20, bottom: 5 },
    };

    switch (chartType) {
      case "area":
        return (
          <AreaChart {...commonProps}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
            <XAxis 
              dataKey="time"
              tickFormatter={(value) => new Date(value).toLocaleTimeString()}
              stroke="#6b7280"
            />
            <YAxis stroke="#6b7280" domain={domains.y} />
            <Tooltip
              labelFormatter={(value) => new Date(value as Date).toLocaleString()}
              contentStyle={{ backgroundColor: "#ffffff", border: "1px solid #e5e7eb" }}
            />
            <Legend />
            {series.map((s, idx) => (
              <Area
                key={`${s.deviceId}-${s.metric}`}
                type="monotone"
                dataKey={`${s.deviceId}-${s.metric}`}
                name={`${s.deviceId}: ${s.metric}`}
                stroke={colors[idx % colors.length]}
                fill={colors[idx % colors.length]}
                fillOpacity={0.3}
                strokeWidth={2}
              />
            ))}
          </AreaChart>
        );

      case "bar":
        return (
          <BarChart {...commonProps}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
            <XAxis 
              dataKey="time"
              tickFormatter={(value) => new Date(value).toLocaleTimeString()}
              stroke="#6b7280"
            />
            <YAxis stroke="#6b7280" domain={domains.y} />
            <Tooltip
              labelFormatter={(value) => new Date(value as Date).toLocaleString()}
              contentStyle={{ backgroundColor: "#ffffff", border: "1px solid #e5e7eb" }}
            />
            <Legend />
            {series.map((s, idx) => (
              <Bar
                key={`${s.deviceId}-${s.metric}`}
                dataKey={`${s.deviceId}-${s.metric}`}
                name={`${s.deviceId}: ${s.metric}`}
                fill={colors[idx % colors.length]}
              />
            ))}
          </BarChart>
        );

      default:
        return (
          <LineChart {...commonProps}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
            <XAxis 
              dataKey="time"
              tickFormatter={(value) => new Date(value).toLocaleTimeString()}
              stroke="#6b7280"
            />
            <YAxis stroke="#6b7280" domain={domains.y} />
            <Tooltip
              labelFormatter={(value) => new Date(value as Date).toLocaleString()}
              contentStyle={{ backgroundColor: "#ffffff", border: "1px solid #e5e7eb" }}
            />
            <Legend />
            {series.map((s, idx) => (
              <Line
                key={`${s.deviceId}-${s.metric}`}
                type="monotone"
                dataKey={`${s.deviceId}-${s.metric}`}
                name={`${s.deviceId}: ${s.metric}`}
                stroke={colors[idx % colors.length]}
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 6 }}
              />
            ))}
          </LineChart>
        );
    }
  };

  return (
    <div className={`rounded-lg border border-zinc-200 bg-white p-4 ${className}`} style={{ height }}>
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-sm font-semibold text-zinc-900">Telemetry Visualization</h3>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => {}}
            className={`px-2 py-1 text-xs rounded ${
              chartType === "line" 
                ? "bg-zinc-900 text-white" 
                : "bg-zinc-100 text-zinc-700 hover:bg-zinc-200"
            }`}
          >
            Line
          </button>
          <button
            type="button"
            onClick={() => {}}
            className={`px-2 py-1 text-xs rounded ${
              chartType === "area" 
                ? "bg-zinc-900 text-white" 
                : "bg-zinc-100 text-zinc-700 hover:bg-zinc-200"
            }`}
          >
            Area
          </button>
          <button
            type="button"
            onClick={() => {}}
            className={`px-2 py-1 text-xs rounded ${
              chartType === "bar" 
                ? "bg-zinc-900 text-white" 
                : "bg-zinc-100 text-zinc-700 hover:bg-zinc-200"
            }`}
          >
            Bar
          </button>
        </div>
      </div>
      <ResponsiveContainer width="100%" height={height - 60}>
        {renderChart()}
      </ResponsiveContainer>
    </div>
  );
}

interface MetricCardProps {
  title: string;
  value: number | string;
  unit?: string;
  trend?: "up" | "down" | "stable";
  trendValue?: number;
  color?: string;
}

export function MetricCard({ 
  title, 
  value, 
  unit, 
  trend, 
  trendValue, 
  color = "#3b82f6" 
}: MetricCardProps) {
  const trendIcon = trend === "up" ? "↑" : trend === "down" ? "↓" : "→";
  const trendColor = trend === "up" ? "text-green-600" : trend === "down" ? "text-red-600" : "text-zinc-600";

  return (
    <div className="rounded-lg border border-zinc-200 bg-white p-4">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-zinc-600">{title}</p>
          <div className="flex items-baseline gap-1 mt-1">
            <span className="text-2xl font-semibold text-zinc-900" style={{ color }}>
              {value}
            </span>
            {unit && <span className="text-sm text-zinc-500">{unit}</span>}
          </div>
          {trend && trendValue !== undefined && (
            <div className={`flex items-center gap-1 mt-1 text-xs ${trendColor}`}>
              <span>{trendIcon}</span>
              <span>{Math.abs(trendValue)}%</span>
            </div>
          )}
        </div>
        <div 
          className="w-12 h-12 rounded-full opacity-20"
          style={{ backgroundColor: color }}
        />
      </div>
    </div>
  );
}
