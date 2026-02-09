import type {
  TelemetryQuery,
  TelemetryResponse,
  DeviceHealth,
  Alert,
  AlertListResponse,
  AlertSearchParams,
  AlertRule,
  CreateAlertRuleRequest,
  UpdateAlertRuleRequest,
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

// Telemetry
export async function queryTelemetry(query: TelemetryQuery): Promise<TelemetryResponse> {
  const res = await fetch(`${API_BASE_URL}/api/v1/telemetry/query`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(query),
  });
  return handleResponse<TelemetryResponse>(res);
}

export async function getDeviceHealth(deviceId: string): Promise<DeviceHealth> {
  const res = await fetch(`${API_BASE_URL}/api/v1/devices/${encodeURIComponent(deviceId)}/health`);
  return handleResponse<DeviceHealth>(res);
}

export async function getDeviceTelemetry(deviceId: string, timeRange?: string): Promise<TelemetryResponse> {
  const search = new URLSearchParams();
  if (timeRange) search.set("timeRange", timeRange);
  const res = await fetch(`${API_BASE_URL}/api/v1/devices/${encodeURIComponent(deviceId)}/telemetry?${search.toString()}`);
  return handleResponse<TelemetryResponse>(res);
}

export async function getDeviceAlerts(deviceId: string, params?: AlertSearchParams): Promise<AlertListResponse> {
  const search = new URLSearchParams();
  search.set("deviceId", deviceId);
  if (params) {
    if (params.severity) search.set("severity", params.severity);
    if (params.status) search.set("status", params.status);
    if (params.type) search.set("type", params.type);
    if (params.page != null) search.set("page", String(params.page));
    if (params.pageSize != null) search.set("pageSize", String(params.pageSize));
    if (params.sortBy) search.set("sortBy", params.sortBy);
    if (params.sortOrder) search.set("sortOrder", params.sortOrder);
  }
  const res = await fetch(`${API_BASE_URL}/api/v1/alerts?${search.toString()}`);
  return handleResponse<AlertListResponse>(res);
}

export async function getAllDevicesHealth(): Promise<DeviceHealth[]> {
  const res = await fetch(`${API_BASE_URL}/api/v1/devices/health`);
  return handleResponse<DeviceHealth[]>(res);
}

// Alerts
export async function getAlerts(params?: AlertSearchParams): Promise<AlertListResponse> {
  const search = new URLSearchParams();
  if (params) {
    if (params.severity) search.set("severity", params.severity);
    if (params.status) search.set("status", params.status);
    if (params.deviceId) search.set("deviceId", params.deviceId);
    if (params.type) search.set("type", params.type);
    if (params.page != null) search.set("page", String(params.page));
    if (params.pageSize != null) search.set("pageSize", String(params.pageSize));
    if (params.sortBy) search.set("sortBy", params.sortBy);
    if (params.sortOrder) search.set("sortOrder", params.sortOrder);
  }
  const res = await fetch(`${API_BASE_URL}/api/v1/alerts?${search.toString()}`);
  return handleResponse<AlertListResponse>(res);
}

export async function acknowledgeAlert(alertId: string): Promise<Alert> {
  const res = await fetch(`${API_BASE_URL}/api/v1/alerts/${encodeURIComponent(alertId)}/acknowledge`, {
    method: "POST",
  });
  return handleResponse<Alert>(res);
}

export async function resolveAlert(alertId: string): Promise<Alert> {
  const res = await fetch(`${API_BASE_URL}/api/v1/alerts/${encodeURIComponent(alertId)}/resolve`, {
    method: "POST",
  });
  return handleResponse<Alert>(res);
}

// Alert Rules
export async function getAlertRules(): Promise<AlertRule[]> {
  const res = await fetch(`${API_BASE_URL}/api/v1/alert-rules`);
  return handleResponse<AlertRule[]>(res);
}

export async function createAlertRule(body: CreateAlertRuleRequest): Promise<AlertRule> {
  const res = await fetch(`${API_BASE_URL}/api/v1/alert-rules`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return handleResponse<AlertRule>(res);
}

export async function updateAlertRule(id: string, body: UpdateAlertRuleRequest): Promise<AlertRule> {
  const res = await fetch(`${API_BASE_URL}/api/v1/alert-rules/${encodeURIComponent(id)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return handleResponse<AlertRule>(res);
}

export async function deleteAlertRule(id: string): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1/alert-rules/${encodeURIComponent(id)}`, {
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
