import type { ComponentType, ReactNode } from 'react';

/**
 * The list → view → edit framework, adapted from centiwise-gateway/web
 * (src/framework). A module describes its records; the shell and
 * ResourceSection do the rest, so every screen behaves the same way.
 */

/** portfolio: no school is chosen (platform or group view). school: one school. */
export type ScopeKind = 'portfolio' | 'school';

/** How far the signed-in user may reach. Decided by the API, mirrored here. */
export interface Session {
  scope: 'school' | 'group' | 'platform';
  name: string;
  role: string;
  /** Fixed for school staff. */
  schoolId?: string;
  /** Fixed for a group user. */
  groupId?: string;
}

/** What every screen is told: who is asking, and the context chosen in the bar. */
export interface Ctx {
  session: Session;
  scope: ScopeKind;
  groupId?: string;
  schoolId?: string;
}

export interface Page<T> {
  items: T[];
  /** The cursor for the next page, when there is one. */
  next?: string;
}

export interface Column<T> {
  header: string;
  cell: (row: T) => ReactNode;
  align?: 'right';
}

export interface FieldSpec<T> {
  label: string;
  value: (row: T) => ReactNode;
}

// Form values are whatever the fields produce: text, numbers, booleans, lists.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type FormValues = Record<string, any>;

export interface Option {
  value: string;
  label: string;
  /** A second line that helps tell two similar options apart. */
  hint?: string;
}

export interface FormField {
  name: string;
  label: string;
  type: 'text' | 'number' | 'select' | 'tags' | 'textarea' | 'datetime' | 'checkbox' | 'picker';
  required?: boolean;
  options?: Option[] | ((ctx: Ctx) => Promise<Option[]>);
  help?: string;
  placeholder?: string;
  showIf?: (values: FormValues) => boolean;
  /** For a picker: finds the records matching what is typed. The value is the chosen ids. */
  search?: (ctx: Ctx, query: string) => Promise<Option[]>;
  /** Pieces of text a click appends to the field, such as {{parent_name}}. */
  inserts?: Option[];
  /** A line under the field computed from what is typed, such as an SMS unit count. */
  meter?: (value: string) => string;
}

/** Something shown once after an action, such as a new password. */
export interface Reveal {
  title: string;
  note: string;
  items: { label: string; value: string }[];
}

export interface ActionDef<T> {
  id: string;
  label: string;
  /** Staff roles that may use it. Empty or absent: everyone who can see the section. */
  roles?: string[];
  when?: (row: T, ctx: Ctx) => boolean;
  tone?: 'primary' | 'danger';
  fields?: FormField[] | ((row: T, ctx: Ctx) => FormField[]);
  initial?: (row: T, ctx: Ctx) => FormValues;
  /** Shown above the submit button. An action with neither fields nor this runs on click. */
  confirm?: string;
  /**
   * Asked before the action runs, with what was typed. Its answer is shown and
   * the user must confirm it: "Sends to 212 parents, 2 units each, about KES 339".
   */
  preview?: (ctx: Ctx, row: T, values: FormValues) => Promise<string>;
  submitLabel?: string;
  run: (ctx: Ctx, row: T, values: FormValues, idempotencyKey: string) => Promise<unknown>;
  /** Something from the result to show once, where the record opens. */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  reveal?: (result: any) => Reveal | undefined;
  /** For a create action: the id of the new record, so the view can open it. */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  createdId?: (result: any) => string | undefined;
  /** The record no longer exists after this action; return to the list. */
  removes?: boolean;
}

interface SectionBase {
  id: string;
  label: string;
  scopes: ScopeKind[];
  roles?: string[];
  /** Kinds of session that may see it. Absent: all of them. */
  sessions?: Session['scope'][];
  /** What this page is for, in a sentence. */
  purpose: string;
}

/** A section whose records follow list, then view, then edit. */
export interface ResourceDef<T> extends SectionBase {
  kind: 'resource';
  /** Singular name of a record, for headings and empty states. */
  noun: string;
  list: (ctx: Ctx, filters: FormValues, cursor?: string) => Promise<Page<T>>;
  rowId: (row: T) => string;
  title: (row: T) => string;
  status?: (row: T) => string | undefined;
  columns: Column<T>[];
  filters?: FormField[];
  defaultFilters?: (ctx: Ctx) => FormValues;
  /** Fetches one record. Without it the view shows the row from the list. */
  get?: (ctx: Ctx, id: string) => Promise<T>;
  /** While true for the open record, it is re-fetched every few seconds. */
  live?: (row: T) => boolean;
  fields: FieldSpec<T>[];
  extra?: ComponentType<{ row: T; ctx: Ctx }>;
  create?: ActionDef<undefined>;
  actions?: ActionDef<T>[];
}

/** A section that is one page: a dashboard, an import, a settings form. */
export interface PageDef extends SectionBase {
  kind: 'page';
  page: ComponentType<{ ctx: Ctx }>;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type SectionDef = ResourceDef<any> | PageDef;

/** What a module tells the shell about itself. No module edits the shell or imports another module. */
export interface ModuleManifest {
  id: string;
  label: string;
  sections: SectionDef[];
}

/** Keeps a record's type inside one definition while the registry stores them all together. */
export function resource<T>(def: Omit<ResourceDef<T>, 'kind'>): SectionDef {
  return { ...def, kind: 'resource' };
}

export function page(def: Omit<PageDef, 'kind'>): SectionDef {
  return { ...def, kind: 'page' };
}

/**
 * Hides what the role cannot use. The API decides; this only keeps a control
 * the user could not use off the screen.
 */
export function allowed(ctx: Ctx, item: { roles?: string[] }): boolean {
  return !item.roles || item.roles.length === 0 || item.roles.includes(ctx.session.role);
}

export function sectionVisible(ctx: Ctx, section: SectionDef): boolean {
  if (section.sessions && !section.sessions.includes(ctx.session.scope)) return false;
  return section.scopes.includes(ctx.scope) && allowed(ctx, section);
}
