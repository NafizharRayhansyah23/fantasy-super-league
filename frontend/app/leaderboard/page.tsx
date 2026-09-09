"use client";

import { useEffect, useState } from "react";
import { leaderboardApi, LeaderboardEntry } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { Trophy, Search } from "lucide-react";

export default function LeaderboardPage() {
  const { user } = useAuth();
  const [entries, setEntries] = useState<LeaderboardEntry[]>([]);
  const [filtered, setFiltered] = useState<LeaderboardEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");

  useEffect(() => {
    leaderboardApi
      .get()
      .then((res) => {
        setEntries(res.data || []);
        setFiltered(res.data || []);
      })
      .catch(() => setEntries([]))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (!search) {
      setFiltered(entries);
    } else {
      const q = search.toLowerCase();
      setFiltered(
        entries.filter(
          (e) =>
            e.team_name.toLowerCase().includes(q) ||
            e.username.toLowerCase().includes(q)
        )
      );
    }
  }, [search, entries]);

  const myEntry = entries.find((e) => e.user_id === user?.id);

  const rankColors: Record<number, string> = {
    1: "var(--fsl-gold)",
    2: "#94A3B8",
    3: "#B47A3C",
  };

  const rankEmoji: Record<number, string> = {
    1: "🥇",
    2: "🥈",
    3: "🥉",
  };

  return (
    <div style={{ padding: "2rem 0 4rem" }}>
      <div className="container-app" style={{ maxWidth: 720 }}>
        {/* Header */}
        <div style={{ marginBottom: "2rem" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", marginBottom: "0.5rem" }}>
            <Trophy size={28} color="var(--fsl-gold)" />
            <h1 className="heading-lg">Leaderboard</h1>
          </div>
          <p style={{ color: "var(--text-secondary)" }}>
            Ranking {entries.length} manajer terbaik Fantasy Super League
          </p>
        </div>

        {/* My rank card */}
        {myEntry && (
          <div
            className="card card-glow"
            style={{
              marginBottom: "1.5rem",
              display: "flex",
              alignItems: "center",
              gap: "1rem",
              background: "linear-gradient(135deg, rgba(0, 208, 132, 0.08), rgba(59, 130, 246, 0.05))",
            }}
          >
            <div
              style={{
                fontSize: "1.5rem",
                width: 48,
                height: 48,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                background: "var(--bg-elevated)",
                borderRadius: 12,
                fontWeight: 800,
                fontFamily: "Space Grotesk, sans-serif",
              }}
            >
              #{myEntry.rank}
            </div>
            <div style={{ flex: 1 }}>
              <p style={{ fontWeight: 700 }}>Tim Kamu: {myEntry.team_name}</p>
              <p style={{ fontSize: "0.8125rem", color: "var(--text-secondary)" }}>
                @{myEntry.username}
              </p>
            </div>
            <div className="points-chip">
              <span className="points-chip-value">{myEntry.total_points}</span>
              <span className="points-chip-label">pts</span>
            </div>
          </div>
        )}

        {/* Search */}
        <div style={{ position: "relative", marginBottom: "1.5rem" }}>
          <Search
            size={16}
            style={{
              position: "absolute",
              left: "1rem",
              top: "50%",
              transform: "translateY(-50%)",
              color: "var(--text-muted)",
            }}
          />
          <input
            id="leaderboard-search"
            type="text"
            className="form-input"
            placeholder="Cari nama tim atau username..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ paddingLeft: "2.75rem" }}
          />
        </div>

        {/* Table */}
        <div className="card" style={{ padding: 0, overflow: "hidden" }}>
          {/* Table header */}
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "48px 1fr auto",
              gap: "1rem",
              padding: "0.875rem 1.25rem",
              borderBottom: "1px solid var(--bg-border)",
              fontSize: "0.75rem",
              color: "var(--text-muted)",
              textTransform: "uppercase",
              letterSpacing: "0.05em",
              fontWeight: 600,
            }}
          >
            <span>#</span>
            <span>Manajer</span>
            <span>Poin</span>
          </div>

          {loading ? (
            <div style={{ padding: "2rem", display: "flex", flexDirection: "column", gap: "0.75rem" }}>
              {Array.from({ length: 8 }).map((_, i) => (
                <div key={i} className="skeleton" style={{ height: 56, borderRadius: 12 }} />
              ))}
            </div>
          ) : filtered.length === 0 ? (
            <div style={{ textAlign: "center", padding: "4rem 1.5rem", color: "var(--text-muted)" }}>
              <Trophy size={40} style={{ opacity: 0.3, marginBottom: "1rem" }} />
              <p>{entries.length === 0 ? "Belum ada data leaderboard" : "Tidak ada hasil pencarian"}</p>
            </div>
          ) : (
            <div style={{ padding: "0.75rem" }}>
              {filtered.map((entry, idx) => {
                const isMe = entry.user_id === user?.id;
                const isTop3 = entry.rank <= 3;

                return (
                  <div
                    key={entry.user_id}
                    className={`leaderboard-row ${isTop3 ? `top-${entry.rank}` : ""}`}
                    style={{
                      display: "grid",
                      gridTemplateColumns: "48px 1fr auto",
                      gap: "1rem",
                      alignItems: "center",
                      background: isMe ? "rgba(0, 208, 132, 0.06)" : undefined,
                      border: isMe ? "1px solid rgba(0, 208, 132, 0.2)" : undefined,
                      animation: `fadeInUp 0.3s ease ${idx * 0.03}s both`,
                    }}
                  >
                    {/* Rank */}
                    <div style={{ display: "flex", justifyContent: "center" }}>
                      {isTop3 ? (
                        <span style={{ fontSize: "1.375rem" }}>{rankEmoji[entry.rank]}</span>
                      ) : (
                        <span
                          style={{
                            width: 32,
                            height: 32,
                            borderRadius: 8,
                            background: "var(--bg-elevated)",
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                            fontSize: "0.8125rem",
                            fontWeight: 700,
                            color: isMe ? "var(--fsl-green)" : "var(--text-secondary)",
                          }}
                        >
                          {entry.rank}
                        </span>
                      )}
                    </div>

                    {/* User info */}
                    <div style={{ minWidth: 0 }}>
                      <div
                        style={{
                          fontWeight: 600,
                          fontSize: "0.9375rem",
                          color: isTop3 ? rankColors[entry.rank] : "var(--text-primary)",
                          display: "flex",
                          alignItems: "center",
                          gap: "0.375rem",
                          overflow: "hidden",
                          textOverflow: "ellipsis",
                          whiteSpace: "nowrap",
                        }}
                      >
                        {entry.team_name}
                        {isMe && (
                          <span
                            style={{
                              fontSize: "0.6875rem",
                              background: "var(--fsl-green)",
                              color: "#000",
                              padding: "1px 6px",
                              borderRadius: 4,
                              fontWeight: 700,
                              flexShrink: 0,
                            }}
                          >
                            KAMU
                          </span>
                        )}
                      </div>
                      <div style={{ fontSize: "0.75rem", color: "var(--text-muted)" }}>
                        @{entry.username}
                      </div>
                    </div>

                    {/* Points */}
                    <div
                      style={{
                        fontFamily: "Space Grotesk, sans-serif",
                        fontWeight: 800,
                        fontSize: "1.125rem",
                        color: isTop3 ? rankColors[entry.rank] : "var(--text-primary)",
                        minWidth: 56,
                        textAlign: "right",
                      }}
                    >
                      {entry.total_points}
                      <span style={{ fontSize: "0.75rem", color: "var(--text-muted)", fontWeight: 400, marginLeft: 2 }}>
                        pts
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
