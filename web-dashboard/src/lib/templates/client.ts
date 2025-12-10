import type {
  Template,
  TemplateListResponse,
  TemplateSearchParams,
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

export async function getTemplates(
  params?: TemplateSearchParams
): Promise<TemplateListResponse> {
  const search = new URLSearchParams();
  if (params) {
    if (params.q) search.set("q", params.q);
    if (params.category) search.set("category", params.category);
    if (params.tags?.length) search.set("tags", params.tags.join(","));
    if (params.language) search.set("language", params.language);
    if (params.framework) search.set("framework", params.framework);
    if (params.page != null) search.set("page", String(params.page));
    if (params.pageSize != null) search.set("pageSize", String(params.pageSize));
    if (params.sortBy) search.set("sortBy", params.sortBy);
    if (params.sortOrder) search.set("sortOrder", params.sortOrder);
  }
  const res = await fetch(
    `${API_BASE_URL}/api/v1/templates?${search.toString()}`
  );
  return handleResponse<TemplateListResponse>(res);
}

export async function getTemplate(id: string): Promise<Template> {
  const res = await fetch(`${API_BASE_URL}/api/v1/templates/${encodeURIComponent(id)}`);
  return handleResponse<Template>(res);
}
