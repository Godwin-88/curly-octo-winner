import { communications } from '@/modules/communications';
import { finance } from '@/modules/finance';
import { platform } from '@/modules/platform';
import { school } from '@/modules/school';
import { sectionVisible, type Ctx, type ModuleManifest, type SectionDef } from './types';

/**
 * The modules that follow list → view → edit, in order. Adding a module is one
 * import and one entry here. The modules not listed yet still live under
 * app/(admin) and are reached from the sidebar.
 */
export const MODULES: ModuleManifest[] = [platform, communications, finance, school];

export function visibleSections(ctx: Ctx, module: ModuleManifest): SectionDef[] {
  return module.sections.filter((section) => sectionVisible(ctx, section));
}

/** Modules sold separately: a school has them or it does not. The API enforces it; this hides the menu. */
export const SOLD_MODULES = ['communications', 'finance', 'academic'];

export function moduleEnabled(id: string, enabled: string[] | undefined): boolean {
  return !enabled || !SOLD_MODULES.includes(id) || enabled.includes(id);
}

/**
 * A module appears when the user can open at least one of its sections in this
 * scope and the school has it. `enabled` is the school's modules; undefined
 * hides nothing.
 */
export function visibleModules(ctx: Ctx, enabled?: string[]): ModuleManifest[] {
  return MODULES.filter((module) => moduleEnabled(module.id, enabled) && visibleSections(ctx, module).length > 0);
}
