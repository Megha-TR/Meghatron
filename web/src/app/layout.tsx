import './globals.css';
import Sidebar from '@/components/Sidebar';

export const metadata = {
  title: 'Meghatron — Container Orchestration',
  description: 'Meghatron container orchestration dashboard',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className="h-full">
      <body className="h-full bg-gray-900 text-gray-100">
        <div className="flex h-full">
          <Sidebar />
          <main className="flex-1 ml-64 min-h-screen overflow-auto">
            <div className="p-8">
              {children}
            </div>
          </main>
        </div>
      </body>
    </html>
  );
}
