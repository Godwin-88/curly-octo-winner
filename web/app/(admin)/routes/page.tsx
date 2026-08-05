'use client';

import { useEffect, useState } from 'react';
import { Plus, MapPin, Trash2, Users, UserPlus } from 'lucide-react';
import { api, Route, Vehicle, Learner, Assignment } from '@/lib/api';

export default function RoutesPage() {
  const token = ''; // TODO: Get from auth context
  const [routes, setRoutes] = useState<Route[]>([]);
  const [vehicles, setVehicles] = useState<Vehicle[]>([]);
  const [learners, setLearners] = useState<Learner[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ name: '', description: '', vehicle_id: '' });

  const [expandedRoute, setExpandedRoute] = useState<string | null>(null);
  const [assignments, setAssignments] = useState<Assignment[]>([]);
  const [newStop, setNewStop] = useState({ name: '', landmark: '' });
  const [newAssignment, setNewAssignment] = useState({ learner_id: '', stop_id: '', direction: 'both' });

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [routeData, vehicleData, learnerData] = await Promise.all([
        api.listRoutes(token),
        api.listVehicles({}, token),
        api.listLearners({}, token),
      ]);
      setRoutes(routeData);
      setVehicles(vehicleData);
      setLearners(learnerData);
    } catch (e: any) {
      setError(e.message || 'Failed to load routes');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const toggleRoute = async (routeId: string) => {
    if (expandedRoute === routeId) {
      setExpandedRoute(null);
      return;
    }
    setExpandedRoute(routeId);
    try {
      const data = await api.listRouteAssignments(routeId, token);
      setAssignments(data);
    } catch (e: any) {
      setError(e.message || 'Failed to load assignments');
    }
  };

  const handleCreateRoute = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      await api.createRoute({
        name: form.name,
        description: form.description || undefined,
        vehicle_id: form.vehicle_id || undefined,
      }, token);
      setShowForm(false);
      setForm({ name: '', description: '', vehicle_id: '' });
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to create route');
    }
  };

  const handleAddStop = async (routeId: string) => {
    if (!newStop.name) return;
    setError('');
    try {
      const route = routes.find((r) => r.id === routeId);
      const seq = (route?.stops?.length || 0) + 1;
      await api.createRouteStop(routeId, { name: newStop.name, sequence: seq, landmark: newStop.landmark || undefined }, token);
      setNewStop({ name: '', landmark: '' });
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to add stop');
    }
  };

  const handleRemoveStop = async (stopId: string) => {
    setError('');
    try {
      await api.deleteRouteStop(stopId, token);
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to remove stop');
    }
  };

  const handleAssign = async (routeId: string) => {
    if (!newAssignment.learner_id || !newAssignment.stop_id) return;
    setError('');
    try {
      await api.assignLearnerToRoute(routeId, {
        learner_id: newAssignment.learner_id,
        stop_id: newAssignment.stop_id,
        direction: newAssignment.direction,
      }, token);
      setNewAssignment({ learner_id: '', stop_id: '', direction: 'both' });
      toggleRoute(routeId);
    } catch (e: any) {
      setError(e.message || 'Failed to assign learner');
    }
  };

  const handleRemoveAssignment = async (assignmentId: string) => {
    setError('');
    try {
      await api.removeRouteAssignment(assignmentId, token);
      if (expandedRoute) toggleRoute(expandedRoute);
    } catch (e: any) {
      setError(e.message || 'Failed to remove assignment');
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Routes</h1>
          <p className="text-sm text-gray-500">Manage transport routes, stops and learner assignments</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="btn-primary flex items-center gap-2">
          <Plus size={16} /> Add Route
        </button>
      </div>

      {/* Add form */}
      {showForm && (
        <div className="card p-5 mb-6">
          <h2 className="font-semibold mb-4">Add New Route</h2>
          <form onSubmit={handleCreateRoute} className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Route Name *</label>
              <input
                required
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                className="w-full px-3 py-2 border rounded-md text-sm"
                placeholder="Route C - Thika"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Description</label>
              <input
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Vehicle</label>
              <select
                value={form.vehicle_id}
                onChange={(e) => setForm({ ...form, vehicle_id: e.target.value })}
                className="w-full px-3 py-2 border rounded-md text-sm"
              >
                <option value="">Unassigned</option>
                {vehicles.filter((v) => v.status === 'active').map((v) => (
                  <option key={v.id} value={v.id}>{v.registration} - {v.make} {v.model}</option>
                ))}
              </select>
            </div>
            <div className="sm:col-span-3 flex gap-2 pt-2">
              <button type="submit" className="btn-primary text-sm">Save Route</button>
              <button type="button" onClick={() => setShowForm(false)} className="btn-secondary text-sm">Cancel</button>
            </div>
          </form>
        </div>
      )}

      {error && <div className="bg-red-50 text-red-700 p-3 rounded-md mb-4 text-sm">{error}</div>}

      {/* Route cards */}
      <div className="space-y-4">
        {loading ? (
          <div className="card p-8 text-center text-gray-400">Loading routes...</div>
        ) : routes.length === 0 ? (
          <div className="card p-8 text-center text-gray-400">
            <MapPin className="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p>No routes found</p>
            <p className="text-xs mt-2">Create a route to get started</p>
          </div>
        ) : (
          routes.map((route) => {
            const vehicle = vehicles.find((v) => v.id === route.vehicle_id);
            return (
              <div key={route.id} className="card">
                <button
                  onClick={() => toggleRoute(route.id)}
                  className="w-full p-4 flex items-center justify-between hover:bg-gray-50 transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-lg bg-blue-100 text-blue-600 flex items-center justify-center">
                      <MapPin size={20} />
                    </div>
                    <div>
                      <p className="font-semibold">{route.name}</p>
                      <p className="text-xs text-gray-500">
                        {route.description || 'No description'} · {route.stops?.length || 0} stops · {vehicle ? vehicle.registration : 'No vehicle assigned'}
                      </p>
                    </div>
                  </div>
                  <span className={`px-2 py-0.5 rounded-full text-xs ${route.active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
                    {route.active ? 'Active' : 'Inactive'}
                  </span>
                </button>

                {expandedRoute === route.id && (
                  <div className="border-t p-4 space-y-6">
                    {/* Stops */}
                    <div>
                      <h3 className="font-semibold text-sm mb-3 flex items-center gap-2">
                        <MapPin size={14} /> Stops ({route.stops?.length || 0})
                      </h3>
                      <div className="space-y-2">
                        {route.stops?.map((s) => (
                          <div key={s.id} className="flex items-center justify-between bg-gray-50 px-3 py-2 rounded">
                            <div className="flex items-center gap-2 text-sm">
                              <span className="text-xs bg-blue-600 text-white w-5 h-5 rounded-full flex items-center justify-center">{s.sequence}</span>
                              <div>
                                <span className="font-medium">{s.name}</span>
                                {s.landmark && <span className="text-xs text-gray-500 ml-2">{s.landmark}</span>}
                              </div>
                            </div>
                            <button onClick={() => handleRemoveStop(s.id)} className="text-red-500 hover:text-red-700">
                              <Trash2 size={14} />
                            </button>
                          </div>
                        ))}
                      </div>
                      <div className="flex gap-2 mt-3">
                        <input
                          value={newStop.name}
                          onChange={(e) => setNewStop({ ...newStop, name: e.target.value })}
                          placeholder="Enter stop name (e.g. Kahawa West)"
                          className="flex-1 px-3 py-2 border rounded-md text-sm"
                        />
                        <input
                          value={newStop.landmark}
                          onChange={(e) => setNewStop({ ...newStop, landmark: e.target.value })}
                          placeholder="Landmark (optional)"
                          className="w-40 px-3 py-2 border rounded-md text-sm"
                        />
                        <button onClick={() => handleAddStop(route.id)} className="btn-secondary text-sm">Add Stop</button>
                      </div>
                    </div>

                    {/* Assignments */}
                    <div>
                      <h3 className="font-semibold text-sm mb-3 flex items-center gap-2">
                        <Users size={14} /> Assigned Learners ({assignments.length})
                      </h3>
                      {assignments.length > 0 && (
                        <div className="space-y-1 mb-3">
                          {assignments.map((a) => (
                            <div key={a.id} className="flex items-center justify-between bg-gray-50 px-3 py-2 rounded text-sm">
                              <div>
                                <span className="font-medium">{a.learner_name}</span>
                                <span className="text-xs text-gray-500 ml-2">{a.grade} · {a.stream || '—'}</span>
                                <span className="text-xs text-gray-500 ml-2">@{a.stop_name || '—'}</span>
                                <span className="ml-2 text-xs bg-blue-100 text-blue-700 px-2 py-0.5 rounded-full">{a.direction}</span>
                              </div>
                              <button onClick={() => handleRemoveAssignment(a.id)} className="text-red-500 hover:text-red-700">
                                <Trash2 size={14} />
                              </button>
                            </div>
                          ))}
                        </div>
                      )}
                      <div className="flex gap-2">
                        <select
                          value={newAssignment.learner_id}
                          onChange={(e) => setNewAssignment({ ...newAssignment, learner_id: e.target.value })}
                          className="flex-1 px-3 py-2 border rounded-md text-sm"
                        >
                          <option value="">Select learner...</option>
                          {learners.filter((l) => l.is_active).map((l) => (
                            <option key={l.id} value={l.id}>{l.full_name} ({l.grade})</option>
                          ))}
                        </select>
                        <select
                          value={newAssignment.stop_id}
                          onChange={(e) => setNewAssignment({ ...newAssignment, stop_id: e.target.value })}
                          className="flex-1 px-3 py-2 border rounded-md text-sm"
                        >
                          <option value="">Select stop...</option>
                          {route.stops?.map((s) => (
                            <option key={s.id} value={s.id}>{s.sequence}. {s.name}</option>
                          ))}
                        </select>
                        <select
                          value={newAssignment.direction}
                          onChange={(e) => setNewAssignment({ ...newAssignment, direction: e.target.value })}
                          className="px-3 py-2 border rounded-md text-sm"
                        >
                          <option value="both">Both ways</option>
                          <option value="to_school">To school</option>
                          <option value="from_school">From school</option>
                        </select>
                        <button onClick={() => handleAssign(route.id)} className="btn-secondary text-sm flex items-center gap-1">
                          <UserPlus size={14} /> Assign
                        </button>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}