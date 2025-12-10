"use client";

import type { Deployment, DeploymentProgress } from "@/lib/ota/types";

interface DeploymentRolloutProps {
  deployment: Deployment;
  progress: DeploymentProgress | null;
  onAction: (action: "start" | "pause" | "resume" | "cancel") => void;
  actionLoading: string | null;
}

export function DeploymentRollout({ deployment, progress, onAction, actionLoading }: DeploymentRolloutProps) {
  const total = progress?.totalDevices ?? 0;
  const completed = progress?.completedDevices ?? 0;
  const failed = progress?.failedDevices ?? 0;
  const pending = progress?.pendingDevices ?? 0;
  const percentage = total > 0 ? (completed / total) * 100 : 0;

  function canStart() {
    return deployment.status === "draft";
  }

  function canPause() {
    return deployment.status === "running";
  }

  function canResume() {
    return deployment.status === "paused";
  }

  function canCancel() {
    return ["draft", "pending", "running", "paused"].includes(deployment.status);
  }

  return (
    <div className="space-y-6">
      {/* Controls */}
      <div className="rounded-lg border border-zinc-200 bg-white p-4">
        <h2 className="text-base font-semibold text-zinc-900 mb-4">Controls</h2>
        <div className="flex flex-wrap gap-2">
          {canStart() && (
            <button
              type="button"
              onClick={() => onAction("start")}
              disabled={actionLoading === "start"}
              className="px-4 py-2 text-sm font-medium text-white bg-green-700 rounded-md hover:bg-green-800 disabled:opacity-50"
            >
              {actionLoading === "start" ? "Starting..." : "Start Deployment"}
            </button>
          )}
          {canPause() && (
            <button
              type="button"
              onClick={() => onAction("pause")}
              disabled={actionLoading === "pause"}
              className="px-4 py-2 text-sm font-medium text-yellow-700 border border-yellow-300 rounded-md hover:bg-yellow-50 disabled:opacity-50"
            >
              {actionLoading === "pause" ? "Pausing..." : "Pause"}
            </button>
          )}
          {canResume() && (
            <button
              type="button"
              onClick={() => onAction("resume")}
              disabled={actionLoading === "resume"}
              className="px-4 py-2 text-sm font-medium text-blue-700 border border-blue-300 rounded-md hover:bg-blue-50 disabled:opacity-50"
            >
              {actionLoading === "resume" ? "Resuming..." : "Resume"}
            </button>
          )}
          {canCancel() && (
            <button
              type="button"
              onClick={() => onAction("cancel")}
              disabled={actionLoading === "cancel"}
              className="px-4 py-2 text-sm font-medium text-red-700 border border-red-300 rounded-md hover:bg-red-50 disabled:opacity-50"
            >
              {actionLoading === "cancel" ? "Cancelling..." : "Cancel"}
            </button>
          )}
        </div>
      </div>

      {/* Progress Overview */}
      <div className="rounded-lg border border-zinc-200 bg-white p-4">
        <h2 className="text-base font-semibold text-zinc-900 mb-4">Progress Overview</h2>
        {total === 0 ? (
          <p className="text-sm text-zinc-600">No devices targeted.</p>
        ) : (
          <div className="space-y-4">
            {/* Overall progress bar */}
            <div>
              <div className="flex items-center justify-between text-sm mb-1">
                <span className="font-medium text-zinc-700">Overall</span>
                <span className="text-zinc-600">{completed}/{total} ({percentage.toFixed(1)}%)</span>
              </div>
              <div className="w-full bg-zinc-200 rounded-full h-2">
                <div
                  className="bg-green-600 h-2 rounded-full transition-all"
                  style={{ width: `${percentage}%` }}
                />
              </div>
            </div>

            {/* Status breakdown */}
            <div className="grid grid-cols-3 gap-4 text-center text-sm">
              <div>
                <div className="text-2xl font-semibold text-green-600">{completed}</div>
                <div className="text-zinc-600">Completed</div>
              </div>
              <div>
                <div className="text-2xl font-semibold text-yellow-600">{pending}</div>
                <div className="text-zinc-600">Pending</div>
              </div>
              <div>
                <div className="text-2xl font-semibold text-red-600">{failed}</div>
                <div className="text-zinc-600">Failed</div>
              </div>
            </div>

            {/* Timeline */}
            {progress?.startedAt && (
              <div className="text-xs text-zinc-600 space-y-1">
                <div>Started: {new Date(progress.startedAt).toLocaleString()}</div>
                {progress.estimatedCompletionAt && (
                  <div>Est. completion: {new Date(progress.estimatedCompletionAt).toLocaleString()}</div>
                )}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Phases (for phased/canary) */}
      {deployment.strategy !== "immediate" && progress?.phases && (
        <div className="rounded-lg border border-zinc-200 bg-white p-4">
          <h2 className="text-base font-semibold text-zinc-900 mb-4">Phases</h2>
          <div className="space-y-3">
            {progress.phases.map((phase, idx) => (
              <div key={idx} className="border-l-2 border-zinc-200 pl-4">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm font-medium text-zinc-900">{phase.phase.name}</span>
                  <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                    phase.status === "completed" ? "bg-green-100 text-green-800" :
                    phase.status === "running" ? "bg-blue-100 text-blue-800" :
                    phase.status === "failed" ? "bg-red-100 text-red-800" :
                    "bg-zinc-100 text-zinc-800"
                  }`}>
                    {phase.status}
                  </span>
                </div>
                <p className="text-xs text-zinc-600 mb-2">
                  {phase.completedDevices}/{phase.totalDevices} devices
                </p>
                <div className="w-full bg-zinc-200 rounded-full h-1.5">
                  <div
                    className="bg-blue-600 h-1.5 rounded-full transition-all"
                    style={{ width: phase.totalDevices > 0 ? (phase.completedDevices / phase.totalDevices) * 100 : 0 }}
                  />
                </div>
                {phase.phase.durationMinutes && (
                  <p className="text-xs text-zinc-500 mt-1">
                    Duration: {phase.phase.durationMinutes}m
                  </p>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
