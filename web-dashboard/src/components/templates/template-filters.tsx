"use client";

import { useState } from "react";
import type { TemplateSearchParams } from "@/lib/templates/types";

interface TemplateFiltersProps {
  params: TemplateSearchParams;
  onParamsChange: (params: TemplateSearchParams) => void;
}

const categories = [
  "Sensors",
  "Actuators",
  "Communication",
  "Display",
  "Power",
  "Input",
  "Storage",
  "Misc",
];

const languages = [
  { value: "cpp", label: "C++" },
  { value: "c", label: "C" },
] as const;

const sortOptions = [
  { value: "name", label: "Name" },
  { value: "createdAt", label: "Created" },
  { value: "updatedAt", label: "Updated" },
  { value: "version", label: "Version" },
] as const;

export function TemplateFilters({ params, onParamsChange }: TemplateFiltersProps) {
  const [tagInput, setTagInput] = useState("");

  function update(updates: Partial<TemplateSearchParams>) {
    onParamsChange({ ...params, ...updates, page: 1 });
  }

  function addTag() {
    const tag = tagInput.trim();
    if (tag && !params.tags?.includes(tag)) {
      update({ tags: [...(params.tags || []), tag] });
      setTagInput("");
    }
  }

  function removeTag(tag: string) {
    update({ tags: params.tags?.filter((t) => t !== tag) });
  }

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-zinc-700 mb-1">
          Search
        </label>
        <input
          type="text"
          placeholder="Search templates..."
          value={params.q ?? ""}
          onChange={(e) => update({ q: e.target.value || undefined })}
          className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-zinc-700 mb-1">
          Category
        </label>
        <select
          value={params.category ?? ""}
          onChange={(e) => update({ category: e.target.value || undefined })}
          className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
        >
          <option value="">All categories</option>
          {categories.map((cat) => (
            <option key={cat} value={cat}>
              {cat}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label className="block text-sm font-medium text-zinc-700 mb-1">
          Language
        </label>
        <select
          value={params.language ?? ""}
          onChange={(e) => update({ language: (e.target.value as "cpp" | "c") || undefined })}
          className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
        >
          <option value="">All languages</option>
          {languages.map((lang) => (
            <option key={lang.value} value={lang.value}>
              {lang.label}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label className="block text-sm font-medium text-zinc-700 mb-1">
          Tags
        </label>
        <div className="flex gap-1 mb-2">
          <input
            type="text"
            placeholder="Add tag..."
            value={tagInput}
            onChange={(e) => setTagInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                addTag();
              }
            }}
            className="flex-1 rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
          />
          <button
            type="button"
            onClick={addTag}
            className="px-3 py-2 text-sm font-medium text-white bg-zinc-900 rounded-md hover:bg-zinc-800"
          >
            Add
          </button>
        </div>
        {params.tags && params.tags.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {params.tags.map((tag) => (
              <span
                key={tag}
                className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-zinc-100 text-zinc-700"
              >
                {tag}
                <button
                  type="button"
                  onClick={() => removeTag(tag)}
                  className="ml-1 text-zinc-500 hover:text-zinc-700"
                >
                  ×
                </button>
              </span>
            ))}
          </div>
        )}
      </div>

      <div>
        <label className="block text-sm font-medium text-zinc-700 mb-1">
          Sort by
        </label>
        <div className="flex gap-2">
          <select
            value={params.sortBy ?? "updatedAt"}
            onChange={(e) => update({ sortBy: e.target.value as "name" | "createdAt" | "updatedAt" | "version" })}
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
