import type {
  FirmwareRelease,
  FirmwareReleaseListResponse,
  CreateFirmwareReleaseRequest,
  Deployment,
  DeploymentListResponse,
  DeploymentProgress,
  DeviceUpdateStatus,
  RollbackRequest,
  Rollback,
  ApiError,
} from "./types";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let err: ApiError | undefined;
    try {
      err = (await res.json()) as ApiError;
    } catch {}
    throw new Error(err?.error ?? `Request failed with status ${res.status}`);
  }
  return (await res.json()) as T;
}

// Firmware Releases
export async function getFirmwareReleases(page = 1, pageSize = 20): Promise<FirmwareReleaseListResponse> {
  const res = await fetch(`${API_BASE_URL}/api/v1/ota/releases?page=${page}&pageSize=${pageSize}`);
  return handleResponse<FirmwareReleaseListResponse>(res);
}

export async function getFirmwareRelease(id: string): Promise<FirmwareRelease> {
  const res = await fetch(`${API_BASE_URL}/api/v1/ota/releases/${encodeURIComponent(id)}`);
  return handleResponse<FirmwareRelease>(res);
}

export async function createFirmwareRelease(body: CreateFirmwareReleaseRequest): Promise<FirmwareRelease> {
  const formData = new FormData();
  formData.append("version", body.version);
  formData.append("templateId", body.templateId);
  if (body.description) formData.append("description", body.description);
  if (body.binary) formData.append("binary", body.binary);
  if (body.metadata) formData.append("metadata", JSON.stringify(body.metadata));

  const res = await fetch(`${API_BASE_URL}/api/v1/ota/releases`, {
    method: "POST",
    body: formData,
  });
  return handleResponse<FirmwareRelease>(res);
}

export async function deleteFirmwareRelease(id: string): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1/ota/releases/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
  if (!res.ok) {
    let err: ApiError | undefined;
    try {
      err = (await res.json()) as ApiError;
    } catch {}
    throw new Error(err?.error ?? `Delete failed with status ${res.status}`);
  }
}

// Deployments
export async function getDeployments(page = 1, pageSize = 20): Promise<DeploymentListResponse> {
  const res = await fetch(`${API_BASE_URL}/api/v1/ota/deployments?page=${page}&pageSize=${pageSize}`);
  return handleResponse<DeploymentListResponse>(res);
}

export async function getDeployment(id: string): Promise<Deployment> {
  const res = await fetch(`${API_BASE_URL}/api/v1/ota/deployments/${encodeURIComponent(id)}`);
  return handleResponse<Deployment>(res);
}

export async function createDeployment(deployment: Omit<Deployment, "id" | "status" | "createdAt" | "createdBy">): Promise<Deployment> {
  const res = await fetch(`${API_BASE_URL}/api/v1/ota/deployments`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(deployment),
  });
  return handleResponse<Deployment>(res);
}

export async function startDeployment(id: string): Promise<Deployment> {
  const res = await fetch(`${API_BASE_URL}/api/v1/ota/deployments/${encodeURIComponent(id)}/start`, {
    method: "POST",
  });
  return handleResponse<Deployment>(res);
}

export async function pauseDeployment(id: string): Promise<Deployment> {
  const res = await fetch(`${API_BASE_URL}/api/v1/ota/deployments/${encodeURIComponent(id)}/pause`, {
    method: "POST",
  });
  return handleResponse<Deployment>(res);
}

export async function resumeDeployment(id: string): Promise<Deployment> {
  const res = await fetch(`${API_BASE_URL}/api/v1/ota/deployments/${encodeURIComponent(id)}/resume`, {
    method: "POST",
  });
  return handleResponse<Deployment>(res);
}

export async function cancelDeployment(id: string): Promise<Deployment> {
  const res = await fetch(`${API_BASE_URL}/api/v1/ota/deployments/${encodeURIComponent(id)}/cancel`, {
    method: "POST",
  });
  return handleResponse<Deployment>(res);
}

// Progress & Status
export async function getDeploymentProgress(id: string): Promise<DeploymentProgress> {
  const res = await fetch(`${API_BASE_URL}/api/v1/ota/deployments/${encodeURIComponent(id)}/progress`);
  return handleResponse<DeploymentProgress>(res);
}

export async function getDeploymentDeviceStatuses(deploymentId: string): Promise<DeviceUpdateStatus[]> {
  const res = await fetch(`${API_BASE_URL}/api/v1/ota/deployments/${encodeURIComponent(deploymentId)}/devices`);
  return handleResponse<DeviceUpdateStatus[]>(res);
}

// Rollback
export async function initiateRollback(body: RollbackRequest): Promise<Rollback> {
  const res = await fetch(`${API_BASE_URL}/api/v1/ota/deployments/${encodeURIComponent(body.deploymentId)}/rollback`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ reason: body.reason, toVersion: body.toVersion }),
  });
  return handleResponse<Rollback>(res);
}

export async function getRollback(rollbackId: string): Promise<Rollback> {
  const res = await fetch(`${API_BASE_URL}/api/v1/ota/rollbacks/${encodeURIComponent(rollbackId)}`);
  return handleResponse<Rollback>(res);
}
