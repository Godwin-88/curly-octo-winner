'use client';

import { useEffect, useState } from 'react';
import { Plus, Play, CheckCircle, XCircle, MapPin, Navigation, UserCheck } from 'lucide-react';
import { api, Trip, Route, Vehicle, Learner, TripCheckin } from '@/lib/api';

const STATUS_STYLES: Record<string, string> = {
  scheduled: 'bg-blue-100 text-blue-700',
  in_progress: 'bg-green-100 text-green-700',
  completed: 'bg-gray-100 text-gray-500',
  cancelled: 'bg-red-100 text-red-700',
};

export default function TripsPage() {
  const token = ''; // TODO: Get from auth context
  const [trips, setTrips] = useState<Trip[]>([]);
  const [routes, setRoutes] = useState<Route[]>([]);
  const [vehicles, setVehicles] = useState<Vehicle[]>([]);
  const [learners, setLearners] = useState<Learner[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ route_id: '', vehicle_id: '', direction: 'to_school', scheduled_departure: '', notes: '' });

  const [detailTrip, setDetailTrip] = useState<Trip | null>(null);
  const [checkins, setCheckins] = useState<TripCheckin[]>([]);
  const [checkinForm, setCheckinForm] = useState({ learner_id: '', stop_id: '', action: 'boarded' });

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [tripData, routeData, vehicleData, learnerData] = await Promise.all([
        api.listTrips({ status: statusFilter || undefined }, token),
        api.listRoutes(token),
        api.listVehicles({}, token),
        api.listLearners({}, token),
      ]);
      setTrips(tripData);
      setRoutes(routeData);
      setVehicles(vehicleData);
      setLearners(learnerData);
    } catch (e: any) {
      setError(e.message || 'Failed to load trips');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, statusFilter]);

  const handleCreateTrip = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      await api.createTrip({
        route_id: form.route_id,
        vehicle_id: form.vehicle_id || undefined,
        direction: form.direction,
        scheduled_departure: new Date(form.scheduled_departure).toISOString(),
        notes: form.notes || undefined,
      }, token);
      setShowForm(false);
      setForm({ route_id: '', vehicle_id: '', direction: 'to_school', scheduled_departure: '', notes: '' });
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to create trip');
    }
  };

  const handleTransition = async (trip: Trip, action: 'start' | 'complete' | 'cancel') => {
    setError('');
    try {
      if (action === 'start') await api.startTrip(trip.id, token);
      if (action === 'complete') await api.completeTrip(trip.id, token);
      if (action === 'cancel') await api.cancelTrip(trip.id, token);
      load();
      if (detailTrip?.id === trip.id) {
        const updated = await api.getTrip(trip.id, token);
        setDetailTrip(updated);
      }
    } catch (e: any) {
      setError(e.message || `Failed to ${action} trip`);
    }
  };

  const openDetail = async (trip: Trip) => {
    setDetailTrip(trip);
    try {
      const checkinData = await api.listTripCheckins(trip.id, token);
      setCheckins(checkinData);
    } catch (e: any) {
      setError(e.message || 'Failed to load check-ins');
    }
  };

  const handleCheckIn = async () => {
    if (!detailTrip || !checkinForm.learner_id) return;
    setError('');
    try {
      await api.checkInLearner(detailTrip.id, {
        learner_id: checkinForm.learner_id,
        stop_id: checkinForm.stop_id || undefined,
        action: checkinForm.action,
      }, token);
      setCheckinForm({ learner_id: '', stop_id: '', action: 'boarded' });
      openDetail(detailTrip);
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to record check-in');
    }
  };

  const stats = {
    scheduled: trips.filter((t) => t.status === 'scheduled').length,
    inProgress: trips.filter((t) => t.status === 'in_progress').length,
    completed: trips.filter((t) => t.status === 'completed').length,
    totalBoardings: trips.reduce((sum, t) => sum + t.boarded_count, 0),
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Trips</h1>
          <p className="text-sm text-gray-500">Schedule and track transport trips in real-time</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="btn-primary flex items-center gap-2">
          <Plus size={16} /> Schedule Trip
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-4 gap-4 mb-6">
        <div className="card p-4">
          <p className="text-sm text-gray-500">Scheduled</p>
          <p className="text-xl font-bold">{stats.scheduled}</p>
        </div>
        <div className="card p-4">
          <p className="text-sm text-gray-500">In Progress</p>
          <p className="text-xl font-bold text-green-600">{stats.inProgress}</p>
        </div>
        <div className="card p-4">
          <p className="text-sm text-gray-500">Completed</p>
          <p className="text-xl font-bold">{stats.completed}</p>
        </div>
        <div className="card p-4">
          <p className="text-sm text-gray-500">Boardings</p>
          <p className="text-xl font-bold">{stats.totalBoardings}</p>
        </div>
      </div>

      {/* Add form */}
      {showForm && (
        <div className="card p-5 mb-6">
          <h2 className="font-semibold mb-4">Schedule New Trip</h2>
          <form onSubmit={handleCreateTrip} className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Route *</label>
              <select
                required
                value={form.route_id}
                onChange={(e) => setForm({ ...form, route_id: e.target.value })}
                className="w-full px-3 py-2 border rounded-md text-sm"
              >
                <option value="">Select route...</option>
                {routes.filter((r) => r.active).map((r) => (
                  <option key={r.id} value={r.id}>{r.name}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Vehicle</label>
              <select
                value={form.vehicle_id}
                onChange={(e) => setForm({ ...form, vehicle_id: e.target.value })}
                className="w-full px-3 py-2 border rounded-md text-sm"
              >
                <option value="">Auto</option>
                {vehicles.filter((v) => v.status === 'active').map((v) => (
                  <option key={v.id} value={v.id}>{v.registration}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Direction</label>
              <select
                value={form.direction}
                onChange={(e) => setForm({ ...form, direction: e.target.value })}
                className="w-full px-3 py-2 border rounded-md text-sm"
              >
                <option value="to_school">To School</option>
                <option value="from_school">From School</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Departure *</label>
              <input
                required
                type="datetime-local"
                value={form.scheduled_departure}
                onChange={(e) => setForm({ ...form, scheduled_departure: e.target.value })}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Notes</label>
              <input
                value={form.notes}
                onChange={(e) => setForm({ ...form, notes: e.target.value })}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>
            <div className="sm:col-span-2 lg:col-span-5 flex gap-2 pt-2">
              <button type="submit" className="btn-primary text-sm">Schedule Trip</button>
              <button type="button" onClick={() => setShowForm(false)} className="btn-secondary text-sm">Cancel</button>
            </div>
          </form>
        </div>
      )}

      {/* Status filter */}
      <div className="card p-4 mb-6">
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="px-3 py-2 border rounded-md text-sm"
        >
          <option value="">All Statuses</option>
          <option value="scheduled">Scheduled</option>
          <option value="in_progress">In Progress</option>
          <option value="completed">Completed</option>
          <option value="cancelled">Cancelled</option>
        </select>
      </div>

      {error && <div className="bg-red-50 text-red-700 p-3 rounded-md mb-4 text-sm">{error}</div>}

      {/* Trips list */}
      <div className="space-y-4">
        {loading ? (
          <div className="card p-8 text-center text-gray-400">Loading trips...</div>
        ) : trips.length === 0 ? (
          <div className="card p-8 text-center text-gray-400">
            <Navigation className="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p>No trips found</p>
            <p className="text-xs mt-2">Schedule a trip to get started</p>
          </div>
        ) : (
          trips.map((trip) => (
            <div key={trip.id} className="card p-4">
              <div className="flex items-center justify-between">
                <button onClick={() => openDetail(trip)} className="flex items-center gap-3 text-left hover:opacity-80">
                  <div className="w-10 h-10 rounded-lg bg-blue-100 text-blue-600 flex items-center justify-center">
                    <Navigation size={20} />
                  </div>
                  <div>
                    <p className="font-semibold">{trip.route_name || 'Unknown route'}</p>
                    <p className="text-xs text-gray-500">
                      {new Date(trip.scheduled_departure).toLocaleString()} · {trip.direction === 'to_school' ? 'To School' : 'From School'} · {trip.vehicle_registration || 'No vehicle'}
                      {trip.last_latitude && trip.last_longitude && (
                        <span className="ml-2 text-green-600 flex items-center gap-1">
                          <MapPin size={12} /> Live
                        </span>
                      )}
                    </p>
                  </div>
                </button>
                <div className="flex items-center gap-2">
                  {trip.status === 'in_progress' && (
                    <span className="text-xs text-gray-500 mr-2">{trip.boarded_count} boarded</span>
                  )}
                  <span className={`px-2 py-0.5 rounded-full text-xs ${STATUS_STYLES[trip.status] || 'bg-gray-100 text-gray-500'}`}>
                    {trip.status}
                  </span>
                  {trip.status === 'scheduled' && (
                    <button onClick={() => handleTransition(trip, 'start')} className="btn-primary text-xs flex items-center gap-1 px-2 py-1">
                      <Play size={12} /> Start
                    </button>
                  )}
                  {trip.status === 'in_progress' && (
                    <button onClick={() => handleTransition(trip, 'complete')} className="btn-secondary text-xs flex items-center gap-1 px-2 py-1">
                      <CheckCircle size={12} /> Complete
                    </button>
                  )}
                  {(trip.status === 'scheduled' || trip.status === 'in_progress') && (
                    <button onClick={() => handleTransition(trip, 'cancel')} className="text-red-500 hover:text-red-700 text-xs flex items-center gap-1 px-2 py-1">
                      <XCircle size={12} /> Cancel
                    </button>
                  )}
                </div>
              </div>

              {/* Detail panel */}
              {detailTrip?.id === trip.id && (
                <div className="border-t mt-4 pt-4">
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-4">
                    <div>
                      <h3 className="font-semibold text-sm mb-2 flex items-center gap-2">
                        <MapPin size={14} /> Live Position
                      </h3>
                      {trip.last_latitude && trip.last_longitude ? (
                        <div className="bg-gray-50 rounded p-3 text-sm">
                          <p className="font-mono text-xs">{trip.last_latitude.toFixed(6)}, {trip.last_longitude.toFixed(6)}</p>
                          <p className="text-xs text-gray-500 mt-1">Last reported: {trip.last_reported ? new Date(trip.last_reported).toLocaleTimeString() : 'N/A'}</p>
                        </div>
                      ) : (
                        <p className="text-sm text-gray-400">No position data yet. Start the trip and report GPS pings.</p>
                      )}
                    </div>
                    <div>
                      <h3 className="font-semibold text-sm mb-2 flex items-center gap-2">
                        <UserCheck size={14} /> Check-ins ({checkins.length})
                      </h3>
                      {checkins.length > 0 && (
                        <div className="space-y-1 mb-3">
                          {checkins.map((c) => (
                            <div key={c.id} className="bg-gray-50 px-3 py-1.5 rounded text-xs flex justify-between">
                              <span>{c.learner_name} <span className="text-gray-500">@ {c.stop_name || '—'}</span></span>
                              <span className={`font-medium ${c.action === 'boarded' ? 'text-green-600' : 'text-orange-600'}`}>
                                {c.action} · {new Date(c.checked_at).toLocaleTimeString()}
                              </span>
                            </div>
                          ))}
                        </div>
                      )}
                      <div className="flex gap-2">
                        <select
                          value={checkinForm.learner_id}
                          onChange={(e) => setCheckinForm({ ...checkinForm, learner_id: e.target.value })}
                          className="flex-1 px-2 py-1.5 border rounded-md text-xs"
                        >
                          <option value="">Select learner...</option>
                          {learners.filter((l) => l.is_active).map((l) => (
                            <option key={l.id} value={l.id}>{l.full_name}</option>
                          ))}
                        </select>
                        <select
                          value={checkinForm.stop_id}
                          onChange={(e) => setCheckinForm({ ...checkinForm, stop_id: e.target.value })}
                          className="flex-1 px-2 py-1.5 border rounded-md text-xs"
                        >
                          <option value="">Stop (optional)</option>
                          {routes.find((r) => r.id === trip.route_id)?.stops?.map((s) => (
                            <option key={s.id} value={s.id}>{s.name}</option>
                          ))}
                        </select>
                        <select
                          value={checkinForm.action}
                          onChange={(e) => setCheckinForm({ ...checkinForm, action: e.target.value })}
                          className="px-2 py-1.5 border rounded-md text-xs"
                        >
                          <option value="boarded">Boarded</option>
                          <option value="alighted">Alighted</option>
                        </select>
                        <button onClick={handleCheckIn} className="btn-secondary text-xs px-3 py-1.5">Record</button>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  );
}