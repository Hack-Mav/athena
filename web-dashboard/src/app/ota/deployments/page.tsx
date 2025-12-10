"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { getDeployments } from "@/lib/ota/client";
import type { Deployment, DeploymentStatus } from "@/lib/ota/types";

export default function DeploymentsPage() {
  const router = useRouter();
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Batch state updates to avoid cascading renders
    Promise.resolve().then(() => {
      setLoading(true);
      setError(null);
    });
    getDeployments(1, 50)
      .then((res) => setDeployments(res.deployments))
      .catch((err) => {
        setError(err instanceof Error ? err.message : "Failed to load deployments");
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  function statusColor(status: DeploymentStatus) {
    switch (status) {
      case "completed":
        return "bg-green-100 text-green-800";
      case "running":
        return "bg-blue-100 text-blue-800";
      case "paused":
        return "bg-yellow-100 text-yellow-800";
      case "failed":
        return "bg-red-100 text-red-800";
      case "rolled_back":
        return "bg-zinc-100 text-zinc-800";
      default:
        return "bg-zinc-100 text-zinc-800";
    }
  }

  return (
    <DashboardShell title="Deployments">
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold text-zinc-900">All Deployments</h2>
          <button
            type="button"
            onClick={() => router.push("/ota")}
            className="px-4 py-2 text-sm font-medium text-zinc-700 border border-zinc-300 rounded-md hover:bg-zinc-50"
          >
            Back to OTA
          </button>
        </div>

        {loading && (
          <div className="flex items-center justify-center py-12">
            <p className="text-sm text-zinc-600">Loading deployments...</p>
          </div>
        )}

        {error && (
          <div className="rounded-lg border border-red-200 bg-red-50 p-4">
            <p className="text-sm text-red-600">{error}</p>
          </div>
        )}

        {!loading && !error && (
          <>
            {deployments.length === 0 ? (
              <div className="rounded-lg border border-zinc-200 bg-white p-8 text-center">
                <p className="text-sm text-zinc-600 mb-4">No deployments yet.</p>
                <button
                  type="button"
                  onClick={() => router.push("/ota")}
                  className="px-4 py-2 text-sm font-medium text-white bg-zinc-900 rounded-md hover:bg-zinc-800"
                >
                  Create Deployment
                </button>
              </div>
            ) : (
              <div className="space-y-3">
                {deployments.map((dep) => (
                  <div key={dep.id} className="rounded-lg border border-zinc-200 bg-white p-4">
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <h3 className="text-sm font-medium text-zinc-900">{dep.name}</h3>
                          <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${statusColor(dep.status)}`}>
                            {dep.status}
                          </span>
                        </div>
                        <p className="text-xs text-zinc-600 mb-1">
                          Firmware: {dep.firmwareVersion} • Strategy: {dep.strategy} • {dep.targetGroups.length} target groups
                        </p>
                        <p className="text-xs text-zinc-600">
                          Created {new Date(dep.createdAt).toLocaleString()}
                          {dep.startedAt && ` • Started ${new Date(dep.startedAt).toLocaleString()}`}
                          {dep.completedAt && ` • Completed ${new Date(dep.completedAt).toLocaleString()}`}
                        </p>
                        {dep.description && (
                          <p className="text-xs text-zinc-700 mt-2">{dep.description}</p>
                        )}
                      </div>
                      <div className="flex gap-2 flex-shrink-0">
                        <button
                          type="button"
                          onClick={() => router.push(`/ota/deployments/${dep.id}`)}
                          className="px-2 py-1 text-xs font-medium text-zinc-700 border border-zinc-300 rounded hover:bg-zinc-50"
                        >
                          View
                        </button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </>
        )}
      </div>
    </DashboardShell>
  );
}
