"use client";

import { useState } from "react";
import type { Alert } from "@/lib/telemetry/types";
import { acknowledgeAlert, resolveAlert } from "@/lib/telemetry/client";

interface AlertListProps {
  alerts: Alert[];
  onRefresh?: () => void;
}

function severityColor(severity: Alert["severity"]) {
  switch (severity) {
    case "critical":
      return "bg-red-100 text-red-800";
    case "warning":
      return "bg-yellow-100 text-yellow-800";
    case "info":
      return "bg-blue-100 text-blue-800";
    default:
      return "bg-zinc-100 text-zinc-800";
  }
}

function statusColor(status: Alert["status"]) {
  switch (status) {
    case "active":
      return "bg-red-100 text-red-800";
    case "acknowledged":
      return "bg-yellow-100 text-yellow-800";
    case "resolved":
      return "bg-green-100 text-green-800";
    default:
      return "bg-zinc-100 text-zinc-800";
  }
}

export function AlertList({ alerts, onRefresh }: AlertListProps) {
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  async function handleAcknowledge(alertId: string) {
    setActionLoading(alertId);
    try {
      await acknowledgeAlert(alertId);
      onRefresh?.();
    } catch {
      // ignore errors
    } finally {
      setActionLoading(null);
    }
  }

  async function handleResolve(alertId: string) {
    setActionLoading(alertId);
    try {
      await resolveAlert(alertId);
      onRefresh?.();
    } catch {
      // ignore errors
    } finally {
      setActionLoading(null);
    }
  }

  if (alerts.length === 0) {
    return (
      <div className="rounded-lg border border-zinc-200 bg-white p-8 text-center">
        <p className="text-sm text-zinc-600">No alerts found.</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {alerts.map((alert) => (
        <div key={alert.id} className="rounded-lg border border-zinc-200 bg-white p-4">
          <div className="mb-2 flex items-start justify-between gap-4">
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-1">
                <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${severityColor(alert.severity)}`}>
                  {alert.severity}
                </span>
                <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${statusColor(alert.status)}`}>
                  {alert.status}
                </span>
                <span className="text-xs text-zinc-500">{alert.type}</span>
              </div>
              <h3 className="text-sm font-medium text-zinc-900">{alert.message}</h3>
              <p className="text-xs text-zinc-600 mt-1">
                {alert.deviceName} • {new Date(alert.createdAt).toLocaleString()}
              </p>
            </div>
            <div className="flex gap-2 flex-shrink-0">
              {alert.status === "active" && (
                <button
                  type="button"
                  onClick={() => handleAcknowledge(alert.id)}
                  disabled={actionLoading === alert.id}
                  className="px-2 py-1 text-xs font-medium text-yellow-700 border border-yellow-300 rounded hover:bg-yellow-50 disabled:opacity-50"
                >
                  Acknowledge
                </button>
              )}
              {alert.status !== "resolved" && (
                <button
                  type="button"
                  onClick={() => handleResolve(alert.id)}
                  disabled={actionLoading === alert.id}
                  className="px-2 py-1 text-xs font-medium text-green-700 border border-green-300 rounded hover:bg-green-50 disabled:opacity-50"
                >
                  Resolve
                </button>
              )}
            </div>
          </div>

          {(alert.details || alert.acknowledgedAt || alert.resolvedAt) && (
            <details className="text-xs text-zinc-600">
              <summary className="cursor-pointer">Details</summary>
              <div className="mt-2 space-y-1">
                {alert.acknowledgedAt && (
                  <div>
                    <span className="font-medium">Acknowledged:</span>{" "}
                    {new Date(alert.acknowledgedAt).toLocaleString()}
                    {alert.acknowledgedBy && ` by ${alert.acknowledgedBy}`}
                  </div>
                )}
                {alert.resolvedAt && (
                  <div>
                    <span className="font-medium">Resolved:</span>{" "}
                    {new Date(alert.resolvedAt).toLocaleString()}
                  </div>
                )}
                {alert.details && (
                  <div>
                    <span className="font-medium">Extra:</span>{" "}
                    <pre className="mt-1 bg-zinc-100 p-2 rounded text-xs overflow-x-auto">
                      {JSON.stringify(alert.details, null, 2)}
                    </pre>
                  </div>
                )}
              </div>
            </details>
          )}
        </div>
      ))}
    </div>
  );
}
