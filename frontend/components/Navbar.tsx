"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";
import { useAuth } from "@/lib/auth-context";
import {
  LayoutDashboard,
  Users,
  Trophy,
  LogOut,
  Menu,
  X,
  ChevronDown,
} from "lucide-react";

const navLinks = [
  { href: "/squad", label: "Tim Saya", icon: LayoutDashboard },
  { href: "/players", label: "Pemain", icon: Users },
  { href: "/leaderboard", label: "Leaderboard", icon: Trophy },
];

export default function Navbar() {
  const pathname = usePathname();
  const router = useRouter();
  const { user, isLoggedIn, logout } = useAuth();
  const [menuOpen, setMenuOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);

  const handleLogout = () => {
    logout();
    router.push("/");
    setUserMenuOpen(false);
    setMenuOpen(false);
  };

  return (
    <nav className="navbar">
      <div className="container-app navbar-inner">
        {/* Logo */}
        <Link href="/" className="navbar-logo">
          <div className="navbar-logo-icon">⚽</div>
          <span className="navbar-logo-text">
            Fantasy <span>Super League</span>
          </span>
        </Link>

        {/* Desktop Nav */}
        {isLoggedIn && (
          <ul className="navbar-nav">
            {navLinks.map(({ href, label, icon: Icon }) => (
              <li key={href}>
                <Link
                  href={href}
                  className={`navbar-link ${pathname === href ? "active" : ""}`}
                >
                  <Icon size={16} />
                  {label}
                </Link>
              </li>
            ))}
          </ul>
        )}

        {/* Right actions */}
        <div className="navbar-actions">
          {isLoggedIn ? (
            <div style={{ position: "relative" }}>
              <button
                className="btn btn-ghost btn-sm"
                onClick={() => setUserMenuOpen(!userMenuOpen)}
                style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}
              >
                <div
                  style={{
                    width: 28,
                    height: 28,
                    borderRadius: "50%",
                    background: "linear-gradient(135deg, var(--fsl-green), var(--fsl-blue))",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    fontSize: "0.75rem",
                    fontWeight: 700,
                    color: "#000",
                  }}
                >
                  {user?.username?.charAt(0).toUpperCase()}
                </div>
                <span style={{ fontSize: "0.875rem", fontWeight: 500 }}>
                  {user?.username}
                </span>
                <ChevronDown size={14} />
              </button>

              {userMenuOpen && (
                <div
                  style={{
                    position: "absolute",
                    top: "calc(100% + 8px)",
                    right: 0,
                    background: "#FFFFFF",
                    border: "1px solid #D5E5F7",
                    borderRadius: 12,
                    padding: "0.5rem",
                    minWidth: 180,
                    boxShadow: "0 16px 48px rgba(11,36,71,0.12)",
                    zIndex: 50,
                    animation: "scaleIn 0.15s ease",
                  }}
                >
                  <div
                    style={{
                      padding: "0.5rem 0.75rem",
                      borderBottom: "1px solid #D5E5F7",
                      marginBottom: "0.25rem",
                    }}
                  >
                    <p style={{ fontSize: "0.75rem", color: "#7E96B3", marginBottom: 2 }}>
                      Tim kamu
                    </p>
                    <p style={{ fontSize: "0.875rem", fontWeight: 600, color: "#0B2447" }}>
                      {user?.team_name || "Belum ada tim"}
                    </p>
                  </div>
                  <button
                    className="btn btn-ghost btn-sm"
                    style={{ width: "100%", justifyContent: "flex-start", color: "var(--fsl-red)" }}
                    onClick={handleLogout}
                  >
                    <LogOut size={14} />
                    Keluar
                  </button>
                </div>
              )}
            </div>
          ) : (
            <>
              <Link href="/auth/login" className="btn btn-ghost btn-sm">
                Masuk
              </Link>
              <Link href="/auth/register" className="btn btn-primary btn-sm">
                Daftar
              </Link>
            </>
          )}

          {/* Mobile menu toggle */}
          {isLoggedIn && (
            <button
              className="btn btn-ghost btn-sm mobile-menu-toggle"
              onClick={() => setMenuOpen(!menuOpen)}
              aria-label="Toggle menu"
              aria-expanded={menuOpen}
            >
              {menuOpen ? <X size={20} /> : <Menu size={20} />}
            </button>
          )}
        </div>
      </div>

      {/* Mobile dropdown */}
      {isLoggedIn && menuOpen && (
        <div className="navbar-mobile-menu">
          <ul>
            {navLinks.map(({ href, label, icon: Icon }) => (
              <li key={href}>
                <Link
                  href={href}
                  className={`navbar-link ${pathname === href ? "active" : ""}`}
                  onClick={() => setMenuOpen(false)}
                >
                  <Icon size={16} />
                  {label}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      )}
    </nav>
  );
}
