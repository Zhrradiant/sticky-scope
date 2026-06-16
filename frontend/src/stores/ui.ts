import { defineStore } from 'pinia'

interface State {
  isExpanded: boolean
}

export const useUiStore = defineStore('ui', {
  state: (): State => ({ isExpanded: false }),
  actions: {
    setExpanded(v: boolean) {
      this.isExpanded = v
    },
    toggleExpanded() {
      this.isExpanded = !this.isExpanded
    },
  },
})
