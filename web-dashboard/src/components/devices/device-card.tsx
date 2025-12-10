"use client";

import Link from "next/link";
import type { Device } from "@/lib/devices/types";

interface DeviceCardProps {
  device: Device;
}

function statusColor(status: Device) {
  switch (status.status) {
    case "online":
      return "bg-green-100 text-green-800";
    case "offline":
      return "bg-zinc-100 text-zinc-800";
    case "compiling":
      return "bg-blue-100 text-blue-800";
    case "flashing":
      return "bg-yellow-100 text-yellow-800";
    case "error":
      return "bg-red-100 text-red-800";
    default:
      return "bg-zinc-100 text-zinc-800";
  }
}

export function DeviceCard({ device }: DeviceCardProps) {
  return (
    <Link
      href={`/provisioning/devices/${device.id}`}
      className="block rounded-lg border border-zinc-200 bg-white p-4 shadow-sm transition-shadow hover:shadow-md"
    >
      <div className="mb-2 flex items-start justify-between">
        <h2 className="text-base font-semibold text-zinc-900 leading-tight">
          {device.name}
        </h2>
        <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${statusColor(device)}`}>
          {device.status}
        </span>
      </div>
      {device.description && (
        <p className="mb-3 text-sm text-zinc-700 line-clamp-2">{device.description}</p>
      )}
      <div className="space-y-1 text-xs text-zinc-600">
        <div>Template: {device.templateName}</div>
        {device.serialPort && <div>Serial: {device.serialPort}</div>}
        {device.board && <div>Board: {device.board}</div>}
        {device.firmwareVersion && <div>Firmware: {device.firmwareVersion}</div>}
        <div>Last seen: {new Date(device.lastSeen).toLocaleString()}</div>
      </div>
    </Link>
  );
}
