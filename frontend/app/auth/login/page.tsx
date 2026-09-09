"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { authApi } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import toast from "react-hot-toast";
import { Eye, EyeOff, LogIn } from "lucide-react";

export default function LoginPage() {
  const router = useRouter();
  const { login } = useAuth();
  const [form, setForm] = useState({ email: "", password: "" });
  const [showPw, setShowPw] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await authApi.login(form);
      login(res.token, res.user);
      toast.success(`Selamat datang kembali, ${res.user.username}!`);
      router.push("/squad");
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Login gagal";
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      style={{
        minHeight: "calc(100vh - 72px)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: "2rem 1rem",
        background:
          "radial-gradient(ellipse 60% 50% at 50% 0%, rgba(0,208,132,0.08) 0%, transparent 70%)",
      }}
    >
      <div style={{ width: "100%", maxWidth: 440 }}>
        {/* Header */}
        <div style={{ textAlign: "center", marginBottom: "2rem" }}>
          <div
            style={{
              width: 60,
              height: 60,
              borderRadius: 16,
              background: "linear-gradient(135deg, var(--fsl-green), var(--fsl-blue))",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontSize: "1.75rem",
              margin: "0 auto 1rem",
            }}
          >
            ⚽
          </div>
          <h1 className="heading-md" style={{ marginBottom: "0.5rem" }}>
            Masuk ke akunmu
          </h1>
          <p style={{ color: "var(--text-secondary)", fontSize: "0.9rem" }}>
            Belum punya akun?{" "}
            <Link href="/auth/register" style={{ color: "var(--fsl-green)", fontWeight: 600 }}>
              Daftar gratis
            </Link>
          </p>
        </div>

        {/* Card */}
        <div className="card animate-scaleIn">
          {error && (
            <div
              style={{
                padding: "0.75rem 1rem",
                background: "rgba(239, 68, 68, 0.1)",
                border: "1px solid rgba(239, 68, 68, 0.3)",
                borderRadius: 10,
                color: "var(--fsl-red)",
                fontSize: "0.875rem",
                marginBottom: "1.25rem",
              }}
            >
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: "1.25rem" }}>
            <div className="form-group">
              <label className="form-label" htmlFor="login-email">
                Email
              </label>
              <input
                id="login-email"
                type="email"
                className="form-input"
                placeholder="nama@email.com"
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
                required
                autoComplete="email"
              />
            </div>

            <div className="form-group">
              <label className="form-label" htmlFor="login-password">
                Password
              </label>
              <div style={{ position: "relative" }}>
                <input
                  id="login-password"
                  type={showPw ? "text" : "password"}
                  className="form-input"
                  placeholder="••••••••"
                  value={form.password}
                  onChange={(e) => setForm({ ...form, password: e.target.value })}
                  required
                  autoComplete="current-password"
                  style={{ paddingRight: "3rem" }}
                />
                <button
                  type="button"
                  onClick={() => setShowPw(!showPw)}
                  style={{
                    position: "absolute",
                    right: "0.875rem",
                    top: "50%",
                    transform: "translateY(-50%)",
                    background: "none",
                    border: "none",
                    cursor: "pointer",
                    color: "var(--text-muted)",
                    display: "flex",
                  }}
                  aria-label={showPw ? "Sembunyikan password" : "Tampilkan password"}
                >
                  {showPw ? <EyeOff size={18} /> : <Eye size={18} />}
                </button>
              </div>
            </div>

            <button
              type="submit"
              className="btn btn-primary"
              style={{ width: "100%", padding: "0.875rem", fontSize: "1rem", marginTop: "0.25rem" }}
              disabled={loading}
              id="login-submit"
            >
              {loading ? (
                <div className="spinner" style={{ width: 20, height: 20 }} />
              ) : (
                <>
                  <LogIn size={18} /> Masuk
                </>
              )}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
