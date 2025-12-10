"use client";

import { useState, useEffect } from "react";
import type { Device, ProvisioningJob, JobStatus, StepStatus } from "@/lib/devices/types";
import { getProvisioningJob } from "@/lib/devices/client";

interface ProvisioningWorkflowProps {
  device: Device;
  initialJob?: ProvisioningJob | null;
  onJobChange: (job: ProvisioningJob | null) => void;
}

export function ProvisioningWorkflow({ device, initialJob, onJobChange }: ProvisioningWorkflowProps) {
  const [job, setJob] = useState<ProvisioningJob | null>(initialJob || null);
  const [templateId, setTemplateId] = useState(device.templateId);
  const [config, setConfig] = useState(device.config);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Poll job status if running
  useEffect(() => {
    if (!job || (job.status !== "running" && job.status !== "pending")) return;
    const interval = setInterval(async () => {
      try {
        const updated = await getProvisioningJob(job.id);
        setJob(updated);
        onJobChange(updated);
        if (updated.status === "completed" || updated.status === "failed") {
          clearInterval(interval);
        }
      } catch {
        // ignore polling errors
      }
    }, 2000);
    return () => clearInterval(interval);
  }, [job, onJobChange]);

  async function handleStart() {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080"}/api/v1/devices/provision`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ deviceId: device.id, templateId, config }),
      });
      if (!res.ok) throw new Error("Failed to start provisioning");
      const newJob: ProvisioningJob = await res.json();
      setJob(newJob);
      onJobChange(newJob);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to start provisioning");
    } finally {
      setLoading(false);
    }
  }

  function statusColor(status: JobStatus | StepStatus) {
    switch (status) {
      case "completed":
        return "text-green-600";
      case "running":
        return "text-blue-600";
      case "failed":
        return "text-red-600";
      default:
        return "text-zinc-600";
    }
  }

  return (
    <div className="space-y-6">
      {/* Configuration */}
      <div className="rounded-lg border border-zinc-200 bg-white p-4">
        <h2 className="text-base font-semibold text-zinc-900 mb-4">Provisioning Configuration</h2>
        <div className="space-y-4">
          <div>
            <label htmlFor="template-id" className="block text-sm font-medium text-zinc-700 mb-1">
              Template ID
            </label>
            <input
              id="template-id"
              type="text"
              value={templateId}
              onChange={(e) => setTemplateId(e.target.value)}
              className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
            />
          </div>
          <div>
            <label htmlFor="config-json" className="block text-sm font-medium text-zinc-700 mb-1">
              Configuration (JSON)
            </label>
            <textarea
              id="config-json"
              value={JSON.stringify(config, null, 2)}
              onChange={(e) => {
                try {
                  setConfig(JSON.parse(e.target.value));
                } catch {
                  // ignore invalid JSON for now
                }
              }}
              rows={6}
              className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm font-mono outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
            />
          </div>
          <button
            type="button"
            onClick={handleStart}
            disabled={loading || !!job}
            className="px-4 py-2 text-sm font-medium text-white bg-zinc-900 rounded-md hover:bg-zinc-800 disabled:opacity-50"
          >
            {loading ? "Starting..." : "Start Provisioning"}
          </button>
          {error && (
            <p className="text-sm text-red-600">{error}</p>
          )}
        </div>
      </div>

      {/* Job progress */}
      {job && (
        <div className="rounded-lg border border-zinc-200 bg-white p-4">
          <h2 className="text-base font-semibold text-zinc-900 mb-4">Job Progress</h2>
          <div className="space-y-4">
            <div className="flex items-center justify-between text-sm">
              <span>Status</span>
              <span className={`font-medium ${statusColor(job.status)}`}>{job.status}</span>
            </div>
            {job.startedAt && (
              <div className="flex items-center justify-between text-sm">
                <span>Started</span>
                <span>{new Date(job.startedAt).toLocaleString()}</span>
              </div>
            )}
            {job.completedAt && (
              <div className="flex items-center justify-between text-sm">
                <span>Completed</span>
                <span>{new Date(job.completedAt).toLocaleString()}</span>
              </div>
            )}
            {job.error && (
              <div className="rounded-md border border-red-200 bg-red-50 p-3">
                <p className="text-sm text-red-600">{job.error}</p>
              </div>
            )}

            {/* Steps */}
            {job.steps.length > 0 && (
              <div className="space-y-3">
                <h3 className="text-sm font-medium text-zinc-900">Steps</h3>
                {job.steps.map((step) => (
                  <div key={step.id} className="rounded-md border border-zinc-200 p-3">
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-sm font-medium text-zinc-900">{step.name}</span>
                      <span className={`text-xs font-medium ${statusColor(step.status)}`}>{step.status}</span>
                    </div>
                    {step.description && (
                      <p className="text-xs text-zinc-600 mb-2">{step.description}</p>
                    )}
                    {step.progress !== undefined && (
                      <div className="w-full bg-zinc-200 rounded-full h-1.5">
                        <div
                          className="bg-blue-600 h-1.5 rounded-full transition-all"
                          style={{ width: `${step.progress}%` }}
                        />
                      </div>
                    )}
                    {step.error && (
                      <p className="text-xs text-red-600 mt-2">{step.error}</p>
                    )}
                    {step.logs && step.logs.length > 0 && (
                      <details className="mt-2">
                        <summary className="text-xs text-zinc-600 cursor-pointer">View logs</summary>
                        <pre className="text-xs bg-zinc-100 p-2 rounded mt-1 overflow-x-auto">
                          {step.logs.join("\n")}
                        </pre>
                      </details>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
