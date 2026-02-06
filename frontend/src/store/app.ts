import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface Tab {
  id: string
  name: string
  content: string
  isDirty: boolean
  isPinned: boolean
  formatOptions: {
    indent: '2' | '4' | 'tab'
    quotes: 'double' | 'single'
    trimWhitespace: boolean
  }
}

export const useAppStore = defineStore('app', () => {
  const tabs = ref<Tab[]>([])
  const activeTabId = ref<string | null>(null)
  const isDarkMode = ref(true)
  const themeColor = ref<'dark' | 'yellow' | 'green' | 'blue'>('dark')
  const globalFilter = ref('')

  const activeTab = computed(() => 
    tabs.value.find(t => t.id === activeTabId.value) || null
  )

  // 自动保存到 localStorage
  function saveToStorage() {
    const data = {
      tabs: tabs.value,
      activeTabId: activeTabId.value,
      themeColor: themeColor.value,
      isDarkMode: isDarkMode.value,
    }
    localStorage.setItem('json_repair_state', JSON.stringify(data))
  }

  // 从 localStorage 恢复
  function loadFromStorage() {
    const saved = localStorage.getItem('json_repair_state')
    if (saved) {
      try {
        const data = JSON.parse(saved)
        tabs.value = data.tabs
        activeTabId.value = data.activeTabId
        if (data.themeColor) themeColor.value = data.themeColor
        if (data.isDarkMode !== undefined) isDarkMode.value = data.isDarkMode

        // 如果没有标签，创建一个默认的
        if (tabs.value.length === 0) {
          createTab('Untitled', '')
        }
      } catch (e) {
        console.error('Failed to load state from storage', e)
        createTab('Untitled', '')
      }
    } else {
      createTab('Untitled', '')
    }
  }

  function createTab(name = 'New Tab', content = '') {
    const id = crypto.randomUUID()
    const newTab: Tab = {
      id,
      name,
      content,
      isDirty: false,
      isPinned: false,
      formatOptions: {
        indent: '4',
        quotes: 'double',
        trimWhitespace: true
      }
    }
    tabs.value.push(newTab)
    activeTabId.value = id
    saveToStorage()
    return newTab
  }

  function closeTab(id: string) {
    const index = tabs.value.findIndex(t => t.id === id)
    if (index === -1) return

    // 如果是固定标签，不允许关闭
    if (tabs.value[index].isPinned) return

    if (activeTabId.value === id) {
      const nextTab = tabs.value[index + 1] || tabs.value[index - 1]
      activeTabId.value = nextTab ? nextTab.id : null
    }
    tabs.value.splice(index, 1)
    saveToStorage()
  }

  function updateTabContent(id: string, content: string) {
    const tab = tabs.value.find(t => t.id === id)
    if (tab) {
      tab.content = content
      tab.isDirty = true
      saveToStorage()
    }
  }

  function togglePin(id: string) {
    const tab = tabs.value.find(t => t.id === id)
    if (tab) {
      tab.isPinned = !tab.isPinned
      saveToStorage()
    }
  }

  function renameTab(id: string, newName: string) {
    const tab = tabs.value.find(t => t.id === id)
    if (tab) {
      tab.name = newName
      saveToStorage()
    }
  }

  function closeOtherTabs(id: string) {
    tabs.value = tabs.value.filter(t => t.id === id || t.isPinned)
    activeTabId.value = id
    saveToStorage()
  }

  function closeTabsToLeft(id: string) {
    const index = tabs.value.findIndex(t => t.id === id)
    if (index === -1) return
    tabs.value = tabs.value.filter((t, i) => i >= index || t.isPinned)
    saveToStorage()
  }

  function closeTabsToRight(id: string) {
    const index = tabs.value.findIndex(t => t.id === id)
    if (index === -1) return
    tabs.value = tabs.value.filter((t, i) => i <= index || t.isPinned)
    saveToStorage()
  }

  function closeAllTabs() {
    tabs.value = tabs.value.filter(t => t.isPinned)
    if (tabs.value.length > 0) {
      activeTabId.value = tabs.value[0].id
    } else {
      createTab('Untitled', '')
    }
    saveToStorage()
  }

  return {
    tabs,
    activeTabId,
    activeTab,
    isDarkMode,
    themeColor,
    globalFilter,
    createTab,
    closeTab,
    updateTabContent,
    togglePin,
    renameTab,
    closeOtherTabs,
    closeTabsToLeft,
    closeTabsToRight,
    closeAllTabs,
    loadFromStorage,
    saveToStorage
  }
})
