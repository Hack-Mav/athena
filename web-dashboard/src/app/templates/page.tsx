"use client";

import { useState, useEffect } from "react";
import { DashboardShell } from "@/components/layout/dashboard-shell";
import { TemplateFilters } from "@/components/templates/template-filters";
import { TemplateCard } from "@/components/templates/template-card";
import { getTemplates } from "@/lib/templates/client";
import type { Template, TemplateSearchParams } from "@/lib/templates/types";

const DEFAULT_PAGE_SIZE = 12;

export default function TemplatesPage() {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [params, setParams] = useState<TemplateSearchParams>({
    page: 1,
    pageSize: DEFAULT_PAGE_SIZE,
    sortBy: "updatedAt",
    sortOrder: "desc",
  });

  useEffect(() => {
    // Batch state updates to avoid cascading renders
    Promise.resolve().then(() => {
      setLoading(true);
      setError(null);
    });
    getTemplates(params)
      .then((res) => {
        setTemplates(res.templates);
        setTotal(res.total);
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : "Failed to load templates");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [params]);

  const totalPages = Math.ceil(total / (params.pageSize ?? DEFAULT_PAGE_SIZE));
  const currentPage = params.page ?? 1;

  return (
    <DashboardShell title="Template Catalog">
      <div className="flex flex-col md:flex-row gap-6">
        {/* Filters sidebar */}
        <aside className="w-full md:w-64 flex-shrink-0">
          <div className="rounded-lg border border-zinc-200 bg-white p-4">
            <h2 className="mb-4 text-sm font-semibold text-zinc-900">Filters</h2>
            <TemplateFilters params={params} onParamsChange={setParams} />
          </div>
        </aside>

        {/* Main content */}
        <div className="flex-1">
          {loading && (
            <div className="flex items-center justify-center py-12">
              <p className="text-sm text-zinc-600">Loading templates...</p>
            </div>
          )}

          {error && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-4">
              <p className="text-sm text-red-600">{error}</p>
            </div>
          )}

          {!loading && !error && (
            <>
              <div className="mb-4 text-sm text-zinc-600">
                {total} template{total !== 1 ? "s" : ""}
              </div>

              {templates.length === 0 ? (
                <div className="rounded-lg border border-zinc-200 bg-white p-8 text-center">
                  <p className="text-sm text-zinc-600">No templates found.</p>
                </div>
              ) : (
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                  {templates.map((template) => (
                    <TemplateCard key={template.id} template={template} />
                  ))}
                </div>
              )}

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="mt-6 flex items-center justify-center gap-2">
                  <button
                    type="button"
                    disabled={currentPage <= 1}
                    onClick={() => setParams({ ...params, page: currentPage - 1 })}
                    className="px-3 py-1 text-sm rounded-md border border-zinc-300 bg-white disabled:opacity-50 disabled:cursor-not-allowed hover:bg-zinc-50"
                  >
                    Previous
                  </button>
                  <span className="text-sm text-zinc-600">
                    Page {currentPage} of {totalPages}
                  </span>
                  <button
                    type="button"
                    disabled={currentPage >= totalPages}
                    onClick={() => setParams({ ...params, page: currentPage + 1 })}
                    className="px-3 py-1 text-sm rounded-md border border-zinc-300 bg-white disabled:opacity-50 disabled:cursor-not-allowed hover:bg-zinc-50"
                  >
                    Next
                  </button>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </DashboardShell>
  );
}
