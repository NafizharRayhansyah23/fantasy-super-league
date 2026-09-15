"use client";

import type { CSSProperties } from "react";
import { useState } from "react";

// Siluet abu-abu ala foto profil kosong. Memakai currentColor agar
// mengikuti warna teks container (var(--text-muted) di tema kita).
export function AvatarSilhouette({ label }: { label?: string }) {
  return (
    <svg
      viewBox="0 0 64 64"
      role="img"
      aria-label={label ?? "Foto pemain tidak tersedia"}
      style={{ width: "72%", height: "72%", display: "block" }}
    >
      <circle cx="32" cy="22" r="12" fill="currentColor" />
      <path d="M6 58c3-14 13-21 26-21s23 7 26 21" fill="currentColor" />
    </svg>
  );
}

interface PlayerAvatarProps {
  src?: string | null;
  alt: string;
  className?: string;
  style?: CSSProperties;
}

// Avatar pemain: tampilkan foto bila ada & berhasil dimuat,
// jika tidak tampilkan siluet abu-abu. Beri key={playerId} di pemakaian bila
// src bisa berganti (mis. modal/pitch) agar state error ikut reset.
export default function PlayerAvatar({ src, alt, className = "", style }: PlayerAvatarProps) {
  const [failed, setFailed] = useState(false);
  const showImg = !!src && !failed;

  return (
    <div className={`player-avatar ${className}`} style={style}>
      {showImg ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={src as string}
          alt={alt}
          className="player-avatar-img"
          draggable={false}
          onError={() => setFailed(true)}
        />
      ) : (
        <AvatarSilhouette label={alt} />
      )}
    </div>
  );
}
