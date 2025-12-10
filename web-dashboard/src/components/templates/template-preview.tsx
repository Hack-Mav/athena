"use client";

import { useState } from "react";
import Image from "next/image";
import type { WiringDiagram, TemplateExample } from "@/lib/templates/types";

interface TemplatePreviewProps {
  wiring?: WiringDiagram;
  documentation?: string;
  examples?: TemplateExample[];
}

export function TemplatePreview({ wiring, documentation, examples }: TemplatePreviewProps) {
  const [activeTab, setActiveTab] = useState<"wiring" | "docs" | "examples">("wiring");

  return (
    <div className="space-y-4">
      {/* Tabs */}
      <div className="flex border-b border-zinc-200">
        <button
          type="button"
          onClick={() => setActiveTab("wiring")}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === "wiring"
              ? "border-zinc-900 text-zinc-900"
              : "border-transparent text-zinc-600 hover:text-zinc-900"
          }`}
        >
          Wiring
        </button>
        <button
          type="button"
          onClick={() => setActiveTab("docs")}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === "docs"
              ? "border-zinc-900 text-zinc-900"
              : "border-transparent text-zinc-600 hover:text-zinc-900"
          }`}
        >
          Documentation
        </button>
        <button
          type="button"
          onClick={() => setActiveTab("examples")}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === "examples"
              ? "border-zinc-900 text-zinc-900"
              : "border-transparent text-zinc-600 hover:text-zinc-900"
          }`}
        >
          Examples
        </button>
      </div>

      {/* Tab content */}
      <div className="min-h-[300px]">
        {activeTab === "wiring" && (
          <WiringTab wiring={wiring} />
        )}
        {activeTab === "docs" && (
          <DocsTab documentation={documentation} />
        )}
        {activeTab === "examples" && (
          <ExamplesTab examples={examples} />
        )}
      </div>
    </div>
  );
}

function WiringTab({ wiring }: { wiring?: WiringDiagram }) {
  if (!wiring) {
    return (
      <div className="text-center py-12">
        <p className="text-sm text-zinc-600">No wiring diagram available.</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {wiring.description && (
        <p className="text-sm text-zinc-700">{wiring.description}</p>
      )}
      {wiring.image && (
        <div className="rounded-lg border border-zinc-200 overflow-hidden">
          <Image
            src={wiring.image}
            alt="Wiring diagram"
            width={800}
            height={600}
            className="w-full h-auto"
          />
        </div>
      )}
      {wiring.svg && (
        <div
          className="rounded-lg border border-zinc-200 p-4"
          dangerouslySetInnerHTML={{ __html: wiring.svg }}
        />
      )}
      {wiring.connections && wiring.connections.length > 0 && (
        <div className="rounded-lg border border-zinc-200 bg-white p-4">
          <h3 className="text-sm font-semibold text-zinc-900 mb-3">Connections</h3>
          <ul className="space-y-2 text-sm">
            {wiring.connections.map((conn, idx) => (
              <li key={idx} className="flex items-center gap-2">
                <span className="font-mono text-zinc-700">
                  {conn.from.pin}
                </span>
                <span className="text-zinc-500">→</span>
                <span className="font-mono text-zinc-700">
                  {conn.to.pin}
                </span>
                {conn.wireColor && (
                  <span
                    className="inline-block w-3 h-3 rounded-full border border-zinc-300"
                    style={{ backgroundColor: conn.wireColor }}
                    title={conn.wireColor}
                  />
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

function DocsTab({ documentation }: { documentation?: string }) {
  if (!documentation) {
    return (
      <div className="text-center py-12">
        <p className="text-sm text-zinc-600">No documentation available.</p>
      </div>
    );
  }

  // Simple markdown-like rendering (basic only)
  const html = documentation
    .replace(/^### (.*$)/gim, "<h3 class='text-base font-semibold text-zinc-900 mt-4 mb-2'>$1</h3>")
    .replace(/^## (.*$)/gim, "<h2 class='text-lg font-semibold text-zinc-900 mt-6 mb-3'>$1</h2>")
    .replace(/^# (.*$)/gim, "<h1 class='text-xl font-semibold text-zinc-900 mt-8 mb-4'>$1</h1>")
    .replace(/\*\*(.*)\*\*/gim, "<strong>$1</strong>")
    .replace(/\*(.*)\*/gim, "<em>$1</em>")
    .replace(/`([^`]+)`/gim, "<code class='px-1 py-0.5 bg-zinc-100 text-zinc-800 rounded text-xs font-mono'>$1</code>")
    .replace(/\n\n/g, "</p><p class='mb-4 text-sm text-zinc-700'>")
    .replace(/\n/g, "<br />");

  return (
    <div
      className="prose prose-sm max-w-none text-sm text-zinc-700"
      dangerouslySetInnerHTML={{ __html: `<p class='mb-4'>${html}</p>` }}
    />
  );
}

function ExamplesTab({ examples }: { examples?: TemplateExample[] }) {
  const [selectedIdx, setSelectedIdx] = useState(0);
  
  if (!examples || examples.length === 0) {
    return (
      <div className="text-center py-12">
        <p className="text-sm text-zinc-600">No examples available.</p>
      </div>
    );
  }

  const selected = examples[selectedIdx];

  return (
    <div className="space-y-4">
      {/* Example selector */}
      <div className="flex gap-2 flex-wrap">
        {examples.map((ex, idx) => (
          <button
            key={idx}
            type="button"
            onClick={() => setSelectedIdx(idx)}
            className={`px-3 py-1 text-sm rounded-md border transition-colors ${
              idx === selectedIdx
                ? "border-zinc-900 bg-zinc-900 text-white"
                : "border-zinc-300 bg-white text-zinc-700 hover:bg-zinc-50"
            }`}
          >
            {ex.name}
          </button>
        ))}
      </div>

      {/* Selected example */}
      {selected && (
        <div className="rounded-lg border border-zinc-200 bg-white p-4">
          <h3 className="text-sm font-semibold text-zinc-900 mb-2">
            {selected.name}
          </h3>
          {selected.description && (
            <p className="text-sm text-zinc-600 mb-4">{selected.description}</p>
          )}
          <div className="relative">
            <pre className="text-xs bg-zinc-900 text-zinc-100 p-4 rounded-md overflow-x-auto">
              <code>{selected.code}</code>
            </pre>
            <button
              type="button"
              onClick={() => navigator.clipboard.writeText(selected.code)}
              className="absolute top-2 right-2 px-2 py-1 text-xs bg-zinc-700 text-zinc-100 rounded hover:bg-zinc-600"
            >
              Copy
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
