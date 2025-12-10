"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { DeviceFilters } from "@/components/devices/device-filters";
import { DeviceCard } from "@/components/devices/device-card";
import { getDevices } from "@/lib/devices/client";
import type { Device, DeviceSearchParams } from "@/lib/devices/types";

const DEFAULT_PAGE_SIZE = 12;

export default function ProvisioningPage() {
  const router = useRouter();
  const [devices, setDevices] = useState<Device[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [params, setParams] = useState<DeviceSearchParams>({
    page: 1,
    pageSize: DEFAULT_PAGE_SIZE,
    sortBy: "updatedAt",
    sortOrder: "desc",
  });

  useEffect(() => {
    // Batch state updates to avoid cascading renders
    Promise.resolve().then(() => {
      setLoading(true);
      setError(null);
    });
    getDevices(params)
      .then((res) => {
        setDevices(res.devices);
        setTotal(res.total);
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : "Failed to load devices");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [params]);

  const totalPages = Math.ceil(total / (params.pageSize ?? DEFAULT_PAGE_SIZE));
  const currentPage = params.page ?? 1;

  return (
    <DashboardShell title="Device Provisioning">
      <div className="flex flex-col md:flex-row gap-6">
        {/* Filters sidebar */}
        <aside className="w-full md:w-64 flex-shrink-0">
          <div className="rounded-lg border border-zinc-200 bg-white p-4">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-sm font-semibold text-zinc-900">Filters</h2>
              <button
                type="button"
                onClick={() => router.push("/provisioning/devices/new")}
                className="px-2 py-1 text-xs font-medium text-white bg-zinc-900 rounded hover:bg-zinc-800"
              >
                + Add device
              </button>
            </div>
            <DeviceFilters params={params} onParamsChange={setParams} />
          </div>
        </aside>

        {/* Main content */}
        <div className="flex-1">
          {loading && (
            <div className="flex items-center justify-center py-12">
              <p className="text-sm text-zinc-600">Loading devices...</p>
            </div>
          )}

          {error && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-4">
              <p className="text-sm text-red-600">{error}</p>
            </div>
          )}

          {!loading && !error && (
            <>
              <div className="mb-4 text-sm text-zinc-600">
                {total} device{total !== 1 ? "s" : ""}
              </div>

              {devices.length === 0 ? (
                <div className="rounded-lg border border-zinc-200 bg-white p-8 text-center">
                  <p className="text-sm text-zinc-600 mb-4">No devices found.</p>
                  <button
                    type="button"
                    onClick={() => router.push("/provisioning/devices/new")}
                    className="px-4 py-2 text-sm font-medium text-white bg-zinc-900 rounded-md hover:bg-zinc-800"
                  >
                    Add your first device
                  </button>
                </div>
              ) : (
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                  {devices.map((device) => (
                    <DeviceCard key={device.id} device={device} />
                  ))}
                </div>
              )}

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="mt-6 flex items-center justify-center gap-2">
                  <button
                    type="button"
                    disabled={currentPage <= 1}
                    onClick={() => setParams({ ...params, page: currentPage - 1 })}
                    className="px-3 py-1 text-sm rounded-md border border-zinc-300 bg-white disabled:opacity-50 disabled:cursor-not-allowed hover:bg-zinc-50"
                  >
                    Previous
                  </button>
                  <span className="text-sm text-zinc-600">
                    Page {currentPage} of {totalPages}
                  </span>
                  <button
                    type="button"
                    disabled={currentPage >= totalPages}
                    onClick={() => setParams({ ...params, page: currentPage + 1 })}
                    className="px-3 py-1 text-sm rounded-md border border-zinc-300 bg-white disabled:opacity-50 disabled:cursor-not-allowed hover:bg-zinc-50"
                  >
                    Next
                  </button>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </DashboardShell>
  );
}
