import type { Metadata } from "next";
import "./globals.css";
import Navbar from "@/components/Navbar";
import { AuthProvider } from "@/lib/auth-context";
import { Toaster } from "react-hot-toast";

// NOTE: font dimuat via <link> (bukan next/font/google) agar dev/build
// tidak gagal saat fonts.googleapis.com tidak terjangkau. CSS sudah punya
// fallback system-ui sehingga halaman tetap tampil tanpa font Google.
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
      <head>
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        <link
          href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800;900&family=Space+Grotesk:wght@400;500;600;700&display=swap"
          rel="stylesheet"
        />
      </head>
      <body>
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
