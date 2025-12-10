import type {
  Device,
  DeviceListResponse,
  DeviceSearchParams,
  CreateDeviceRequest,
  UpdateDeviceRequest,
  ProvisioningJob,
  ProvisionRequest,
  SerialMessage,
  SerialPortInfo,
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

// Devices
export async function getDevices(
  params?: DeviceSearchParams
): Promise<DeviceListResponse> {
  const search = new URLSearchParams();
  if (params) {
    if (params.q) search.set("q", params.q);
    if (params.status) search.set("status", params.status);
    if (params.templateId) search.set("templateId", params.templateId);
    if (params.page != null) search.set("page", String(params.page));
    if (params.pageSize != null) search.set("pageSize", String(params.pageSize));
    if (params.sortBy) search.set("sortBy", params.sortBy);
    if (params.sortOrder) search.set("sortOrder", params.sortOrder);
  }
  const res = await fetch(
    `${API_BASE_URL}/api/v1/devices?${search.toString()}`
  );
  return handleResponse<DeviceListResponse>(res);
}

export async function getDevice(id: string): Promise<Device> {
  const res = await fetch(`${API_BASE_URL}/api/v1/devices/${encodeURIComponent(id)}`);
  return handleResponse<Device>(res);
}

export async function createDevice(body: CreateDeviceRequest): Promise<Device> {
  const res = await fetch(`${API_BASE_URL}/api/v1/devices`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return handleResponse<Device>(res);
}

export async function updateDevice(id: string, body: UpdateDeviceRequest): Promise<Device> {
  const res = await fetch(`${API_BASE_URL}/api/v1/devices/${encodeURIComponent(id)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return handleResponse<Device>(res);
}

export async function deleteDevice(id: string): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1/devices/${encodeURIComponent(id)}`, {
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

// Provisioning
export async function provisionDevice(body: ProvisionRequest): Promise<ProvisioningJob> {
  const res = await fetch(`${API_BASE_URL}/api/v1/devices/provision`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return handleResponse<ProvisioningJob>(res);
}

export async function getProvisioningJob(jobId: string): Promise<ProvisioningJob> {
  const res = await fetch(`${API_BASE_URL}/api/v1/provisioning/jobs/${encodeURIComponent(jobId)}`);
  return handleResponse<ProvisioningJob>(res);
}

// Serial
export async function getSerialPorts(): Promise<SerialPortInfo[]> {
  const res = await fetch(`${API_BASE_URL}/api/v1/serial/ports`);
  return handleResponse<SerialPortInfo[]>(res);
}

export async function getSerialMessages(deviceId: string, since?: string): Promise<SerialMessage[]> {
  const search = since ? `?since=${encodeURIComponent(since)}` : "";
  const res = await fetch(`${API_BASE_URL}/api/v1/devices/${encodeURIComponent(deviceId)}/serial${search}`);
  return handleResponse<SerialMessage[]>(res);
}
