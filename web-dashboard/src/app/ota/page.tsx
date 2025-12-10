"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { FirmwareReleaseCard } from "@/components/ota/firmware-release-card";
import { getFirmwareReleases, getDeployments } from "@/lib/ota/client";
import type { FirmwareRelease, Deployment } from "@/lib/ota/types";

export default function OtaPage() {
  const router = useRouter();
  const [releases, setReleases] = useState<FirmwareRelease[]>([]);
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showUpload, setShowUpload] = useState(false);

  useEffect(() => {
    // Batch state updates to avoid cascading renders
    Promise.resolve().then(() => {
      setLoading(true);
      setError(null);
    });
    
    Promise.all([
      getFirmwareReleases(1, 6),
      getDeployments(1, 3),
    ])
      .then(([rels, deps]) => {
        setReleases(rels.releases);
        setDeployments(deps.deployments);
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : "Failed to load OTA data");
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  async function handleUpload(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const formData = new FormData(e.currentTarget);
    const version = formData.get("version") as string;
    const templateId = formData.get("templateId") as string;
    const binary = formData.get("binary") as File;

    if (!version || !templateId || !binary) return;

    try {
      await fetch(`${process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080"}/api/v1/ota/releases`, {
        method: "POST",
        body: formData,
      });
      setShowUpload(false);
      // Refresh releases
      const rels = await getFirmwareReleases(1, 6);
      setReleases(rels.releases);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload failed");
    }
  }

  return (
    <DashboardShell title="OTA Management">
      <div className="space-y-6">
        {loading && (
          <div className="flex items-center justify-center py-12">
            <p className="text-sm text-zinc-600">Loading OTA data...</p>
          </div>
        )}

        {error && (
          <div className="rounded-lg border border-red-200 bg-red-50 p-4">
            <p className="text-sm text-red-600">{error}</p>
          </div>
        )}

        {!loading && !error && (
          <>
            {/* Quick actions */}
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                onClick={() => setShowUpload(true)}
                className="px-4 py-2 text-sm font-medium text-white bg-zinc-900 rounded-md hover:bg-zinc-800"
              >
                + Upload Release
              </button>
              <button
                type="button"
                onClick={() => router.push("/ota/deployments")}
                className="px-4 py-2 text-sm font-medium text-zinc-700 border border-zinc-300 rounded-md hover:bg-zinc-50"
              >
                View Deployments
              </button>
            </div>

            {/* Recent Releases */}
            <div>
              <h2 className="text-base font-semibold text-zinc-900 mb-4">Recent Releases</h2>
              {releases.length === 0 ? (
                <div className="rounded-lg border border-zinc-200 bg-white p-8 text-center">
                  <p className="text-sm text-zinc-600 mb-4">No firmware releases yet.</p>
                  <button
                    type="button"
                    onClick={() => setShowUpload(true)}
                    className="px-4 py-2 text-sm font-medium text-white bg-zinc-900 rounded-md hover:bg-zinc-800"
                  >
                    Upload first release
                  </button>
                </div>
              ) : (
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                  {releases.map((release) => (
                    <FirmwareReleaseCard key={release.id} release={release} />
                  ))}
                </div>
              )}
            </div>

            {/* Recent Deployments */}
            {deployments.length > 0 && (
              <div>
                <h2 className="text-base font-semibold text-zinc-900 mb-4">Recent Deployments</h2>
                <div className="space-y-3">
                  {deployments.map((dep) => (
                    <div key={dep.id} className="rounded-lg border border-zinc-200 bg-white p-4">
                      <div className="flex items-start justify-between">
                        <div>
                          <h3 className="text-sm font-medium text-zinc-900">{dep.name}</h3>
                          <p className="text-xs text-zinc-600">
                            {dep.firmwareVersion} • {dep.targetGroups.length} target groups
                          </p>
                          <p className="text-xs text-zinc-600 mt-1">
                            {new Date(dep.createdAt).toLocaleString()}
                          </p>
                        </div>
                        <div className="flex items-center gap-2">
                          <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                            dep.status === "completed" ? "bg-green-100 text-green-800" :
                            dep.status === "running" ? "bg-blue-100 text-blue-800" :
                            dep.status === "failed" ? "bg-red-100 text-red-800" :
                            "bg-zinc-100 text-zinc-800"
                          }`}>
                            {dep.status}
                          </span>
                          <button
                            type="button"
                            onClick={() => router.push(`/ota/deployments/${dep.id}`)}
                            className="text-xs text-zinc-700 hover:text-zinc-900"
                          >
                            View
                          </button>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </>
        )}

        {/* Upload Modal */}
        {showUpload && (
          <div className="fixed inset-0 bg-zinc-900 bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg max-w-md w-full p-6">
              <h3 className="text-lg font-semibold text-zinc-900 mb-4">Upload Firmware Release</h3>
              <form onSubmit={handleUpload} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-zinc-700 mb-1">Version</label>
                  <input
                    name="version"
                    type="text"
                    required
                    className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-zinc-700 mb-1">Template ID</label>
                  <input
                    name="templateId"
                    type="text"
                    required
                    className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-zinc-700 mb-1">Description</label>
                  <textarea
                    name="description"
                    rows={2}
                    className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-zinc-700 mb-1">Binary (.bin/.hex)</label>
                  <input
                    name="binary"
                    type="file"
                    accept=".bin,.hex"
                    required
                    className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                  />
                </div>
                <div className="flex gap-2 pt-2">
                  <button
                    type="submit"
                    className="flex-1 px-4 py-2 text-sm font-medium text-white bg-zinc-900 rounded-md hover:bg-zinc-800"
                  >
                    Upload
                  </button>
                  <button
                    type="button"
                    onClick={() => setShowUpload(false)}
                    className="flex-1 px-4 py-2 text-sm font-medium text-zinc-700 border border-zinc-300 rounded-md hover:bg-zinc-50"
                  >
                    Cancel
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
