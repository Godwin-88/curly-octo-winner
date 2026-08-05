'use client';

import Link from 'next/link';
import { BookOpen, ClipboardList, CalendarCheck, TrendingUp } from 'lucide-react';

export default function AcademicPage() {
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Academic Operations</h1>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <Link href="/academic/curriculum" className="card p-6 hover:shadow-md transition-shadow">
          <BookOpen className="w-8 h-8 text-blue-600 mb-3" />
          <h3 className="font-semibold text-lg">Curriculum</h3>
          <p className="text-sm text-gray-500 mt-1">
            KICD strand/sub-strand catalogue and learning outcomes.
          </p>
        </Link>
        <Link href="/academic/assessments" className="card p-6 hover:shadow-md transition-shadow">
          <ClipboardList className="w-8 h-8 text-green-600 mb-3" />
          <h3 className="font-semibold text-lg">Assessments</h3>
          <p className="text-sm text-gray-500 mt-1">
            Formative observations, rubric builder, and report cards.
          </p>
        </Link>
        <Link href="/academic/attendance" className="card p-6 hover:shadow-md transition-shadow">
          <CalendarCheck className="w-8 h-8 text-purple-600 mb-3" />
          <h3 className="font-semibold text-lg">Attendance</h3>
          <p className="text-sm text-gray-500 mt-1">
            Daily roll call, absence tracking, and chronic absenteeism alerts.
          </p>
        </Link>
        <Link href="/academic/learners" className="card p-6 hover:shadow-md transition-shadow">
          <TrendingUp className="w-8 h-8 text-orange-600 mb-3" />
          <h3 className="font-semibold text-lg">Learner Portfolio</h3>
          <p className="text-sm text-gray-500 mt-1">
            CBC competency tracking and progress dashboards.
          </p>
        </Link>
      </div>
    </div>
  );
}