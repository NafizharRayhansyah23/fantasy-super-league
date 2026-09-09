"use client";

import React, { createContext, useContext, useEffect, useState } from "react";
import { authApi, getToken, getUser, removeToken, setToken, setUser, User } from "@/lib/api";

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isLoggedIn: boolean;
  login: (token: string, user: User) => void;
  logout: () => void;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUserState] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const init = async () => {
      const token = getToken();
      if (token) {
        const cached = getUser();
        if (cached) setUserState(cached);
        try {
          const fresh = await authApi.getMe();
          setUserState(fresh);
          setUser(fresh);
        } catch {
          removeToken();
          setUserState(null);
        }
      }
      setIsLoading(false);
    };
    init();
  }, []);

  const login = (token: string, userData: User) => {
    setToken(token);
    setUser(userData);
    setUserState(userData);
  };

  const logout = () => {
    removeToken();
    setUserState(null);
  };

  const refreshUser = async () => {
    try {
      const fresh = await authApi.getMe();
      setUserState(fresh);
      setUser(fresh);
    } catch {
      logout();
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading,
        isLoggedIn: !!user,
        login,
        logout,
        refreshUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
