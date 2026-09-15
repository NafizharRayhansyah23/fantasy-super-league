import Link from "next/link";
import { ArrowLeft, Target, Crown, Wallet, ArrowLeftRight } from "lucide-react";

const APPEARANCE = [
  { action: "Tidak main (0 menit)", pts: "0", note: "Langsung 0, aksi lain diabaikan" },
  { action: "Main < 60 menit", pts: "+1", note: "Semua posisi" },
  { action: "Main ≥ 60 menit", pts: "+2", note: "Semua posisi" },
];

const ATTACK = [
  { action: "Gol (GK / DEF)", pts: "+6 / gol", note: "Kiper & Bek" },
  { action: "Gol (MID)", pts: "+5 / gol", note: "Gelandang" },
  { action: "Gol (FWD)", pts: "+4 / gol", note: "Penyerang" },
  { action: "Assist", pts: "+3 / assist", note: "Semua posisi" },
];

const DEFENSE = [
  { action: "Clean sheet (GK / DEF, ≥60 mnt)", pts: "+4", note: "Syarat main ≥60 menit" },
  { action: "Clean sheet (MID, ≥60 mnt)", pts: "+1", note: "FWD tidak dapat" },
  { action: "Penyelamatan (GK)", pts: "+1 / 3 saves", note: "Pembulatan ke bawah" },
  { action: "Kebobolan (GK / DEF)", pts: "−1 / 2 gol", note: "Pembulatan ke bawah" },
];

const DISCIPLINE = [
  { action: "Kartu kuning", pts: "−1", note: "Per kartu" },
  { action: "Kartu merah", pts: "−3", note: "Per kartu" },
  { action: "Bonus", pts: "+bonus", note: "Ditentukan admin (1–3)" },
  { action: "Batas bawah", pts: "min −4", note: "Poin 1 laga tidak < −4" },
];

function RuleTable({ rows }: { rows: { action: string; pts: string; note: string }[] }) {
  return (
    <table style={{ width: "100%", borderCollapse: "separate", borderSpacing: "0 6px" }}>
      <tbody>
        {rows.map((row) => (
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
              <div style={{ fontSize: "0.75rem", color: "var(--text-muted)", marginTop: 2 }}>
                {row.note}
              </div>
            </td>
            <td
              style={{
                padding: "0.625rem 0.75rem",
                background: "var(--bg-elevated)",
                borderRadius: "0 8px 8px 0",
                textAlign: "right",
                fontWeight: 700,
                whiteSpace: "nowrap",
                color: row.pts.startsWith("+")
                  ? "var(--fsl-green)"
                  : row.pts.startsWith("−") || row.pts.startsWith("-")
                    ? "var(--fsl-red)"
                    : "var(--text-primary)",
              }}
            >
              {row.pts}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export default function ScoringPage() {
  return (
    <div style={{ padding: "2rem 0 4rem" }}>
      <div className="container-app" style={{ maxWidth: 860 }}>
        <Link href="/" className="btn btn-ghost btn-sm" style={{ marginBottom: "1.5rem" }}>
          <ArrowLeft size={14} /> Kembali
        </Link>

        <h1 className="heading-lg" style={{ marginBottom: "0.5rem" }}>
          Sistem Poin & Aturan
        </h1>
        <p style={{ color: "var(--text-secondary)", marginBottom: "2rem" }}>
          Mengikuti logika scoring backend (gaya FPL, disesuaikan untuk BRI Super League).
        </p>

        <div className="card" style={{ marginBottom: "1.25rem" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", marginBottom: "1rem" }}>
            <Target size={20} color="var(--fsl-green)" />
            <h2 className="heading-md">Penampilan</h2>
          </div>
          <RuleTable rows={APPEARANCE} />
        </div>

        <div className="card" style={{ marginBottom: "1.25rem" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", marginBottom: "1rem" }}>
            <Target size={20} color="var(--fsl-gold)" />
            <h2 className="heading-md">Serangan</h2>
          </div>
          <RuleTable rows={ATTACK} />
        </div>

        <div className="card" style={{ marginBottom: "1.25rem" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", marginBottom: "1rem" }}>
            <Target size={20} color="var(--fsl-blue)" />
            <h2 className="heading-md">Bertahan</h2>
          </div>
          <RuleTable rows={DEFENSE} />
        </div>

        <div className="card" style={{ marginBottom: "1.25rem" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", marginBottom: "1rem" }}>
            <Target size={20} color="var(--fsl-red)" />
            <h2 className="heading-md">Disiplin & Bonus</h2>
          </div>
          <RuleTable rows={DISCIPLINE} />
        </div>

        <div
          className="card"
          style={{
            marginBottom: "1.25rem",
            borderColor: "rgba(245, 158, 11, 0.3)",
            background: "rgba(245, 158, 11, 0.05)",
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", marginBottom: "0.75rem" }}>
            <Crown size={20} color="var(--fsl-gold)" />
            <h2 className="heading-md">Kapten</h2>
          </div>
          <p style={{ color: "var(--text-secondary)", fontSize: "0.9rem", lineHeight: 1.6 }}>
            Poin kapten dikalikan <strong style={{ color: "var(--text-primary)" }}>2×</strong>.
            Hanya pemain starter yang menyumbang poin tim. Wajib pilih 1 kapten dan 1 wakil kapten.
          </p>
        </div>

        <div className="card" style={{ marginBottom: "1.25rem" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", marginBottom: "0.75rem" }}>
            <Wallet size={20} color="var(--fsl-green)" />
            <h2 className="heading-md">Skuad & Budget</h2>
          </div>
          <p style={{ color: "var(--text-secondary)", fontSize: "0.9rem", lineHeight: 1.7 }}>
            15 pemain (2 GK, 5 DEF, 5 MID, 3 FWD) dengan total maksimal{" "}
            <strong style={{ color: "var(--text-primary)" }}>Rp 100.0jt</strong>. 11 starter,
            4 cadangan.
          </p>
        </div>

        <div className="card">
          <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", marginBottom: "0.75rem" }}>
            <ArrowLeftRight size={20} color="var(--fsl-purple)" />
            <h2 className="heading-md">Transfer</h2>
          </div>
          <p style={{ color: "var(--text-secondary)", fontSize: "0.9rem", lineHeight: 1.7 }}>
            1 transfer gratis per gameweek. Kelebihan transfer memotong{" "}
            <strong style={{ color: "var(--fsl-red)" }}>−4 poin</strong> kecuali memakai Wildcard
            (1× semusim, tanpa minus poin).
          </p>
        </div>
      </div>
    </div>
  );
}
