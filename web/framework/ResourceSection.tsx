'use client';

import { useInfiniteQuery, useQuery, useQueryClient } from '@tanstack/react-query';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useState } from 'react';
import { confirmLeave, pathTo } from '@/shell/context';
import { ActionForm, newIdempotencyKey } from '@/ui/Form';
import { Button, Definitions, Empty, ErrorNote, RevealBox, Spinner, Status } from '@/ui/kit';
import { heldReveal, holdReveal } from './revealStore';
import { allowed, type ActionDef, type Ctx, type FormValues, type ResourceDef, type Reveal } from './types';

interface Props<T> {
  moduleId: string;
  def: ResourceDef<T>;
  ctx: Ctx;
  /** The record in the address, if any. */
  record?: string;
}

/**
 * List, then view, then edit, for one kind of record. The list is on the left; choosing a row
 * opens the view beside it; an action turns the view into a form. Each step has its own address:
 * .../{section}, .../{section}/{id}, and ?do={action}.
 */
export function ResourceSection<T>({ moduleId, def, ctx, record }: Props<T>) {
  const search = useSearchParams();
  const router = useRouter();
  const queryClient = useQueryClient();
  const [filters, setFilters] = useState<FormValues>(() => def.defaultFilters?.(ctx) ?? {});
  const [done, setDone] = useState('');
  const [failure, setFailure] = useState<unknown>();

  const doing = search.get('do');
  // The school is part of every key: nothing loaded for one school is shown for another.
  const scopeKey = [moduleId, def.id, ctx.groupId ?? 'all', ctx.schoolId ?? 'all'];
  const sectionPath = pathTo(ctx, moduleId, def.id);
  const recordPath = record ? `${sectionPath}/${encodeURIComponent(record)}` : sectionPath;
  const [reveal, setRevealState] = useState<Reveal | undefined>(() => heldReveal(recordPath));
  const setReveal = (next: Reveal | undefined, path = recordPath) => {
    holdReveal(path, next);
    setRevealState(next);
  };

  const list = useInfiniteQuery({
    queryKey: [...scopeKey, 'list', filters],
    queryFn: ({ pageParam }) => def.list(ctx, filters, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.next,
    refetchInterval: (query) => {
      const rows = query.state.data?.pages.flatMap((page) => page.items) ?? [];
      return def.live && rows.some((row) => def.live!(row)) ? 4000 : false;
    },
  });
  const rows = list.data?.pages.flatMap((page) => page.items) ?? [];
  const fromList = record ? rows.find((row) => def.rowId(row) === record) : undefined;

  const one = useQuery({
    queryKey: [...scopeKey, 'one', record],
    queryFn: () => def.get!(ctx, record!),
    enabled: Boolean(record && def.get),
    refetchInterval: (query) => (def.live && query.state.data && def.live(query.state.data) ? 3000 : false),
  });
  const row: T | undefined = def.get ? (one.data ?? fromList) : fromList;

  const mayCreate = def.create && allowed(ctx, def.create) && (!def.create.when || def.create.when(undefined, ctx)) ? def.create : undefined;
  const creating = doing === 'create' ? mayCreate : undefined;
  const actions = row ? (def.actions ?? []).filter((action) => allowed(ctx, action) && (!action.when || action.when(row, ctx))) : [];
  const action = row && doing ? actions.find((candidate) => candidate.id === doing) : undefined;

  function setDoing(id?: string) {
    if (!confirmLeave()) return;
    setFailure(undefined);
    router.replace(id ? `${recordPath}?do=${id}` : recordPath);
  }

  async function finish<R>(chosen: ActionDef<R>, target: R, values: FormValues, key: string) {
    const result = await chosen.run(ctx, target, values, key);
    await queryClient.invalidateQueries({ queryKey: scopeKey });
    const shown = chosen.reveal?.(result);
    const created = chosen.createdId?.(result);
    setReveal(shown, created ? `${sectionPath}/${encodeURIComponent(created)}` : recordPath);
    setDone(shown ? '' : `${chosen.label}: done.`);
    if (created) router.replace(`${sectionPath}/${encodeURIComponent(created)}`);
    else if (chosen.removes) router.replace(sectionPath);
    else router.replace(recordPath);
  }

  /** An action with nothing to fill in and nothing to confirm runs at once. */
  function start(chosen: ActionDef<T>) {
    setDone('');
    const fields = typeof chosen.fields === 'function' ? chosen.fields(row!, ctx) : chosen.fields;
    if ((fields && fields.length > 0) || chosen.confirm || chosen.preview) {
      setDoing(chosen.id);
      return;
    }
    setFailure(undefined);
    finish(chosen, row!, {}, newIdempotencyKey()).catch(setFailure);
  }

  const showPanel = Boolean(record || creating);

  return (
    <div className="flex h-full min-h-0 flex-col gap-3 lg:flex-row">
      {/* LIST */}
      <section
        aria-label={`${def.label} list`}
        className={`min-h-0 flex-col rounded-xl border border-gray-200 bg-white lg:flex lg:w-[46%] lg:shrink-0 ${showPanel ? 'hidden' : 'flex flex-1'}`}
      >
        <header className="flex flex-wrap items-center justify-between gap-2 border-b border-gray-200 px-4 py-3">
          <div className="min-w-0">
            <h1 className="text-lg font-bold">{def.label}</h1>
            <p className="text-xs text-gray-600">{def.purpose}</p>
          </div>
          {mayCreate && (
            <Link
              href={`${sectionPath}?do=create`}
              onClick={(event) => {
                if (!confirmLeave()) event.preventDefault();
                setDone('');
              }}
              className="inline-flex items-center justify-center rounded-lg bg-blue-700 px-3.5 py-2 text-sm font-semibold text-white hover:bg-blue-800"
            >
              {mayCreate.label}
            </Link>
          )}
        </header>
        {def.filters && (
          <details className="border-b border-gray-200 px-4 py-2">
            <summary className="cursor-pointer text-sm font-semibold text-gray-700">
              Filters{Object.keys(filters).length > 0 ? ` (${Object.keys(filters).length} applied)` : ''}
            </summary>
            <div className="pb-2 pt-3">
              <ActionForm
                ctx={ctx}
                fields={def.filters}
                initial={filters}
                submitLabel="Apply filters"
                onSubmit={async (values) => setFilters(values)}
              />
            </div>
          </details>
        )}
        <div className="min-h-0 flex-1 overflow-auto">
          {list.isLoading && <Spinner />}
          {list.error && <div className="p-4"><ErrorNote error={list.error} /></div>}
          {list.isSuccess && rows.length === 0 && (
            <Empty title={`No ${def.noun}s yet`}>
              {mayCreate ? `Use "${mayCreate.label}" to add the first one.` : 'Nothing to show for this school.'}
            </Empty>
          )}
          {rows.length > 0 && (
            <table className="w-full text-left text-sm">
              <caption className="sr-only">{def.label}</caption>
              <thead className="sticky top-0 bg-white">
                <tr className="border-b border-gray-200 text-xs uppercase tracking-wide text-gray-600">
                  {def.columns.map((column) => (
                    <th key={column.header} scope="col" className={`px-4 py-2 font-semibold ${column.align === 'right' ? 'text-right' : ''}`}>
                      {column.header}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {rows.map((item) => {
                  const id = def.rowId(item);
                  const selected = id === record;
                  return (
                    <tr key={id} className={`border-b border-gray-200 last:border-0 ${selected ? 'bg-blue-50' : 'hover:bg-gray-50'}`}>
                      {def.columns.map((column, index) => (
                        <td key={column.header} className={`px-4 py-2.5 align-top ${column.align === 'right' ? 'text-right tabular-nums' : ''}`}>
                          {index === 0 ? (
                            <Link
                              href={`${sectionPath}/${encodeURIComponent(id)}`}
                              aria-current={selected ? 'true' : undefined}
                              onClick={(event) => {
                                if (!confirmLeave()) event.preventDefault();
                                setDone('');
                              }}
                              className="font-semibold text-blue-800 underline-offset-2 hover:underline"
                            >
                              {column.cell(item)}
                            </Link>
                          ) : (
                            column.cell(item)
                          )}
                        </td>
                      ))}
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
          {list.hasNextPage && (
            <div className="p-3 text-center">
              <Button onClick={() => void list.fetchNextPage()} disabled={list.isFetchingNextPage}>
                {list.isFetchingNextPage ? 'Loading…' : 'Load more'}
              </Button>
            </div>
          )}
        </div>
      </section>

      {/* VIEW and EDIT */}
      <section
        aria-label={`${def.noun} detail`}
        className={`min-h-0 flex-1 flex-col overflow-auto rounded-xl border border-gray-200 bg-white lg:flex ${showPanel ? 'flex' : 'hidden'}`}
      >
        {!showPanel && (
          <Empty title={`Choose a ${def.noun}`}>Select a row in the list to see it here. What you may do with it appears above its details.</Empty>
        )}

        {creating && (
          <div className="space-y-4 p-4">
            <PanelHeader title={creating.label} onClose={() => { if (confirmLeave()) router.push(sectionPath); }} />
            <ActionForm
              ctx={ctx}
              fields={typeof creating.fields === 'function' ? creating.fields(undefined, ctx) : creating.fields ?? []}
              initial={creating.initial?.(undefined, ctx)}
              confirm={creating.confirm}
              preview={creating.preview ? (values) => creating.preview!(ctx, undefined, values) : undefined}
              submitLabel={creating.submitLabel ?? creating.label}
              onSubmit={(values, key) => finish(creating, undefined, values, key)}
              onCancel={() => router.push(sectionPath)}
            />
          </div>
        )}

        {record && !creating && (
          <div className="space-y-4 p-4">
            {!row && (one.isLoading || list.isLoading) && <Spinner />}
            {!row && one.error && <ErrorNote error={one.error} />}
            {!row && !one.isLoading && !list.isLoading && !one.error && (
              <Empty title={`This ${def.noun} is not here`}>It does not exist, or it belongs to a different school from the one chosen above.</Empty>
            )}
            {row && (
              <>
                <PanelHeader
                  title={def.title(row)}
                  status={def.status?.(row)}
                  onClose={() => { if (confirmLeave()) router.push(sectionPath); }}
                />
                {reveal && <RevealBox {...reveal} onDone={() => setReveal(undefined)} />}
                {done && <p role="status" className="rounded-lg bg-emerald-50 p-3 text-sm font-semibold text-emerald-900">{done}</p>}
                {failure !== undefined && !action && <ErrorNote error={failure} />}

                {action ? (
                  <div className="space-y-3 rounded-xl border border-blue-300 p-4">
                    <h3 className="font-bold">{action.label}</h3>
                    <ActionForm
                      key={action.id}
                      ctx={ctx}
                      fields={typeof action.fields === 'function' ? action.fields(row, ctx) : action.fields ?? []}
                      initial={action.initial?.(row, ctx)}
                      confirm={action.confirm}
                      preview={action.preview ? (values) => action.preview!(ctx, row, values) : undefined}
                      danger={action.tone === 'danger'}
                      submitLabel={action.submitLabel ?? action.label}
                      onSubmit={(values, key) => finish(action, row, values, key)}
                      onCancel={() => setDoing()}
                    />
                  </div>
                ) : (
                  actions.length > 0 && (
                    <div className="flex flex-wrap gap-2" role="group" aria-label="Actions">
                      {actions.map((candidate) => (
                        <Button key={candidate.id} tone={candidate.tone ?? 'plain'} onClick={() => start(candidate)}>
                          {candidate.label}
                        </Button>
                      ))}
                    </div>
                  )
                )}

                <Definitions items={def.fields.map((field) => ({ label: field.label, value: field.value(row) }))} />
                {def.extra && <def.extra row={row} ctx={ctx} />}
              </>
            )}
          </div>
        )}
      </section>
    </div>
  );
}

function PanelHeader({ title, status, onClose }: { title: string; status?: string; onClose: () => void }) {
  return (
    <header className="flex items-start justify-between gap-3">
      <div className="min-w-0 space-y-1">
        <h2 className="break-words text-lg font-bold">{title}</h2>
        {status && <Status value={status} />}
      </div>
      <Button small onClick={onClose} aria-label="Close and return to the list">Close</Button>
    </header>
  );
}
