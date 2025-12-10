"use client";

import { useState, useEffect, useCallback } from "react";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { AlertList } from "@/components/monitoring/alert-list";
import { getAlerts, getAlertRules, createAlertRule, updateAlertRule, deleteAlertRule } from "@/lib/telemetry/client";
import type { Alert, AlertRule, AlertSearchParams, CreateAlertRuleRequest, UpdateAlertRuleRequest } from "@/lib/telemetry/types";

export default function AlertsPage() {
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showAddRule, setShowAddRule] = useState(false);
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null);
  const [filter, setFilter] = useState<AlertSearchParams>({});

  const fetchData = useCallback(async () => {
    try {
      const [alertsData, rulesData] = await Promise.all([
        getAlerts(filter),
        getAlertRules(),
      ]);
      setAlerts(alertsData.alerts);
      setRules(rulesData);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load alerts");
    } finally {
      setLoading(false);
    }
  }, [filter]);

  useEffect(() => {
    setLoading(true);
    fetchData();
  }, [fetchData]);

  async function handleCreateRule(body: CreateAlertRuleRequest) {
    try {
      await createAlertRule(body);
      setShowAddRule(false);
      fetchData();
    } catch {
      // ignore errors
    }
  }

  async function handleUpdateRule(id: string, updates: Partial<AlertRule>) {
    try {
      await updateAlertRule(id, updates);
      setEditingRule(null);
      fetchData();
    } catch {
      // ignore errors
    }
  }

  async function handleDeleteRule(id: string) {
    try {
      await deleteAlertRule(id);
      fetchData();
    } catch {
      // ignore errors
    }
  }

  return (
    <DashboardShell title="Alerts & Rules">
      <div className="space-y-6">
        {loading && (
          <div className="flex items-center justify-center py-12">
            <p className="text-sm text-zinc-600">Loading alerts...</p>
          </div>
        )}

        {error && (
          <div className="rounded-lg border border-red-200 bg-red-50 p-4">
            <p className="text-sm text-red-600">{error}</p>
          </div>
        )}

        {!loading && !error && (
          <>
            {/* Filters */}
            <div className="rounded-lg border border-zinc-200 bg-white p-4">
              <div className="flex flex-wrap gap-4 items-center">
                <select
                  value={filter.severity ?? ""}
                  onChange={(e) => setFilter({ ...filter, severity: (e.target.value as "info" | "warning" | "critical" | undefined) || undefined })}
                  className="rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                >
                  <option value="">All severities</option>
                  <option value="info">Info</option>
                  <option value="warning">Warning</option>
                  <option value="critical">Critical</option>
                </select>
                <select
                  value={filter.status ?? ""}
                  onChange={(e) => setFilter({ ...filter, status: (e.target.value as "active" | "acknowledged" | "resolved" | undefined) || undefined })}
                  className="rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                >
                  <option value="">All statuses</option>
                  <option value="active">Active</option>
                  <option value="acknowledged">Acknowledged</option>
                  <option value="resolved">Resolved</option>
                </select>
                <button
                  type="button"
                  onClick={() => setShowAddRule(true)}
                  className="ml-auto px-4 py-2 text-sm font-medium text-white bg-zinc-900 rounded-md hover:bg-zinc-800"
                >
                  + Add rule
                </button>
              </div>
            </div>

            {/* Alerts */}
            <div>
              <h2 className="text-base font-semibold text-zinc-900 mb-4">Alerts</h2>
              <AlertList alerts={alerts} onRefresh={fetchData} />
            </div>

            {/* Alert Rules */}
            <div>
              <h2 className="text-base font-semibold text-zinc-900 mb-4">Alert Rules</h2>
              <div className="space-y-3">
                {rules.map((rule) => (
                  <div key={rule.id} className="rounded-lg border border-zinc-200 bg-white p-4">
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <h3 className="text-sm font-medium text-zinc-900">{rule.name}</h3>
                          <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                            rule.enabled ? "bg-green-100 text-green-800" : "bg-zinc-100 text-zinc-800"
                          }`}>
                            {rule.enabled ? "Enabled" : "Disabled"}
                          </span>
                          <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                            rule.severity === "critical" ? "bg-red-100 text-red-800" :
                            rule.severity === "warning" ? "bg-yellow-100 text-yellow-800" :
                            "bg-blue-100 text-blue-800"
                          }`}>
                            {rule.severity}
                          </span>
                        </div>
                        {rule.description && (
                          <p className="text-xs text-zinc-600 mb-2">{rule.description}</p>
                        )}
                        <p className="text-xs text-zinc-600">
                          {rule.metric} {rule.condition} {rule.threshold}
                          {rule.deviceId && ` • Device: ${rule.deviceId}`}
                          {" • Cooldown: "}{rule.cooldownMinutes}m
                        </p>
                      </div>
                      <div className="flex gap-2 flex-shrink-0">
                        <button
                          type="button"
                          onClick={() => setEditingRule(rule)}
                          className="px-2 py-1 text-xs font-medium text-zinc-700 border border-zinc-300 rounded hover:bg-zinc-50"
                        >
                          Edit
                        </button>
                        <button
                          type="button"
                          onClick={() => handleUpdateRule(rule.id, { enabled: !rule.enabled })}
                          className="px-2 py-1 text-xs font-medium text-zinc-700 border border-zinc-300 rounded hover:bg-zinc-50"
                        >
                          {rule.enabled ? "Disable" : "Enable"}
                        </button>
                        <button
                          type="button"
                          onClick={() => handleDeleteRule(rule.id)}
                          className="px-2 py-1 text-xs font-medium text-red-700 border border-red-300 rounded hover:bg-red-50"
                        >
                          Delete
                        </button>
                      </div>
                    </div>
                  </div>
                ))}
                {rules.length === 0 && (
                  <div className="rounded-lg border border-zinc-200 bg-white p-8 text-center">
                    <p className="text-sm text-zinc-600">No alert rules configured.</p>
                  </div>
                )}
              </div>
            </div>

            {/* Add/Edit Rule Modal */}
            {(showAddRule || editingRule) && (
              <AlertRuleForm
                rule={editingRule}
                onSubmit={editingRule ? (r) => handleUpdateRule(editingRule.id, r as UpdateAlertRuleRequest) : (r) => handleCreateRule(r as CreateAlertRuleRequest)}
                onCancel={() => {
                  setShowAddRule(false);
                  setEditingRule(null);
                }}
              />
            )}
          </>
        )}
      </div>
    </DashboardShell>
  );
}

// Minimal inline form for add/edit alert rule
function AlertRuleForm({
  rule,
  onSubmit,
  onCancel,
}: {
  rule: AlertRule | null;
  onSubmit: (data: CreateAlertRuleRequest | UpdateAlertRuleRequest) => void;
  onCancel: () => void;
}) {
  const [form, setForm] = useState({
    name: rule?.name ?? "",
    description: rule?.description ?? "",
    enabled: rule?.enabled ?? true,
    deviceId: rule?.deviceId ?? "",
    metric: rule?.metric ?? "",
    condition: rule?.condition ?? "gt" as const,
    threshold: rule?.threshold ?? 0,
    severity: rule?.severity ?? "warning" as const,
    cooldownMinutes: rule?.cooldownMinutes ?? 5,
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (rule) {
      // For updates, only send the fields that can be updated
      const updateData: UpdateAlertRuleRequest = {
        name: form.name,
        description: form.description,
        enabled: form.enabled,
        metric: form.metric,
        condition: form.condition,
        threshold: form.threshold,
        severity: form.severity,
        cooldownMinutes: form.cooldownMinutes,
      };
      onSubmit(updateData);
    } else {
      // For creation, send all required fields
      const createData: CreateAlertRuleRequest = {
        name: form.name,
        description: form.description,
        enabled: form.enabled,
        deviceId: form.deviceId,
        metric: form.metric,
        condition: form.condition,
        threshold: form.threshold,
        severity: form.severity,
        cooldownMinutes: form.cooldownMinutes,
      };
      onSubmit(createData);
    }
  }

  return (
    <div className="fixed inset-0 bg-zinc-900 bg-opacity-50 flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-lg max-w-md w-full p-6">
        <h3 className="text-lg font-semibold text-zinc-900 mb-4">
          {rule ? "Edit Alert Rule" : "Add Alert Rule"}
        </h3>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-zinc-700 mb-1">Name</label>
            <input
              type="text"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-zinc-700 mb-1">Description</label>
            <input
              type="text"
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
              className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-zinc-700 mb-1">Metric</label>
            <input
              type="text"
              value={form.metric}
              onChange={(e) => setForm({ ...form, metric: e.target.value })}
              className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
              required
            />
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="block text-sm font-medium text-zinc-700 mb-1">Condition</label>
              <select
                value={form.condition}
                onChange={(e) => setForm({ ...form, condition: e.target.value as "gt" | "lt" | "eq" | "ne" | "gte" | "lte" })}
                className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
              >
                <option value="gt">&gt;</option>
                <option value="gte">&gt;=</option>
                <option value="lt">&lt;</option>
                <option value="lte">&lt;=</option>
                <option value="eq">=</option>
                <option value="ne">!=</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-zinc-700 mb-1">Threshold</label>
              <input
                type="number"
                step="any"
                value={form.threshold}
                onChange={(e) => setForm({ ...form, threshold: parseFloat(e.target.value) })}
                className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                required
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="block text-sm font-medium text-zinc-700 mb-1">Severity</label>
              <select
                value={form.severity}
                onChange={(e) => setForm({ ...form, severity: e.target.value as "info" | "warning" | "critical" })}
                className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
              >
                <option value="info">Info</option>
                <option value="warning">Warning</option>
                <option value="critical">Critical</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-zinc-700 mb-1">Cooldown (min)</label>
              <input
                type="number"
                min="0"
                value={form.cooldownMinutes}
                onChange={(e) => setForm({ ...form, cooldownMinutes: parseInt(e.target.value, 10) })}
                className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
                required
              />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="enabled"
              checked={form.enabled}
              onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
              className="rounded border-zinc-300 text-zinc-900 focus:ring-zinc-900"
            />
            <label htmlFor="enabled" className="text-sm text-zinc-700">Enabled</label>
          </div>
          <div className="flex gap-2 pt-2">
            <button
              type="submit"
              className="flex-1 px-4 py-2 text-sm font-medium text-white bg-zinc-900 rounded-md hover:bg-zinc-800"
            >
              {rule ? "Update" : "Create"}
            </button>
            <button
              type="button"
              onClick={onCancel}
              className="flex-1 px-4 py-2 text-sm font-medium text-zinc-700 border border-zinc-300 rounded-md hover:bg-zinc-50"
            >
              Cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
