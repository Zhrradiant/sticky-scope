import { defineStore } from 'pinia'

interface State {
  isExpanded: boolean
  isCollapsed: boolean
}

export const useUiStore = defineStore('ui', {
  state: (): State => ({
    isExpanded: false,
    isCollapsed: false,
  }),
  actions: {
    setExpanded(v: boolean) {
      this.isExpanded = v
    },
    toggleExpanded() {
      this.isExpanded = !this.isExpanded
    },
    setCollapsed(v: boolean) {
      this.isCollapsed = v
    },
    toggleCollapsed() {
      this.setCollapsed(!this.isCollapsed)
    },
  },
})
