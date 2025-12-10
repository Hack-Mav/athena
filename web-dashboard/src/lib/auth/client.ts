import type {
  LoginRequest,
  LoginResponse,
  RefreshRequest,
  RefreshResponse,
  LogoutRequest,
  MeResponse,
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

export async function login(body: LoginRequest): Promise<LoginResponse> {
  const res = await fetch(`${API_BASE_URL}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return handleResponse<LoginResponse>(res);
}

export async function refresh(body: RefreshRequest): Promise<RefreshResponse> {
  const res = await fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return handleResponse<RefreshResponse>(res);
}

export async function logout(body: LogoutRequest): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/api/v1/auth/logout`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    let err: ApiError | undefined;
    try {
      err = (await res.json()) as ApiError;
    } catch {}
    throw new Error(err?.error ?? `Logout failed with status ${res.status}`);
  }
}

export async function me(accessToken: string): Promise<MeResponse> {
  const res = await fetch(`${API_BASE_URL}/api/v1/auth/me`, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });
  return handleResponse<MeResponse>(res);
}
