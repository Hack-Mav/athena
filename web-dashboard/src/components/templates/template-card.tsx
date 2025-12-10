"use client";

import Link from "next/link";
import type { Template } from "@/lib/templates/types";

interface TemplateCardProps {
  template: Template;
}

export function TemplateCard({ template }: TemplateCardProps) {
  return (
    <Link
      href={`/templates/${template.id}`}
      className="block rounded-lg border border-zinc-200 bg-white p-4 shadow-sm transition-shadow hover:shadow-md"
    >
      <div className="mb-2">
        <h2 className="text-base font-semibold text-zinc-900 leading-tight">
          {template.name}
        </h2>
        <p className="text-sm text-zinc-600">v{template.version} by {template.author}</p>
      </div>
      <p className="mb-3 text-sm text-zinc-700 line-clamp-2">
        {template.description}
      </p>
      <div className="flex flex-wrap gap-1 mb-2">
        {template.tags.map((tag) => (
          <span
            key={tag}
            className="inline-block text-xs px-2 py-0.5 rounded-full bg-zinc-100 text-zinc-700"
          >
            {tag}
          </span>
        ))}
      </div>
      <div className="flex items-center justify-between text-xs text-zinc-500">
        <span>{template.category}</span>
        <span>{new Date(template.updatedAt).toLocaleDateString()}</span>
      </div>
    </Link>
  );
}
