"use client";

import Link from "next/link";
import type { DeviceHealth } from "@/lib/telemetry/types";

interface DeviceHealthGridProps {
  devices: DeviceHealth[];
}

function statusColor(status: DeviceHealth["status"]) {
  switch (status) {
    case "healthy":
      return "bg-green-100 text-green-800";
    case "warning":
      return "bg-yellow-100 text-yellow-800";
    case "critical":
      return "bg-red-100 text-red-800";
    case "offline":
      return "bg-zinc-100 text-zinc-800";
    default:
      return "bg-zinc-100 text-zinc-800";
  }
}

function formatBytes(bytes?: number) {
  if (!bytes) return "—";
  const units = ["B", "KB", "MB", "GB"];
  let i = 0;
  while (bytes >= 1024 && i < units.length - 1) {
    bytes /= 1024;
    i++;
  }
  return `${bytes.toFixed(1)} ${units[i]}`;
}

function formatUptime(seconds?: number) {
  if (!seconds) return "—";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m ${s}s`;
}

export function DeviceHealthGrid({ devices }: DeviceHealthGridProps) {
  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {devices.map((health) => (
        <Link
          key={health.deviceId}
          href={`/provisioning/devices/${health.deviceId}`}
          className="block rounded-lg border border-zinc-200 bg-white p-4 shadow-sm transition-shadow hover:shadow-md"
        >
          <div className="mb-3 flex items-center justify-between">
            <h3 className="text-base font-semibold text-zinc-900 truncate">
              Device {health.deviceId}
            </h3>
            <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${statusColor(health.status)}`}>
              {health.status}
            </span>
          </div>

          <div className="space-y-2 text-xs text-zinc-600">
            <div className="flex justify-between">
              <span>Last seen</span>
              <span>{new Date(health.lastSeen).toLocaleString()}</span>
            </div>
            {health.uptime !== undefined && (
              <div className="flex justify-between">
                <span>Uptime</span>
                <span>{formatUptime(health.uptime)}</span>
              </div>
            )}
            {health.cpuUsage !== undefined && (
              <div className="flex justify-between">
                <span>CPU</span>
                <span>{health.cpuUsage.toFixed(1)}%</span>
              </div>
            )}
            {health.freeMemory !== undefined && (
              <div className="flex justify-between">
                <span>Free mem</span>
                <span>{formatBytes(health.freeMemory)}</span>
              </div>
            )}
            {health.signalStrength !== undefined && (
              <div className="flex justify-between">
                <span>Signal</span>
                <span>{health.signalStrength} dBm</span>
              </div>
            )}
            {health.batteryLevel !== undefined && (
              <div className="flex justify-between">
                <span>Battery</span>
                <span>{health.batteryLevel}%</span>
              </div>
            )}
            {health.errors !== undefined && health.errors > 0 && (
              <div className="flex justify-between">
                <span>Errors</span>
                <span className="text-red-600">{health.errors}</span>
              </div>
            )}
          </div>
        </Link>
      ))}
    </div>
  );
}
