'use client';

import { useEffect, useState, useRef } from 'react';
import { ArrowRight, BookOpen, Users, BarChart3, Shield, Bell, MapPin, Phone, Mail, Facebook, Twitter, Linkedin } from 'lucide-react';

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
  const stats = [
    { icon: <Users className="w-6 h-6" />, value: '150+', label: 'Schools Served' },
    { icon: <BookOpen className="w-6 h-6" />, value: '45K+', label: 'Learner Records' },
    { icon: <BarChart3 className="w-6 h-6" />, value: '98%', label: 'Uptime SLA' },
    { icon: <Shield className="w-6 h-6" />, value: '24/7', label: 'Support' },
  ];

  const features = [
    { icon: <Bell className="w-8 h-8" />, title: 'Communications Hub', desc: 'Bulk SMS, WhatsApp Business, and inbox — reach every parent instantly with school-branded messages.' },
    { icon: <BookOpen className="w-8 h-8" />, title: 'CBC Academic Management', desc: 'Competency-based curriculum tracking, formative assessments, and report card generation aligned to KICD frameworks.' },
    { icon: <Users className="w-8 h-8" />, title: 'Learner Records', desc: 'Enrollment, progression, document management, and transfers — a single source of truth for every learner.' },
    { icon: <MapPin className="w-8 h-8" />, title: 'Transport Tracking', desc: 'Real-time GPS fleet tracking, route management, boarding check-ins, and fee reconciliation.' },
    { icon: <BarChart3 className="w-8 h-8" />, title: 'Finance & Fees', desc: 'M-Pesa STK push fee collection, invoicing, supplier payments, and financial reporting.' },
    { icon: <Shield className="w-8 h-8" />, title: 'Secure & Compliant', desc: 'Multi-tenant architecture with row-level security, GDPR-aligned data handling, and Kenya DPA compliance.' },
  ];

  const steps = [
    { num: '01', title: 'Set Up Your School', desc: 'Create your school profile, configure tenants, and invite staff — takes less than 5 minutes.' },
    { num: '02', title: 'Add Learners & Staff', desc: 'Import or manually enter learner records, staff profiles, and class structures.' },
    { num: '03', title: 'Start Managing', desc: 'Send communications, track assessments, manage transport, and collect fees — all in one place.' },
    { num: '04', title: 'Grow with Insights', desc: 'Access real-time analytics, CBC report cards, and at-risk learner dashboards.' },
  ];

  return (
    <main className="min-h-screen">
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
                  <a href="/(auth)/login" className="inline-flex items-center justify-center px-6 py-3 bg-white text-blue-900 font-semibold rounded-lg hover:bg-blue-50 transition-colors shadow-lg">
                    Get Started Free
                    <ArrowRight className="ml-2 w-5 h-5" />
                  </a>
                  <a href="#how-it-works" className="inline-flex items-center justify-center px-6 py-3 border-2 border-blue-300 text-white font-semibold rounded-lg hover:bg-blue-700 transition-colors">
                    See How It Works
                  </a>
                </div>
              </div>
            </AnimatedSection>
            <AnimatedSection>
              <div className="bg-white/10 backdrop-blur-lg rounded-2xl p-8 border border-white/20">
                <div className="grid grid-cols-2 gap-4">
                  {stats.map((stat, i) => (
                    <div key={i} className="bg-white/5 rounded-xl p-4 text-center border border-white/10">
                      <div className="text-white flex justify-center">{stat.icon}</div>
                      <div className="mt-3 text-3xl font-bold text-white">{stat.value}</div>
                      <div className="text-sm text-blue-200 mt-1">{stat.label}</div>
                    </div>
                  ))}
                </div>
              </div>
            </AnimatedSection>
          </div>
        </div>
        <div className="absolute bottom-0 left-0 right-0 h-16 bg-gradient-to-t from-gray-50 to-transparent" />
      </section>

      {/* Features */}
      <section className="py-20 md:py-28 bg-white">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <AnimatedSection>
            <div className="text-center max-w-3xl mx-auto">
              <h2 className="text-3xl md:text-4xl font-bold text-gray-900">Everything Your School Needs</h2>
              <p className="mt-4 text-lg text-gray-600">A unified platform that covers communications, academics, learner records, transport, finance, and HR — all CBC-aligned.</p>
            </div>
          </AnimatedSection>
          <div className="mt-16 grid sm:grid-cols-2 lg:grid-cols-3 gap-8">
            {features.map((feature, i) => (
              <AnimatedSection key={i}>
                <div className="group bg-gray-50 rounded-xl p-6 border border-gray-100 hover:shadow-lg hover:border-blue-200 transition-all duration-300">
                  <div className="w-12 h-12 bg-blue-100 text-blue-600 rounded-lg flex items-center justify-center group-hover:bg-blue-600 group-hover:text-white transition-colors duration-300">
                    {feature.icon}
                  </div>
                  <h3 className="mt-4 text-lg font-semibold text-gray-900">{feature.title}</h3>
                  <p className="mt-2 text-gray-600 leading-relaxed">{feature.desc}</p>
                </div>
              </AnimatedSection>
            ))}
          </div>
        </div>
      </section>

      {/* Stats */}
      <section className="py-16 bg-gray-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8">
            {stats.map((stat, i) => (
              <AnimatedSection key={i}>
                <div className="text-center">
                  <div className="inline-flex items-center justify-center w-14 h-14 bg-blue-100 text-blue-600 rounded-full mb-4">
                    {stat.icon}
                  </div>
                  <div className="text-4xl font-bold text-gray-900">{stat.value}</div>
                  <div className="mt-2 text-sm font-medium text-gray-600">{stat.label}</div>
                </div>
              </AnimatedSection>
            ))}
          </div>
        </div>
      </section>

      {/* How It Works */}
      <section id="how-it-works" className="py-20 md:py-28 bg-white">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <AnimatedSection>
            <div className="text-center max-w-3xl mx-auto">
              <h2 className="text-3xl md:text-4xl font-bold text-gray-900">How It Works</h2>
              <p className="mt-4 text-lg text-gray-600">Get up and running in four simple steps.</p>
            </div>
          </AnimatedSection>
          <div className="mt-16 grid md:grid-cols-4 gap-8 relative">
            <div className="hidden md:block absolute top-12 left-1/6 right-1/6 h-0.5 bg-blue-200" />
            {steps.map((step, i) => (
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
            <p className="mt-4 text-lg text-blue-100 max-w-2xl mx-auto">Join 150+ Kenyan schools already using Shule360 to streamline operations and improve outcomes.</p>
            <div className="mt-8 flex flex-col sm:flex-row gap-4 justify-center">
              <a href="/(auth)/login" className="inline-flex items-center justify-center px-8 py-4 bg-white text-blue-900 font-semibold rounded-lg hover:bg-blue-50 transition-colors shadow-lg text-lg">
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
                <li><a href="#features" className="hover:text-white transition-colors">Features</a></li>
                <li><a href="#how-it-works" className="hover:text-white transition-colors">How It Works</a></li>
                <li><a href="#" className="hover:text-white transition-colors">Pricing</a></li>
                <li><a href="#" className="hover:text-white transition-colors">Integrations</a></li>
              </ul>
            </div>
            <div>
              <h4 className="text-white font-semibold mb-3">Support</h4>
              <ul className="space-y-2 text-sm">
                <li><a href="#" className="hover:text-white transition-colors">Help Center</a></li>
                <li><a href="#" className="hover:text-white transition-colors">Documentation</a></li>
                <li><a href="#" className="hover:text-white transition-colors">API Reference</a></li>
                <li><a href="#" className="hover:text-white transition-colors">Status</a></li>
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