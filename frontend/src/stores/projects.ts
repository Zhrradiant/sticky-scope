import { defineStore } from 'pinia'
import { App, model, StartAllMonitoring } from '@/api'
import type { AddProgress } from '@/types'

interface State {
  projects: model.ProjectInfo[]
  activeId: string | null
  addingProject: boolean
  addProgress: AddProgress | null
}

export const useProjectsStore = defineStore('projects', {
  state: (): State => ({
    projects: [],
    activeId: null,
    addingProject: false,
    addProgress: null,
  }),
  getters: {
    active(state): model.ProjectInfo | null {
      return state.projects.find((p) => p.id === state.activeId) ?? null
    },
  },
  actions: {
    async load() {
      this.projects = await App.ListProjects()
      // Kick off background watchers now that the list is rendered. This must
      // NOT be awaited: StartAllMonitoring launches goroutines per project and
      // returns immediately, but keeping it fire-and-forget guarantees the UI
      // never blocks on watcher registration even if the call shape changes.
      void StartAllMonitoring()
      if (this.activeId && !this.projects.some((p) => p.id === this.activeId)) {
        this.activeId = null
      }
      if (!this.activeId && this.projects.length) {
        // Prefer the project passed via --project-path (sticky note spawn).
        const stickyId = await App.StickyProjectID()
        if (stickyId && this.projects.some((p) => p.id === stickyId)) {
          this.activeId = stickyId
        } else {
          this.activeId = this.projects[0].id
        }
      }
    },
    select(id: string) {
      this.activeId = id
    },
    async addByPath(path: string) {
      this.addingProject = true
      this.addProgress = { message: '', current: 0, total: 0 }
      try {
        const p = await App.AddProject(path)
        await this.load()
        this.activeId = p.id
        return p
      } finally {
        this.addingProject = false
        this.addProgress = null
      }
    },
    setAddProgress(p: AddProgress) {
      this.addProgress = p
      if (p.message === 'done') {
        this.addingProject = false
      }
    },
    async pickAndAdd() {
      const dir = await App.SelectDirectory()
      if (!dir) return null
      return this.addByPath(dir)
    },
    async remove(id: string) {
      await App.RemoveProject(id)
      if (this.activeId === id) this.activeId = null
      await this.load()
    },
    async updateIgnore(id: string, extraPatterns: string[], useGitignore: boolean) {
      await App.UpdateIgnore(id, extraPatterns, useGitignore)
      await this.load()
    },
  },
})
