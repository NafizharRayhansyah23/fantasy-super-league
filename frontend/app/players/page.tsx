"use client";

import { useEffect, useState, useCallback } from "react";
import { playersApi, clubsApi, Player, Club, PlayerFilter, formatPrice, POSITION_LABELS } from "@/lib/api";
import { Search, Filter, ChevronLeft, ChevronRight, Star } from "lucide-react";
import PlayerAvatar from "@/components/PlayerAvatar";

const POSITIONS = ["GK", "DEF", "MID", "FWD"];

function PlayerCard({ player, onSelect }: { player: Player; onSelect?: (p: Player) => void }) {
  const posClass = `badge-${player.position.toLowerCase()}`;

  return (
    <div
      className="player-card animate-fadeInUp"
      onClick={() => onSelect?.(player)}
      style={{ cursor: onSelect ? "pointer" : "default" }}
    >
      {/* Photo */}
      <div style={{ position: "relative" }}>
        <PlayerAvatar key={player.id} src={player.photo_url} alt={player.name} className="player-card-photo" />
        <div style={{ position: "absolute", top: 8, right: 8 }}>
          <span className={`badge ${posClass}`}>{player.position}</span>
        </div>
        {player.is_national_team && (
          <div style={{ position: "absolute", top: 8, left: 8 }}>
            <span style={{ fontSize: "1rem" }} title="Timnas Indonesia">🇮🇩</span>
          </div>
        )}
      </div>

      {/* Body */}
      <div className="player-card-body">
        <div className="player-card-name">{player.name}</div>
        <div className="player-card-club">
          {player.jersey_number ? `#${player.jersey_number} · ` : ""}
          {player.club?.short_name || player.club?.name || "—"}
        </div>
      </div>

      {/* Footer */}
      <div className="player-card-footer">
        <span className="player-card-price">{formatPrice(player.price)}</span>
        <span className="player-card-pts">
          <Star size={10} style={{ display: "inline", marginRight: 2, verticalAlign: "middle" }} />
          {player.total_points} pts
        </span>
      </div>
    </div>
  );
}

export default function PlayersPage() {
  const [players, setPlayers] = useState<Player[]>([]);
  const [clubs, setClubs] = useState<Club[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<PlayerFilter>({ page: 1, limit: 20 });
  const [total, setTotal] = useState(0);
  const [pages, setPages] = useState(1);
  const [search, setSearch] = useState("");
  const [selectedPlayer, setSelectedPlayer] = useState<Player | null>(null);

  const fetchPlayers = useCallback(async (f: PlayerFilter) => {
    setLoading(true);
    try {
      const res = await playersApi.getAll(f);
      setPlayers(res.data || []);
      setTotal(res.total || 0);
      setPages(res.pages || 1);
    } catch {
      setPlayers([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    clubsApi.getAll().then((res) => setClubs(res.data || [])).catch(() => {});
  }, []);

  useEffect(() => {
    const timeout = setTimeout(() => {
      fetchPlayers({ ...filter, search, page: 1 });
    }, 350);
    return () => clearTimeout(timeout);
  }, [search, filter, fetchPlayers]);

  const setPosition = (pos: string) => {
    setFilter((f) => ({ ...f, position: f.position === pos ? undefined : pos, page: 1 }));
  };

  const setClub = (id: string) => {
    setFilter((f) => ({ ...f, club_id: f.club_id === id ? undefined : id, page: 1 }));
  };

  const goToPage = (p: number) => {
    const newFilter = { ...filter, search, page: p };
    setFilter(newFilter);
    fetchPlayers(newFilter);
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  return (
    <div style={{ padding: "2rem 0 4rem" }}>
      <div className="container-app">
        {/* Header */}
        <div style={{ marginBottom: "2rem" }}>
          <h1 className="heading-lg" style={{ marginBottom: "0.5rem" }}>
            Daftar Pemain
          </h1>
          <p style={{ color: "var(--text-secondary)" }}>
            {total > 0 ? `${total} pemain dari 16 klub BRI Super League` : "Memuat data pemain..."}
          </p>
        </div>

        {/* Search & Filters */}
        <div style={{ display: "flex", flexDirection: "column", gap: "1rem", marginBottom: "2rem" }}>
          {/* Search bar */}
          <div style={{ position: "relative" }}>
            <Search
              size={18}
              style={{
                position: "absolute",
                left: "1rem",
                top: "50%",
                transform: "translateY(-50%)",
                color: "var(--text-muted)",
              }}
            />
            <input
              id="player-search"
              type="text"
              className="form-input"
              placeholder="Cari nama pemain..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              style={{ paddingLeft: "3rem" }}
            />
          </div>

          {/* Position filter */}
          <div className="filter-bar">
            <div style={{ display: "flex", alignItems: "center", gap: "0.5rem", color: "var(--text-muted)", fontSize: "0.8125rem" }}>
              <Filter size={14} /> Posisi:
            </div>
            {POSITIONS.map((pos) => (
              <button
                key={pos}
                className={`filter-chip ${filter.position === pos ? "active" : ""}`}
                onClick={() => setPosition(pos)}
                id={`filter-pos-${pos.toLowerCase()}`}
              >
                {POSITION_LABELS[pos]} ({pos})
              </button>
            ))}
          </div>

          {/* Club filter */}
          {clubs.length > 0 && (
            <div className="filter-bar" style={{ flexWrap: "wrap" }}>
              <div style={{ display: "flex", alignItems: "center", gap: "0.5rem", color: "var(--text-muted)", fontSize: "0.8125rem" }}>
                <Filter size={14} /> Klub:
              </div>
              {clubs.map((club) => (
                <button
                  key={club.id}
                  className={`filter-chip ${filter.club_id === club.id ? "active" : ""}`}
                  onClick={() => setClub(club.id)}
                >
                  {club.short_name || club.name}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Player grid */}
        {loading ? (
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fill, minmax(160px, 1fr))",
              gap: "1rem",
            }}
          >
            {Array.from({ length: 12 }).map((_, i) => (
              <div key={i}>
                <div className="skeleton" style={{ aspectRatio: "3/4", borderRadius: 14 }} />
                <div className="skeleton" style={{ height: 16, marginTop: 8, borderRadius: 6 }} />
                <div className="skeleton" style={{ height: 12, marginTop: 6, width: "60%", borderRadius: 6 }} />
              </div>
            ))}
          </div>
        ) : players.length === 0 ? (
          <div style={{ textAlign: "center", padding: "4rem 0", color: "var(--text-muted)" }}>
            <div style={{ fontSize: "3rem", marginBottom: "1rem" }}>🔍</div>
            <p style={{ fontSize: "1.125rem" }}>Tidak ada pemain yang ditemukan</p>
            <p style={{ marginTop: "0.5rem", fontSize: "0.9rem" }}>
              Coba ubah filter atau kata kunci pencarian
            </p>
          </div>
        ) : (
          <div
            className="stagger"
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fill, minmax(160px, 1fr))",
              gap: "1rem",
            }}
          >
            {players.map((player) => (
              <PlayerCard
                key={player.id}
                player={player}
                onSelect={setSelectedPlayer}
              />
            ))}
          </div>
        )}

        {/* Pagination */}
        {pages > 1 && (
          <div
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              gap: "0.5rem",
              marginTop: "2.5rem",
            }}
          >
            <button
              className="btn btn-ghost btn-sm"
              onClick={() => goToPage((filter.page || 1) - 1)}
              disabled={(filter.page || 1) <= 1}
            >
              <ChevronLeft size={16} />
            </button>

            {Array.from({ length: Math.min(pages, 7) }, (_, i) => {
              const page = i + 1;
              return (
                <button
                  key={page}
                  className={`btn btn-sm ${filter.page === page ? "btn-primary" : "btn-ghost"}`}
                  onClick={() => goToPage(page)}
                >
                  {page}
                </button>
              );
            })}

            <button
              className="btn btn-ghost btn-sm"
              onClick={() => goToPage((filter.page || 1) + 1)}
              disabled={(filter.page || 1) >= pages}
            >
              <ChevronRight size={16} />
            </button>
          </div>
        )}
      </div>

      {/* Player detail modal */}
      {selectedPlayer && (
        <div className="overlay" onClick={() => setSelectedPlayer(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2 className="heading-sm">{selectedPlayer.name}</h2>
              <button
                className="btn btn-ghost btn-sm"
                onClick={() => setSelectedPlayer(null)}
                aria-label="Tutup"
              >
                ✕
              </button>
            </div>

            <div style={{ display: "flex", gap: "1.5rem", marginBottom: "1.5rem" }}>
              <div
                style={{
                  width: 100,
                  height: 120,
                  borderRadius: 12,
                  background: "var(--bg-elevated)",
                  overflow: "hidden",
                  flexShrink: 0,
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  fontSize: "3rem",
                }}
              >
                <PlayerAvatar
                  key={selectedPlayer.id}
                  src={selectedPlayer.photo_url}
                  alt={selectedPlayer.name}
                />
              </div>

              <div style={{ flex: 1 }}>
                <div style={{ display: "flex", gap: "0.5rem", flexWrap: "wrap", marginBottom: "0.75rem" }}>
                  <span className={`badge badge-${selectedPlayer.position.toLowerCase()}`}>
                    {POSITION_LABELS[selectedPlayer.position]}
                  </span>
                  {selectedPlayer.is_national_team && (
                    <span className="badge" style={{ background: "rgba(220,38,38,0.2)", color: "#F87171" }}>
                      🇮🇩 Timnas
                    </span>
                  )}
                </div>
                <p style={{ fontSize: "0.9rem", color: "var(--text-secondary)", marginBottom: "0.375rem" }}>
                  {selectedPlayer.club?.name || "Klub tidak diketahui"}
                </p>
                {selectedPlayer.jersey_number > 0 && (
                  <p style={{ fontSize: "0.875rem", color: "var(--text-muted)" }}>
                    Nomor punggung: #{selectedPlayer.jersey_number}
                  </p>
                )}
                <div style={{ display: "flex", gap: "1rem", marginTop: "1rem" }}>
                  <div className="points-chip">
                    <span className="points-chip-value">{selectedPlayer.total_points}</span>
                    <span className="points-chip-label">Total Pts</span>
                  </div>
                  <div className="points-chip">
                    <span className="points-chip-value" style={{ fontSize: "1.25rem" }}>
                      {formatPrice(selectedPlayer.price)}
                    </span>
                    <span className="points-chip-label">Harga</span>
                  </div>
                </div>
              </div>
            </div>

            <button
              className="btn btn-secondary"
              style={{ width: "100%", justifyContent: "center" }}
              onClick={() => setSelectedPlayer(null)}
            >
              Tutup
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
