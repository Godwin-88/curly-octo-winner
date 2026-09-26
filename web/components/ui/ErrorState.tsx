// Standard "something went wrong" panel used by pages (failed fetches) and
// route error boundaries alike. Offers a retry action when one makes sense.

'use client';

export function ErrorState({
  title = 'Something went wrong',
  message,
  onRetry,
}: {
  title?: string;
  message?: string;
  onRetry?: () => void;
}) {
  return (
    <div
      role="alert"
      className="card p-8 flex flex-col items-center justify-center text-center gap-2"
    >
      <div className="text-3xl" aria-hidden="true">
        ⚠️
      </div>
      <h3 className="text-base font-semibold text-gray-900">{title}</h3>
      {message && <p className="text-sm text-gray-500 max-w-md">{message}</p>}
      {onRetry && (
        <button type="button" onClick={onRetry} className="btn-primary mt-3">
          Try again
        </button>
      )}
    </div>
  );
}
