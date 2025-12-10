"use client";

import { useEffect, useState } from "react";

export function useAuth() {
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [refreshToken, setRefreshToken] = useState<string | null>(null);
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (typeof window === "undefined") return;
    const access = window.localStorage.getItem("athena_access_token");
    const refresh = window.localStorage.getItem("athena_refresh_token");
    
    // Batch state updates to avoid cascading renders
    Promise.resolve().then(() => {
      setAccessToken(access);
      setRefreshToken(refresh);
      setInitialized(true);
    });
  }, []);

  const isAuthenticated = !!accessToken;

  return { accessToken, refreshToken, isAuthenticated, initialized };
}
