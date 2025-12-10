"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { ProvisioningWorkflow } from "@/components/devices/provisioning-workflow";
import { SerialMonitor } from "@/components/devices/serial-monitor";
import { getDevice } from "@/lib/devices/client";
import type { Device, ProvisioningJob } from "@/lib/devices/types";

export default function DeviceDetailPage() {
  const params = useParams();
  const deviceId = typeof params.id === "string" ? params.id : params.id?.[0];

  const [device, setDevice] = useState<Device | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<"overview" | "workflow" | "serial">("overview");
  const [currentJob, setCurrentJob] = useState<ProvisioningJob | null>(null);

  useEffect(() => {
    if (!deviceId) return;
    // Batch state updates to avoid cascading renders
    Promise.resolve().then(() => {
      setLoading(true);
      setError(null);
    });
    getDevice(deviceId)
      .then((dev) => setDevice(dev))
      .catch((err) => {
        setError(err instanceof Error ? err.message : "Failed to load device");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [deviceId]);

  if (loading) {
    return (
      <DashboardShell title="Device Details">
        <div className="flex items-center justify-center py-12">
          <p className="text-sm text-zinc-600">Loading device...</p>
        </div>
      </DashboardShell>
    );
  }

  if (error || !device) {
    return (
      <DashboardShell title="Device Details">
        <div className="rounded-lg border border-red-200 bg-red-50 p-4">
          <p className="text-sm text-red-600">{error || "Device not found"}</p>
        </div>
      </DashboardShell>
    );
  }

  return (
    <DashboardShell title={device.name}>
      <div className="space-y-6">
        {/* Tabs */}
        <div className="flex border-b border-zinc-200">
          <button
            type="button"
            onClick={() => setActiveTab("overview")}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === "overview"
                ? "border-zinc-900 text-zinc-900"
                : "border-transparent text-zinc-600 hover:text-zinc-900"
            }`}
          >
            Overview
          </button>
          <button
            type="button"
            onClick={() => setActiveTab("workflow")}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === "workflow"
                ? "border-zinc-900 text-zinc-900"
                : "border-transparent text-zinc-600 hover:text-zinc-900"
            }`}
          >
            Provisioning
          </button>
          <button
            type="button"
            onClick={() => setActiveTab("serial")}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === "serial"
                ? "border-zinc-900 text-zinc-900"
                : "border-transparent text-zinc-600 hover:text-zinc-900"
            }`}
          >
            Serial Monitor
          </button>
        </div>

        {/* Tab content */}
        {activeTab === "overview" && (
          <div className="grid gap-6 lg:grid-cols-3">
            <div className="lg:col-span-2 space-y-6">
              <div className="rounded-lg border border-zinc-200 bg-white p-4">
                <h2 className="text-base font-semibold text-zinc-900 mb-4">Device Info</h2>
                <dl className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <dt className="font-medium text-zinc-700">Name</dt>
                    <dd className="text-zinc-600">{device.name}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt className="font-medium text-zinc-700">Status</dt>
                    <dd className="text-zinc-600">{device.status}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt className="font-medium text-zinc-700">Template</dt>
                    <dd className="text-zinc-600">{device.templateName}</dd>
                  </div>
                  {device.serialPort && (
                    <div className="flex justify-between">
                      <dt className="font-medium text-zinc-700">Serial Port</dt>
                      <dd className="text-zinc-600">{device.serialPort}</dd>
                    </div>
                  )}
                  {device.board && (
                    <div className="flex justify-between">
                      <dt className="font-medium text-zinc-700">Board</dt>
                      <dd className="text-zinc-600">{device.board}</dd>
                    </div>
                  )}
                  {device.firmwareVersion && (
                    <div className="flex justify-between">
                      <dt className="font-medium text-zinc-700">Firmware Version</dt>
                      <dd className="text-zinc-600">{device.firmwareVersion}</dd>
                    </div>
                  )}
                  <div className="flex justify-between">
                    <dt className="font-medium text-zinc-700">Created</dt>
                    <dd className="text-zinc-600">{new Date(device.createdAt).toLocaleString()}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt className="font-medium text-zinc-700">Last Seen</dt>
                    <dd className="text-zinc-600">{new Date(device.lastSeen).toLocaleString()}</dd>
                  </div>
                </dl>
              </div>

              {device.description && (
                <div className="rounded-lg border border-zinc-200 bg-white p-4">
                  <h2 className="text-base font-semibold text-zinc-900 mb-2">Description</h2>
                  <p className="text-sm text-zinc-700">{device.description}</p>
                </div>
              )}
            </div>

            <aside className="space-y-4">
              <div className="rounded-lg border border-zinc-200 bg-white p-4">
                <h3 className="text-sm font-semibold text-zinc-900 mb-3">Actions</h3>
                <div className="space-y-2">
                  <button
                    type="button"
                    onClick={() => setActiveTab("workflow")}
                    className="w-full px-3 py-2 text-sm font-medium text-white bg-zinc-900 rounded-md hover:bg-zinc-800"
                  >
                    Re-provision device
                  </button>
                  <button
                    type="button"
                    onClick={() => setActiveTab("serial")}
                    className="w-full px-3 py-2 text-sm font-medium text-zinc-700 border border-zinc-300 rounded-md hover:bg-zinc-50"
                  >
                    Open serial monitor
                  </button>
                </div>
              </div>
            </aside>
          </div>
        )}

        {activeTab === "workflow" && (
          <ProvisioningWorkflow
            device={device}
            initialJob={currentJob}
            onJobChange={setCurrentJob}
          />
        )}

        {activeTab === "serial" && (
          <SerialMonitor deviceId={device.id} />
        )}
      </div>
    </DashboardShell>
  );
}
