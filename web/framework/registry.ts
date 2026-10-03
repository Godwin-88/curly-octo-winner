import { communications } from '@/modules/communications';
import { platform } from '@/modules/platform';
import { sectionVisible, type Ctx, type ModuleManifest, type SectionDef } from './types';

/**
 * The modules that follow list → view → edit, in order. Adding a module is one
 * import and one entry here. The modules not listed yet still live under
 * app/(admin) and are reached from the sidebar.
 */
export const MODULES: ModuleManifest[] = [platform, communications];

export function visibleSections(ctx: Ctx, module: ModuleManifest): SectionDef[] {
  return module.sections.filter((section) => sectionVisible(ctx, section));
}

/** A module appears when the user can open at least one of its sections in this scope. */
export function visibleModules(ctx: Ctx): ModuleManifest[] {
  return MODULES.filter((module) => visibleSections(ctx, module).length > 0);
}
