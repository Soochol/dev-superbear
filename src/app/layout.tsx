import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "NEXUS — AI-Native Investment Intelligence",
  description: "Agent-based investment analysis platform",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="ko" className="dark">
      <head>
        <link
          rel="stylesheet"
          as="style"
          crossOrigin="anonymous"
          href="https://cdn.jsdelivr.net/gh/orioncactus/pretendard@v1.3.9/dist/web/variable/pretendardvariable-dynamic-subset.min.css"
        />
      </head>
      <body className="font-sans bg-nexus-bg text-nexus-text-primary min-h-screen antialiased">
        {children}
      </body>
    </html>
  );
}
