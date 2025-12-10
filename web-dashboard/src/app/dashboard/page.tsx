"use client";

import { DashboardShell } from "@/components/layout/dashboard-shell";

export default function DashboardPage() {
  return (
    <DashboardShell title="Overview">
      <div className="grid gap-4 md:grid-cols-3">
        <div className="rounded-lg border border-zinc-200 bg-white p-4">
          <h2 className="mb-2 text-sm font-semibold text-zinc-900">
            Devices
          </h2>
          <p className="text-sm text-zinc-600">
            High-level overview of registered devices and their status.
          </p>
        </div>
        <div className="rounded-lg border border-zinc-200 bg-white p-4">
          <h2 className="mb-2 text-sm font-semibold text-zinc-900">
            Telemetry
          </h2>
          <p className="text-sm text-zinc-600">
            Recent telemetry activity and health indicators.
          </p>
        </div>
        <div className="rounded-lg border border-zinc-200 bg-white p-4">
          <h2 className="mb-2 text-sm font-semibold text-zinc-900">
            OTA Updates
          </h2>
          <p className="text-sm text-zinc-600">
            Summary of recent firmware deployments.
          </p>
        </div>
      </div>
    </DashboardShell>
  );
}
