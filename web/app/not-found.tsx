import Link from 'next/link';

export default function NotFound() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 p-6">
      <div className="card max-w-md w-full p-8 text-center space-y-3">
        <div className="text-4xl" aria-hidden="true">
          🧭
        </div>
        <h1 className="text-xl font-bold text-gray-900">Page not found</h1>
        <p className="text-sm text-gray-500">
          The page you are looking for doesn&apos;t exist or was moved.
        </p>
        <div className="flex items-center justify-center gap-3 pt-2">
          <Link href="/dashboard" className="btn-primary">
            Go to dashboard
          </Link>
          <Link href="/" className="btn-secondary">
            Home
          </Link>
        </div>
      </div>
    </div>
  );
}
