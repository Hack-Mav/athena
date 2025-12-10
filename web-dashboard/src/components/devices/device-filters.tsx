"use client";

import type { DeviceSearchParams, DeviceStatus } from "@/lib/devices/types";

interface DeviceFiltersProps {
  params: DeviceSearchParams;
  onParamsChange: (params: DeviceSearchParams) => void;
}

const statuses: { value: DeviceStatus; label: string }[] = [
  { value: "online", label: "Online" },
  { value: "offline", label: "Offline" },
  { value: "compiling", label: "Compiling" },
  { value: "flashing", label: "Flashing" },
  { value: "error", label: "Error" },
];

const sortOptions = [
  { value: "name", label: "Name" },
  { value: "createdAt", label: "Created" },
  { value: "updatedAt", label: "Updated" },
  { value: "lastSeen", label: "Last seen" },
] as const;

export function DeviceFilters({ params, onParamsChange }: DeviceFiltersProps) {
  function update(updates: Partial<DeviceSearchParams>) {
    onParamsChange({ ...params, ...updates, page: 1 });
  }

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-zinc-700 mb-1">
          Search
        </label>
        <input
          type="text"
          placeholder="Search devices..."
          value={params.q ?? ""}
          onChange={(e) => update({ q: e.target.value || undefined })}
          className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-zinc-700 mb-1">
          Status
        </label>
        <select
          value={params.status ?? ""}
          onChange={(e) => update({ status: (e.target.value as DeviceStatus) || undefined })}
          className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
        >
          <option value="">All statuses</option>
          {statuses.map((st) => (
            <option key={st.value} value={st.value}>
              {st.label}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label className="block text-sm font-medium text-zinc-700 mb-1">
          Template
        </label>
        <input
          type="text"
          placeholder="Template ID..."
          value={params.templateId ?? ""}
          onChange={(e) => update({ templateId: e.target.value || undefined })}
          className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-zinc-700 mb-1">
          Sort by
        </label>
        <div className="flex gap-2">
          <select
            value={params.sortBy ?? "updatedAt"}
            onChange={(e) => update({ sortBy: e.target.value as "name" | "createdAt" | "updatedAt" | "lastSeen" })}
            className="flex-1 rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
          >
            {sortOptions.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
          <select
            value={params.sortOrder ?? "desc"}
            onChange={(e) => update({ sortOrder: e.target.value as "asc" | "desc" })}
            className="rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
          >
            <option value="asc">A–Z</option>
            <option value="desc">Z–A</option>
          </select>
        </div>
      </div>
    </div>
  );
}
