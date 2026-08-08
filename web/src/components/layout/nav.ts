import type { FunctionalComponent } from 'vue'

export interface NavItem {
  to: string
  label: string
  icon: FunctionalComponent
}

export interface NavSection {
  title: string
  items: NavItem[]
}
