"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import {
  teamApi,
  playersApi,
  FantasyTeam,
  Player,
  PlayerFilter,
  SaveTeamRequest,
  POSITION_SLOTS,
  formatPrice,
} from "@/lib/api";
import toast from "react-hot-toast";
import PlayerAvatar from "@/components/PlayerAvatar";
import {
  Save,
  RefreshCw,
  Search,
  X,
  AlertCircle,
  Check,
  ArrowRight,
} from "lucide-react";

const getErrorMessage = (err: unknown, fallback: string) =>
  err instanceof Error ? err.message : fallback;

// --- Types ---
interface SquadSlot {
  index: number;
  position: "GK" | "DEF" | "MID" | "FWD";
  player: Player | null;
  isCaptain: boolean;
  isViceCaptain: boolean;
  isStarting: boolean;
}

// Initialize empty slots (pure, di luar komponen agar stabil)
const getEmptySlots = (): SquadSlot[] => {
  const s: SquadSlot[] = [];
  POSITION_SLOTS.GK.forEach((i) =>
    s.push({ index: i, position: "GK", player: null, isCaptain: false, isViceCaptain: false, isStarting: i === 0 })
  );
  POSITION_SLOTS.DEF.forEach((i) =>
    s.push({ index: i, position: "DEF", player: null, isCaptain: false, isViceCaptain: false, isStarting: i < 5 })
  );
  POSITION_SLOTS.MID.forEach((i) =>
    s.push({ index: i, position: "MID", player: null, isCaptain: false, isViceCaptain: false, isStarting: i < 10 })
  );
  POSITION_SLOTS.FWD.forEach((i) =>
    s.push({ index: i, position: "FWD", player: null, isCaptain: false, isViceCaptain: false, isStarting: i < 13 })
  );
  return s.sort((a, b) => a.index - b.index);
};

export default function SquadPage() {
  const router = useRouter();
  const { isLoggedIn, isLoading } = useAuth();

  // Team state
  const [team, setTeam] = useState<FantasyTeam | null>(null);
  const [hasTeam, setHasTeam] = useState(false);
  const [slots, setSlots] = useState<SquadSlot[]>([]);
  const [initialSlots, setInitialSlots] = useState<SquadSlot[]>([]);
  const [budget, setBudget] = useState(100.0);

  // Player selection state
  const [players, setPlayers] = useState<Player[]>([]);
  const [selectingForSlot, setSelectingForSlot] = useState<number | null>(null);
  const [search, setSearch] = useState("");
  const [loadingPlayers, setLoadingPlayers] = useState(false);

  // UI state
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [showConfirmTransfer, setShowConfirmTransfer] = useState(false);
  const [transferPlan, setTransferPlan] = useState<{
    slotIdx: number;
    playerOut: Player;
    playerIn: Player;
    useWildcard: boolean;
  } | null>(null);

  const loadTeam = useCallback(async () => {
    setLoading(true);
    try {
      const res = await teamApi.getMyTeam();
      setHasTeam(res.has_team);
      if (res.has_team && res.data) {
        setTeam(res.data);
        setBudget(res.data.remaining_budget);

        // Map saved players to slots
        const newSlots = getEmptySlots();
        res.data.players.forEach((p) => {
          if (p.player) {
            newSlots[p.position_slot] = {
              index: p.position_slot,
              position: p.player.position,
              player: p.player,
              isCaptain: p.is_captain,
              isViceCaptain: p.is_vice_captain,
              isStarting: p.is_starting,
            };
          }
        });
        setSlots(newSlots);
        setInitialSlots(JSON.parse(JSON.stringify(newSlots))); // deep copy
      } else {
        setSlots(getEmptySlots());
        setBudget(100.0);
      }
    } catch {
      setSlots(getEmptySlots());
      setBudget(100.0);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!isLoading && !isLoggedIn) {
      router.push("/auth/login");
      return;
    }

    if (isLoggedIn) {
      // Fetch awal saat login — bukan sync state biasa
      // eslint-disable-next-line react-hooks/set-state-in-effect
      loadTeam();
    }
  }, [isLoggedIn, isLoading, router, loadTeam]);

  // Fetch players for modal
  useEffect(() => {
    if (selectingForSlot !== null) {
      const targetSlot = slots[selectingForSlot];
      if (!targetSlot) return;
      const targetPos = targetSlot.position;
      const fetchP = async () => {
        setLoadingPlayers(true);
        try {
          const filter: PlayerFilter = { position: targetPos, limit: 50 };
          if (search) filter.search = search;
          const res = await playersApi.getAll(filter);
          setPlayers(res.data || []);
        } catch {
          setPlayers([]);
        } finally {
          setLoadingPlayers(false);
        }
      };
      const timeout = setTimeout(fetchP, 300);
      return () => clearTimeout(timeout);
    }
  }, [selectingForSlot, search, slots]);

  const openPlayerSelect = (slotIdx: number) => {
    setSelectingForSlot(slotIdx);
    setSearch("");
  };

  const handleSelectPlayer = (player: Player) => {
    if (selectingForSlot === null) return;

    // Check if player already in team
    if (slots.some((s) => s.player?.id === player.id)) {
      toast.error("Pemain ini sudah ada di tim kamu!");
      return;
    }

    const currentSlot = slots[selectingForSlot];
    if (!currentSlot) return;
    const playerOut = currentSlot.player;

    // If making a transfer on an existing team
    if (hasTeam && playerOut && initialSlots[selectingForSlot]?.player?.id === playerOut.id) {
      setTransferPlan({
        slotIdx: selectingForSlot,
        playerOut,
        playerIn: player,
        useWildcard: false,
      });
      setShowConfirmTransfer(true);
      return;
    }

    // Normal selection (new team or empty slot)
    const newSlots = [...slots];
    newSlots[selectingForSlot] = {
      ...newSlots[selectingForSlot],
      player,
    };
    
    // Recalculate budget
    const priceDiff = (playerOut?.price || 0) - player.price;
    setBudget((b) => b + priceDiff);
    
    setSlots(newSlots);
    setSelectingForSlot(null);
  };

  const handleRemovePlayer = (slotIdx: number, e: React.MouseEvent) => {
    e.stopPropagation();
    if (hasTeam) {
      toast.error("Gunakan fitur transfer untuk mengganti pemain pada tim yang sudah disimpan.");
      return;
    }

    const playerOut = slots[slotIdx]?.player;
    if (playerOut) {
      const newSlots = [...slots];
      newSlots[slotIdx] = { ...newSlots[slotIdx], player: null, isCaptain: false, isViceCaptain: false };
      setSlots(newSlots);
      setBudget((b) => b + playerOut.price);
    }
  };

  const handleSetRole = (slotIdx: number, role: 'captain' | 'vice', e: React.MouseEvent) => {
    e.stopPropagation();
    if (!slots[slotIdx].player) return;

    const newSlots = [...slots];
    if (role === 'captain') {
      newSlots.forEach(s => s.isCaptain = false);
      newSlots[slotIdx].isCaptain = true;
      if (newSlots[slotIdx].isViceCaptain) newSlots[slotIdx].isViceCaptain = false;
    } else {
      newSlots.forEach(s => s.isViceCaptain = false);
      newSlots[slotIdx].isViceCaptain = true;
      if (newSlots[slotIdx].isCaptain) newSlots[slotIdx].isCaptain = false;
    }
    setSlots(newSlots);
  };

  const handleConfirmTransfer = async () => {
    if (!transferPlan) return;
    
    setSaving(true);
    try {
      const res = await teamApi.makeTransfer({
        player_out_id: transferPlan.playerOut.id,
        player_in_id: transferPlan.playerIn.id,
        use_wildcard: transferPlan.useWildcard,
      });
      
      toast.success(res.message);
      setSelectingForSlot(null);
      setShowConfirmTransfer(false);
      setTransferPlan(null);
      await loadTeam(); // Reload to get updated budget and transfers
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, "Transfer gagal"));
    } finally {
      setSaving(false);
    }
  };

  const handleSaveTeam = async () => {
    // Validate
    const missing = slots.filter(s => !s.player).length;
    if (missing > 0) {
      toast.error(`Masih ada ${missing} posisi kosong di tim kamu!`);
      return;
    }
    if (budget < 0) {
      toast.error("Budget melebihi batas (Rp 100 juta)!");
      return;
    }
    if (!slots.some(s => s.isCaptain)) {
      toast.error("Pilih satu kapten!");
      return;
    }
    if (!slots.some(s => s.isViceCaptain)) {
      toast.error("Pilih satu wakil kapten!");
      return;
    }

    setSaving(true);
    try {
      const req: SaveTeamRequest = {
        team_name: team?.team_name || "Tim Saya",
        players: slots.map(s => ({
          player_id: s.player!.id,
          is_captain: s.isCaptain,
          is_vice_captain: s.isViceCaptain,
          is_starting: s.isStarting,
          position_slot: s.index,
        }))
      };
      
      const res = await teamApi.saveTeam(req);
      toast.success(res.message);
      await loadTeam();
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, "Gagal menyimpan tim"));
    } finally {
      setSaving(false);
    }
  };

  // --- Render Helpers ---

  const renderPitchRow = (slotIndices: number[], _title: string, top: string) => (
    <div className="pitch-section" style={{ top, height: "25%", zIndex: 10 }}>
      {slotIndices.map((idx) => {
        const slot = slots[idx];
        if (!slot) return null;
        const p = slot.player;
        
        return (
          <div key={idx} className="pitch-player" onClick={() => openPlayerSelect(idx)}>
            <div className={`pitch-player-avatar ${!p ? 'empty' : ''}`}>
              {!p ? (
                "+"
              ) : (
                <>
                  <PlayerAvatar key={p.id} src={p.photo_url} alt={p.name} />
                  {slot.isCaptain && <div className="captain-badge">C</div>}
                  {slot.isViceCaptain && <div className="captain-badge" style={{ background: "var(--bg-elevated)", color: "#fff", border: "1px solid var(--fsl-gold)" }}>V</div>}
                  {!hasTeam && (
                    <button 
                      onClick={(e) => handleRemovePlayer(idx, e)}
                      style={{
                        position: "absolute", top: -4, right: -4, background: "var(--fsl-red)",
                        color: "white", borderRadius: "50%", width: 16, height: 16,
                        display: "flex", alignItems: "center", justifyContent: "center",
                        border: "none", cursor: "pointer", zIndex: 3
                      }}
                    >
                      <X size={10} />
                    </button>
                  )}
                </>
              )}
            </div>
            {p ? (
              <>
                <div className="pitch-player-name">{p.name}</div>
                <div className="pitch-player-price">{formatPrice(p.price)}</div>
                
                {/* Role selectors (only show when editing roles, simplified here for MVP) */}
                <div style={{ display: "flex", gap: 2, marginTop: 2 }} onClick={e => e.stopPropagation()}>
                   <button 
                    onClick={(e) => handleSetRole(idx, 'captain', e)}
                    style={{ background: slot.isCaptain ? "var(--fsl-gold)" : "rgba(255,255,255,0.2)", color: slot.isCaptain ? "#000" : "#fff", border: "none", borderRadius: 2, fontSize: 9, padding: "1px 4px", cursor: "pointer" }}
                   >C</button>
                   <button 
                    onClick={(e) => handleSetRole(idx, 'vice', e)}
                    style={{ background: slot.isViceCaptain ? "var(--fsl-gold)" : "rgba(255,255,255,0.2)", color: slot.isViceCaptain ? "#000" : "#fff", border: "none", borderRadius: 2, fontSize: 9, padding: "1px 4px", cursor: "pointer" }}
                   >V</button>
                </div>
              </>
            ) : (
              <div className="pitch-player-name" style={{ background: "rgba(255,255,255,0.15)", color: "rgba(255,255,255,0.5)" }}>
                {slot.position}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );

  if (isLoading || loading) {
    return <div style={{ display: "flex", justifyContent: "center", padding: "4rem" }}><div className="spinner" /></div>;
  }

  const selectedCount = slots.filter(s => s.player).length;

  return (
    <div style={{ padding: "2rem 0 4rem" }}>
      <div className="container-app">
        
        {/* Header & Status */}
        <div style={{ display: "flex", flexWrap: "wrap", justifyContent: "space-between", alignItems: "flex-end", gap: "1rem", marginBottom: "2rem" }}>
          <div>
            <h1 className="heading-lg" style={{ marginBottom: "0.25rem" }}>
              {team?.team_name || "Tim Saya"}
            </h1>
            <p style={{ color: "var(--text-secondary)" }}>
              Pilih 15 pemain (2 GK, 5 DEF, 5 MID, 3 FWD)
            </p>
          </div>
          
          <div style={{ display: "flex", gap: "1rem" }}>
            <div className="points-chip">
              <span className="points-chip-value" style={{ fontSize: "1.25rem" }}>{selectedCount}/15</span>
              <span className="points-chip-label">Pemain</span>
            </div>
            <div className="points-chip">
              <span className="points-chip-value" style={{ fontSize: "1.25rem", color: budget < 0 ? "var(--fsl-red)" : "var(--fsl-green)" }}>
                {budget.toFixed(1)}
              </span>
              <span className="points-chip-label">Sisa Budget</span>
            </div>
            {hasTeam && (
               <div className="points-chip" style={{ background: "rgba(59, 130, 246, 0.1)", borderColor: "rgba(59, 130, 246, 0.3)" }}>
               <span className="points-chip-value" style={{ fontSize: "1.25rem", color: "var(--fsl-blue)" }}>
                 {team?.free_transfers}
               </span>
               <span className="points-chip-label" style={{ color: "var(--fsl-blue)" }}>Transfer</span>
             </div>
            )}
          </div>
        </div>

        {/* Budget Bar */}
        <div style={{ marginBottom: "2rem" }}>
          <div className="budget-bar-container">
            <div 
              className={`budget-bar-fill ${budget < 0 ? 'warning' : ''}`}
              style={{ width: `${Math.min(100, Math.max(0, ((100 - budget) / 100) * 100))}%` }}
            />
          </div>
          <div style={{ display: "flex", justifyContent: "space-between", fontSize: "0.75rem", color: "var(--text-muted)", marginTop: "0.5rem" }}>
            <span>Rp 0</span>
            <span>Budget Terpakai: Rp {(100 - budget).toFixed(1)}jt</span>
            <span>Rp 100jt</span>
          </div>
        </div>

        {/* Action Bar */}
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "1rem", background: "var(--bg-card)", padding: "1rem", borderRadius: 12, border: "1px solid var(--bg-border)" }}>
           <div style={{ fontSize: "0.875rem", color: "var(--text-secondary)" }}>
              {hasTeam 
                ? "Ganti pemain dengan klik pada pemain di lapangan. Poin kapten dikalikan 2." 
                : "Lengkapi tim kamu lalu klik Simpan Tim."}
           </div>
           
           {!hasTeam && (
              <button 
                className="btn btn-primary" 
                onClick={handleSaveTeam}
                disabled={saving || selectedCount < 15 || budget < 0}
              >
                {saving ? <div className="spinner" style={{ width: 16, height: 16 }} /> : <Save size={16} />}
                Simpan Tim
              </button>
           )}
        </div>

        {/* PITCH VISUALIZATION */}
        <div className="pitch-container" style={{ margin: "2rem auto" }}>
           <div className="pitch">
              {renderPitchRow([0], "GK", "0%")}
              {renderPitchRow([2, 3, 4, 5, 6], "DEF", "25%")}
              {renderPitchRow([7, 8, 9, 10, 11], "MID", "50%")}
              {renderPitchRow([12, 13, 14], "FWD", "75%")}
           </div>

           {/* Bench / Reserves */}
           <div style={{ marginTop: "1rem", padding: "1rem", background: "rgba(0,0,0,0.3)", borderRadius: 12, position: "relative", display: "flex", justifyContent: "center", gap: "1rem" }}>
              <div style={{ fontSize: "0.75rem", color: "var(--text-muted)", position: "absolute", left: "1rem", top: "1rem" }}>Cadangan</div>
              <div style={{ display: "flex", gap: "1rem" }}>
                 {[1].map((idx) => {
                    const slot = slots[idx];
                    const p = slot?.player;
                    if (!slot) return null;
                    return (
                       <div key={idx} className="pitch-player" onClick={() => openPlayerSelect(idx)} style={{ position: "relative" }}>
                           <div className={`pitch-player-avatar ${!p ? "empty" : ""}`}>
                              {!p ? "+" : (
                                 <PlayerAvatar key={p.id} src={p.photo_url} alt={p.name} />
                              )}
                           </div>
                          <div className="pitch-player-name" style={!p ? { background: "rgba(255,255,255,0.15)", color: "rgba(255,255,255,0.5)" } : undefined}>
                             {p ? p.name : slot.position}
                          </div>
                          {p && <div className="pitch-player-price">{formatPrice(p.price)}</div>}
                       </div>
                    );
                 })}
              </div>
           </div>
        </div>

      </div>

      {/* PLAYER SELECTION MODAL */}
      {selectingForSlot !== null && (
        <div className="overlay" onClick={() => setSelectingForSlot(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()} style={{ height: "80vh", display: "flex", flexDirection: "column" }}>
            <div className="modal-header" style={{ marginBottom: "1rem" }}>
              <h2 className="heading-sm">
                Pilih {slots[selectingForSlot].position}
              </h2>
              <button className="btn btn-ghost btn-sm" onClick={() => setSelectingForSlot(null)}>
                <X size={16} />
              </button>
            </div>

            <div style={{ position: "relative", marginBottom: "1rem" }}>
              <Search size={16} style={{ position: "absolute", left: "1rem", top: "50%", transform: "translateY(-50%)", color: "var(--text-muted)" }} />
              <input
                type="text"
                className="form-input"
                placeholder="Cari pemain..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                style={{ paddingLeft: "2.5rem" }}
              />
            </div>

            <div style={{ flex: 1, overflowY: "auto", paddingRight: "0.5rem" }}>
              {loadingPlayers ? (
                <div style={{ display: "flex", justifyContent: "center", padding: "2rem" }}><div className="spinner" /></div>
              ) : players.length === 0 ? (
                <div style={{ textAlign: "center", padding: "2rem", color: "var(--text-muted)" }}>Tidak ada pemain.</div>
              ) : (
                <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
                  {players.map(p => {
                    const isSelected = slots.some(s => s.player?.id === p.id);
                    return (
                      <div 
                        key={p.id} 
                        className="card card-hover"
                        style={{ 
                          padding: "0.75rem", display: "flex", alignItems: "center", gap: "1rem", 
                          cursor: isSelected ? "not-allowed" : "pointer",
                          opacity: isSelected ? 0.5 : 1,
                          borderColor: isSelected ? "var(--fsl-green)" : ""
                        }}
                        onClick={() => !isSelected && handleSelectPlayer(p)}
                      >
                        <div style={{ width: 40, height: 40, borderRadius: "50%", background: "var(--bg-elevated)", overflow: "hidden", display: "flex", alignItems: "center", justifyContent: "center", flexShrink: 0 }}>
                           <PlayerAvatar key={p.id} src={p.photo_url} alt={p.name} />
                        </div>
                        <div style={{ flex: 1 }}>
                          <div style={{ fontWeight: 600, fontSize: "0.9rem" }}>{p.name}</div>
                          <div style={{ fontSize: "0.75rem", color: "var(--text-secondary)" }}>{p.club?.name}</div>
                        </div>
                        <div style={{ textAlign: "right" }}>
                          <div style={{ fontWeight: 700, color: "var(--fsl-green)", fontSize: "0.9rem" }}>{formatPrice(p.price)}</div>
                          <div style={{ fontSize: "0.7rem", color: "var(--text-muted)" }}>{p.total_points} pts</div>
                        </div>
                        {isSelected && <Check color="var(--fsl-green)" size={16} />}
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* CONFIRM TRANSFER MODAL */}
      {showConfirmTransfer && transferPlan && (
        <div className="overlay" onClick={() => setShowConfirmTransfer(false)}>
           <div className="modal" onClick={e => e.stopPropagation()}>
              <div className="modal-header">
                 <h2 className="heading-sm" style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                    <RefreshCw size={20} color="var(--fsl-blue)" /> Konfirmasi Transfer
                 </h2>
              </div>

              <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "1.5rem", background: "var(--bg-elevated)", padding: "1.5rem", borderRadius: 12 }}>
                 <div style={{ textAlign: "center" }}>
                    <div style={{ color: "var(--fsl-red)", fontSize: "0.75rem", fontWeight: 700, marginBottom: "0.5rem" }}>KELUAR</div>
                    <div style={{ fontWeight: 600 }}>{transferPlan.playerOut.name}</div>
                    <div style={{ fontSize: "0.875rem", color: "var(--text-secondary)" }}>{formatPrice(transferPlan.playerOut.price)}</div>
                 </div>
                 
                 <ArrowRight color="var(--text-muted)" />
                 
                 <div style={{ textAlign: "center" }}>
                    <div style={{ color: "var(--fsl-green)", fontSize: "0.75rem", fontWeight: 700, marginBottom: "0.5rem" }}>MASUK</div>
                    <div style={{ fontWeight: 600 }}>{transferPlan.playerIn.name}</div>
                    <div style={{ fontSize: "0.875rem", color: "var(--text-secondary)" }}>{formatPrice(transferPlan.playerIn.price)}</div>
                 </div>
              </div>

              <div style={{ marginBottom: "2rem" }}>
                 <div style={{ display: "flex", justifyContent: "space-between", marginBottom: "0.5rem", fontSize: "0.875rem" }}>
                    <span style={{ color: "var(--text-secondary)" }}>Budget Saat Ini:</span>
                    <span>{formatPrice(budget)}</span>
                 </div>
                 <div style={{ display: "flex", justifyContent: "space-between", marginBottom: "1rem", fontSize: "0.875rem" }}>
                    <span style={{ color: "var(--text-secondary)" }}>Sisa Budget Setelah Transfer:</span>
                    <span style={{ color: (budget + transferPlan.playerOut.price - transferPlan.playerIn.price) < 0 ? "var(--fsl-red)" : "var(--text-primary)", fontWeight: 700 }}>
                       {formatPrice(budget + transferPlan.playerOut.price - transferPlan.playerIn.price)}
                    </span>
                 </div>

                 <div style={{ padding: "1rem", background: "rgba(59, 130, 246, 0.1)", border: "1px solid rgba(59, 130, 246, 0.2)", borderRadius: 8, display: "flex", gap: "0.75rem" }}>
                    <AlertCircle size={18} color="var(--fsl-blue)" style={{ flexShrink: 0, marginTop: 2 }} />
                    <div style={{ fontSize: "0.8125rem", color: "var(--text-secondary)", lineHeight: 1.5 }}>
                       {team?.free_transfers && team.free_transfers > 0 
                         ? "Kamu memiliki transfer gratis. Transfer ini tidak akan memotong poin." 
                         : "Transfer gratis sudah habis. Transfer ini akan memotong 4 poin dari total poin kamu di gameweek selanjutnya."}
                    </div>
                 </div>
                 
                 {/* Wildcard option (if available) */}
                 {(!team?.wildcard_used && team?.free_transfers === 0) && (
                    <div style={{ marginTop: "1rem", padding: "1rem", border: "1px solid var(--fsl-gold)", borderRadius: 8, display: "flex", alignItems: "center", gap: "0.75rem" }}>
                       <input 
                         type="checkbox" 
                         id="use-wildcard"
                         checked={transferPlan.useWildcard}
                         onChange={(e) => setTransferPlan({...transferPlan, useWildcard: e.target.checked})}
                         style={{ width: 16, height: 16, accentColor: "var(--fsl-gold)" }}
                       />
                       <label htmlFor="use-wildcard" style={{ fontSize: "0.875rem", cursor: "pointer" }}>
                          <span style={{ color: "var(--fsl-gold)", fontWeight: 700 }}>Gunakan Wildcard</span> (Reset transfer tanpa minus poin)
                       </label>
                    </div>
                 )}
              </div>

              <div style={{ display: "flex", gap: "1rem" }}>
                 <button className="btn btn-secondary" style={{ flex: 1 }} onClick={() => setShowConfirmTransfer(false)}>
                    Batal
                 </button>
                 <button 
                   className="btn btn-primary" 
                   style={{ flex: 1 }} 
                   onClick={handleConfirmTransfer}
                   disabled={saving || (budget + transferPlan.playerOut.price - transferPlan.playerIn.price) < 0}
                 >
                    {saving ? <div className="spinner" style={{ width: 16, height: 16 }} /> : "Konfirmasi Transfer"}
                 </button>
              </div>
           </div>
        </div>
      )}

    </div>
  );
}
