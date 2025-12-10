export interface TelemetryPoint {
  timestamp: string; // ISO timestamp
  value: number;
  quality?: "good" | "uncertain" | "bad";
}

export interface TelemetrySeries {
  deviceId: string;
  metric: string;
  unit: string;
  points: TelemetryPoint[];
}

export interface TelemetryQuery {
  deviceIds?: string[];
  metrics?: string[];
  startTime?: string; // ISO timestamp
  endTime?: string; // ISO timestamp
  aggregation?: "raw" | "avg" | "min" | "max" | "sum" | "count";
  interval?: string; // e.g., "1m", "5m", "1h"
  limit?: number;
}

export interface TelemetryResponse {
  series: TelemetrySeries[];
}

export interface DeviceHealth {
  deviceId: string;
  status: "healthy" | "warning" | "critical" | "offline";
  lastSeen: string;
  uptime?: number; // seconds
  freeMemory?: number; // bytes
  cpuUsage?: number; // percent
  signalStrength?: number; // dBm, RSSI, etc.
  batteryLevel?: number; // percent, if applicable
  errors?: number; // recent error count
}

export interface Alert {
  id: string;
  deviceId: string;
  deviceName: string;
  severity: "info" | "warning" | "critical";
  type: string; // e.g., "offline", "high_cpu", "low_memory", "sensor_fault"
  message: string;
  details?: Record<string, unknown>;
  status: "active" | "acknowledged" | "resolved";
  createdAt: string; // ISO timestamp
  acknowledgedAt?: string;
  acknowledgedBy?: string;
  resolvedAt?: string;
}

export interface AlertListResponse {
  alerts: Alert[];
  total: number;
  page: number;
  pageSize: number;
}

export interface AlertSearchParams {
  severity?: "info" | "warning" | "critical";
  status?: "active" | "acknowledged" | "resolved";
  deviceId?: string;
  type?: string;
  page?: number;
  pageSize?: number;
  sortBy?: "createdAt" | "severity";
  sortOrder?: "asc" | "desc";
}

export interface AlertRule {
  id: string;
  name: string;
  description?: string;
  enabled: boolean;
  deviceId?: string; // empty = global rule
  metric: string;
  condition: "gt" | "lt" | "eq" | "ne" | "gte" | "lte";
  threshold: number;
  severity: "info" | "warning" | "critical";
  cooldownMinutes: number;
  createdAt: string;
  updatedAt: string;
}

export interface CreateAlertRuleRequest {
  name: string;
  description?: string;
  enabled?: boolean;
  deviceId?: string;
  metric: string;
  condition: "gt" | "lt" | "eq" | "ne" | "gte" | "lte";
  threshold: number;
  severity: "info" | "warning" | "critical";
  cooldownMinutes: number;
}

export interface UpdateAlertRuleRequest {
  name?: string;
  description?: string;
  enabled?: boolean;
  metric?: string;
  condition?: "gt" | "lt" | "eq" | "ne" | "gte" | "lte";
  threshold?: number;
  severity?: "info" | "warning" | "critical";
  cooldownMinutes?: number;
}

export interface ApiError {
  error: string;
  code: string;
}
