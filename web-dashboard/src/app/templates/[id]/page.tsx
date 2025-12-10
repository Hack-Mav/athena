"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { TemplateConfigForm } from "@/components/templates/template-config-form";
import { TemplatePreview } from "@/components/templates/template-preview";
import { getTemplate } from "@/lib/templates/client";
import type { Template, TemplateConfigFormValues } from "@/lib/templates/types";

export default function TemplateDetailPage() {
  const params = useParams();
  const templateId = typeof params.id === "string" ? params.id : params.id?.[0];

  const [template, setTemplate] = useState<Template | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [configValues, setConfigValues] = useState<TemplateConfigFormValues>({});

  useEffect(() => {
    if (!templateId) return;
    // Batch state updates to avoid cascading renders
    Promise.resolve().then(() => {
      setLoading(true);
      setError(null);
    });
    getTemplate(templateId)
      .then((tpl) => {
        setTemplate(tpl);
        // Initialize default values from schema
        const defaults: TemplateConfigFormValues = {};
        if (tpl.schema.properties) {
          for (const [key, prop] of Object.entries(tpl.schema.properties)) {
            if ("default" in prop && prop.default !== undefined) {
              defaults[key] = prop.default;
            } else if (prop.type === "boolean") {
              defaults[key] = false;
            } else if (prop.type === "number" || prop.type === "integer") {
              defaults[key] = 0;
            } else if (prop.type === "array") {
              defaults[key] = [];
            } else if (prop.type === "object") {
              defaults[key] = {};
            } else {
              defaults[key] = "";
            }
          }
        }
        setConfigValues(defaults);
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : "Failed to load template");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [templateId]);

  if (loading) {
    return (
      <DashboardShell title="Template Details">
        <div className="flex items-center justify-center py-12">
          <p className="text-sm text-zinc-600">Loading template...</p>
        </div>
      </DashboardShell>
    );
  }

  if (error || !template) {
    return (
      <DashboardShell title="Template Details">
        <div className="rounded-lg border border-red-200 bg-red-50 p-4">
          <p className="text-sm text-red-600">{error || "Template not found"}</p>
        </div>
      </DashboardShell>
    );
  }

  return (
    <DashboardShell title={template.name}>
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Main details */}
        <div className="lg:col-span-2 space-y-6">
          {/* Header */}
          <div className="rounded-lg border border-zinc-200 bg-white p-4">
            <h1 className="text-lg font-semibold text-zinc-900 mb-2">
              {template.name}
            </h1>
            <p className="text-sm text-zinc-700 mb-3">{template.description}</p>
            <div className="flex flex-wrap gap-2 text-sm text-zinc-600">
              <span>Version {template.version}</span>
              <span>•</span>
              <span>by {template.author}</span>
              <span>•</span>
              <span>{template.language}</span>
              <span>•</span>
              <span>{template.category}</span>
            </div>
            {template.tags.length > 0 && (
              <div className="flex flex-wrap gap-1 mt-3">
                {template.tags.map((tag) => (
                  <span
                    key={tag}
                    className="inline-block text-xs px-2 py-0.5 rounded-full bg-zinc-100 text-zinc-700"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            )}
          </div>

          {/* Configuration form */}
          <div className="rounded-lg border border-zinc-200 bg-white p-4">
            <h2 className="text-base font-semibold text-zinc-900 mb-4">
              Configuration
            </h2>
            <TemplateConfigForm
              schema={template.schema}
              values={configValues}
              onChange={setConfigValues}
            />
          </div>

          {/* Preview */}
          <div className="rounded-lg border border-zinc-200 bg-white p-4">
            <h2 className="text-base font-semibold text-zinc-900 mb-4">
              Preview
            </h2>
            <TemplatePreview
              wiring={template.wiring}
              documentation={template.documentation}
              examples={template.examples}
            />
          </div>
        </div>

        {/* Sidebar */}
        <aside className="space-y-4">
          {/* Actions */}
          <div className="rounded-lg border border-zinc-200 bg-white p-4">
            <h3 className="text-sm font-semibold text-zinc-900 mb-3">Actions</h3>
            <div className="space-y-2">
              <button
                type="button"
                className="w-full px-3 py-2 text-sm font-medium text-white bg-zinc-900 rounded-md hover:bg-zinc-800"
              >
                Use this template
              </button>
              <button
                type="button"
                className="w-full px-3 py-2 text-sm font-medium text-zinc-700 border border-zinc-300 rounded-md hover:bg-zinc-50"
              >
                Export configuration
              </button>
            </div>
          </div>

          {/* Metadata */}
          <div className="rounded-lg border border-zinc-200 bg-white p-4">
            <h3 className="text-sm font-semibold text-zinc-900 mb-3">Metadata</h3>
            <dl className="space-y-2 text-sm">
              <div>
                <dt className="font-medium text-zinc-700">Created</dt>
                <dd className="text-zinc-600">
                  {new Date(template.createdAt).toLocaleDateString()}
                </dd>
              </div>
              <div>
                <dt className="font-medium text-zinc-700">Updated</dt>
                <dd className="text-zinc-600">
                  {new Date(template.updatedAt).toLocaleDateString()}
                </dd>
              </div>
              {template.framework && (
                <div>
                  <dt className="font-medium text-zinc-700">Framework</dt>
                  <dd className="text-zinc-600">{template.framework}</dd>
                </div>
              )}
            </dl>
          </div>
        </aside>
      </div>
    </DashboardShell>
  );
}
