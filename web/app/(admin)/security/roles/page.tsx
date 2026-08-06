'use client';

import { useEffect, useState } from 'react';
import { ShieldCheck, Check, Plus } from 'lucide-react';
import { api, RolePermissionsResponse, Permission } from '@/lib/api';

const ROLE_LABELS: Record<string, string> = {
  super_admin: 'Super Admin',
  principal: 'Principal',
  teacher: 'Teacher',
  bursar: 'Bursar',
  transport_manager: 'Transport Manager',
  hr: 'HR',
};

export default function RolesPage() {
  const token = ''; // TODO: Get from auth context
  const [roles, setRoles] = useState<RolePermissionsResponse[]>([]);
  const [allPermissions, setAllPermissions] = useState<Permission[]>([]);
  const [selectedRole, setSelectedRole] = useState<string>('principal');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [r, p] = await Promise.all([
        api.listRoles(token),
        api.listPermissions(token),
      ]);
      setRoles(r);
      setAllPermissions(p);
      if (r.length > 0) setSelectedRole(r[0].role);
    } catch (e: any) {
      setError(e.message || 'Failed to load roles');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const categories = Array.from(new Set(allPermissions.map((p) => p.category || 'other')));

  const isGranted = (role: string, code: string) => {
    const rr = roles.find((x) => x.role === role);
    return rr ? rr.permissions.some((p) => p.code === code) : false;
  };

  const togglePermission = async (role: string, code: string, granted: boolean) => {
    setSaving(true);
    setError('');
    try {
      if (granted) {
        await api.revokeRolePermission(role, code, token);
      } else {
        await api.grantRolePermission(role, { permission_code: code }, token);
      }
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to update permission');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Role-Based Access Control</h1>
          <p className="text-gray-500">Grant or revoke permissions for each role</p>
        </div>
      </div>

      {error && <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      {loading ? (
        <p className="text-gray-500 mt-4">Loading...</p>
      ) : (
        <div className="mt-6 flex gap-6">
          {/* Role list */}
          <div className="w-56 shrink-0 space-y-2">
            {roles.map((r) => (
              <button
                key={r.role}
                onClick={() => setSelectedRole(r.role)}
                className={`w-full text-left flex items-center gap-2 px-3 py-2 rounded-md text-sm transition-colors ${
                  selectedRole === r.role ? 'bg-blue-600 text-white' : 'bg-white border text-gray-700 hover:bg-gray-50'
                }`}
              >
                <ShieldCheck size={16} />
                <div className="flex-1">
                  <p className="font-medium">{ROLE_LABELS[r.role] || r.role}</p>
                  <p className={`text-xs ${selectedRole === r.role ? 'text-blue-100' : 'text-gray-400'}`}>
                    {r.permissions.length} permissions
                  </p>
                </div>
              </button>
            ))}
          </div>

          {/* Permission matrix */}
          <div className="flex-1 bg-white rounded-lg shadow border">
            <div className="p-6 pb-4 flex items-center justify-between">
              <div>
                <h2 className="text-lg font-semibold">{ROLE_LABELS[selectedRole] || selectedRole}</h2>
                <p className="text-sm text-gray-500">Toggle permissions for this role</p>
              </div>
              {saving && <p className="text-sm text-blue-600">Saving...</p>}
            </div>
            <div className="p-6 pt-0 space-y-6">
              {categories.map((cat) => (
                <div key={cat}>
                  <h3 className="text-sm font-semibold text-gray-500 uppercase mb-2">{cat}</h3>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                    {allPermissions.filter((p) => (p.category || 'other') === cat).map((p) => {
                      const granted = isGranted(selectedRole, p.code);
                      return (
                        <button
                          key={p.code}
                          onClick={() => togglePermission(selectedRole, p.code, granted)}
                          disabled={saving}
                          className={`flex items-center gap-3 px-3 py-2 rounded-md border text-sm transition-colors disabled:opacity-50 ${
                            granted ? 'bg-green-50 border-green-200 text-green-700' : 'bg-gray-50 border-gray-200 text-gray-600 hover:bg-gray-100'
                          }`}
                        >
                          <span className={`w-5 h-5 rounded flex items-center justify-center ${granted ? 'bg-green-600 text-white' : 'bg-gray-200 text-gray-400'}`}>
                            {granted && <Check size={12} />}
                          </span>
                          <span className="flex-1 text-left">
                            <span className="font-mono text-xs block">{p.code}</span>
                            {p.description && <span className="text-xs text-gray-400">{p.description}</span>}
                          </span>
                        </button>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}