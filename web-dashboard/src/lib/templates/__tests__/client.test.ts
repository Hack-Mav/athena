import { getTemplates } from "../client";

// Mock fetch
const mockFetch = jest.fn();
global.fetch = mockFetch;

beforeEach(() => {
  mockFetch.mockClear();
});

describe("Templates API client", () => {
  it("handles successful response", async () => {
    const mockResponse = {
      devices: [],
      total: 0,
      page: 1,
      pageSize: 12,
    };
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse,
    });

    const result = await getTemplates();
    expect(result).toEqual(mockResponse);
    expect(mockFetch).toHaveBeenCalledWith("http://localhost:8080/api/v1/templates?");
  });

  it("handles HTTP error with custom error message", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 404,
      json: async () => ({ error: "Not found", code: "NOT_FOUND" }),
    });

    await expect(getTemplates()).rejects.toThrow("Not found");
  });

  it("handles HTTP error without JSON body", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: async () => {
        throw new Error("Invalid JSON");
      },
    });

    await expect(getTemplates()).rejects.toThrow("Request failed with status 500");
  });
});
