import { render, screen } from "@testing-library/react";
import { FirmwareReleaseCard } from "../firmware-release-card";
import type { FirmwareRelease } from "@/lib/ota/types";

const mockRelease: FirmwareRelease = {
  id: "rel-1",
  version: "2.1.0",
  templateId: "tpl-1",
  templateName: "Blinky",
  description: "Fixed LED timing issue",
  checksum: "abc123def456",
  size: 1024000,
  createdAt: "2025-01-01T10:00:00Z",
  createdBy: "admin",
  metadata: { breaking: false },
};

describe("FirmwareReleaseCard", () => {
  it("renders release information", () => {
    render(<FirmwareReleaseCard release={mockRelease} />);
    expect(screen.getByText("2.1.0")).toBeInTheDocument();
    expect(screen.getByText(/Template:/)).toBeInTheDocument();
    expect(screen.getByText(/Blinky/)).toBeInTheDocument();
    expect(screen.getByText("Fixed LED timing issue")).toBeInTheDocument();
  });

  it("displays file size", () => {
    render(<FirmwareReleaseCard release={mockRelease} />);
    expect(screen.getByText(/1024|1\.0/)).toBeInTheDocument();
  });

  it("shows creation date", () => {
    render(<FirmwareReleaseCard release={mockRelease} />);
    expect(screen.getByText(/2025/)).toBeInTheDocument();
  });

  it("displays checksum", () => {
    render(<FirmwareReleaseCard release={mockRelease} />);
    expect(screen.getByText(/abc123/)).toBeInTheDocument();
  });

  it("handles missing description", () => {
    const releaseNoDesc = { ...mockRelease, description: undefined };
    render(<FirmwareReleaseCard release={releaseNoDesc} />);
    expect(screen.getByText("2.1.0")).toBeInTheDocument();
  });

  it("shows link to release details", () => {
    render(<FirmwareReleaseCard release={mockRelease} />);
    const link = screen.getByRole("link");
    expect(link).toBeInTheDocument();
    expect(link).toHaveAttribute("href", "/ota/releases/rel-1");
  });
});
