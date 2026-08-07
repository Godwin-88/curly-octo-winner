'use client';

import { useEffect, useState, useRef } from 'react';
import {
  ArrowRight,
  BookOpen,
  Users,
  BarChart3,
  Shield,
  Bell,
  MapPin,
  Wallet,
  FileText,
  Bus,
  CreditCard,
  MessageSquare,
  Brain,
  Briefcase,
  ShoppingCart,
  Lock,
  Phone,
  Mail,
  Facebook,
  Twitter,
  Linkedin,
} from 'lucide-react';

function AnimatedSection({ children, className = '' }: { children: React.ReactNode; className?: string }) {
  const [visible, setVisible] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => { if (entry.isIntersecting) setVisible(true); },
      { threshold: 0.1 }
    );
    if (ref.current) observer.observe(ref.current);
    return () => { if (ref.current) observer.unobserve(ref.current); };
  }, []);

  return (
    <div
      ref={ref}
      className={`transition-all duration-700 ease-out ${visible ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-8'} ${className}`}
    >
      {children}
    </div>
  );
}

export default function Home() {
  const modules = [
    {
      icon: <Bell className="w-8 h-8" />,
      title: 'Communications Hub',
      desc: 'Bulk SMS via Africa\'s Talking, WhatsApp Business messaging, and a dedicated inbox — reach every parent instantly with school-branded messages and track delivery.',
      href: '/communications',
      tags: ['Bulk SMS', 'WhatsApp Business', 'Inbox', 'Delivery Reports'],
    },
    {
      icon: <BookOpen className="w-8 h-8" />,
      title: 'CBC Academic Management',
      desc: 'KICD-aligned curriculum tracking, formative assessments with rubric levels, attendance, and automated report card generation for every learner.',
      href: '/academic',
      tags: ['Curriculum', 'Assessments', 'Attendance', 'Report Cards'],
    },
    {
      icon: <Users className="w-8 h-8" />,
      title: 'Learner Records',
      desc: 'Complete enrollment, progression through grades, document management, guardian profiles, and transfer handling — a single source of truth per learner.',
      href: '/learners',
      tags: ['Enrollment', 'Progression', 'Documents', 'Transfers'],
    },
    {
      icon: <Bus className="w-8 h-8" />,
      title: 'Transport Tracking',
      desc: 'Fleet and vehicle management, route and stop planning, live trip tracking, and boarding check-ins keep parents informed in real time.',
      href: '/trips',
      tags: ['Vehicles', 'Routes', 'Trips', 'Live Tracking'],
    },
    {
      icon: <Wallet className="w-8 h-8" />,
      title: 'Finance & Fees',
      desc: 'Fee structures and billing, invoicing with discounts, M-Pesa STK push collection and receipts, and a full payment ledger with reconciliation.',
      href: '/finance',
      tags: ['Fee Structures', 'Invoices', 'M-Pesa Payments', 'Ledger'],
    },
    {
      icon: <Briefcase className="w-8 h-8" />,
      title: 'Human Resources',
      desc: 'Staff profiles with TSC and KRA records, payroll runs with statutory deductions, leave management, attendance, and TSC-compliant appraisals.',
      href: '/hr',
      tags: ['Payroll', 'Leave', 'Attendance', 'Appraisals'],
    },
    {
      icon: <ShoppingCart className="w-8 h-8" />,
      title: 'Procurement',
      desc: 'Supplier directory, purchase requisitions, purchase orders, goods receipts, and supplier payments with three-way matching.',
      href: '/procurement',
      tags: ['Suppliers', 'Requisitions', 'Purchase Orders', 'Payments'],
    },
    {
      icon: <FileText className="w-8 h-8" />,
      title: 'Reporting & Analytics',
      desc: 'Strand and learning-area performance views, competency distributions, fee collection summaries, attendance rates, and at-risk learner dashboards.',
      href: '/analytics',
      tags: ['Performance', 'Fee Collection', 'Attendance', 'At-Risk'],
    },
    {
      icon: <Brain className="w-8 h-8" />,
      title: 'Digital Intelligence',
      desc: 'An FAQ knowledge base powering a WhatsApp auto-response chatbot, message template suggestions, and AI-assisted communication drafting.',
      href: '/intelligence',
      tags: ['Chatbot', 'FAQ Knowledge Base', 'Auto-Replies'],
    },
    {
      icon: <Shield className="w-8 h-8" />,
      title: 'Digital Security & Compliance',
      desc: 'Role-based access control, refresh-token session management, audit logging, guardian consent tracking, and KDPA-compliant data processing registers.',
      href: '/security',
      tags: ['RBAC', 'Audit Log', 'Consent', 'KDPA Register'],
    },
    {
      icon: <CreditCard className="w-8 h-8" />,
      title: 'Digital Payments',
      desc: 'M-Pesa Daraja integration with STK push and callback handling, plus supplier payments — receipts and reconciliations handled automatically.',
      href: '/finance/payments',
      tags: ['STK Push', 'Callbacks', 'Receipts'],
    },
    {
      icon: <MessageSquare className="w-8 h-8" />,
      title: 'Parent Engagement',
      desc: 'Two-way WhatsApp conversations, opt-in consent management, and reminders that keep guardians connected to their child\'s school life.',
      href: '/communications/inbox',
      tags: ['Two-Way Chat', 'Opt-In Consent', 'Reminders'],
    },
  ];

  return (
    <main className="min-h-screen font-sans">
      {/* Hero */}
      <section className="relative overflow-hidden bg-gradient-to-br from-blue-900 via-blue-800 to-indigo-900">
        <div className="absolute inset-0 opacity-10">
          <div className="absolute top-10 left-10 w-64 h-64 bg-white rounded-full blur-3xl animate-pulse" />
          <div className="absolute bottom-10 right-10 w-96 h-96 bg-blue-400 rounded-full blur-3xl animate-pulse" style={{ animationDelay: '1s' }} />
        </div>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-24 md:py-32">
          <div className="grid md:grid-cols-2 gap-12 items-center">
            <AnimatedSection>
              <div className="text-white">
                <h1 className="text-4xl md:text-6xl font-bold leading-tight tracking-tight">
                  School Management
                  <span className="block text-blue-300">for the CBC Era</span>
                </h1>
                <p className="mt-6 text-lg md:text-xl text-blue-100 max-w-lg">
                  Shule360 digitizes the full school lifecycle — communications, learner records, academics, transport, and finance — purpose-built for Kenyan schools.
                </p>
                <div className="mt-8 flex flex-col sm:flex-row gap-4">
                  <a href="/auth/login" className="inline-flex items-center justify-center px-6 py-3 bg-white text-blue-900 font-semibold rounded-lg hover:bg-blue-50 transition-colors shadow-lg">
                    Get Started Free
                    <ArrowRight className="ml-2 w-5 h-5" />
                  </a>
                  <a href="#modules" className="inline-flex items-center justify-center px-6 py-3 border-2 border-blue-300 text-white font-semibold rounded-lg hover:bg-blue-700 transition-colors">
                    Explore the Modules
                  </a>
                </div>
                <div className="mt-10 flex flex-wrap gap-2">
                  {['Communications', 'Academics', 'Transport', 'Finance', 'HR', 'Security'].map((tag) => (
                    <span key={tag} className="px-3 py-1 bg-white/10 border border-white/20 rounded-full text-sm text-blue-100">
                      {tag}
                    </span>
                  ))}
                </div>
              </div>
            </AnimatedSection>
            <AnimatedSection>
              <div className="bg-white/10 backdrop-blur-lg rounded-2xl p-8 border border-white/20">
                <h3 className="text-xl font-semibold text-white mb-6">One platform, every core module</h3>
                <ul className="space-y-4">
                  {[
                    { name: 'Bulk SMS & WhatsApp', icon: <MessageSquare className="w-5 h-5" /> },
                    { name: 'CBC Curriculum & Assessments', icon: <BookOpen className="w-5 h-5" /> },
                    { name: 'Learner Records & Progression', icon: <Users className="w-5 h-5" /> },
                    { name: 'Transport & Live Trips', icon: <Bus className="w-5 h-5" /> },
                    { name: 'Fees & M-Pesa Collection', icon: <Wallet className="w-5 h-5" /> },
                    { name: 'Payroll, Leave & Appraisals', icon: <Briefcase className="w-5 h-5" /> },
                    { name: 'Procurement & Supplier Payments', icon: <ShoppingCart className="w-5 h-5" /> },
                    { name: 'Analytics, Chatbot & Security', icon: <Shield className="w-5 h-5" /> },
                  ].map((item, i) => (
                    <li key={i} className="flex items-center gap-3 text-blue-100">
                      <span className="w-8 h-8 bg-white/10 rounded-lg flex items-center justify-center shrink-0">{item.icon}</span>
                      <span className="font-medium">{item.name}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </AnimatedSection>
          </div>
        </div>
        <div className="absolute bottom-0 left-0 right-0 h-16 bg-gradient-to-t from-gray-50 to-transparent" />
      </section>

      {/* Modules */}
      <section id="modules" className="py-20 md:py-28 bg-white">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <AnimatedSection>
            <div className="text-center max-w-3xl mx-auto">
              <h2 className="text-3xl md:text-4xl font-bold text-gray-900">Built for Every Part of Your School</h2>
              <p className="mt-4 text-lg text-gray-600">Twelve integrated modules covering academics, operations, and finance — all CBC-aligned and purpose-built for Kenyan schools.</p>
            </div>
          </AnimatedSection>
          <div className="mt-16 grid sm:grid-cols-2 lg:grid-cols-3 gap-8">
            {modules.map((module, i) => (
              <AnimatedSection key={i}>
                <div className="group h-full flex flex-col bg-gray-50 rounded-xl p-6 border border-gray-100 hover:shadow-lg hover:border-blue-200 transition-all duration-300">
                  <div className="w-12 h-12 bg-blue-100 text-blue-600 rounded-lg flex items-center justify-center group-hover:bg-blue-600 group-hover:text-white transition-colors duration-300">
                    {module.icon}
                  </div>
                  <h3 className="mt-4 text-lg font-semibold text-gray-900">{module.title}</h3>
                  <p className="mt-2 text-gray-600 leading-relaxed flex-1">{module.desc}</p>
                  <div className="mt-4 flex flex-wrap gap-2">
                    {module.tags.map((tag) => (
                      <span key={tag} className="px-2 py-0.5 bg-blue-50 text-blue-700 text-xs font-medium rounded-full">
                        {tag}
                      </span>
                    ))}
                  </div>
                </div>
              </AnimatedSection>
            ))}
          </div>
        </div>
      </section>

      {/* How It Works */}
      <section id="how-it-works" className="py-20 md:py-28 bg-gray-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <AnimatedSection>
            <div className="text-center max-w-3xl mx-auto">
              <h2 className="text-3xl md:text-4xl font-bold text-gray-900">How It Works</h2>
              <p className="mt-4 text-lg text-gray-600">Get your school onboard in four simple steps.</p>
            </div>
          </AnimatedSection>
          <div className="mt-16 grid md:grid-cols-4 gap-8 relative">
            <div className="hidden md:block absolute top-12 left-1/6 right-1/6 h-0.5 bg-blue-200" />
            {[
              { num: '01', title: 'Set Up Your School', desc: 'Create your school profile, configure your tenant, and invite staff — takes less than 5 minutes.' },
              { num: '02', title: 'Add Learners & Staff', desc: 'Import or manually enter learner records, guardian profiles, staff details, and class structures.' },
              { num: '03', title: 'Start Managing', desc: 'Send communications, track assessments, manage transport, collect fees, and run payroll — all in one place.' },
              { num: '04', title: 'Grow with Insights', desc: 'Access analytics, CBC report cards, delivery reports, and compliance dashboards.' },
            ].map((step, i) => (
              <AnimatedSection key={i}>
                <div className="relative text-center">
                  <div className="w-24 h-24 bg-blue-600 text-white rounded-full flex items-center justify-center text-2xl font-bold mx-auto shadow-lg shadow-blue-200 relative z-10">
                    {step.num}
                  </div>
                  <h3 className="mt-6 text-lg font-semibold text-gray-900">{step.title}</h3>
                  <p className="mt-2 text-gray-600 leading-relaxed">{step.desc}</p>
                </div>
              </AnimatedSection>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="py-20 md:py-28 bg-gradient-to-r from-blue-600 to-indigo-700">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
          <AnimatedSection>
            <h2 className="text-3xl md:text-4xl font-bold text-white">Ready to Transform Your School?</h2>
            <p className="mt-4 text-lg text-blue-100 max-w-2xl mx-auto">Every core module your school runs on — in one secure, CBC-aligned platform.</p>
            <div className="mt-8 flex flex-col sm:flex-row gap-4 justify-center">
              <a href="/auth/login" className="inline-flex items-center justify-center px-8 py-4 bg-white text-blue-900 font-semibold rounded-lg hover:bg-blue-50 transition-colors shadow-lg text-lg">
                Start Free Today
                <ArrowRight className="ml-2 w-5 h-5" />
              </a>
              <a href="mailto:hello@shule360.ke" className="inline-flex items-center justify-center px-8 py-4 border-2 border-white text-white font-semibold rounded-lg hover:bg-white/10 transition-colors text-lg">
                Contact Us
              </a>
            </div>
          </AnimatedSection>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-gray-900 text-gray-400 py-12">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-8">
            <div>
              <div className="flex items-center gap-2 text-white">
                <BookOpen className="w-6 h-6 text-blue-400" />
                <span className="text-xl font-bold">Shule360</span>
              </div>
              <p className="mt-3 text-sm leading-relaxed">CBC-aligned school management for Kenyan schools. Digitize your school lifecycle today.</p>
              <div className="mt-4 flex gap-3">
                <a href="#" className="text-gray-500 hover:text-white transition-colors"><Facebook className="w-5 h-5" /></a>
                <a href="#" className="text-gray-500 hover:text-white transition-colors"><Twitter className="w-5 h-5" /></a>
                <a href="#" className="text-gray-500 hover:text-white transition-colors"><Linkedin className="w-5 h-5" /></a>
              </div>
            </div>
            <div>
              <h4 className="text-white font-semibold mb-3">Platform</h4>
              <ul className="space-y-2 text-sm">
                <li><a href="#modules" className="hover:text-white transition-colors">Modules</a></li>
                <li><a href="#how-it-works" className="hover:text-white transition-colors">How It Works</a></li>
                <li><a href="#modules" className="hover:text-white transition-colors">Security & Compliance</a></li>
              </ul>
            </div>
            <div>
              <h4 className="text-white font-semibold mb-3">Support</h4>
              <ul className="space-y-2 text-sm">
                <li><a href="#modules" className="hover:text-white transition-colors">Help Center</a></li>
                <li><a href="#modules" className="hover:text-white transition-colors">Documentation</a></li>
                <li><a href="#modules" className="hover:text-white transition-colors">API Reference</a></li>
              </ul>
            </div>
            <div>
              <h4 className="text-white font-semibold mb-3">Contact</h4>
              <ul className="space-y-2 text-sm">
                <li className="flex items-center gap-2"><Mail className="w-4 h-4" /> hello@shule360.ke</li>
                <li className="flex items-center gap-2"><Phone className="w-4 h-4" /> +254 700 000 000</li>
                <li className="flex items-center gap-2"><MapPin className="w-4 h-4" /> Nairobi, Kenya</li>
              </ul>
            </div>
          </div>
          <div className="mt-12 pt-8 border-t border-gray-800 text-center text-sm">
            &copy; {new Date().getFullYear()} Shule360. All rights reserved.
          </div>
        </div>
      </footer>
    </main>
  );
}