import type { NavigationMenuItem } from '@nuxt/ui'
import { t } from '@/features/common/i18n'

/** A menu item with its label in the app's language, found by its id; an item the shell does not know keeps its own. */
export function translated<T extends NavigationMenuItem>(item: T): T {
  const label = typeof item.value === 'string' ? (t.value.shell.menu as Record<string, string>)[item.value] : undefined
  return label ? { ...item, label } : item
}
