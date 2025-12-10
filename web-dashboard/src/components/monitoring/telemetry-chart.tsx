"use client";

import { useMemo } from "react";
import type { TelemetrySeries } from "@/lib/telemetry/types";

interface TelemetryChartProps {
  series: TelemetrySeries[];
  height?: number;
  className?: string;
}

export function TelemetryChart({ series, height = 200, className = "" }: TelemetryChartProps) {
  const { min, max, domain } = useMemo(() => {
    const allPoints = series.flatMap((s) => s.points.map((p) => p.value));
    const minVal = Math.min(...allPoints);
    const maxVal = Math.max(...allPoints);
    const padding = (maxVal - minVal) * 0.1 || 1;
    return {
      min: minVal,
      max: maxVal,
      domain: [minVal - padding, maxVal + padding] as [number, number],
    };
  }, [series]);

  if (series.length === 0) {
    return (
      <div className={`rounded-lg border border-zinc-200 bg-white p-4 text-center ${className}`} style={{ height }}>
        <p className="text-sm text-zinc-600">No data</p>
      </div>
    );
  }

  // Simple SVG line chart (no external library)
  const width = 600;
  const padding = { top: 10, right: 10, bottom: 30, left: 40 };
  const chartWidth = width - padding.left - padding.right;
  const chartHeight = height - padding.top - padding.bottom;

  // Time domain (use latest 24h window)
  const now = new Date();
  const start = new Date(now.getTime() - 24 * 60 * 60 * 1000);
  const timeDomain = [start.getTime(), now.getTime()] as [number, number];

  // Scales
  const xScale = (t: number) => ((t - timeDomain[0]) / (timeDomain[1] - timeDomain[0])) * chartWidth;
  const yScale = (v: number) => chartHeight - ((v - domain[0]) / (domain[1] - domain[0])) * chartHeight;

  // Grid lines
  const yTicks = 5;
  const xTicks = 6;

  return (
    <div className={`rounded-lg border border-zinc-200 bg-white p-4 ${className}`} style={{ height }}>
      <svg width={width} height={height} className="w-full h-full">
        {/* Grid */}
        <g className="text-zinc-300">
          {Array.from({ length: yTicks }).map((_, i) => {
            const y = padding.top + (i * chartHeight) / (yTicks - 1);
            const value = max - ((max - min) * i) / (yTicks - 1);
            return (
              <g key={i}>
                <line x1={padding.left} y1={y} x2={width - padding.right} y2={y} stroke="currentColor" strokeWidth={0.5} />
                <text x={padding.left - 5} y={y + 4} textAnchor="end" fontSize="10" fill="currentColor">
                  {value.toFixed(1)}
                </text>
              </g>
            );
          })}
          {Array.from({ length: xTicks }).map((_, i) => {
            const x = padding.left + (i * chartWidth) / (xTicks - 1);
            const date = new Date(start.getTime() + (i * 24 * 60 * 60 * 1000) / (xTicks - 1));
            return (
              <g key={i}>
                <line x1={x} y1={padding.top} x2={x} y2={height - padding.bottom} stroke="currentColor" strokeWidth={0.5} />
                <text x={x} y={height - padding.bottom + 15} textAnchor="middle" fontSize="10" fill="currentColor">
                  {date.getHours()}:00
                </text>
              </g>
            );
          })}
        </g>

        {/* Lines */}
        {series.map((s, idx) => {
          const color = ["#3b82f6", "#ef4444", "#10b981", "#f59e0b", "#8b5cf6"][idx % 5];
          const d = s.points
            .map((p) => {
              const x = padding.left + xScale(new Date(p.timestamp).getTime());
              const y = padding.top + yScale(p.value);
              return `${idx === 0 && p === s.points[0] ? "M" : "L"} ${x} ${y}`;
            })
            .join(" ");
          return (
            <path
              key={s.deviceId + s.metric}
              d={d}
              stroke={color}
              strokeWidth={2}
              fill="none"
            />
          );
        })}

        {/* Legend */}
        <g className="text-xs">
          {series.map((s, idx) => {
            const color = ["#3b82f6", "#ef4444", "#10b981", "#f59e0b", "#8b5cf6"][idx % 5];
            return (
              <g key={s.deviceId + s.metric}>
                <rect x={width - 150} y={10 + idx * 20} width={12} height={12} fill={color} />
                <text x={width - 132} y={20 + idx * 20} fill="currentColor">
                  {s.deviceId}: {s.metric}
                </text>
              </g>
            );
          })}
        </g>
      </svg>
    </div>
  );
}
