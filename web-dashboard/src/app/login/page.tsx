"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { login } from "@/lib/auth/client";
import type { LoginResponse } from "@/lib/auth/types";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("admin@example.com");
  const [password, setPassword] = useState("admin123");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isDemoMode, setIsDemoMode] = useState(false);

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    
    try {
      let res: LoginResponse;
      
      if (isDemoMode) {
        // Demo mode - simulate API delay and return mock response
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        // Simple demo validation
        if (!email || !password) {
          throw new Error("Email and password are required");
        }
        
        res = {
          accessToken: `demo-token-${Date.now()}`,
          refreshToken: `demo-refresh-${Date.now()}`,
          user: {
            id: "demo-user-1",
            email: email,
            roles: ["admin"]
          }
        };
      } else {
        // Production mode - call actual API
        res = await login({ email, password });
      }
      
      if (typeof window !== "undefined") {
        window.localStorage.setItem("athena_access_token", res.accessToken);
        window.localStorage.setItem("athena_refresh_token", res.refreshToken);
        window.localStorage.setItem("athena_user", JSON.stringify(res.user));
        window.localStorage.setItem("athena_demo_mode", isDemoMode.toString());
      }
      
      router.push("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }, [email, password, router, isDemoMode]);

  // Auto-login on component mount for demo mode
  useEffect(() => {
    if (isDemoMode) {
      const formEvent = new Event("submit") as unknown as React.FormEvent;
      handleSubmit(formEvent);
    }
  }, [isDemoMode, handleSubmit]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-50 px-4">
      <div className="w-full max-w-md rounded-xl border border-zinc-200 bg-white p-8 shadow-sm">
        <div className="mb-6 text-center">
          <h1 className="text-2xl font-semibold text-zinc-900 mb-2">
            ATHENA Dashboard Login
          </h1>
          <div className="flex items-center justify-center gap-2">
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={isDemoMode}
                onChange={(e) => setIsDemoMode(e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
              <span className="ml-3 text-sm font-medium text-zinc-700">
                Demo Mode
              </span>
            </label>
          </div>
          {isDemoMode && (
            <p className="mt-2 text-xs text-zinc-500">
              Demo mode bypasses authentication and uses mock data
            </p>
          )}
        </div>
        
        <form className="space-y-4" onSubmit={handleSubmit}>
          <div className="space-y-1">
            <label className="block text-sm font-medium text-zinc-700">Email</label>
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder={isDemoMode ? "Enter any email" : "Enter your email"}
              className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
            />
          </div>
          <div className="space-y-1">
            <label className="block text-sm font-medium text-zinc-700">Password</label>
            <input
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={isDemoMode ? "Enter any password" : "Enter your password"}
              className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
            />
          </div>
          {error && (
            <p className="text-sm text-red-600" role="alert">
              {error}
            </p>
          )}
          <button
            type="submit"
            disabled={loading}
            className={`flex w-full items-center justify-center rounded-md px-3 py-2 text-sm font-medium text-white transition-colors ${
              isDemoMode 
                ? "bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300" 
                : "bg-zinc-900 hover:bg-zinc-800 disabled:bg-zinc-600"
            } disabled:opacity-60`}
          >
            {loading ? "Signing in..." : (isDemoMode ? "Sign in (Demo)" : "Sign in")}
          </button>
        </form>
        
        {isDemoMode && (
          <div className="mt-6 p-3 bg-blue-50 border border-blue-200 rounded-md">
            <p className="text-xs text-blue-700">
              <strong>Demo Credentials:</strong> Any email and password will work. 
              You&apos;ll be logged in as an admin user with mock data.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
