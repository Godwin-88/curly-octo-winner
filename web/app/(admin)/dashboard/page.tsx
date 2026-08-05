export default function DashboardPage() {
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div className="card p-6">
          <h3 className="text-sm text-gray-500">Total Learners</h3>
          <p className="text-3xl font-bold mt-2">50</p>
        </div>
        <div className="card p-6">
          <h3 className="text-sm text-gray-500">Active Guardians</h3>
          <p className="text-3xl font-bold mt-2">20</p>
        </div>
        <div className="card p-6">
          <h3 className="text-sm text-gray-500">Messages Sent</h3>
          <p className="text-3xl font-bold mt-2">0</p>
        </div>
        <div className="card p-6">
          <h3 className="text-sm text-gray-500">Open Conversations</h3>
          <p className="text-3xl font-bold mt-2">0</p>
        </div>
      </div>
    </div>
  );
}