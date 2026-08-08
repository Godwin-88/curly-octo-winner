'use client';

import { useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
  LayoutDashboard,
  MessageSquare,
  MessageCircle,
  Inbox,
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
    label: 'Communications',
    icon: MessageSquare,
    children: [
      { href: '/communications', label: 'Overview', icon: MessageSquare },
      { href: '/communications/sms', label: 'SMS Campaigns', icon: MessageSquare },
      { href: '/communications/whatsapp', label: 'WhatsApp', icon: MessageCircle },
      { href: '/communications/inbox', label: 'Inbox', icon: Inbox },
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
  { href: '/learners', label: 'Learners', icon: Users },
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
      { href: '/finance/fees', label: 'Fee Structures', icon: FileText },
      { href: '/finance/invoices', label: 'Invoices', icon: Receipt },
      { href: '/finance/payments', label: 'Payments', icon: Smartphone },
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

export default function Sidebar() {
  const pathname = usePathname();
  // Track which group is expanded (only one at a time)
  const [expanded, setExpanded] = useState<string | null>(null);
  // Collapse-all toggle hides sub-menu group labels
  const [collapsed, setCollapsed] = useState(false);
  // Hover flyout for collapsed groups
  const [hovered, setHovered] = useState<string | null>(null);

  const isGroupActive = (children?: ChildItem[]) =>
    children?.some((child) => pathname === child.href) ?? false;

  return (
    <aside className="fixed left-0 top-0 h-screen w-64 bg-gray-900 text-white flex flex-col z-40">
      <div className="p-6 border-b border-gray-800 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold">Shule360</h1>
          <p className="text-sm text-gray-400">School Management</p>
        </div>
        <button
          onClick={() => setCollapsed((c) => !c)}
          className="p-1.5 rounded-md text-gray-400 hover:text-white hover:bg-gray-800 transition-colors"
          title={collapsed ? 'Expand menus' : 'Collapse menus'}
        >
          <PanelLeftClose size={16} />
        </button>
      </div>

      <nav className="flex-1 overflow-y-auto p-4 space-y-1">
        {navItems.map((item) => {
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
                  {item.children.map((child) => {
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
          <div className="w-8 h-8 rounded-full bg-blue-600 flex items-center justify-center text-sm font-bold">
            JK
          </div>
          <div>
            <p className="text-sm font-medium">John Kamau</p>
            <p className="text-xs text-gray-400">Principal</p>
          </div>
        </div>
      </div>
    </aside>
  );
}