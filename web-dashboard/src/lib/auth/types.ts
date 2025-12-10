export type UserRole = "admin" | "operator" | "viewer";

export interface AuthUser {
  id: string;
  email: string;
  roles: UserRole[];
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
}

export interface RefreshRequest {
  refreshToken: string;
}

export interface RefreshResponse {
  accessToken: string;
  refreshToken?: string;
}

export interface LogoutRequest {
  refreshToken: string;
}

export type MeResponse = AuthUser;

export interface ApiError {
  error: string;
  code: string;
}
