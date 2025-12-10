"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";
import { useAuth } from "@/hooks/useAuth";
import { logout } from "@/lib/auth/client";

const navItems = [
  { href: "/dashboard", label: "Overview" },
  { href: "/templates", label: "Templates" },
  { href: "/provisioning", label: "Provisioning" },
  { href: "/monitoring", label: "Monitoring" },
  { href: "/ota", label: "OTA" },
];

export function DashboardShell({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const { refreshToken, isAuthenticated, initialized } = useAuth();

  useEffect(() => {
    if (!initialized) return;
    if (!isAuthenticated) {
      router.replace("/login");
    }
  }, [initialized, isAuthenticated, router]);

  async function handleLogout() {
    if (refreshToken) {
      try {
        await logout({ refreshToken });
      } catch {
        // Continue with local cleanup even if API logout fails
      }
    }
    if (typeof window !== "undefined") {
      window.localStorage.removeItem("athena_access_token");
      window.localStorage.removeItem("athena_refresh_token");
    }
    router.push("/login");
  }

  if (!initialized || !isAuthenticated) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-50">
        <p className="text-sm text-zinc-600">Loading dashboard...</p>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen bg-zinc-50">
      <aside className="hidden w-64 flex-col border-r border-zinc-200 bg-white px-4 py-6 md:flex">
        <div className="mb-8 text-lg font-semibold tracking-tight text-zinc-900">
          ATHENA
        </div>
        <nav className="flex flex-1 flex-col gap-1 text-sm">
          {navItems.map((item) => {
            const active = pathname === item.href;
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`rounded-md px-3 py-2 font-medium transition-colors ${
                  active
                    ? "bg-zinc-900 text-white"
                    : "text-zinc-700 hover:bg-zinc-100 hover:text-zinc-900"
                }`}
              >
                {item.label}
              </Link>
            );
          })}
        </nav>
        <button
          type="button"
          onClick={handleLogout}
          className="mt-4 rounded-md px-3 py-2 text-left text-sm font-medium text-zinc-700 hover:bg-zinc-100 hover:text-zinc-900"
        >
          Log out
        </button>
      </aside>
      <div className="flex min-h-screen flex-1 flex-col">
        <header className="flex items-center justify-between border-b border-zinc-200 bg-white px-4 py-3 md:px-6">
          <h1 className="text-base font-semibold text-zinc-900 md:text-lg">
            {title}
          </h1>
        </header>
        <main className="flex-1 px-4 py-4 md:px-6 md:py-6">{children}</main>
      </div>
    </div>
  );
}
