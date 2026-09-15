import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import Navbar from "@/components/Navbar";
import { AuthProvider } from "@/lib/auth-context";
import { Toaster } from "react-hot-toast";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "Fantasy Super League — Liga Indonesia Fantasy Football",
  description:
    "Main fantasy football berbasis Liga 1 Indonesia BRI Super League. Pilih pemain terbaik, bentuk timmu, dan buktikan jadi manajer terbaik!",
  keywords: "fantasy football, liga 1, liga indonesia, BRI Super League, fantasy premier league",
  openGraph: {
    title: "Fantasy Super League",
    description: "Fantasy football Liga 1 Indonesia",
    type: "website",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="id">
      <body className={inter.className}>
        <AuthProvider>
          <Navbar />
          <main className="page-wrapper">
            {children}
          </main>
        </AuthProvider>
        <Toaster
          position="top-right"
          toastOptions={{
            style: {
              background: "#141B2E",
              color: "#F0F4FF",
              border: "1px solid rgba(255,255,255,0.06)",
              borderRadius: "12px",
              fontSize: "0.875rem",
            },
            success: {
              iconTheme: { primary: "#00D084", secondary: "#000" },
            },
            error: {
              iconTheme: { primary: "#EF4444", secondary: "#fff" },
            },
          }}
        />
      </body>
    </html>
  );
}
