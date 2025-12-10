export interface Template {
  id: string;
  name: string;
  description: string;
  version: string;
  author: string;
  tags: string[];
  category: string;
  language: "cpp" | "c";
  framework?: string;
  createdAt: string; // ISO timestamp
  updatedAt: string; // ISO timestamp
  schema: JsonSchema; // JSON Schema for configuration
  wiring?: WiringDiagram;
  documentation?: string; // markdown or HTML
  examples?: TemplateExample[];
}

export interface JsonSchema {
  $schema?: string;
  type: "object";
  properties?: Record<string, JsonSchemaProperty>;
  required?: string[];
  additionalProperties?: boolean;
  definitions?: Record<string, JsonSchema>;
}

export type JsonSchemaProperty =
  | { type: "string"; title?: string; description?: string; default?: string; enum?: string[]; minLength?: number; maxLength?: number; pattern?: string; format?: "date-time" | "email" | "uri" }
  | { type: "number"; title?: string; description?: string; default?: number; minimum?: number; maximum?: number; exclusiveMinimum?: number; exclusiveMaximum?: number; multipleOf?: number }
  | { type: "integer"; title?: string; description?: string; default?: number; minimum?: number; maximum?: number; exclusiveMinimum?: number; exclusiveMaximum?: number; multipleOf?: number }
  | { type: "boolean"; title?: string; description?: string; default?: boolean }
  | { type: "array"; title?: string; description?: string; default?: unknown[]; items: JsonSchemaProperty; minItems?: number; maxItems?: number; uniqueItems?: boolean }
  | { type: "object"; title?: string; description?: string; default?: Record<string, unknown>; properties?: Record<string, JsonSchemaProperty>; required?: string[]; additionalProperties?: boolean }
  | { type: "null"; title?: string; description?: string };

export interface WiringDiagram {
  image?: string; // URL or base64
  svg?: string; // SVG content
  description?: string;
  connections?: WiringConnection[];
}

export interface WiringConnection {
  from: { pin: string; label?: string };
  to: { pin: string; label?: string };
  wireColor?: string;
}

export interface TemplateExample {
  name: string;
  description?: string;
  code: string;
  language: "cpp" | "c";
}

export interface TemplateListResponse {
  templates: Template[];
  total: number;
  page: number;
  pageSize: number;
}

export interface TemplateSearchParams {
  q?: string; // search query
  category?: string;
  tags?: string[];
  language?: "cpp" | "c";
  framework?: string;
  page?: number;
  pageSize?: number;
  sortBy?: "name" | "createdAt" | "updatedAt" | "version";
  sortOrder?: "asc" | "desc";
}

export interface TemplateConfigFormValues {
  [key: string]: unknown;
}

export interface ApiError {
  error: string;
  code: string;
}
