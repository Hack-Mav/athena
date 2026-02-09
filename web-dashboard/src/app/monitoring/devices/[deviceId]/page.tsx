"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { TelemetryCharts, MetricCard } from "@/components/monitoring/telemetry-charts";
import { AlertList } from "@/components/monitoring/alert-list";
import { 
  getDeviceTelemetry, 
  getDeviceAlerts, 
  getDeviceHealth 
} from "@/lib/telemetry/client";
import type { 
  TelemetrySeries, 
  Alert, 
  DeviceHealth 
} from "@/lib/telemetry/types";

export default function DeviceMonitoringPage() {
  const params = useParams();
  const deviceId = typeof params.deviceId === "string" ? params.deviceId : params.deviceId?.[0];

  const [telemetry, setTelemetry] = useState<TelemetrySeries[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [health, setHealth] = useState<DeviceHealth | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [timeRange, setTimeRange] = useState<"1h" | "6h" | "24h" | "7d">("24h");

  useEffect(() => {
    if (!deviceId) return;
    
    async function fetchData() {
      try {
        setLoading(true);
        setError(null);
        
        const [telemetryData, alertsData, healthData] = await Promise.all([
          getDeviceTelemetry(deviceId as string, timeRange),
          getDeviceAlerts(deviceId as string, { status: "active", pageSize: 10 }),
          getDeviceHealth(deviceId as string),
        ]);
        
        setTelemetry(telemetryData.series);
        setAlerts(alertsData.alerts);
        setHealth(healthData);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load monitoring data");
      } finally {
        setLoading(false);
      }
    }

    fetchData();
    
    // Set up polling for real-time updates
    const interval = setInterval(fetchData, 30000); // Poll every 30 seconds
    return () => clearInterval(interval);
  }, [deviceId, timeRange]);

  if (loading) {
    return (
      <DashboardShell title="Device Monitoring">
        <div className="flex items-center justify-center py-12">
          <p className="text-sm text-zinc-600">Loading monitoring data...</p>
        </div>
      </DashboardShell>
    );
  }

  if (error) {
    return (
      <DashboardShell title="Device Monitoring">
        <div className="rounded-lg border border-red-200 bg-red-50 p-4">
          <p className="text-sm text-red-600">{error}</p>
        </div>
      </DashboardShell>
    );
  }

  // Calculate metrics from telemetry data
  const metrics = telemetry.reduce((acc, series) => {
    const latest = series.points[series.points.length - 1];
    if (latest) {
      acc[series.metric] = latest.value;
    }
    return acc;
  }, {} as Record<string, number>);

  return (
    <DashboardShell title={`Device Monitoring - ${deviceId}`}>
      <div className="space-y-6">
        {/* Time Range Selector */}
        <div className="flex items-center justify-between">
          <div className="flex gap-2">
            {(["1h", "6h", "24h", "7d"] as const).map((range) => (
              <button
                key={range}
                type="button"
                onClick={() => setTimeRange(range)}
                className={`px-3 py-1 text-sm rounded-md ${
                  timeRange === range
                    ? "bg-zinc-900 text-white"
                    : "bg-zinc-100 text-zinc-700 hover:bg-zinc-200"
                }`}
              >
                {range}
              </button>
            ))}
          </div>
        </div>

        {/* Device Status Overview */}
        {health && (
          <div className="grid gap-4 md:grid-cols-4">
            <MetricCard
              title="Device Status"
              value={health.status}
              color={health.status === "healthy" ? "#10b981" : health.status === "offline" ? "#ef4444" : "#f59e0b"}
            />
            <MetricCard
              title="CPU Usage"
              value={health.cpuUsage || 0}
              unit="%"
              trend={(health.cpuUsage || 0) > 80 ? "up" : (health.cpuUsage || 0) < 50 ? "stable" : "up"}
              color={(health.cpuUsage || 0) > 80 ? "#ef4444" : (health.cpuUsage || 0) > 60 ? "#f59e0b" : "#10b981"}
            />
            <MetricCard
              title="Free Memory"
              value={health.freeMemory ? Math.round(health.freeMemory / 1024 / 1024) : 0}
              unit="MB"
              trend={health.freeMemory && health.freeMemory < 1000000 ? "down" : "stable"}
              color={health.freeMemory && health.freeMemory < 1000000 ? "#ef4444" : health.freeMemory && health.freeMemory < 5000000 ? "#f59e0b" : "#10b981"}
            />
            <MetricCard
              title="Uptime"
              value={Math.floor((health.uptime || 0) / 3600)}
              unit="hours"
              color="#3b82f6"
            />
          </div>
        )}

        {/* Telemetry Charts */}
        <div className="space-y-6">
          <div>
            <h2 className="text-base font-semibold text-zinc-900 mb-4">Temperature Trends</h2>
            <TelemetryCharts 
              series={telemetry.filter(s => s.metric.includes("temperature"))}
              height={300}
              chartType="line"
            />
          </div>

          <div>
            <h2 className="text-base font-semibold text-zinc-900 mb-4">System Metrics</h2>
            <TelemetryCharts 
              series={telemetry.filter(s => !s.metric.includes("temperature"))}
              height={300}
              chartType="area"
            />
          </div>
        </div>

        {/* Recent Alerts */}
        <div>
          <h2 className="text-base font-semibold text-zinc-900 mb-4">Recent Alerts</h2>
          {alerts.length === 0 ? (
            <div className="rounded-lg border border-zinc-200 bg-white p-8 text-center">
              <p className="text-sm text-zinc-600">No active alerts for this device.</p>
            </div>
          ) : (
            <AlertList alerts={alerts} />
          )}
        </div>

        {/* Device Information */}
        <div className="rounded-lg border border-zinc-200 bg-white p-4">
          <h2 className="text-base font-semibold text-zinc-900 mb-4">Device Information</h2>
          <div className="grid gap-4 md:grid-cols-2 text-sm">
            <div>
              <span className="font-medium text-zinc-700">Device ID:</span>
              <span className="ml-2 text-zinc-600">{deviceId}</span>
            </div>
            <div>
              <span className="font-medium text-zinc-700">Last Seen:</span>
              <span className="ml-2 text-zinc-600">
                {health?.lastSeen ? new Date(health.lastSeen).toLocaleString() : "Unknown"}
              </span>
            </div>
            <div>
              <span className="font-medium text-zinc-700">Device ID:</span>
              <span className="ml-2 text-zinc-600">{health?.deviceId || "Unknown"}</span>
            </div>
            <div>
              <span className="font-medium text-zinc-700">Status:</span>
              <span className="ml-2 text-zinc-600">{health?.status || "Unknown"}</span>
            </div>
          </div>
        </div>
      </div>
    </DashboardShell>
  );
}
