import { defineStore } from 'pinia'
import { App, model } from '@/api'

interface State {
  changeSets: Record<string, model.ChangeSet>
  // Project IDs whose ChangeSet is still being fetched for the first time
  // (awaiting GetChanges or a changes:updated push). Used to show a loading
  // state instead of a misleading "no changes" message while the background
  // rescan has not reported back yet.
  loading: Record<string, boolean>
  selectedPath: string | null
  fileDiff: model.FileDiff | null
  diffLoading: boolean
  busy: boolean
}

export const useChangesStore = defineStore('changes', {
  state: (): State => ({
    changeSets: {},
    loading: {},
    selectedPath: null,
    fileDiff: null,
    diffLoading: false,
    busy: false,
  }),
  actions: {
    changeSetFor(id: string | null): model.ChangeSet | null {
      if (!id) return null
      return this.changeSets[id] ?? null
    },
    // True when the given project's ChangeSet has not arrived yet (first load).
    isLoading(id: string | null): boolean {
      if (!id) return false
      return !!this.loading[id]
    },
    setChangeSet(cs: model.ChangeSet) {
      this.changeSets[cs.projectId] = cs
      // A push means the first scan landed — clear the loading flag.
      this.loading[cs.projectId] = false
    },
    async fetchChanges(id: string) {
      this.loading[id] = true
      try {
        this.changeSets[id] = await App.GetChanges(id)
      } finally {
        this.loading[id] = false
      }
    },
    async selectFile(id: string, path: string) {
      this.selectedPath = path
      await this.loadDiff(id, path)
    },
    async loadDiff(id: string, path: string) {
      this.diffLoading = true
      try {
        this.fileDiff = await App.GetFileDiff(id, path)
      } finally {
        this.diffLoading = false
      }
    },
    async refreshSelectedDiff(id: string) {
      if (!this.selectedPath) return
      const cs = this.changeSets[id]
      if (cs && cs.files.some((f) => f.path === this.selectedPath)) {
        await this.loadDiff(id, this.selectedPath)
      } else {
        this.fileDiff = null
        this.selectedPath = null
      }
    },
    async afterMutation(id: string) {
      await this.fetchChanges(id)
      await this.refreshSelectedDiff(id)
    },
    async confirmAll(id: string) {
      this.busy = true
      try {
        await App.ConfirmAll(id)
        await this.afterMutation(id)
      } finally {
        this.busy = false
      }
    },
    async deepRescan(id: string) {
      this.busy = true
      try {
        this.changeSets[id] = await App.DeepRescan(id)
        await this.refreshSelectedDiff(id)
      } finally {
        this.busy = false
      }
    },
    resetForProject() {
      this.selectedPath = null
      this.fileDiff = null
    },
  },
})
