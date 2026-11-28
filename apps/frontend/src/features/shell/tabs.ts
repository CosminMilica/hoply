import { computed } from 'vue'
import type { NavigationMenuItem } from '@nuxt/ui'
import { useApplicationMenu } from '@/shared/application/menus'
import { translated } from './menu'

export type NavTab = NavigationMenuItem & { to: NonNullable<NavigationMenuItem['to']> }

/** The layout's primary menu as a flat list of links in the app's language, for the header's tabs and the phone's tab bar. */
export function useNavTabs() {
  const menu = useApplicationMenu('app', 'primary')
  return computed(() => menu.value.flat().filter((item): item is NavTab => Boolean(item.to)).map(translated))
}
