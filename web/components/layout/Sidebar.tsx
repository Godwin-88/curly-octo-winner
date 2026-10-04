'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useAuth } from '@/lib/auth';
import { moduleEnabled } from '@/framework/registry';
import { useSchoolModules } from '@/shell/ContextBar';
import { navPath } from '@/shell/context';
import {
  LayoutDashboard,
  MessageSquare,
  Contact,
  Users,
  GraduationCap,
  Bus,
  Wallet,
  Settings,
  BookOpen,
  ClipboardList,
  CalendarCheck,
  MapPin,
  Navigation,
  FileText,
  FileSpreadsheet,
  Receipt,
  Smartphone,
  BarChart3,
  FileBarChart,
  Briefcase,
  UserCog,
  Banknote,
  CalendarClock,
  ClipboardCheck,
  ShoppingCart,
  Truck,
  PackageCheck,
  CreditCard,
  Brain,
  TrendingUp,
  MessageSquareText,
  ShieldCheck,
  KeyRound,
  UserCheck,
  ChevronDown,
  PanelLeftClose,
  LogOut,
} from 'lucide-react';

interface ChildItem {
  href: string;
  label: string;
  icon: any;
}

interface NavItem {
  href?: string;
  label: string;
  icon: any;
  children?: ChildItem[];
}

const navItems: NavItem[] = [
  { href: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  {
    label: 'School setup',
    icon: ClipboardList,
    children: [
      { href: '/school/setup', label: 'Getting Started', icon: ClipboardList },
      { href: '/school/users', label: 'Users', icon: Users },
    ],
  },
    {
      label: 'Communications',
      icon: MessageSquare,
      children: [
        // These open in the list → view → edit workspace (/w/...); the short
        // addresses forward there. WhatsApp and its inbox are deferred.
        { href: '/communications/messages', label: 'Messages', icon: MessageSquare },
        { href: '/communications/contacts', label: 'Contacts', icon: Contact },
        { href: '/communications/templates', label: 'SMS Templates', icon: MessageSquareText },
      ],
    },
  {
    label: 'Academic',
    icon: GraduationCap,
    children: [
      { href: '/academic', label: 'Overview', icon: GraduationCap },
      { href: '/academic/curriculum', label: 'Curriculum', icon: BookOpen },
      { href: '/academic/assessments', label: 'Assessments', icon: ClipboardList },
      { href: '/academic/attendance', label: 'Attendance', icon: CalendarCheck },
    ],
  },
  {
    label: 'Learners',
    icon: Users,
    children: [
      { href: '/learners', label: 'All Learners', icon: Users },
      { href: '/learners/import', label: 'Import Roster', icon: FileSpreadsheet },
    ],
  },
  {
    label: 'Transport',
    icon: Bus,
    children: [
      { href: '/vehicles', label: 'Vehicles', icon: Bus },
      { href: '/routes', label: 'Routes', icon: MapPin },
      { href: '/trips', label: 'Trips & Tracking', icon: Navigation },
    ],
  },
  {
    label: 'Finance',
    icon: Wallet,
    children: [
      { href: '/finance', label: 'Overview', icon: Wallet },
      { href: '/finance/invoices', label: 'Invoices', icon: Receipt },
      { href: '/finance/payments', label: 'Payments', icon: Smartphone },
      { href: '/finance/paybill', label: 'Paybill Payments', icon: Smartphone },
      { href: '/finance/balances', label: 'Balances', icon: FileText },
      { href: '/finance/fees', label: 'Fee Structures', icon: FileText },
      { href: '/finance/fee-items', label: 'Fee Items', icon: FileText },
    ],
  },
  {
    label: 'Reports & Analytics',
    icon: BarChart3,
    children: [
      { href: '/reports', label: 'Overview', icon: BarChart3 },
      { href: '/reports/cards', label: 'Report Cards', icon: FileBarChart },
      { href: '/analytics', label: 'Analytics', icon: BarChart3 },
    ],
  },
  {
    label: 'Human Resources',
    icon: Briefcase,
    children: [
      { href: '/hr', label: 'Overview', icon: Briefcase },
      { href: '/hr/staff', label: 'Staff Directory', icon: UserCog },
      { href: '/hr/payroll', label: 'Payroll', icon: Banknote },
      { href: '/hr/leave', label: 'Leave', icon: CalendarClock },
      { href: '/hr/attendance', label: 'Attendance', icon: CalendarClock },
      { href: '/hr/appraisals', label: 'Appraisals', icon: ClipboardCheck },
    ],
  },
  {
    label: 'Procurement',
    icon: ShoppingCart,
    children: [
      { href: '/procurement', label: 'Overview', icon: ShoppingCart },
      { href: '/procurement/suppliers', label: 'Suppliers', icon: Truck },
      { href: '/procurement/requisitions', label: 'Requisitions', icon: ClipboardList },
      { href: '/procurement/orders', label: 'Purchase Orders', icon: PackageCheck },
      { href: '/procurement/receipts', label: 'Goods Receipts', icon: PackageCheck },
      { href: '/procurement/payments', label: 'Supplier Payments', icon: CreditCard },
    ],
  },
  {
    label: 'Digital Intelligence',
    icon: Brain,
    children: [
      { href: '/intelligence', label: 'Overview', icon: Brain },
      { href: '/intelligence/financial', label: 'Financial Analytics', icon: TrendingUp },
      { href: '/intelligence/communications', label: 'Communication Analytics', icon: MessageSquareText },
      { href: '/intelligence/ai', label: 'AI Assistant', icon: Brain },
    ],
  },
  {
    label: 'Digital Security',
    icon: ShieldCheck,
    children: [
      { href: '/security', label: 'Overview', icon: ShieldCheck },
      { href: '/security/roles', label: 'Role-Based Access', icon: KeyRound },
      { href: '/security/audit', label: 'Audit Log', icon: ClipboardList },
      { href: '/security/consent', label: 'Parent Consent', icon: UserCheck },
      { href: '/security/data-protection', label: 'Data Protection', icon: FileBarChart },
    ],
  },
  { href: '/settings', label: 'Settings', icon: Settings },
];

// Nav restriction map — mirrors api/internal/middleware role groups in main.go.
// super_admin sees everything; listed roles see only their modules.
const ROLE_NAV: Record<string, string[]> = {
  teacher: ['/dashboard', '/school/setup', '/learners', '/academic'],
  bursar: ['/dashboard', '/school/setup', '/learners', '/finance', '/reports', '/analytics'],
  hr: ['/dashboard', '/school/setup', '/learners', '/hr'],
  transport_manager: ['/dashboard', '/school/setup', '/learners', '/vehicles', '/routes', '/trips'],
  // principal: undefined -> sees all modules
};

export default function Sidebar({
  mobileOpen = false,
  onClose,
}: {
  mobileOpen?: boolean;
  onClose?: () => void;
} = {}) {
  // Inside the workspace the address starts with the context (/w/{group}/{school});
  // navPath strips it so the items below light up the same way everywhere.
  const pathname = navPath(usePathname());
  const { staff, logoutStaff } = useAuth();
  const enabledModules = useSchoolModules();
  // Track which group is expanded (only one at a time)
  const [expanded, setExpanded] = useState<string | null>(null);
  // Collapse-all toggle hides sub-menu group labels
  const [collapsed, setCollapsed] = useState(false);
  // Hover flyout for collapsed groups
  const [hovered, setHovered] = useState<string | null>(null);

  // Mobile drawer UX: close on navigation and on Escape. The drawer is
  // hidden off-canvas below md and slides in over a backdrop rendered by
  // the layout.
  useEffect(() => {
    onClose?.();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pathname]);

  useEffect(() => {
    if (!mobileOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose?.();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [mobileOpen, onClose]);

  const isGroupActive = (children?: ChildItem[]) =>
    children?.some(
      (child) => pathname === child.href || pathname.startsWith(`${child.href}/`)
    ) ?? false;

  // Open the section that owns the current page. Without this the active page
  // is invisible on a fresh load or a direct link: the group header is tinted
  // but its children — including the one you are on — stay collapsed. Keyed on
  // the pathname only, so manually collapsing a section while staying on the
  // same page still sticks.
  useEffect(() => {
    const activeGroup = navItems.find((item) => isGroupActive(item.children));
    if (activeGroup) setExpanded(activeGroup.label);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pathname]);

  // Module-level role filter: a top-level item (or group) is visible if the
  // signed-in role can access it. super_admin / principal see everything.
  const allowedTop = ROLE_NAV[staff?.role ?? ''];
  const canSee = (topLevelHref: string) => {
    if (!allowedTop) return true;
    return allowedTop.some(
      (p) => topLevelHref === p || topLevelHref.startsWith(p + '/')
    );
  };

  const visibleNavItems = navItems.filter((item) => {
    // A module the school has not bought is not offered.
    // Report cards and analytics are part of Academic.
    const moduleId = item.label === 'Reports & Analytics' ? 'academic' : item.label.toLowerCase();
    if (!moduleEnabled(moduleId, enabledModules)) return false;
    if (!item.href) return true; // groups are filtered via children below
    return canSee(item.href);
  });

  return (
    <aside
      id="sidebar"
      aria-label="Main sidebar"
      className={`fixed left-0 top-0 h-screen w-64 bg-gray-900 text-white flex flex-col z-40 transform transition-transform duration-200 ease-out -translate-x-full md:translate-x-0 ${
        mobileOpen ? 'translate-x-0' : ''
      }`}
    >
      <div className="p-6 border-b border-gray-800 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold">Shule360</h1>
          <p className="text-sm text-gray-400">School Management</p>
        </div>
        <button
          onClick={() => setCollapsed((c) => !c)}
          className="p-1.5 rounded-md text-gray-400 hover:text-white hover:bg-gray-800 transition-colors"
          title={collapsed ? 'Expand menus' : 'Collapse menus'}
          aria-label={collapsed ? 'Expand menus' : 'Collapse menus'}
        >
          <PanelLeftClose size={16} aria-hidden="true" />
        </button>
      </div>

      <nav className="flex-1 overflow-y-auto p-4 space-y-1" aria-label="Main navigation">
        {visibleNavItems.map((item) => {
          const Icon = item.icon;

          // Leaf item (no children)
          if (!item.children) {
            const isActive = pathname === item.href;
            return (
              <Link
                key={item.href}
                href={item.href!}
                className={`flex items-center gap-2 px-3 py-2 rounded-md text-sm transition-colors ${
                  isActive ? 'bg-blue-600 text-white' : 'text-gray-300 hover:bg-gray-800'
                }`}
              >
                <Icon size={18} />
                <span>{item.label}</span>
              </Link>
            );
          }

          // Group item with collapsible children
          const isOpen = expanded === item.label;
          const forceOpen = collapsed ? false : isOpen;

          return (
            <div
              key={item.label}
              className="relative"
              onMouseEnter={() => setHovered(item.label)}
              onMouseLeave={() => setHovered(null)}
            >
              <button
                onClick={() => setExpanded(isOpen ? null : item.label)}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-md text-sm transition-colors ${
                  isGroupActive(item.children)
                    ? 'bg-blue-600/20 text-white'
                    : 'text-gray-400 hover:bg-gray-800 hover:text-gray-200'
                }`}
              >
                <span className="flex items-center gap-2">
                  <Icon size={18} />
                  <span>{item.label}</span>
                </span>
                <ChevronDown
                  size={14}
                  className={`transition-transform ${forceOpen ? 'rotate-180' : ''}`}
                />
              </button>

              {forceOpen && (
                <div className="ml-4 mt-1 space-y-1 pb-2 border-l border-gray-700 pl-3">
                  {item.children.filter((c) => canSee(c.href)).map((child) => {
                    const ChildIcon = child.icon;
                    const isChildActive = pathname === child.href;
                    return (
                      <Link
                        key={child.href}
                        href={child.href}
                        className={`flex items-center gap-2 px-3 py-1.5 rounded-md text-sm transition-colors ${
                          isChildActive
                            ? 'bg-blue-600 text-white'
                            : 'text-gray-300 hover:bg-gray-800'
                        }`}
                      >
                        <ChildIcon size={15} />
                        <span>{child.label}</span>
                      </Link>
                    );
                  })}
                </div>
              )}

              {/* Hover flyout when collapsed */}
              {collapsed && hovered === item.label && (
                <div className="absolute left-full top-0 ml-2 w-56 bg-gray-800 rounded-md shadow-lg py-2 border border-gray-700 animate-fade-in">
                  <div className="px-3 py-1.5 text-xs font-semibold text-gray-400 uppercase">
                    {item.label}
                  </div>
                  {item.children.map((child) => {
                    const ChildIcon = child.icon;
                    const isChildActive = pathname === child.href;
                    return (
                      <Link
                        key={child.href}
                        href={child.href}
                        className={`flex items-center gap-2 px-3 py-2 text-sm transition-colors ${
                          isChildActive
                            ? 'bg-blue-600 text-white'
                            : 'text-gray-200 hover:bg-gray-700'
                        }`}
                      >
                        <ChildIcon size={15} />
                        <span>{child.label}</span>
                      </Link>
                    );
                  })}
                </div>
              )}
            </div>
          );
        })}
      </nav>

      <div className="p-4 border-t border-gray-800">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-full bg-blue-600 flex items-center justify-center text-sm font-bold flex-shrink-0">
            {staff?.full_name
              ? staff.full_name
                  .split(' ')
                  .map((p) => p[0])
                  .slice(0, 2)
                  .join('')
                  .toUpperCase()
              : '?'}
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium truncate">{staff?.full_name || 'Signed in'}</p>
            <p className="text-xs text-gray-400 capitalize">{staff?.role || ''}</p>
          </div>
          <button
            onClick={() => logoutStaff('/auth/login')}
            className="p-1.5 rounded-md text-gray-400 hover:text-white hover:bg-gray-800 transition-colors"
            title="Sign out"
            aria-label="Sign out"
          >
            <LogOut size={16} />
          </button>
        </div>
      </div>
    </aside>
  );
}