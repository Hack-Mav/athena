export interface Device {
  id: string;
  name: string;
  description?: string;
  templateId: string;
  templateName: string;
  status: DeviceStatus;
  firmwareVersion?: string;
  lastSeen: string; // ISO timestamp
  createdAt: string; // ISO timestamp
  updatedAt: string; // ISO timestamp
  config: Record<string, unknown>; // Applied configuration
  metadata?: Record<string, string | number | boolean>;
  serialPort?: string;
  board?: string;
  mcu?: string;
}

export type DeviceStatus = "offline" | "online" | "compiling" | "flashing" | "error";

export interface DeviceListResponse {
  devices: Device[];
  total: number;
  page: number;
  pageSize: number;
}

export interface DeviceSearchParams {
  q?: string;
  status?: DeviceStatus;
  templateId?: string;
  page?: number;
  pageSize?: number;
  sortBy?: "name" | "createdAt" | "updatedAt" | "lastSeen";
  sortOrder?: "asc" | "desc";
}

export interface ProvisioningJob {
  id: string;
  deviceId: string;
  templateId: string;
  config: Record<string, unknown>;
  steps: ProvisioningStep[];
  status: JobStatus;
  startedAt?: string;
  completedAt?: string;
  error?: string;
}

export type JobStatus = "pending" | "running" | "completed" | "failed";

export interface ProvisioningStep {
  id: string;
  name: string;
  description?: string;
  status: StepStatus;
  startedAt?: string;
  completedAt?: string;
  error?: string;
  logs?: string[];
  progress?: number; // 0-100
}

export type StepStatus = "pending" | "running" | "completed" | "failed";

export interface CreateDeviceRequest {
  name: string;
  description?: string;
  templateId: string;
  config: Record<string, unknown>;
  serialPort?: string;
  board?: string;
  mcu?: string;
  metadata?: Record<string, string | number | boolean>;
}

export interface UpdateDeviceRequest {
  name?: string;
  description?: string;
  config?: Record<string, unknown>;
  metadata?: Record<string, string | number | boolean>;
}

export interface ProvisionRequest {
  deviceId: string;
  templateId?: string;
  config?: Record<string, unknown>;
}

export interface SerialMessage {
  timestamp: string; // ISO timestamp
  direction: "tx" | "rx";
  data: string;
}

export interface SerialPortInfo {
  port: string;
  description?: string;
  manufacturer?: string;
  serialNumber?: string;
  vendorId?: string;
  productId?: string;
}

export interface ApiError {
  error: string;
  code: string;
}
