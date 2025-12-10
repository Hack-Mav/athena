"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { DeploymentRollout } from "@/components/ota/deployment-rollout";
import { getDeployment, getDeploymentProgress, startDeployment, pauseDeployment, resumeDeployment, cancelDeployment, initiateRollback } from "@/lib/ota/client";
import type { Deployment, DeploymentProgress } from "@/lib/ota/types";

export default function DeploymentDetailPage() {
  const params = useParams();
  const deploymentId = typeof params.id === "string" ? params.id : params.id?.[0];

  const [deployment, setDeployment] = useState<Deployment | null>(null);
  const [progress, setProgress] = useState<DeploymentProgress | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<"overview" | "progress" | "devices" | "rollback">("overview");

  useEffect(() => {
    if (!deploymentId) return;
    setLoading(true);
    setError(null);
    Promise.all([
      getDeployment(deploymentId!),
      getDeploymentProgress(deploymentId!),
    ])
      .then(([dep, prog]) => {
        setDeployment(dep);
        setProgress(prog);
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : "Failed to load deployment");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [deploymentId]);

  // Poll progress if running
  useEffect(() => {
    if (!deployment || (deployment.status !== "running" && deployment.status !== "paused")) return;
    const interval = setInterval(async () => {
      try {
        const prog = await getDeploymentProgress(deploymentId!);
        setProgress(prog);
      } catch {
        // ignore polling errors
      }
    }, 3000);
    return () => clearInterval(interval);
  }, [deployment, deploymentId]);

  async function handleAction(action: "start" | "pause" | "resume" | "cancel") {
    if (!deploymentId) return;
    setActionLoading(action);
    try {
      let updated;
      switch (action) {
        case "start":
          updated = await startDeployment(deploymentId!);
          break;
        case "pause":
          updated = await pauseDeployment(deploymentId!);
          break;
        case "resume":
          updated = await resumeDeployment(deploymentId!);
          break;
        case "cancel":
          updated = await cancelDeployment(deploymentId!);
          break;
      }
      setDeployment(updated);
    } catch {
      // ignore errors
    } finally {
      setActionLoading(null);
    }
  }

  async function handleRollback(reason?: string) {
    if (!deploymentId) return;
    setActionLoading("rollback");
    try {
      await initiateRollback({ deploymentId: deploymentId!, reason });
      // Refresh deployment
      const updated = await getDeployment(deploymentId!);
      setDeployment(updated);
    } catch {
      // ignore errors
    } finally {
      setActionLoading(null);
    }
  }

  if (loading) {
    return (
      <DashboardShell title="Deployment Details">
        <div className="flex items-center justify-center py-12">
          <p className="text-sm text-zinc-600">Loading deployment...</p>
        </div>
      </DashboardShell>
    );
  }

  if (error || !deployment) {
    return (
      <DashboardShell title="Deployment Details">
        <div className="rounded-lg border border-red-200 bg-red-50 p-4">
          <p className="text-sm text-red-600">{error || "Deployment not found"}</p>
        </div>
      </DashboardShell>
    );
  }

  return (
    <DashboardShell title={deployment.name}>
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
            onClick={() => setActiveTab("progress")}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === "progress"
                ? "border-zinc-900 text-zinc-900"
                : "border-transparent text-zinc-600 hover:text-zinc-900"
            }`}
          >
            Rollout Progress
          </button>
          <button
            type="button"
            onClick={() => setActiveTab("devices")}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === "devices"
                ? "border-zinc-900 text-zinc-900"
                : "border-transparent text-zinc-600 hover:text-zinc-900"
            }`}
          >
            Device Status
          </button>
          <button
            type="button"
            onClick={() => setActiveTab("rollback")}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === "rollback"
                ? "border-zinc-900 text-zinc-900"
                : "border-transparent text-zinc-600 hover:text-zinc-900"
            }`}
          >
            Rollback
          </button>
        </div>

        {/* Tab content */}
        {activeTab === "overview" && (
          <div className="rounded-lg border border-zinc-200 bg-white p-4">
            <h2 className="text-base font-semibold text-zinc-900 mb-4">Deployment Overview</h2>
            <dl className="space-y-2 text-sm">
              <div className="flex justify-between">
                <dt className="font-medium text-zinc-700">Status</dt>
                <dd className="text-zinc-600">{deployment.status}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="font-medium text-zinc-700">Firmware Version</dt>
                <dd className="text-zinc-600">{deployment.firmwareVersion}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="font-medium text-zinc-700">Strategy</dt>
                <dd className="text-zinc-600">{deployment.strategy}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="font-medium text-zinc-700">Target Groups</dt>
                <dd className="text-zinc-600">{deployment.targetGroups.length}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="font-medium text-zinc-700">Created</dt>
                <dd className="text-zinc-600">{new Date(deployment.createdAt).toLocaleString()}</dd>
              </div>
              {deployment.startedAt && (
                <div className="flex justify-between">
                  <dt className="font-medium text-zinc-700">Started</dt>
                  <dd className="text-zinc-600">{new Date(deployment.startedAt).toLocaleString()}</dd>
                </div>
              )}
              {deployment.completedAt && (
                <div className="flex justify-between">
                  <dt className="font-medium text-zinc-700">Completed</dt>
                  <dd className="text-zinc-600">{new Date(deployment.completedAt).toLocaleString()}</dd>
                </div>
              )}
            </dl>
            {deployment.description && (
              <div className="mt-4">
                <h3 className="text-sm font-medium text-zinc-900 mb-2">Description</h3>
                <p className="text-sm text-zinc-700">{deployment.description}</p>
              </div>
            )}
          </div>
        )}

        {activeTab === "progress" && (
          <DeploymentRollout deployment={deployment} progress={progress} onAction={handleAction} actionLoading={actionLoading} />
        )}

        {activeTab === "devices" && (
          <div className="rounded-lg border border-zinc-200 bg-white p-4">
            <h2 className="text-base font-semibold text-zinc-900 mb-4">Device Update Status</h2>
            <p className="text-sm text-zinc-600">Device-by-device status will be shown here.</p>
          </div>
        )}

        {activeTab === "rollback" && (
          <div className="rounded-lg border border-zinc-200 bg-white p-4">
            <h2 className="text-base font-semibold text-zinc-900 mb-4">Rollback</h2>
            <div className="space-y-4">
              <p className="text-sm text-zinc-600">
                Initiate a rollback to the previous stable firmware version for this deployment.
              </p>
              <div>
                <label className="block text-sm font-medium text-zinc-700 mb-1">Reason (optional)</label>
                <textarea
                  rows={3}
                  className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                  placeholder="Why are you rolling back?"
                />
              </div>
              <button
                type="button"
                onClick={() => {
                  const reason = (document.querySelector("textarea") as HTMLTextAreaElement)?.value;
                  handleRollback(reason);
                }}
                disabled={actionLoading === "rollback"}
                className="px-4 py-2 text-sm font-medium text-red-700 border border-red-300 rounded-md hover:bg-red-50 disabled:opacity-50"
              >
                {actionLoading === "rollback" ? "Rolling back..." : "Initiate Rollback"}
              </button>
            </div>
          </div>
        )}
      </div>
    </DashboardShell>
  );
}
