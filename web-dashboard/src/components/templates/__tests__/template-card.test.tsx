import { render, screen } from "@testing-library/react";
import { TemplateCard } from "../template-card";
import type { Template } from "@/lib/templates/types";

const mockTemplate: Template = {
  id: "tpl-1",
  name: "Blinky",
  version: "1.0.0",
  description: "Simple LED blink example",
  author: "ATHENA Team",
  category: "examples",
  language: "cpp",
  tags: ["led", "basic"],
  createdAt: "2025-01-01T00:00:00Z",
  updatedAt: "2025-01-01T00:00:00Z",
  metadata: {},
};

describe("TemplateCard", () => {
  it("renders template information", () => {
    render(<TemplateCard template={mockTemplate} />);
    expect(screen.getByText("Blinky")).toBeInTheDocument();
    expect(screen.getByText((content) => content.includes("1.0.0"))).toBeInTheDocument();
    expect(screen.getByText((content) => content.includes("ATHENA Team"))).toBeInTheDocument();
    expect(screen.getByText("examples")).toBeInTheDocument();
    expect(screen.getByText("led")).toBeInTheDocument();
    expect(screen.getByText("basic")).toBeInTheDocument();
  });

  it("links to template detail page", () => {
    render(<TemplateCard template={mockTemplate} />);
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/templates/tpl-1");
  });

  it("truncates long description", () => {
    const longDescTemplate = {
      ...mockTemplate,
      description: "A".repeat(200),
    };
    render(<TemplateCard template={longDescTemplate} />);
    // Should be present but truncated via CSS line-clamp
    expect(screen.getByText(new RegExp("A{100,}"))).toBeInTheDocument();
  });

  it("displays formatted dates", () => {
    render(<TemplateCard template={mockTemplate} />);
    // Date formatting may vary locale, so just check that a date is present
    expect(screen.getByText(/2025/)).toBeInTheDocument();
  });
});
