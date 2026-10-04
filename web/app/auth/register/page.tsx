'use client';

import Link from 'next/link';
import { useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { API_BASE } from '@/lib/api';
import { useAuth } from '@/lib/auth';

// A person registers their own school here and becomes its principal. Three
// short steps, one request: nothing is created until the last step is sent,
// and then the school, its settings and the account are created together.

interface Options {
  open: boolean;
  needs_code: boolean;
  counties: string[];
}

type Values = Record<string, string>;

const STEPS = ['The school', 'You', 'This term'] as const;

/** The fields of each step, so an error from the API can open the step it belongs to. */
const STEP_FIELDS: string[][] = [
  ['school_name', 'ownership', 'county', 'school_phone', 'school_email', 'address'],
  ['admin_name', 'admin_email', 'admin_phone', 'password', 'confirm'],
  ['term', 'year', 'code'],
];

const THIS_YEAR = new Date().getFullYear();

function Field({ id, label, error, hint, children }: { id: string; label: string; error?: string; hint?: string; children: React.ReactNode }) {
  return (
    <div>
      <label htmlFor={id} className="label">{label}</label>
      {children}
      {hint && !error && <p id={`${id}-hint`} className="mt-1 text-xs text-gray-600">{hint}</p>}
      {error && <p id={`${id}-error`} role="alert" className="mt-1 text-sm text-red-700">{error}</p>}
    </div>
  );
}

export default function RegisterPage() {
  const router = useRouter();
  const { loginStaff } = useAuth();
  const [options, setOptions] = useState<Options | null>(null);
  const [unreachable, setUnreachable] = useState(false);
  const [step, setStep] = useState(0);
  const [values, setValues] = useState<Values>({ term: '1', year: String(THIS_YEAR) });
  const [errors, setErrors] = useState<Values>({});
  const [failure, setFailure] = useState('');
  const [busy, setBusy] = useState(false);
  const heading = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    fetch(`${API_BASE}/signup`)
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error('unavailable'))))
      .then((data: Options) => setOptions(data))
      .catch(() => setUnreachable(true));
  }, []);

  // Moving between steps puts the reader at the top of the new one.
  useEffect(() => {
    heading.current?.focus();
  }, [step]);

  const set = (name: string) => (event: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => {
    setValues((current) => ({ ...current, [name]: event.target.value }));
    setErrors((current) => ({ ...current, [name]: '' }));
  };

  const input = (name: string, extra: React.InputHTMLAttributes<HTMLInputElement> = {}) => ({
    id: name,
    className: 'input',
    value: values[name] ?? '',
    onChange: set(name),
    'aria-invalid': errors[name] ? true : undefined,
    'aria-describedby': errors[name] ? `${name}-error` : `${name}-hint`,
    ...extra,
  });

  /** What can be checked without asking the server. The server checks everything again. */
  function check(at: number): Values {
    const found: Values = {};
    const need = (name: string, message: string) => {
      if (!(values[name] ?? '').trim()) found[name] = message;
    };
    if (at === 0) {
      need('school_name', 'Enter the school’s full name.');
      need('ownership', 'Say whether the school is public or private.');
      need('county', 'Choose the county the school is in.');
    }
    if (at === 1) {
      need('admin_name', 'Enter your full name.');
      need('admin_email', 'Enter the email you will sign in with.');
      if ((values.password ?? '').length < 10) found.password = 'Choose a password of at least 10 characters.';
      else if (values.password !== values.confirm) found.confirm = 'The two passwords are not the same.';
    }
    if (at === 2 && options?.needs_code) need('code', 'Enter the registration code you were given.');
    return found;
  }

  function next() {
    const found = check(step);
    setErrors(found);
    if (Object.keys(found).length === 0) setStep(step + 1);
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (step < STEPS.length - 1) return next();
    const found = check(step);
    setErrors(found);
    if (Object.keys(found).length > 0) return;

    setBusy(true);
    setFailure('');
    try {
      const res = await fetch(`${API_BASE}/signup`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          school_name: values.school_name,
          county: values.county,
          ownership: values.ownership,
          school_phone: values.school_phone ?? '',
          school_email: values.school_email ?? '',
          address: values.address ?? '',
          admin_name: values.admin_name,
          admin_email: values.admin_email,
          admin_phone: values.admin_phone ?? '',
          password: values.password,
          term: Number(values.term),
          year: Number(values.year),
          code: values.code ?? '',
        }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        const field: string = data.field ?? '';
        const at = STEP_FIELDS.findIndex((fields) => fields.includes(field));
        if (field && at >= 0) {
          setErrors({ [field]: data.error });
          setStep(at);
        } else {
          setFailure(data.error || 'The school could not be registered. Try again.');
        }
        return;
      }
      // The school exists: sign in to it and show what to set up first.
      try {
        await loginStaff(values.admin_email, values.password);
        router.push('/school/setup');
      } catch {
        router.push('/auth/login');
      }
    } catch {
      setFailure('Could not reach Shule360. Check your connection and try again.');
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 px-4 py-10">
      <main className="mx-auto w-full max-w-xl">
        <div className="mb-6 text-center">
          <p className="text-3xl font-bold text-gray-900">Shule360</p>
          <h1 className="mt-2 text-lg font-semibold text-gray-900">Register your school</h1>
        </div>

        {unreachable && (
          <div role="alert" className="card p-6 text-sm text-red-800">
            Could not reach Shule360. Check your connection and reload this page.
          </div>
        )}
        {!options && !unreachable && <p className="text-center text-sm text-gray-600" role="status">Loading…</p>}

        {options && !options.open && (
          <div className="card space-y-3 p-6 text-sm">
            <p className="font-semibold text-gray-900">Schools are set up by the Shule360 team.</p>
            <p className="text-gray-700">Contact us and we will create your school and send your sign-in details.</p>
            <Link href="/auth/login" className="font-semibold text-blue-700 underline">I already have an account</Link>
          </div>
        )}

        {options?.open && (
          <form onSubmit={submit} noValidate className="card space-y-5 p-6 sm:p-8">
            <ol className="flex gap-2 text-xs font-semibold" aria-label="Steps">
              {STEPS.map((label, index) => (
                <li
                  key={label}
                  aria-current={index === step ? 'step' : undefined}
                  className={`flex-1 rounded-full px-2 py-1.5 text-center ${index === step ? 'bg-blue-700 text-white' : index < step ? 'bg-emerald-100 text-emerald-900' : 'bg-gray-100 text-gray-600'}`}
                >
                  {index + 1}. {label}
                </li>
              ))}
            </ol>

            <h2 ref={heading} tabIndex={-1} className="text-base font-semibold text-gray-900 outline-none">
              {step === 0 && 'About the school'}
              {step === 1 && 'About you, the school’s administrator'}
              {step === 2 && 'Where the school is in the year'}
            </h2>

            {failure && <div role="alert" className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{failure}</div>}

            {step === 0 && (
              <div className="space-y-4">
                <Field id="school_name" label="School name" error={errors.school_name}>
                  <input {...input('school_name', { placeholder: 'Hilltop Primary School', autoComplete: 'organization', autoFocus: true })} />
                </Field>
                <fieldset aria-describedby={errors.ownership ? 'ownership-error' : 'ownership-hint'}>
                  <legend className="label">Kind of school</legend>
                  <div className="mt-1 grid grid-cols-1 gap-2 sm:grid-cols-2">
                    {[
                      { value: 'public', title: 'Public', note: 'A government school. No tuition; levies such as lunch and activities.' },
                      { value: 'private', title: 'Private', note: 'Charges tuition and its own fees.' },
                    ].map((kind) => (
                      <label
                        key={kind.value}
                        className={`flex cursor-pointer items-start gap-2 rounded-md border p-3 text-sm ${values.ownership === kind.value ? 'border-blue-700 bg-blue-50' : 'border-gray-300 bg-white'}`}
                      >
                        <input type="radio" name="ownership" value={kind.value} checked={values.ownership === kind.value} onChange={set('ownership')} className="mt-0.5 h-4 w-4" />
                        <span>
                          <span className="block font-semibold text-gray-900">{kind.title}</span>
                          <span className="block text-gray-700">{kind.note}</span>
                        </span>
                      </label>
                    ))}
                  </div>
                  {errors.ownership
                    ? <p id="ownership-error" role="alert" className="mt-1 text-sm text-red-700">{errors.ownership}</p>
                    : <p id="ownership-hint" className="mt-1 text-xs text-gray-600">This sets the fee items the school starts with. You can change them all afterwards.</p>}
                </fieldset>
                <Field id="county" label="County" error={errors.county}>
                  <select id="county" className="input" value={values.county ?? ''} onChange={set('county')} aria-invalid={errors.county ? true : undefined}>
                    <option value="">Choose…</option>
                    {options.counties.map((county) => <option key={county} value={county}>{county}</option>)}
                  </select>
                </Field>
                <Field id="school_phone" label="School phone (optional)" error={errors.school_phone} hint="Shown on receipts and report cards.">
                  <input {...input('school_phone', { placeholder: '0712 345 678', inputMode: 'tel', autoComplete: 'tel' })} />
                </Field>
                <Field id="school_email" label="School email (optional)" error={errors.school_email}>
                  <input {...input('school_email', { type: 'email', placeholder: 'office@school.ac.ke' })} />
                </Field>
                <Field id="address" label="Postal or physical address (optional)" error={errors.address}>
                  <input {...input('address', { placeholder: 'P.O. Box 123, Nakuru', autoComplete: 'street-address' })} />
                </Field>
              </div>
            )}

            {step === 1 && (
              <div className="space-y-4">
                <p className="text-sm text-gray-700">You become the school’s principal in Shule360: you can do everything, and you add the bursar, teachers and other staff afterwards.</p>
                <Field id="admin_name" label="Your full name" error={errors.admin_name}>
                  <input {...input('admin_name', { autoComplete: 'name', autoFocus: true })} />
                </Field>
                <Field id="admin_email" label="Your email" error={errors.admin_email} hint="You will sign in with this.">
                  <input {...input('admin_email', { type: 'email', autoComplete: 'email', placeholder: 'you@school.ac.ke' })} />
                </Field>
                <Field id="admin_phone" label="Your phone (optional)" error={errors.admin_phone}>
                  <input {...input('admin_phone', { placeholder: '0712 345 678', inputMode: 'tel', autoComplete: 'tel' })} />
                </Field>
                <Field id="password" label="Choose a password" error={errors.password} hint="At least 10 characters.">
                  <input {...input('password', { type: 'password', autoComplete: 'new-password' })} />
                </Field>
                <Field id="confirm" label="Type the password again" error={errors.confirm}>
                  <input {...input('confirm', { type: 'password', autoComplete: 'new-password' })} />
                </Field>
              </div>
            )}

            {step === 2 && (
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <Field id="term" label="Current term" error={errors.term}>
                    <select id="term" className="input" value={values.term} onChange={set('term')}>
                      <option value="1">Term 1</option>
                      <option value="2">Term 2</option>
                      <option value="3">Term 3</option>
                    </select>
                  </Field>
                  <Field id="year" label="Academic year" error={errors.year}>
                    <select id="year" className="input" value={values.year} onChange={set('year')}>
                      {[THIS_YEAR - 1, THIS_YEAR, THIS_YEAR + 1].map((year) => <option key={year} value={year}>{year}</option>)}
                    </select>
                  </Field>
                </div>
                {options.needs_code && (
                  <Field id="code" label="Registration code" error={errors.code} hint="Given to you by the Shule360 team.">
                    <input {...input('code', { autoComplete: 'off' })} />
                  </Field>
                )}
                <dl className="rounded-md bg-gray-50 p-4 text-sm">
                  <dt className="font-semibold text-gray-900">You are about to create</dt>
                  <dd className="mt-1 text-gray-700">{values.school_name}, a {values.ownership} school in {values.county}</dd>
                  <dd className="text-gray-700">with {values.admin_name} ({values.admin_email}) as its principal.</dd>
                </dl>
              </div>
            )}

            <div className="flex items-center justify-between gap-3 pt-2">
              {step > 0 ? (
                <button type="button" className="btn-secondary" onClick={() => setStep(step - 1)} disabled={busy}>Back</button>
              ) : (
                <Link href="/auth/login" className="text-sm font-semibold text-blue-700 underline">I already have an account</Link>
              )}
              <button type="submit" className="btn-primary" disabled={busy}>
                {busy ? 'Creating the school…' : step < STEPS.length - 1 ? 'Continue' : 'Create the school'}
              </button>
            </div>
          </form>
        )}
      </main>
    </div>
  );
}
