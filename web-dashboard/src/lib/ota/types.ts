export interface FirmwareRelease {
  id: string;
  version: string;
  description?: string;
  templateId: string;
  templateName: string;
  checksum: string; // SHA256
  size: number; // bytes
  binaryUrl?: string; // download URL
  createdAt: string; // ISO timestamp
  createdBy: string;
  metadata?: Record<string, string | number | boolean>;
}

export interface CreateFirmwareReleaseRequest {
  version: string;
  description?: string;
  templateId: string;
  binary: File; // multipart upload
  metadata?: Record<string, string | number | boolean>;
}

export interface Deployment {
  id: string;
  name: string;
  description?: string;
  firmwareReleaseId: string;
  firmwareVersion: string;
  targetGroups: DeploymentTargetGroup[];
  strategy: DeploymentStrategy;
  status: DeploymentStatus;
  createdAt: string; // ISO timestamp;
  startedAt?: string;
  completedAt?: string;
  createdBy: string;
  rolloutConfig?: RolloutConfig;
}

export interface DeploymentTargetGroup {
  deviceGroupId?: string;
  deviceIds?: string[];
  name: string;
  criteria?: Record<string, unknown>; // e.g., tags, template, version
}

export type DeploymentStrategy = "immediate" | "phased" | "canary";

export interface RolloutConfig {
  phases?: RolloutPhase[];
  canaryPercentage?: number;
  pauseAfterPhase?: boolean;
  rollbackOnFailure?: boolean;
}

export interface RolloutPhase {
  name: string;
  order: number;
  percentage: number; // % of devices
  durationMinutes?: number; // wait time before next phase
  criteria?: Record<string, unknown>;
}

export type DeploymentStatus = "draft" | "pending" | "running" | "paused" | "completed" | "failed" | "rolled_back";

export interface DeploymentProgress {
  deploymentId: string;
  status: DeploymentStatus;
  totalDevices: number;
  completedDevices: number;
  failedDevices: number;
  pendingDevices: number;
  currentPhase?: string;
  phases?: DeploymentPhaseProgress[];
  startedAt?: string;
  estimatedCompletionAt?: string;
}

export interface DeploymentPhaseProgress {
  phase: RolloutPhase;
  totalDevices: number;
  completedDevices: number;
  failedDevices: number;
  pendingDevices: number;
  status: "pending" | "running" | "completed" | "failed";
  startedAt?: string;
  completedAt?: string;
}

export interface DeviceUpdateStatus {
  deviceId: string;
  deviceName: string;
  deploymentId: string;
  status: UpdateStatus;
  currentVersion?: string;
  targetVersion: string;
  startedAt?: string;
  completedAt?: string;
  error?: string;
  logs?: string[];
}

export type UpdateStatus = "pending" | "downloading" | "installing" | "rebooting" | "completed" | "failed" | "rollback_initiated" | "rolled_back";

export interface RollbackRequest {
  deploymentId: string;
  reason?: string;
  toVersion?: string; // if not provided, rollback to previous stable
}

export interface Rollback {
  id: string;
  deploymentId: string;
  reason?: string;
  fromVersion: string;
  toVersion: string;
  status: "pending" | "running" | "completed" | "failed";
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  error?: string;
}

export interface FirmwareReleaseListResponse {
  releases: FirmwareRelease[];
  total: number;
  page: number;
  pageSize: number;
}

export interface DeploymentListResponse {
  deployments: Deployment[];
  total: number;
  page: number;
  pageSize: number;
}

export interface ApiError {
  error: string;
  code: string;
}
