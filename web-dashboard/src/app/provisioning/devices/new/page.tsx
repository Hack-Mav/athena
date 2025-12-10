"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { createDevice } from "@/lib/devices/client";
import type { CreateDeviceRequest } from "@/lib/devices/types";

export default function NewDevicePage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [formData, setFormData] = useState<CreateDeviceRequest>({
    name: "",
    templateId: "",
    config: {},
    metadata: {},
  });

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      await createDevice(formData);
      router.push("/provisioning");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create device");
    } finally {
      setLoading(false);
    }
  }

  function updateField<K extends keyof CreateDeviceRequest>(
    field: K,
    value: CreateDeviceRequest[K]
  ) {
    setFormData(prev => ({ ...prev, [field]: value }));
  }

  return (
    <DashboardShell title="Add New Device">
      <div className="max-w-2xl">
        {error && (
          <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-4">
            <p className="text-sm text-red-600">{error}</p>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Basic Information */}
          <div className="rounded-lg border border-zinc-200 bg-white p-6">
            <h2 className="text-base font-semibold text-zinc-900 mb-4">Basic Information</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">
                  Device Name *
                </label>
                <input
                  type="text"
                  required
                  value={formData.name}
                  onChange={(e) => updateField("name", e.target.value)}
                  className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                  placeholder="My Arduino Device"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">
                  Template *
                </label>
                <select
                  required
                  value={formData.templateId}
                  onChange={(e) => updateField("templateId", e.target.value)}
                  className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                >
                  <option value="">Select a template...</option>
                  <option value="arduino-uno-led">Arduino Uno - LED Blink</option>
                  <option value="arduino-uno-sensor">Arduino Uno - Temperature Sensor</option>
                  <option value="esp32-wifi">ESP32 - WiFi Connection</option>
                </select>
              </div>
            </div>
          </div>

          {/* Device Configuration */}
          <div className="rounded-lg border border-zinc-200 bg-white p-6">
            <h2 className="text-base font-semibold text-zinc-900 mb-4">Device Configuration</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">
                  Board Type
                </label>
                <select
                  value={formData.config.boardType || ""}
                  onChange={(e) => updateField("config", { ...formData.config, boardType: e.target.value })}
                  className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                >
                  <option value="">Select board type...</option>
                  <option value="arduino-uno">Arduino Uno</option>
                  <option value="arduino-nano">Arduino Nano</option>
                  <option value="esp32">ESP32</option>
                  <option value="esp8266">ESP8266</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">
                  Port (Optional)
                </label>
                <input
                  type="text"
                  value={formData.config.port || ""}
                  onChange={(e) => updateField("config", { ...formData.config, port: e.target.value })}
                  className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                  placeholder="/dev/ttyUSB0 or COM3"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">
                  WiFi SSID (for wireless devices)
                </label>
                <input
                  type="text"
                  value={formData.config.wifiSsid || ""}
                  onChange={(e) => updateField("config", { ...formData.config, wifiSsid: e.target.value })}
                  className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                  placeholder="Your WiFi Network"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">
                  WiFi Password (for wireless devices)
                </label>
                <input
                  type="password"
                  value={formData.config.wifiPassword || ""}
                  onChange={(e) => updateField("config", { ...formData.config, wifiPassword: e.target.value })}
                  className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                  placeholder="WiFi Password"
                />
              </div>
            </div>
          </div>

          {/* Metadata */}
          <div className="rounded-lg border border-zinc-200 bg-white p-6">
            <h2 className="text-base font-semibold text-zinc-900 mb-4">Additional Information</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">
                  Description
                </label>
                <textarea
                  rows={3}
                  value={formData.metadata.description || ""}
                  onChange={(e) => updateField("metadata", { ...formData.metadata, description: e.target.value })}
                  className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                  placeholder="Optional description of this device..."
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">
                  Location
                </label>
                <input
                  type="text"
                  value={formData.metadata.location || ""}
                  onChange={(e) => updateField("metadata", { ...formData.metadata, location: e.target.value })}
                  className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                  placeholder="Lab Room 101, Office Shelf, etc."
                />
              </div>
            </div>
          </div>

          {/* Actions */}
          <div className="flex gap-3">
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 text-sm font-medium text-white bg-zinc-900 rounded-md hover:bg-zinc-800 disabled:opacity-60 disabled:cursor-not-allowed"
            >
              {loading ? "Creating Device..." : "Create Device"}
            </button>
            <button
              type="button"
              onClick={() => router.push("/provisioning")}
              className="px-4 py-2 text-sm font-medium text-zinc-700 border border-zinc-300 rounded-md hover:bg-zinc-50"
            >
              Cancel
            </button>
          </div>
        </form>
      </div>
    </DashboardShell>
  );
}
