"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { authApi } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import toast from "react-hot-toast";
import { Eye, EyeOff, UserPlus } from "lucide-react";

export default function RegisterPage() {
  const router = useRouter();
  const { login } = useAuth();
  const [form, setForm] = useState({
    username: "",
    email: "",
    password: "",
    team_name: "",
  });
  const [showPw, setShowPw] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (form.password.length < 8) {
      setError("Password minimal 8 karakter");
      return;
    }
    setLoading(true);

    try {
      const res = await authApi.register(form);
      login(res.token, res.user);
      toast.success("Akun berhasil dibuat! Selamat datang! 🎉");
      router.push("/squad");
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Registrasi gagal";
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  const fields = [
    { id: "reg-username", key: "username", label: "Username", type: "text", placeholder: "nama_kamu", autocomplete: "username" },
    { id: "reg-email", key: "email", label: "Email", type: "email", placeholder: "nama@email.com", autocomplete: "email" },
    { id: "reg-teamname", key: "team_name", label: "Nama Tim Fantasy", type: "text", placeholder: "Misal: Garuda XI", autocomplete: "off" },
  ];

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
      <div style={{ width: "100%", maxWidth: 480 }}>
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
            Buat akun gratis
          </h1>
          <p style={{ color: "var(--text-secondary)", fontSize: "0.9rem" }}>
            Sudah punya akun?{" "}
            <Link href="/auth/login" style={{ color: "var(--fsl-green)", fontWeight: 600 }}>
              Masuk di sini
            </Link>
          </p>
        </div>

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

          <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: "1.125rem" }}>
            {fields.map(({ id, key, label, type, placeholder, autocomplete }) => (
              <div className="form-group" key={key}>
                <label className="form-label" htmlFor={id}>
                  {label}
                </label>
                <input
                  id={id}
                  type={type}
                  className="form-input"
                  placeholder={placeholder}
                  value={form[key as keyof typeof form]}
                  onChange={(e) => setForm({ ...form, [key]: e.target.value })}
                  required
                  autoComplete={autocomplete}
                />
              </div>
            ))}

            <div className="form-group">
              <label className="form-label" htmlFor="reg-password">
                Password
              </label>
              <div style={{ position: "relative" }}>
                <input
                  id="reg-password"
                  type={showPw ? "text" : "password"}
                  className="form-input"
                  placeholder="Min. 8 karakter"
                  value={form.password}
                  onChange={(e) => setForm({ ...form, password: e.target.value })}
                  required
                  autoComplete="new-password"
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
                  aria-label="Toggle password"
                >
                  {showPw ? <EyeOff size={18} /> : <Eye size={18} />}
                </button>
              </div>
              {/* Password strength hint */}
              {form.password.length > 0 && (
                <div style={{ display: "flex", gap: 4, marginTop: 6 }}>
                  {[1, 2, 3, 4].map((i) => (
                    <div
                      key={i}
                      style={{
                        flex: 1,
                        height: 3,
                        borderRadius: 2,
                        background:
                          form.password.length >= i * 3
                            ? i <= 1 ? "var(--fsl-red)"
                            : i <= 2 ? "var(--fsl-gold)"
                            : "var(--fsl-green)"
                            : "var(--bg-border)",
                        transition: "background 0.3s ease",
                      }}
                    />
                  ))}
                </div>
              )}
            </div>

            <button
              type="submit"
              id="register-submit"
              className="btn btn-primary"
              style={{ width: "100%", padding: "0.875rem", fontSize: "1rem", marginTop: "0.25rem" }}
              disabled={loading}
            >
              {loading ? (
                <div className="spinner" style={{ width: 20, height: 20 }} />
              ) : (
                <>
                  <UserPlus size={18} /> Buat Akun
                </>
              )}
            </button>

            <p style={{ fontSize: "0.8rem", color: "var(--text-muted)", textAlign: "center" }}>
              Dengan mendaftar, kamu menyetujui syarat dan ketentuan penggunaan.
            </p>
          </form>
        </div>
      </div>
    </div>
  );
}
