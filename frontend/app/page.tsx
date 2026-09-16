"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useAuth } from "@/lib/auth-context";
import {
  gameweeksApi, leaderboardApi, playersApi, clubsApi,
  Gameweek, LeaderboardEntry, Player, Club, formatPrice,
} from "@/lib/api";
import { ArrowRight, Trophy, Users, Star, Zap, Shield, Target, Crown, ChevronRight } from "lucide-react";
import PlayerAvatar from "@/components/PlayerAvatar";

const FEATURES = [
  {
    icon: Users,
    title: "15 Pemain Liga 1",
    desc: "Pilih 15 pemain terbaik dari 16 klub BRI Super League dengan budget Rp 100 juta.",
    color: "var(--fsl-green)",
  },
  {
    icon: Trophy,
    title: "Poin Real-time",
    desc: "Poin dihitung otomatis berdasarkan performa nyata: gol, assist, clean sheet, dan lainnya.",
    color: "var(--fsl-gold)",
  },
  {
    icon: Zap,
    title: "Transfer Tiap Pekan",
    desc: "1 transfer gratis per gameweek. Gunakan Wildcard untuk reset tim penuh 1× semusim.",
    color: "var(--fsl-blue)",
  },
  {
    icon: Shield,
    title: "Kapten & Wakil",
    desc: "Poin kapten dikalikan 2. Pilih strategi kapten yang tepat setiap gameweek.",
    color: "var(--fsl-purple)",
  },
];

const SCORING = [
  { action: "Tampil ≥60 mnt", pts: "+2", pos: "Semua" },
  { action: "Gol", pts: "+4–6", pos: "FWD/MID/DEF-GK" },
  { action: "Assist", pts: "+3", pos: "Semua" },
  { action: "Clean Sheet", pts: "+4", pos: "GK/DEF" },
  { action: "Kartu Kuning", pts: "−1", pos: "Semua" },
  { action: "Kartu Merah", pts: "−3", pos: "Semua" },
];

export default function HomePage() {
  const { isLoggedIn } = useAuth();
  const [activeGw, setActiveGw] = useState<Gameweek | null>(null);
  const [topManagers, setTopManagers] = useState<LeaderboardEntry[]>([]);
  const [topScorers, setTopScorers] = useState<Player[]>([]);
  const [clubs, setClubs] = useState<Club[]>([]);

  useEffect(() => {
    gameweeksApi.getAll().then((res) => {
      const gw = res.data?.find((g) => g.is_active);
      setActiveGw(gw || null);
    }).catch(() => {});

    leaderboardApi.get().then((res) => {
      setTopManagers(res.data?.slice(0, 5) || []);
    }).catch(() => {});

    playersApi.getAll({ limit: 5 }).then((res) => {
      setTopScorers(res.data || []);
    }).catch(() => {});

    clubsApi.getAll().then((res) => {
      setClubs(res.data || []);
    }).catch(() => {});
  }, []);

  const featured = topScorers[0] || null;

  return (
    <div className="landing-light">
      {/* ─── HERO ─── */}
      <section className="hero">
        <div className="hero-bg" />
        <div className="hero-grid" />
        <div className="container-app" style={{ width: "100%", zIndex: 1 }}>
          <div
            className="hero-grid-2col"
            style={{
              paddingTop: "3rem",
              paddingBottom: "2.5rem",
              display: "grid",
              gridTemplateColumns: "1.2fr 0.8fr",
              gap: "3rem",
              alignItems: "center",
            }}
          >
          <div style={{ maxWidth: 560 }}>
            {/* Active gameweek badge */}
            {activeGw && (
              <div
                className="animate-fadeIn"
                style={{
                  display: "inline-flex",
                  alignItems: "center",
                  gap: "0.5rem",
                  padding: "0.35rem 0.875rem",
                  background: "rgba(0, 82, 156, 0.08)",
                  border: "1px solid rgba(0, 82, 156, 0.25)",
                  borderRadius: 999,
                  fontSize: "0.8125rem",
                  fontWeight: 600,
                  color: "var(--fsl-green)",
                  marginBottom: "1.5rem",
                }}
              >
                <span
                  style={{
                    width: 8,
                    height: 8,
                    borderRadius: "50%",
                    background: "var(--fsl-green)",
                    animation: "pulse-green 2s infinite",
                  }}
                />
                🔵 LIVE — {activeGw.name}
              </div>
            )}

            <h1 className="heading-hero animate-fadeInUp" style={{ marginBottom: "1.25rem" }}>
              Fantasy Football<br />
              Liga Indonesia
            </h1>

            <p
              className="animate-fadeInUp"
              style={{
                fontSize: "1.125rem",
                color: "var(--text-secondary)",
                lineHeight: 1.7,
                marginBottom: "2.5rem",
                animationDelay: "0.1s",
                maxWidth: 520,
              }}
            >
              Buat tim fantasy-mu dari pemain-pemain BRI Super League 2026–27.
              Kompetisi, kumpulkan poin, dan buktikan kamu manajer terbaik Indonesia!
            </p>

            <div
              className="animate-fadeInUp"
              style={{
                display: "flex",
                gap: "1rem",
                flexWrap: "wrap",
                animationDelay: "0.2s",
              }}
            >
              {isLoggedIn ? (
                <>
                  <Link href="/squad" className="btn btn-primary btn-lg">
                    Tim Saya <ArrowRight size={18} />
                  </Link>
                  <Link href="/players" className="btn btn-secondary btn-lg">
                    <Users size={18} /> Browse Pemain
                  </Link>
                </>
              ) : (
                <>
                  <Link href="/auth/register" className="btn btn-primary btn-lg">
                    Mulai Gratis <ArrowRight size={18} />
                  </Link>
                  <Link href="/auth/login" className="btn btn-secondary btn-lg">
                    Sudah punya akun? Masuk
                  </Link>
                </>
              )}
            </div>

            {/* Quick stats */}
            <div
              className="animate-fadeInUp"
              style={{
                display: "flex",
                gap: "2rem",
                marginTop: "3rem",
                animationDelay: "0.3s",
                flexWrap: "wrap",
              }}
            >
              {[
                { label: "Klub Liga 1", value: String(clubs.length || 18) },
                { label: "Budget Fantasy", value: "Rp 100jt" },
                { label: "Pemain / Tim", value: "15" },
                { label: "Transfer/Pekan", value: "1 Gratis" },
              ].map((stat) => (
                <div key={stat.label}>
                  <div
                    style={{
                      fontSize: "1.5rem",
                      fontWeight: 800,
                      color: "var(--text-primary)",
                      fontFamily: "Space Grotesk, sans-serif",
                    }}
                  >
                    {stat.value}
                  </div>
                  <div style={{ fontSize: "0.8125rem", color: "var(--text-muted)" }}>
                    {stat.label}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Featured player card (ala hero referensi) */}
          <div className="animate-fadeInUp" style={{ animationDelay: "0.25s" }}>
            <div className="card featured-card" style={{ overflow: "hidden", padding: 0 }}>
              <div
                style={{
                  background: "linear-gradient(135deg, #00529C 0%, #003E7A 60%, #0B2447 100%)",
                  padding: "1.25rem 1.25rem 0",
                  color: "#fff",
                }}
              >
                <div style={{ display: "flex", alignItems: "center", gap: "0.5rem", marginBottom: "1rem" }}>
                  <Crown size={16} color="#F59E0B" />
                  <span style={{ fontSize: "0.8125rem", fontWeight: 700, textTransform: "uppercase", letterSpacing: "0.08em", opacity: 0.9 }}>
                    Pemain Unggulan
                  </span>
                </div>
                {featured ? (
                  <div style={{ display: "flex", gap: "1rem", alignItems: "flex-end" }}>
                    <div style={{ width: 120, height: 140, borderRadius: "12px 12px 0 0", overflow: "hidden", flexShrink: 0, background: "rgba(255,255,255,0.15)" }}>
                      <PlayerAvatar key={featured.id} src={featured.photo_url} alt={featured.name} />
                    </div>
                    <div style={{ paddingBottom: "1.25rem", minWidth: 0 }}>
                      <div style={{ fontSize: "1.25rem", fontWeight: 800, lineHeight: 1.2, marginBottom: "0.25rem" }}>
                        {featured.name}
                      </div>
                      <div style={{ fontSize: "0.8125rem", opacity: 0.85, marginBottom: "0.75rem" }}>
                        {featured.club?.name || "—"} · {featured.position}
                      </div>
                      <div style={{ display: "inline-flex", alignItems: "baseline", gap: "0.375rem", background: "rgba(255,255,255,0.15)", borderRadius: 10, padding: "0.375rem 0.75rem" }}>
                        <span style={{ fontSize: "1.5rem", fontWeight: 900 }}>{formatPrice(featured.price)}</span>
                        <span style={{ fontSize: "0.75rem", opacity: 0.85 }}>{featured.total_points} pts</span>
                      </div>
                    </div>
                  </div>
                ) : (
                  <div style={{ padding: "2rem 0", textAlign: "center", opacity: 0.8, fontSize: "0.9rem" }}>
                    Belum ada data pemain
                  </div>
                )}
              </div>
              <Link
                href="/players"
                className="btn btn-secondary btn-sm"
                style={{ margin: "1rem 1.25rem 1.25rem", justifyContent: "center" }}
              >
                Lihat Semua Pemain <ChevronRight size={14} />
              </Link>
            </div>
          </div>

          </div>
        </div>
      </section>

      {/* ─── CLUB STRIP ─── */}
      <section style={{ padding: "2.5rem 0", background: "var(--bg-surface)", borderTop: "1px solid var(--bg-border)", borderBottom: "1px solid var(--bg-border)" }}>
        <div className="container-app">
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "1.25rem" }}>
            <h2 className="heading-md">Klub BRI Super League</h2>
            <Link href="/players" className="btn btn-ghost btn-sm">
              Semua Pemain <ChevronRight size={14} />
            </Link>
          </div>
          <div className="club-strip">
            {clubs.map((club) => (
              <Link key={club.id} href="/players" className="club-chip" title={club.name}>
                <span className="club-chip-logo">
                  <span className="club-chip-initials">
                    {(club.short_name || club.name).slice(0, 3).toUpperCase()}
                  </span>
                  {club.logo_url ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={club.logo_url}
                      alt={club.name}
                      loading="lazy"
                      onError={(e) => {
                        (e.target as HTMLImageElement).style.display = "none";
                      }}
                    />
                  ) : null}
                </span>
                <span className="club-chip-name">{club.name}</span>
              </Link>
            ))}
            {clubs.length === 0 && (
              <p style={{ color: "var(--text-muted)", fontSize: "0.875rem" }}>Memuat klub…</p>
            )}
          </div>
        </div>
      </section>

      {/* ─── TOP PEMAIN ─── */}
      <section style={{ padding: "5rem 0" }}>
        <div className="container-app">
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "0.5rem" }}>
            <h2 className="heading-lg">Top Pemain</h2>
            <Link href="/players" className="btn btn-ghost btn-sm">
              Semua <ChevronRight size={14} />
            </Link>
          </div>
          <p style={{ color: "var(--text-secondary)", fontSize: "1.0625rem", marginBottom: "2rem" }}>
            Peringkat pemain berdasarkan total poin fantasy
          </p>
          {topScorers.length === 0 ? (
            <div className="card" style={{ textAlign: "center", color: "var(--text-muted)", padding: "3rem" }}>
              Belum ada data pemain
            </div>
          ) : (
            <div
              className="stagger"
              style={{
                display: "grid",
                gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))",
                gap: "1.25rem",
              }}
            >
              {topScorers.map((p, i) => (
                <div key={p.id} className="card card-hover" style={{ padding: 0, overflow: "hidden" }}>
                  <div style={{ position: "relative" }}>
                    <PlayerAvatar key={p.id} src={p.photo_url} alt={p.name} className="player-card-photo" />
                    <div className={`rank-badge ${i < 3 ? `rank-${i + 1}` : "rank-other"}`} style={{ position: "absolute", top: 8, left: 8 }}>
                      {i + 1}
                    </div>
                  </div>
                  <div style={{ padding: "0.875rem 1rem 1rem" }}>
                    <div style={{ fontWeight: 700, fontSize: "0.9375rem", whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
                      {p.name}
                    </div>
                    <div style={{ fontSize: "0.75rem", color: "var(--text-secondary)", marginBottom: "0.75rem" }}>
                      {p.club?.short_name || p.club?.name || "—"} · {p.position}
                    </div>
                    <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
                      <span style={{ fontWeight: 800, fontSize: "1.125rem", color: "var(--fsl-green)" }}>
                        {p.total_points} <span style={{ fontSize: "0.75rem", fontWeight: 400, color: "var(--text-muted)" }}>pts</span>
                      </span>
                      <span style={{ fontSize: "0.8125rem", fontWeight: 600, color: "var(--text-secondary)" }}>
                        {formatPrice(p.price)}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </section>

      {/* ─── FEATURES ─── */}
      <section style={{ padding: "5rem 0", background: "var(--bg-surface)" }}>
        <div className="container-app">
          <div style={{ textAlign: "center", marginBottom: "3rem" }}>
            <h2 className="heading-lg" style={{ marginBottom: "0.75rem" }}>
              Cara Bermain
            </h2>
            <p style={{ color: "var(--text-secondary)", fontSize: "1.0625rem" }}>
              Sistem scoring mirip FPL, disesuaikan untuk Liga 1 Indonesia
            </p>
          </div>

          <div
            className="stagger"
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fit, minmax(240px, 1fr))",
              gap: "1.25rem",
            }}
          >
            {FEATURES.map(({ icon: Icon, title, desc, color }) => (
              <div key={title} className="card card-hover">
                <div
                  style={{
                    width: 48,
                    height: 48,
                    borderRadius: 12,
                    background: `${color}18`,
                    border: `1px solid ${color}30`,
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    marginBottom: "1rem",
                  }}
                >
                  <Icon size={22} color={color} />
                </div>
                <h3 className="heading-sm" style={{ marginBottom: "0.5rem" }}>
                  {title}
                </h3>
                <p style={{ color: "var(--text-secondary)", fontSize: "0.9rem", lineHeight: 1.6 }}>
                  {desc}
                </p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ─── SCORING + LEADERBOARD ─── */}
      <section style={{ padding: "5rem 0" }}>
        <div className="container-app">
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))",
              gap: "2rem",
            }}
          >
            {/* Scoring table */}
            <div className="card">
              <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", marginBottom: "1.5rem" }}>
                <Target size={20} color="var(--fsl-green)" />
                <h2 className="heading-md">Sistem Poin</h2>
              </div>
              <table style={{ width: "100%", borderCollapse: "separate", borderSpacing: "0 6px" }}>
                <thead>
                  <tr>
                    <th style={{ textAlign: "left", fontSize: "0.75rem", color: "var(--text-muted)", padding: "0 0.75rem", textTransform: "uppercase", letterSpacing: "0.05em" }}>
                      Aksi
                    </th>
                    <th style={{ textAlign: "center", fontSize: "0.75rem", color: "var(--text-muted)", padding: "0 0.75rem", textTransform: "uppercase", letterSpacing: "0.05em" }}>
                      Poin
                    </th>
                    <th style={{ textAlign: "right", fontSize: "0.75rem", color: "var(--text-muted)", padding: "0 0.75rem", textTransform: "uppercase", letterSpacing: "0.05em" }}>
                      Posisi
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {SCORING.map((row) => (
                    <tr key={row.action}>
                      <td
                        style={{
                          padding: "0.625rem 0.75rem",
                          background: "var(--bg-elevated)",
                          borderRadius: "8px 0 0 8px",
                          fontSize: "0.875rem",
                        }}
                      >
                        {row.action}
                      </td>
                      <td
                        style={{
                          padding: "0.625rem 0.75rem",
                          background: "var(--bg-elevated)",
                          textAlign: "center",
                          fontWeight: 700,
                          color: row.pts.startsWith("+") ? "var(--fsl-green)" : "var(--fsl-red)",
                        }}
                      >
                        {row.pts}
                      </td>
                      <td
                        style={{
                          padding: "0.625rem 0.75rem",
                          background: "var(--bg-elevated)",
                          borderRadius: "0 8px 8px 0",
                          textAlign: "right",
                          fontSize: "0.8125rem",
                          color: "var(--text-secondary)",
                        }}
                      >
                        {row.pos}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <Link href="/scoring" className="btn btn-ghost btn-sm" style={{ marginTop: "1rem", width: "100%", justifyContent: "center" }}>
                Lihat semua aturan
              </Link>
            </div>

            {/* Top leaderboard preview */}
            <div className="card">
              <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "1.5rem" }}>
                <div style={{ display: "flex", alignItems: "center", gap: "0.75rem" }}>
                  <Trophy size={20} color="var(--fsl-gold)" />
                  <h2 className="heading-md">Top Manajer</h2>
                </div>
                <Link href="/leaderboard" className="btn btn-ghost btn-sm">
                  Semua <ArrowRight size={14} />
                </Link>
              </div>

              {topManagers.length === 0 ? (
                <div style={{ textAlign: "center", padding: "2rem", color: "var(--text-muted)" }}>
                  <Star size={32} style={{ marginBottom: "0.75rem", opacity: 0.4 }} />
                  <p>Belum ada data leaderboard</p>
                </div>
              ) : (
                <div style={{ display: "flex", flexDirection: "column", gap: "0.375rem" }}>
                  {topManagers.map((entry) => (
                    <div key={entry.user_id} className={`leaderboard-row top-${entry.rank}`}>
                      <div className={`rank-badge rank-${entry.rank <= 3 ? entry.rank : "other"}`}>
                        {entry.rank}
                      </div>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <p style={{ fontWeight: 600, fontSize: "0.9rem", whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
                          {entry.team_name}
                        </p>
                        <p style={{ fontSize: "0.75rem", color: "var(--text-secondary)" }}>
                          @{entry.username}
                        </p>
                      </div>
                      <div className="points-chip" style={{ padding: "0.5rem 0.75rem", minWidth: 60 }}>
                        <span className="points-chip-value" style={{ fontSize: "1.25rem" }}>
                          {entry.total_points}
                        </span>
                        <span className="points-chip-label">pts</span>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      </section>

      {/* ─── CTA ─── */}
      {!isLoggedIn && (
        <section
          style={{
            padding: "5rem 0",
            background: "linear-gradient(135deg, rgba(0, 82, 156, 0.07), rgba(0, 82, 156, 0.02))",
            borderTop: "1px solid var(--bg-border)",
          }}
        >
          <div className="container-app" style={{ textAlign: "center" }}>
            <h2 className="heading-lg" style={{ marginBottom: "1rem" }}>
              Siap jadi manajer terbaik?
            </h2>
            <p style={{ color: "var(--text-secondary)", marginBottom: "2rem", fontSize: "1.0625rem" }}>
              Daftar gratis, pilih timmu, dan mulai kompetisi!
            </p>
            <Link href="/auth/register" className="btn btn-primary btn-lg">
              Buat Tim Sekarang <ArrowRight size={18} />
            </Link>
          </div>
        </section>
      )}

      {/* Footer */}
      <footer
        style={{
          borderTop: "1px solid var(--bg-border)",
          padding: "2rem 0",
          textAlign: "center",
          color: "var(--text-muted)",
          fontSize: "0.8125rem",
        }}
      >
        <div className="container-app">
          <p>
            Fantasy Super League — Tidak terafiliasi dengan BRI Super League atau PSSI secara resmi.
          </p>
          <p style={{ marginTop: "0.375rem" }}>
            Data pemain bersumber dari{" "}
            <a href="https://ileague.id" target="_blank" rel="noopener noreferrer" style={{ color: "var(--fsl-green)" }}>
              ileague.id
            </a>
          </p>
        </div>
      </footer>
    </div>
  );
}
