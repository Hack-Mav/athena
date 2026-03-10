"use client";

import { useEffect, useState } from "react";
import type { AuthUser } from "@/lib/auth/types";

export function useAuth() {
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [refreshToken, setRefreshToken] = useState<string | null>(null);
  const [user, setUser] = useState<AuthUser | null>(null);
  const [isDemoMode, setIsDemoMode] = useState(false);
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (typeof window === "undefined") return;
    
    const access = window.localStorage.getItem("athena_access_token");
    const refresh = window.localStorage.getItem("athena_refresh_token");
    const userStr = window.localStorage.getItem("athena_user");
    const demoMode = window.localStorage.getItem("athena_demo_mode") === "true";
    
    // Batch state updates to avoid cascading renders
    Promise.resolve().then(() => {
      setAccessToken(access);
      setRefreshToken(refresh);
      setUser(userStr ? JSON.parse(userStr) : null);
      setIsDemoMode(demoMode);
      setInitialized(true);
    });
  }, []);

  const isAuthenticated = !!accessToken;

  return { 
    accessToken, 
    refreshToken, 
    user, 
    isDemoMode,
    isAuthenticated, 
    initialized 
  };
}
