"use client";

import { useState, useEffect } from "react";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { DeviceHealthGrid } from "@/components/monitoring/device-health-grid";
import { AlertList } from "@/components/monitoring/alert-list";
import { getAllDevicesHealth, getAlerts } from "@/lib/telemetry/client";
import type { DeviceHealth, Alert } from "@/lib/telemetry/types";

export default function MonitoringPage() {
  const [health, setHealth] = useState<DeviceHealth[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function fetchData() {
    try {
      const [healthData, alertsData] = await Promise.all([
        getAllDevicesHealth(),
        getAlerts({ status: "active", pageSize: 10 }),
      ]);
      setHealth(healthData);
      setAlerts(alertsData.alerts);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load monitoring data");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    setLoading(true);
    fetchData();
    // Poll every 10 seconds
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, []);

  return (
    <DashboardShell title="Device Monitoring">
      <div className="space-y-6">
        {loading && (
          <div className="flex items-center justify-center py-12">
            <p className="text-sm text-zinc-600">Loading monitoring data...</p>
          </div>
        )}

        {error && (
          <div className="rounded-lg border border-red-200 bg-red-50 p-4">
            <p className="text-sm text-red-600">{error}</p>
          </div>
        )}

        {!loading && !error && (
          <>
            {/* Device Health Overview */}
            <div>
              <h2 className="text-base font-semibold text-zinc-900 mb-4">Device Health</h2>
              {health.length === 0 ? (
                <div className="rounded-lg border border-zinc-200 bg-white p-8 text-center">
                  <p className="text-sm text-zinc-600">No devices found.</p>
                </div>
              ) : (
                <DeviceHealthGrid devices={health} />
              )}
            </div>

            {/* Recent Alerts */}
            <div>
              <h2 className="text-base font-semibold text-zinc-900 mb-4">Recent Alerts</h2>
              <AlertList alerts={alerts} onRefresh={fetchData} />
            </div>
          </>
        )}
      </div>
    </DashboardShell>
  );
}
