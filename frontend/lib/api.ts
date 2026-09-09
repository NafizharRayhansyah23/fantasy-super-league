// API base URL
const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

// ─── Auth helpers ─────────────────────────────────────────
export const getToken = (): string | null => {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("fsl_token");
};

export const setToken = (token: string) => {
  localStorage.setItem("fsl_token", token);
};

export const removeToken = () => {
  localStorage.removeItem("fsl_token");
  localStorage.removeItem("fsl_user");
};

export const getUser = () => {
  if (typeof window === "undefined") return null;
  const raw = localStorage.getItem("fsl_user");
  return raw ? JSON.parse(raw) : null;
};

export const setUser = (user: object) => {
  localStorage.setItem("fsl_user", JSON.stringify(user));
};

// ─── Base fetch wrapper ────────────────────────────────────
async function apiFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_URL}${path}`, { ...options, headers });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Request failed" }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }

  return res.json();
}

// ─── Auth API ──────────────────────────────────────────────
export const authApi = {
  register: (data: {
    username: string;
    email: string;
    password: string;
    team_name: string;
  }) => apiFetch<{ token: string; user: User }>("/auth/register", {
    method: "POST",
    body: JSON.stringify(data),
  }),

  login: (data: { email: string; password: string }) =>
    apiFetch<{ token: string; user: User }>("/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  getMe: () => apiFetch<User>("/auth/me"),
};

// ─── Players API ───────────────────────────────────────────
export const playersApi = {
  getAll: (params: PlayerFilter = {}) => {
    const qs = new URLSearchParams(
      Object.entries(params)
        .filter(([, v]) => v !== undefined && v !== "")
        .map(([k, v]) => [k, String(v)])
    ).toString();
    return apiFetch<PlayerListResponse>(`/players${qs ? "?" + qs : ""}`);
  },

  getById: (id: string) =>
    apiFetch<{ player: Player; stats: PlayerStats[] }>(`/players/${id}`),
};

// ─── Clubs API ─────────────────────────────────────────────
export const clubsApi = {
  getAll: () => apiFetch<{ data: Club[] }>("/clubs"),
};

// ─── Gameweeks API ─────────────────────────────────────────
export const gameweeksApi = {
  getAll: () => apiFetch<{ data: Gameweek[] }>("/gameweeks"),
};

// ─── Team API ──────────────────────────────────────────────
export const teamApi = {
  getMyTeam: () => apiFetch<{ data: FantasyTeam; has_team: boolean }>("/team"),

  saveTeam: (data: SaveTeamRequest) =>
    apiFetch<{ message: string; remaining_budget: number }>("/team", {
      method: "POST",
      body: JSON.stringify(data),
    }),

  makeTransfer: (data: TransferRequest) =>
    apiFetch<{ message: string; points_deducted: number; remaining_budget: number }>(
      "/team/transfer",
      { method: "POST", body: JSON.stringify(data) }
    ),

  getGameweekPoints: (gwNum: number) =>
    apiFetch<{ data: PlayerPoints[]; total_points: number }>(
      `/team/points/${gwNum}`
    ),
};

// ─── Leaderboard API ───────────────────────────────────────
export const leaderboardApi = {
  get: () => apiFetch<{ data: LeaderboardEntry[] }>("/leaderboard"),
};

// ─── Types ─────────────────────────────────────────────────
export interface User {
  id: string;
  username: string;
  email: string;
  team_name: string;
  total_points: number;
  created_at: string;
}

export interface Club {
  id: string;
  name: string;
  short_name: string;
  slug: string;
  logo_url: string;
  stadium: string;
  city: string;
  tier: number;
}

export interface Player {
  id: string;
  name: string;
  slug: string;
  club_id: string;
  club?: Club;
  position: "GK" | "DEF" | "MID" | "FWD";
  nationality: string;
  is_national_team: boolean;
  price: number;
  photo_url: string;
  jersey_number: number;
  is_active: boolean;
  total_points: number;
}

export interface PlayerStats {
  gameweek_id: string;
  gameweek_name: string;
  gameweek_num: number;
  minutes_played: number;
  goals: number;
  assists: number;
  clean_sheet: boolean;
  yellow_cards: number;
  red_cards: number;
  saves: number;
  bonus: number;
  fantasy_points: number;
}

export interface Gameweek {
  id: string;
  number: number;
  name: string;
  is_active: boolean;
  is_finished: boolean;
  deadline: string;
}

export interface FantasyTeamPlayer {
  id: string;
  team_id: string;
  player_id: string;
  player?: Player;
  is_captain: boolean;
  is_vice_captain: boolean;
  is_starting: boolean;
  position_slot: number;
}

export interface FantasyTeam {
  id: string;
  user_id: string;
  team_name: string;
  total_budget: number;
  remaining_budget: number;
  wildcard_used: boolean;
  free_transfers: number;
  total_points: number;
  players: FantasyTeamPlayer[];
}

export interface PlayerPoints {
  player_id: string;
  name: string;
  position: string;
  photo_url: string;
  is_captain: boolean;
  is_vice_captain: boolean;
  is_starting: boolean;
  goals: number;
  assists: number;
  minutes_played: number;
  clean_sheet: boolean;
  yellow_cards: number;
  red_cards: number;
  saves: number;
  bonus: number;
  fantasy_points: number;
  total_points: number;
}

export interface LeaderboardEntry {
  rank: number;
  user_id: string;
  username: string;
  team_name: string;
  total_points: number;
}

export interface PlayerFilter {
  position?: string;
  club_id?: string;
  search?: string;
  min_price?: number;
  max_price?: number;
  page?: number;
  limit?: number;
}

export interface PlayerListResponse {
  data: Player[];
  total: number;
  page: number;
  limit: number;
  pages: number;
}

export interface SaveTeamRequest {
  team_name: string;
  players: Array<{
    player_id: string;
    is_captain: boolean;
    is_vice_captain: boolean;
    is_starting: boolean;
    position_slot: number;
  }>;
}

export interface TransferRequest {
  player_out_id: string;
  player_in_id: string;
  use_wildcard: boolean;
}

// ─── Position helpers ──────────────────────────────────────
export const POSITION_LABELS: Record<string, string> = {
  GK: "Kiper",
  DEF: "Bek",
  MID: "Gelandang",
  FWD: "Penyerang",
};

export const POSITION_SLOTS = {
  GK: [0, 1],
  DEF: [2, 3, 4, 5, 6],
  MID: [7, 8, 9, 10, 11],
  FWD: [12, 13, 14],
};

export const formatPrice = (price: number) =>
  `Rp ${price.toFixed(1)}jt`;

export const formatPoints = (pts: number) =>
  pts >= 0 ? `+${pts}` : `${pts}`;
