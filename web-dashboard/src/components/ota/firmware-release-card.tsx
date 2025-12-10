"use client";

import Link from "next/link";
import type { FirmwareRelease } from "@/lib/ota/types";

interface FirmwareReleaseCardProps {
  release: FirmwareRelease;
}

function formatBytes(bytes: number) {
  const units = ["B", "KB", "MB", "GB"];
  let i = 0;
  while (bytes >= 1024 && i < units.length - 1) {
    bytes /= 1024;
    i++;
  }
  return `${bytes.toFixed(1)} ${units[i]}`;
}

export function FirmwareReleaseCard({ release }: FirmwareReleaseCardProps) {
  return (
    <Link
      href={`/ota/releases/${release.id}`}
      className="block rounded-lg border border-zinc-200 bg-white p-4 shadow-sm transition-shadow hover:shadow-md"
    >
      <div className="mb-2">
        <h2 className="text-base font-semibold text-zinc-900 leading-tight">
          {release.version}
        </h2>
        <p className="text-sm text-zinc-600">Template: {release.templateName}</p>
      </div>
      {release.description && (
        <p className="mb-3 text-sm text-zinc-700 line-clamp-2">{release.description}</p>
      )}
      <div className="space-y-1 text-xs text-zinc-600">
        <div className="flex justify-between">
          <span>Size</span>
          <span>{formatBytes(release.size)}</span>
        </div>
        <div className="flex justify-between">
          <span>SHA256</span>
          <span className="font-mono">{release.checksum.slice(0, 12)}...</span>
        </div>
        <div className="flex justify-between">
          <span>Created</span>
          <span>{new Date(release.createdAt).toLocaleDateString()}</span>
        </div>
        <div className="flex justify-between">
          <span>By</span>
          <span>{release.createdBy}</span>
        </div>
      </div>
    </Link>
  );
}
