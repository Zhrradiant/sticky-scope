import { ref, readonly, type Ref } from 'vue'

/**
 * A single item in the context menu.
 */
export interface ContextMenuItem {
  /** Display label */
  label: string
  /** Optional icon class or emoji */
  icon?: string
  /** Called when the item is clicked */
  action: () => void
  /** When true the item appears faded and is not clickable */
  disabled?: boolean
  /** When true renders a separator line after this item */
  separator?: boolean
  /** When true renders in danger/destructive style */
  danger?: boolean
}

export interface ContextMenuState {
  visible: Ref<boolean>
  x: Ref<number>
  y: Ref<number>
  items: Ref<ContextMenuItem[]>
}

/**
 * Manages a single-instance context menu. Call openMenu(e, items) to show it;
 * the caller must also render <ContextMenu> bound to the returned reactive state
 * in the same component tree.
 */
export function useContextMenu(): ContextMenuState & {
  openMenu: (e: MouseEvent, items: ContextMenuItem[]) => void
  closeMenu: () => void
} {
  const visible = ref(false)
  const x = ref(0)
  const y = ref(0)
  const items = ref<ContextMenuItem[]>([])

  function openMenu(e: MouseEvent, menuItems: ContextMenuItem[]) {
    e.preventDefault()
    e.stopPropagation()
    items.value = menuItems
    x.value = e.clientX
    y.value = e.clientY
    visible.value = true
  }

  function closeMenu() {
    visible.value = false
    // Delay clearing items so the closing animation/teleport can still read them
    setTimeout(() => {
      items.value = []
    }, 150)
  }

  return {
    visible: readonly(visible) as Ref<boolean>,
    x: readonly(x) as Ref<number>,
    y: readonly(y) as Ref<number>,
    items: readonly(items) as Ref<ContextMenuItem[]>,
    openMenu,
    closeMenu,
  }
}
