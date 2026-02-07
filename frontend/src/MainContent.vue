<template>
  <div 
    class="h-screen flex flex-col transition-colors duration-300 overflow-hidden"
    :class="themeClasses"
    :style="{ backgroundColor: currentBgColor }"
    @contextmenu.prevent
    @dragover.prevent
    @drop.prevent="handleGlobalDrop"
  >
    <!-- 顶部控制区 -->
    <header class="flex flex-col border-b border-gray-300 dark:border-gray-700 shrink-0">
      <!-- 辅助工具栏 -->
      <div 
        class="h-10 flex items-center px-4 gap-3 border-b shrink-0 z-10 shadow-sm"
        :style="{ backgroundColor: currentBgColor }"
        :class="store.isDarkMode ? 'border-gray-800' : 'border-gray-200'"
      >
        <div class="flex items-center gap-2">
          <n-button size="small" type="primary" secondary @click="handleFormat">
            <template #icon><n-icon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><path d="M432 32H80a64 64 0 00-64 64v320a64 64 0 0064 64h352a64 64 0 0064-64V96a64 64 0 00-64-64zM96 384H64v-32h32zm0-64H64v-32h32zm0-64H64v-32h32zm0-64H64V96h32zm224 192H128v-32h192zm96 0h-64v-32h64zm0-64H128v-32h288zm0-64H128v-32h288zm0-64H128V96h288z" fill="currentColor"/></svg></n-icon></template>
            格式化
          </n-button>
          <n-divider vertical class="h-5" />
          <n-button size="small" type="info" secondary @click="handleMinify">
            <template #icon><n-icon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><path d="M432 32H80a64 64 0 00-64 64v320a64 64 0 0064 64h352a64 64 0 0064-64V96a64 64 0 00-64-64zM96 384H64v-32h32zm0-64H64v-32h32zm0-64H64v-32h32zm0-64H64V96h32zm224 192H128v-32h192zm96 0h-64v-32h64zm0-64H128v-32h288zm0-64H128v-32h288zm0-64H128V96h288z" fill="currentColor"/></svg></n-icon></template>
            压缩
          </n-button>
        </div>

        <n-divider vertical class="h-5" />

        <n-button size="small" type="success" secondary @click="handleImportClipboard">
          <template #icon><clipboard-icon /></template>
          粘贴导入
        </n-button>

        <n-divider vertical class="h-5" />

        <n-dropdown :options="exportOptions" @select="handleExport">
          <n-button size="small" type="warning" secondary>
            <template #icon><n-icon><download-icon /></n-icon></template>
            导出为...
          </n-button>
        </n-dropdown>

        <n-divider vertical class="h-5" />

        <div class="flex items-center gap-2 shrink-0">
          <span class="text-xs text-gray-400 whitespace-nowrap">缩进:</span>
          <n-select 
            v-if="store.activeTab"
            v-model:value="store.activeTab.formatOptions.indent"
            size="small"
            :options="indentOptions"
            class="w-24"
            @update:value="handleIndentChange"
          />
        </div>

        <n-divider vertical class="h-5" />

        <div class="flex items-center gap-2 shrink-0">
          <n-tooltip trigger="hover">
            <template #trigger>
              <div class="flex items-center gap-1">
                <span class="text-xs text-gray-400">去空格:</span>
                <n-switch
                  v-if="store.activeTab"
                  v-model:value="store.activeTab.formatOptions.trimWhitespace"
                  size="small"
                  @update:value="handleFormat"
                />
              </div>
            </template>
            去除字符串两边不可见符号 (换行、空格等)
          </n-tooltip>
        </div>
        <n-divider vertical class="h-5" />
        <div class="flex-1"></div>
        <n-divider vertical class="h-5" />
        <div class="flex items-center gap-2 shrink-0">
          <span class="text-xs text-gray-400 whitespace-nowrap">主题:</span>
          <n-select 
            v-model:value="store.themeColor"
            size="small"
            :options="themeSelectOptions"
            style="width: 280px"
            @update:value="handleThemeChange"
          />
        </div>
      </div>

      <!-- 标签栏 -->
      <div 
        class="h-11 flex items-center px-3 overflow-x-auto no-scrollbar border-b gap-2"
        :style="{ backgroundColor: currentBgColor }"
        :class="store.isDarkMode ? 'border-gray-800' : 'border-gray-200'"
      >
        <div 
          v-for="(tab, index) in store.tabs" 
          :key="tab.id"
          class="flex items-center h-full"
          @contextmenu.prevent="handleTabContextMenu($event, tab)"
        >
          <div 
            @click="store.activeTabId = tab.id"
            class="group flex items-center h-8 px-4 min-w-[120px] max-w-[220px] text-sm cursor-pointer transition-all duration-150 relative select-none rounded-lg border hover:shadow-sm"
            :class="store.activeTabId === tab.id 
              ? (store.isDarkMode 
                ? 'bg-[#2a2a2a] text-white font-bold border-[#6fb8ff] shadow-md' 
                : 'bg-white text-primary-700 font-bold border-primary shadow-md')
              : (store.isDarkMode 
                ? 'bg-[#242424] text-[#cfcfcf] border-transparent hover:bg-[#2f2f2f] hover:border-[#3a3a3a]' 
                : 'bg-[#f6f8fb] text-gray-600 border-transparent hover:bg-white hover:border-gray-200')"
          >
            <span class="truncate flex-1">
              {{ tab.name }}{{ tab.isDirty ? '*' : '' }}
            </span>

            <div class="flex items-center ml-2 gap-1.5">
              <div 
                v-if="tab.isPinned"
                class="flex items-center justify-center"
                :class="store.isDarkMode ? 'text-blue-400' : 'text-blue-500'"
                @click.stop="store.togglePin(tab.id)"
              >
                <n-icon size="14"><lock-icon /></n-icon>
              </div>

              <div 
                v-if="!tab.isPinned"
                class="flex items-center justify-center w-5 h-5 rounded-full transition-all opacity-0 group-hover:opacity-100"
                :class="store.isDarkMode ? 'hover:bg-white/10 text-[#cfcfcf]' : 'hover:bg-gray-300 text-gray-500'"
                @click.stop="store.closeTab(tab.id)"
              >
                <n-icon size="14"><close-icon /></n-icon>
              </div>
            </div>
          </div>
          <n-divider v-if="index < store.tabs.length - 1" vertical class="h-5 mx-1" />
        </div>

        <div 
          class="flex items-center justify-center w-8 h-8 rounded-lg transition-colors cursor-pointer shrink-0"
          :class="store.isDarkMode ? 'text-[#cfcfcf] hover:bg-[#2b2b2b]' : 'text-gray-500 hover:bg-gray-200'"
          @click="createNewTab"
        >
          <n-icon size="20"><add-icon /></n-icon>
        </div>
      </div>

      <!-- 标签右键菜单 -->
      <n-dropdown
        placement="bottom-start"
        trigger="manual"
        :x="tabContextMenuX"
        :y="tabContextMenuY"
        :options="tabMenuOptions"
        :show="showTabContextMenu"
        :on-clickoutside="() => showTabContextMenu = false"
        @select="handleTabMenuSelect"
      />
    </header>

    <!-- 中部工作区 (85%) -->
    <main class="h-[85%] flex overflow-hidden" :style="{ backgroundColor: currentBgColor }">
      <n-split direction="horizontal" :default-size="0.5" :min="0.1" :max="0.9" class="w-full">
        <template #1>
          <!-- 左侧编辑区 -->
          <div class="h-full relative border-r border-gray-300 dark:border-gray-700" :style="{ backgroundColor: currentBgColor }">
            <n-upload
              v-if="!store.activeTab"
              multiple
              directory-dnd
              :show-file-list="false"
              @before-upload="handleFileUpload"
              class="h-full"
            >
              <div class="h-full flex flex-col items-center justify-center text-gray-500 border-2 border-dashed border-gray-300 dark:border-gray-700 m-4 rounded-lg">
                <n-icon size="48" class="mb-2"><file-icon /></n-icon>
                <div class="text-lg">拖拽 JSON 文件到此处或点击上传</div>
                <div class="text-sm mt-2">支持多文件上传</div>
              </div>
            </n-upload>

            <monaco-editor 
              v-else
              ref="editorRef"
              :value="store.activeTab.content"
              :theme="monacoTheme"
              @update:value="handleContentUpdate"
              @cursor-path="handleCursorPath"
              @paste="handlePaste"
            />
          </div>
        </template>
        <template #2>
          <!-- 右侧视图区 -->
          <div class="h-full overflow-auto" :style="{ backgroundColor: currentBgColor }">
            <tree-view 
              v-if="store.activeTab" 
              ref="treeRef"
              :data="jsonObj" 
              @node-click="handleNodeClick" 
            />
          </div>
        </template>
      </n-split>
    </main>

    <!-- 底部状态区 (5%) -->
    <footer 
      class="h-[5%] flex items-center px-4 text-xs gap-4"
      :style="{ backgroundColor: currentBgColor }"
      :class="store.isDarkMode ? 'text-gray-400 border-t border-gray-800' : 'text-gray-500 border-t border-gray-200'"
    >
      <div v-if="store.activeTab" class="flex items-center gap-4">
        <span>字符数: {{ store.activeTab.content.length }}</span>
        <span>大小: {{ (store.activeTab.content.length / 1024).toFixed(2) }} KB</span>
        <n-divider vertical />
        <span>缩进: {{ store.activeTab.formatOptions.indent }} 空格</span>
        <n-divider vertical />
        <span>编码: UTF-8</span>
      </div>
      <div class="flex-1"></div>
      <div class="flex flex-col items-end gap-0.5" style="padding-right: 10px;">
        <div class="cursor-pointer hover:text-blue-500 transition-colors text-xs" @click="handleOpenLink('https://github.com/zzguang83325/json_formatter_fixer')">
          https://github.com/zzguang83325/json_formatter_fixer
        </div>
      </div>
    </footer>

    <!-- 代码预览对话框 -->
    <n-modal v-model:show="showCodeModal" preset="card" :style="{ width: '800px' }" :title="codeModalTitle">
      <template #header-extra>
        <n-button size="small" type="primary" @click="handleCopyCode">
          <template #icon><n-icon><copy-icon /></n-icon></template>
          复制
        </n-button>
      </template>
      <div v-if="exportType === 'sql'" class="database-select-container" :class="{ 'dark': store.isDarkMode }">
        <span class="input-label">数据库:</span>
        <n-select 
          v-model:value="selectedDatabase" 
          :options="databaseOptions" 
          size="small" 
          style="width: 200px"
          @update:value="handleDatabaseChange"
        />
      </div>
      <div v-if="exportType === 'sql'" class="table-name-input-container" :class="{ 'dark': store.isDarkMode }">
        <span class="input-label">表名:</span>
        <n-input 
          v-model:value="sqlTableName" 
          size="small" 
          style="width: 200px"
          @update:value="handleTableNameChange"
        />
      </div>
      <div v-if="exportType !== 'yaml' && exportType !== 'sql'" class="class-name-input-container" :class="{ 'dark': store.isDarkMode }">
        <span class="input-label">{{ getClassNameLabel() }}</span>
        <n-input 
          v-model:value="codeClassName" 
          size="small" 
          style="width: 200px"
          @update:value="handleClassNameChange"
        />
      </div>
      <div class="code-preview-container">
        <pre class="code-preview" :class="store.isDarkMode ? 'code-dark' : 'code-light'">{{ codeModalContent }}</pre>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch, h, defineComponent, nextTick } from 'vue'
import { 
  NButton, NIcon, NDivider, NSelect, NCheckbox, NSwitch,
  NDropdown, NInput, NTooltip, NUpload, NSplit, NDrawer, NDrawerContent, 
  NModal, NDataTable, useMessage, useDialog
} from 'naive-ui'
import { 
  AddOutline as AddIcon, 
  CloseOutline as CloseIcon,
  ClipboardOutline as ClipboardIcon,
  CopyOutline as CopyIcon,
  LockClosedOutline as LockIcon,
  LockOpenOutline as UnlockIcon,
  FileTrayOutline as FileIcon,
  HelpCircleOutline as HelpIcon,
  DownloadOutline as DownloadIcon
} from '@vicons/ionicons5'
import { useAppStore } from './store/app'
import MonacoEditor from './components/MonacoEditor.vue'
import TreeView from './components/TreeView.vue'
import { 
  FormatJSON, MinifyJSON, ProcessJSON, 
  ConvertToYAML, ConvertToJavaClass, ConvertToGoStruct,
  ConvertToPythonClass, ConvertToTypeScriptInterface, ConvertToCSharpClass, ConvertToSQL,
  GetPathOffset, GetPathByOffset 
} from '../wailsjs/go/main/App'
import { BrowserOpenURL } from '../wailsjs/runtime/runtime'

const store = useAppStore()
const message = useMessage()
const dialog = useDialog()

const editorRef = ref<any>(null)
const treeRef = ref<any>(null)

// 代码预览对话框相关
const showCodeModal = ref(false)
const codeModalTitle = ref('')
const codeModalContent = ref('')
const exportType = ref('')
const codeClassName = ref('')
const sqlTableName = ref('table1')
const originalJsonContent = ref('')
const selectedDatabase = ref('mysql')

// 标签右键菜单相关
const showTabContextMenu = ref(false)
const tabContextMenuX = ref(0)
const tabContextMenuY = ref(0)
const currentTabContext = ref<any>(null)

const tabMenuOptions = computed(() => {
  if (!currentTabContext.value) return []
  const tab = currentTabContext.value
  return [
    { label: '重命名', key: 'rename' },
    { label: tab.isPinned ? '取消固定' : '固定', key: 'toggle-pin' },
    { label: '关闭', key: 'close', disabled: tab.isPinned },
    { label: '关闭其它', key: 'close-others' },
    { label: '关闭左侧', key: 'close-left' },
    { label: '关闭右侧', key: 'close-right' },
    { label: '全部关闭', key: 'close-all' }
  ]
})

function handleTabContextMenu(e: MouseEvent, tab: any) {
  showTabContextMenu.value = false
  nextTick(() => {
    tabContextMenuX.value = e.clientX
    tabContextMenuY.value = e.clientY
    currentTabContext.value = tab
    showTabContextMenu.value = true
  })
}

function handleTabMenuSelect(key: string) {
  showTabContextMenu.value = false
  if (!currentTabContext.value) return
  
  const tabId = currentTabContext.value.id
  
  switch (key) {
    case 'rename':
      const currentName = currentTabContext.value.name
      dialog.info({
        title: '重命名标签',
        content: () => h(NInput, {
          defaultValue: currentName,
          onUpdateValue: (v) => { (window as any).tempRenameValue = v }
        }),
        positiveText: '确定',
        negativeText: '取消',
        onPositiveClick: () => {
          const newName = (window as any).tempRenameValue || currentName
          store.renameTab(tabId, newName)
          delete (window as any).tempRenameValue
        }
      })
      break
    case 'toggle-pin':
      store.togglePin(tabId)
      break
    case 'close':
      store.closeTab(tabId)
      break
    case 'close-others':
      store.closeOtherTabs(tabId)
      break
    case 'close-left':
      store.closeTabsToLeft(tabId)
      break
    case 'close-right':
      store.closeTabsToRight(tabId)
      break
    case 'close-all':
      store.closeAllTabs()
      break
  }
}

const isJsonValid = computed(() => jsonObj.value !== null)
const jsonObj = computed(() => {
  if (!store.activeTab?.content) return null
  try {
    return JSON.parse(store.activeTab.content)
  } catch (e) {
    return null
  }
})

const indentOptions = [
  { label: '2 Spaces', value: '2' },
  { label: '4 Spaces', value: '4' },
  { label: 'Tab', value: 'tab' }
]

const exportOptions = [
  { label: 'YAML', key: 'yaml' },
  { label: 'Java Class', key: 'java' },
  { label: 'Go Struct', key: 'go' },
  { label: 'Python Class', key: 'python' },
  { label: 'TypeScript Interface', key: 'typescript' },
  { label: 'C# Class', key: 'csharp' },
  { label: 'SQL', key: 'sql' }
]

const themeOptions = [
  { value: 'dark', bg: 'bg-[#1e1e1e]' },
  { value: 'yellow', bg: 'bg-[#fdf6e3]' },
  { value: 'green', bg: 'bg-[#e8f5e9]' },
  { value: 'blue', bg: 'bg-[#e3f2fd]' }
]

const themeSelectOptions = [
  { label: '深色主题', value: 'dark' },
  { label: '复古黄', value: 'yellow' },
  { label: '清新绿', value: 'green' },
  { label: '天空蓝', value: 'blue' }
]

const databaseOptions = [
  { label: 'MySQL', value: 'mysql' },
  { label: 'PostgreSQL', value: 'postgresql' },
  { label: 'SQLite', value: 'sqlite' },
  { label: 'SQL Server', value: 'sqlserver' },
  { label: 'Oracle', value: 'oracle' }
]

const themeClasses = computed(() => {
  switch (store.themeColor) {
    case 'yellow': return 'bg-[#fdf6e3] text-[#657b83] theme-light'
    case 'green': return 'bg-[#e8f5e9] text-[#2e7d32] theme-light'
    case 'blue': return 'bg-[#e3f2fd] text-[#1565c0] theme-light'
    case 'dark': return 'bg-[#1e1e1e] text-gray-100 dark theme-dark'
    default: return 'bg-white text-gray-900 theme-light'
  }
})

const workspaceBgClass = computed(() => {
  switch (store.themeColor) {
    case 'yellow': return 'bg-[#fdf6e3]'
    case 'green': return 'bg-[#e8f5e9]'
    case 'blue': return 'bg-[#e3f2fd]'
    default: return 'bg-white dark:bg-[#1e1e1e]'
  }
})

const currentBgColor = computed(() => {
  switch (store.themeColor) {
    case 'yellow': return '#fdf6e3'
    case 'green': return '#e8f5e9'
    case 'blue': return '#e3f2fd'
    case 'dark': return '#1e1e1e'
    default: return '#ffffff'
  }
})

const monacoTheme = computed(() => {
  return store.isDarkMode ? 'vs-dark' : 'vs'
})

function handleThemeChange(color: any) {
  store.themeColor = color
  store.isDarkMode = color === 'dark'
  // 强制设置 body 的 class 以确保全局深色模式样式生效
  if (store.isDarkMode) {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
  store.saveToStorage()
}

function handleKeyDown(e: KeyboardEvent) {
  // Ctrl + W: 关闭当前标签
  if (e.ctrlKey && e.key.toLowerCase() === 'w') {
    e.preventDefault()
    if (store.activeTabId) {
      const tabName = store.activeTab?.name || ''
      store.closeTab(store.activeTabId)
      message.info(`已关闭标签: ${tabName}`)
    }
  }
  // Ctrl + N: 新建标签 (顺便加上，常用)
  if (e.ctrlKey && e.key.toLowerCase() === 'n') {
    e.preventDefault()
    createNewTab()
  }
}

onMounted(() => {
  store.loadFromStorage()
  // 初始化时同步一次 dark class
  if (store.isDarkMode) {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
  window.addEventListener('keydown', handleKeyDown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeyDown)
})

function createNewTab() {
  store.createTab()
}

function handleContentUpdate(content: string) {
  if (store.activeTab) {
    store.updateTabContent(store.activeTabId!, content)
  }
}

function handleNodeClick(path: string) {
  if (editorRef.value) {
    editorRef.value.revealPath(path)
  }
}

function handleCursorPath(path: string) {
  if (treeRef.value) {
    treeRef.value.revealPath(path)
  }
}

async function handlePaste() {
  // 不再自动处理粘贴内容，保持原始状态
}

async function handleFormat() {
  if (!store.activeTab) return
  await autoProcessContent(store.activeTab.content)
}

async function handleMinify() {
  if (!store.activeTab) return
  try {
    const trimWhitespace = store.activeTab.formatOptions.trimWhitespace || false
    const res = await MinifyJSON(store.activeTab.content, trimWhitespace)
    if (res.success) {
      store.updateTabContent(store.activeTabId!, res.data)
      message.success('压缩成功')
    } else {
      message.error('压缩失败: ' + res.error)
    }
  } catch (e: any) {
    message.error('压缩失败: ' + (e.message || '未知错误'))
  }
}

function handleOpenLink(url: string) {
  BrowserOpenURL(url)
}

function handleIndentChange() {
  store.saveToStorage()
  if (store.activeTab?.content) {
    handleFormat()
  }
}

async function autoProcessContent(content: string, tabName?: string) {
  try {
    const indent = store.activeTab?.formatOptions.indent || '4'
    const trimWhitespace = store.activeTab?.formatOptions.trimWhitespace || false
    const res = await ProcessJSON(content, indent, trimWhitespace)
    if (res.success) {
      if (tabName) {
        store.createTab(tabName, res.data)
      } else if (store.activeTabId) {
        store.updateTabContent(store.activeTabId, res.data)
      }
      if (res.repaired) {
        message.warning('检测到 JSON 格式错误，已自动修复并格式化')
      } else {
        message.success('格式化成功')
      }
    } else {
      if (tabName) {
        // 如果解析失败，仍然创建标签，但显示原始错误内容
        store.createTab(tabName, content)
      }
      message.error(res.error || '无法解析该内容')
    }
  } catch (e: any) {
    message.error('处理失败: ' + (e.message || '未知错误'))
  }
}

async function handleImportClipboard() {
  try {
    const text = await navigator.clipboard.readText()
    if (!text) {
      message.warning('剪贴板为空')
      return
    }
    // 粘贴导入时不再自动格式化，直接更新内容
    if (store.activeTabId) {
      store.updateTabContent(store.activeTabId, text)
      message.success('已从剪贴板粘贴')
    } else {
      store.createTab('剪贴板内容', text)
      message.success('已创建新标签并粘贴')
    }
  } catch (e: any) {
    message.error('无法读取剪贴板: ' + (e.message || '权限不足'))
  }
}

function handleRenameTab(tab: any) {
  const newName = ref(tab.name)
  dialog.create({
    title: '重命名标签',
    content: () => h(NInput, {
      value: newName.value,
      'onUpdate:value': (v: string) => newName.value = v,
      placeholder: '输入新名称',
      autofocus: true
    }),
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: () => {
      if (newName.value) {
        tab.name = newName.value
        store.saveToStorage()
      }
    }
  })
}

function handleFileUpload(data: { file: { file: File | null } }) {
  const file = data.file.file
  if (!file) return false
  
  const reader = new FileReader()
  reader.onload = async (e) => {
    const content = e.target?.result as string
    store.createTab(file.name, content)
  }
  reader.readAsText(file)
  return false
}

function handleGlobalDrop(e: DragEvent) {
  const files = e.dataTransfer?.files
  if (!files || files.length === 0) return

  for (let i = 0; i < files.length; i++) {
    const file = files[i]
    if (file.name.toLowerCase().endsWith('.json') || file.type === 'application/json' || file.type === '') {
      const reader = new FileReader()
      reader.onload = async (event) => {
        const content = event.target?.result as string
        store.createTab(file.name, content)
      }
      reader.readAsText(file)
    }
  }
}

async function handleExport(key: string) {
  if (!store.activeTab) return
  let content = store.activeTab.content
  let filename = store.activeTab.name
  
  try {
    const trimWhitespace = store.activeTab.formatOptions.trimWhitespace || false
    if (key === 'yaml') {
      exportType.value = 'yaml'
      originalJsonContent.value = content
      const res = await ConvertToYAML(content, trimWhitespace)
      if (res.success) {
        codeModalTitle.value = 'YAML'
        codeModalContent.value = res.data
        showCodeModal.value = true
      } else {
        throw new Error(res.error)
      }
    } else if (key === 'java') {
      exportType.value = 'java'
      originalJsonContent.value = content
      const res = await ConvertToJavaClass(content, trimWhitespace, '')
      if (res.success) {
        codeModalTitle.value = 'Java Class'
        codeModalContent.value = res.data
        codeClassName.value = 'RootClass'
        showCodeModal.value = true
      } else {
        throw new Error(res.error)
      }
    } else if (key === 'go') {
      exportType.value = 'go'
      originalJsonContent.value = content
      const res = await ConvertToGoStruct(content, trimWhitespace, '')
      if (res.success) {
        codeModalTitle.value = 'Go Struct'
        codeModalContent.value = res.data
        codeClassName.value = 'RootStruct'
        showCodeModal.value = true
      } else {
        throw new Error(res.error)
      }
    } else if (key === 'python') {
      exportType.value = 'python'
      originalJsonContent.value = content
      const res = await ConvertToPythonClass(content, trimWhitespace, '')
      if (res.success) {
        codeModalTitle.value = 'Python Class'
        codeModalContent.value = res.data
        codeClassName.value = 'RootClass'
        showCodeModal.value = true
      } else {
        throw new Error(res.error)
      }
    } else if (key === 'typescript') {
      exportType.value = 'typescript'
      originalJsonContent.value = content
      const res = await ConvertToTypeScriptInterface(content, trimWhitespace, '')
      if (res.success) {
        codeModalTitle.value = 'TypeScript Interface'
        codeModalContent.value = res.data
        codeClassName.value = 'RootInterface'
        showCodeModal.value = true
      } else {
        throw new Error(res.error)
      }
    } else if (key === 'csharp') {
      exportType.value = 'csharp'
      originalJsonContent.value = content
      const res = await ConvertToCSharpClass(content, trimWhitespace, '')
      if (res.success) {
        codeModalTitle.value = 'C# Class'
        codeModalContent.value = res.data
        codeClassName.value = 'RootClass'
        showCodeModal.value = true
      } else {
        throw new Error(res.error)
      }
    } else if (key === 'sql') {
      exportType.value = 'sql'
      originalJsonContent.value = content
      selectedDatabase.value = 'mysql'
      sqlTableName.value = 'table1'
      const res = await ConvertToSQL(content, trimWhitespace, 'mysql', 'table1')
      if (res.success) {
        codeModalTitle.value = 'SQL'
        codeModalContent.value = res.data
        showCodeModal.value = true
      } else {
        throw new Error(res.error)
      }
    }
  } catch (e: any) {
    message.error('导出失败: ' + (e.message || '未知错误'))
  }
}

async function handleCopyCode() {
  try {
    await navigator.clipboard.writeText(codeModalContent.value)
    message.success('复制成功')
  } catch (e: any) {
    message.error('复制失败: ' + (e.message || '未知错误'))
  }
}

async function handleClassNameChange(newName: string) {
  if (!newName || newName.trim() === '' || exportType.value === 'yaml') {
    return
  }
  
  try {
    const trimWhitespace = store.activeTab?.formatOptions.trimWhitespace || false
    
    if (exportType.value === 'java') {
      const res = await ConvertToJavaClass(originalJsonContent.value, trimWhitespace, newName)
      if (res.success) {
        codeModalContent.value = res.data
      }
    } else if (exportType.value === 'go') {
      const res = await ConvertToGoStruct(originalJsonContent.value, trimWhitespace, newName)
      if (res.success) {
        codeModalContent.value = res.data
      }
    } else if (exportType.value === 'python') {
      const res = await ConvertToPythonClass(originalJsonContent.value, trimWhitespace, newName)
      if (res.success) {
        codeModalContent.value = res.data
      }
    } else if (exportType.value === 'typescript') {
      const res = await ConvertToTypeScriptInterface(originalJsonContent.value, trimWhitespace, newName)
      if (res.success) {
        codeModalContent.value = res.data
      }
    } else if (exportType.value === 'csharp') {
      const res = await ConvertToCSharpClass(originalJsonContent.value, trimWhitespace, newName)
      if (res.success) {
        codeModalContent.value = res.data
      }
    }
  } catch (e: any) {
    message.error('生成失败: ' + (e.message || '未知错误'))
  }
}

async function handleDatabaseChange(database: string) {
  if (!database || exportType.value !== 'sql') {
    return
  }
  
  try {
    const trimWhitespace = store.activeTab?.formatOptions.trimWhitespace || false
    const res = await ConvertToSQL(originalJsonContent.value, trimWhitespace, database, sqlTableName.value)
    if (res.success) {
      codeModalContent.value = res.data
    }
  } catch (e: any) {
    message.error('生成失败: ' + (e.message || '未知错误'))
  }
}

async function handleTableNameChange(tableName: string) {
  if (!tableName || exportType.value !== 'sql') {
    return
  }
  
  try {
    const trimWhitespace = store.activeTab?.formatOptions.trimWhitespace || false
    const res = await ConvertToSQL(originalJsonContent.value, trimWhitespace, selectedDatabase.value, tableName)
    if (res.success) {
      codeModalContent.value = res.data
    }
  } catch (e: any) {
    message.error('生成失败: ' + (e.message || '未知错误'))
  }
}

function getClassNameLabel() {
  switch (exportType.value) {
    case 'java': return '类名:'
    case 'go': return '结构体名:'
    case 'python': return '类名:'
    case 'typescript': return '接口名:'
    case 'csharp': return '类名:'
    default: return '名称:'
  }
}
</script>

<style scoped>
.code-preview-container {
  max-height: 500px;
  overflow-y: auto;
}

.code-preview {
  margin: 0;
  padding: 16px;
  border-radius: 6px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.code-light {
  background-color: #f8f9fa;
  color: #212529;
  border: 1px solid #dee2e6;
}

.code-dark {
  background-color: #1e1e1e;
  color: #d4d4d4;
  border: 1px solid #3c3c3c;
}

.class-name-input-container {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e0e0e0;
}

.class-name-input-container.dark {
  border-bottom-color: #3c3c3c;
}

.class-name-input-container .input-label {
  font-size: 14px;
  font-weight: 500;
  color: #333;
}

.class-name-input-container.dark .input-label {
  color: #d4d4d4;
}

.database-select-container {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e0e0e0;
}

.database-select-container.dark {
  border-bottom-color: #3c3c3c;
}

.database-select-container .input-label {
  font-size: 14px;
  font-weight: 500;
  color: #333;
}

.database-select-container.dark .input-label {
  color: #d4d4d4;
}

.table-name-input-container {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e0e0e0;
}

.table-name-input-container.dark {
  border-bottom-color: #3c3c3c;
}

.table-name-input-container .input-label {
  font-size: 14px;
  font-weight: 500;
  color: #333;
}

.table-name-input-container.dark .input-label {
  color: #d4d4d4;
}
</style>
