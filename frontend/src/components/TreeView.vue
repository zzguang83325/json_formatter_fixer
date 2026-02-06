<template>
  <div class="h-full flex flex-col" @contextmenu.prevent>
    <!-- JSONPath 查询栏 -->
    <div class="p-2 border-b border-gray-300 dark:border-gray-700 flex items-center gap-2">
      <div class="flex items-center gap-1 shrink-0">
        <span class="text-xs text-gray-400">JsonPath:</span>
        <n-tooltip trigger="hover">
          <template #trigger>
            <n-icon class="text-gray-400 cursor-help hover:text-blue-500" @click="isHelpVisible = true">
              <help-icon />
            </n-icon>
          </template>
          JSONPath 查询
        </n-tooltip>
      </div>
      <n-input 
        v-model:value="jsonPath" 
        placeholder="$.store.." 
        size="small" 
        class="flex-1"
        @keyup.enter="handleQuery"
      />
      <n-button size="small" type="primary" secondary @click="handleQuery">查询</n-button>
    </div>

    <!-- 树节点过滤栏 -->
    <div class="p-2 border-b border-gray-300 dark:border-gray-700 flex items-center gap-1">
      <n-input v-model:value="store.globalFilter" placeholder="过滤节点..." size="small" clearable class="flex-1" @update:value="handleFilterChange">
        <template #prefix>
          <n-icon><search-icon /></n-icon>
        </template>
        <template #suffix>
          <div v-if="store.globalFilter && matchedPaths.length > 0" class="text-[10px] text-gray-400 mr-1 select-none">
            {{ currentIndex + 1 }}/{{ matchedPaths.length }}
          </div>
        </template>
      </n-input>
      
      <div v-if="store.globalFilter && matchedPaths.length > 0" class="flex items-center">
        <n-tooltip trigger="hover" placement="bottom">
          <template #trigger>
            <n-button quaternary circle size="tiny" @click="navPrev">
              <template #icon><n-icon><up-icon /></n-icon></template>
            </n-button>
          </template>
          上一个匹配
        </n-tooltip>
        <n-tooltip trigger="hover" placement="bottom">
          <template #trigger>
            <n-button quaternary circle size="tiny" @click="navNext">
              <template #icon><n-icon><down-icon /></n-icon></template>
            </n-button>
          </template>
          下一个匹配
        </n-tooltip>
      </div>

      <n-tooltip trigger="hover" placement="bottom-end">
        <template #trigger>
          <n-icon class="text-gray-400 cursor-help ml-1"><help-icon /></n-icon>
        </template>
        <div class="text-xs">
          <strong>过滤技巧：</strong><br/>
          - 输入文本：同时搜索键名和键值<br/>
          - 自动展开：匹配到的节点会自动展开<br/>
          - 同步高亮：左侧编辑器会同步搜索该文本<br/>
          - 导航定位：点击箭头在匹配项间跳转
        </div>
      </n-tooltip>
    </div>
    <div class="flex-1 overflow-auto p-2 font-mono text-sm select-none">
      <div v-if="data === null" class="text-gray-400 italic">
        No data to display
      </div>
      <div v-else>
        <tree-node 
          ref="rootNode"
          :name="rootName" 
          :value="data" 
          :depth="0" 
          path="$" 
          :filter="store.globalFilter" 
          @node-click="handleNodeClick"
          @node-context-menu="handleNodeContextMenu"
        />
      </div>
    </div>
  </div>

  <!-- 右键菜单 -->
  <n-dropdown
    placement="bottom-start"
    trigger="manual"
    :x="contextMenuX"
    :y="contextMenuY"
    :options="menuOptions"
    :show="showContextMenu"
    :on-clickoutside="() => showContextMenu = false"
    @select="handleMenuSelect"
  />

  <!-- 查询结果抽屉 -->
  <n-drawer v-model:show="isQueryVisible" :width="500" placement="right">
    <n-drawer-content title="JSONPath 查询结果" closable>
      <div class="h-full flex flex-col">
        <div class="mb-2 text-xs text-gray-400">
          路径: <code class="bg-gray-200 dark:bg-gray-800 px-1 rounded">{{ jsonPath }}</code>
        </div>
        <div class="flex-1 border rounded overflow-hidden">
          <monaco-editor 
            :value="JSON.stringify(queryResult, null, 4)" 
            :read-only="true"
          />
        </div>
        <div class="mt-4 flex justify-end gap-2">
          <n-button size="small" type="primary" @click="store.createTab('查询结果', JSON.stringify(queryResult, null, 4))">
            提取到新标签页
          </n-button>
        </div>
      </div>
    </n-drawer-content>
  </n-drawer>

  <!-- 帮助说明模态框 -->
  <n-modal v-model:show="isHelpVisible" preset="card" style="width: 600px" title="JSONPath 使用说明">
    <div class="space-y-4">
      <p class="text-sm text-gray-500">JSONPath 是一种在 JSON 文档中定位信息的查询语言。以下是常用语法对照表：</p>
      <n-data-table 
        size="small"
        :columns="helpColumns" 
        :data="helpData" 
        :bordered="false"
      />
      <div class="bg-blue-50 dark:bg-blue-900/20 p-3 rounded text-xs leading-loose">
        <strong>进阶技巧：</strong><br/>
        - <code>$..name</code> 可查找 JSON 中所有层级的 name 字段<br/>
        - <code>$.items[-1:]</code> 获取列表中的最后一项<br/>
        - <code>$.items[?(@.active)]</code> 筛选所有 active 为 true 的对象
      </div>
      <div class="flex justify-end mt-4 pt-4 border-t border-gray-100 dark:border-gray-800">
        <n-button size="small" @click="isHelpVisible = false">关闭</n-button>
      </div>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, nextTick, h } from 'vue'
import { 
  NInput, NIcon, NTooltip, NButton, NDrawer, 
  NDrawerContent, NModal, NDataTable, NDropdown, useMessage 
} from 'naive-ui'
import { 
  SearchOutline as SearchIcon, 
  HelpCircleOutline as HelpIcon,
  ChevronUpOutline as UpIcon,
  ChevronDownOutline as DownIcon
} from '@vicons/ionicons5'
import TreeNode from './TreeNode.vue'
import MonacoEditor from './MonacoEditor.vue'
import { JSONPath } from 'jsonpath-plus'

import { useAppStore } from '../store/app'

const props = defineProps<{
  data: any
  rootName?: string
}>()

const emit = defineEmits<{
  (e: 'node-click', path: string): void
}>()

const store = useAppStore()
const message = useMessage()
const rootNode = ref<any>(null)
const matchedPaths = ref<string[]>([])
const currentIndex = ref(-1)

// 右键菜单相关
const showContextMenu = ref(false)
const contextMenuX = ref(0)
const contextMenuY = ref(0)
const currentContextNode = ref<{ name?: string, value: any, path: string } | null>(null)

const menuOptions = [
  { label: '复制 键名', key: 'copy-key' },
  { label: '复制 键值', key: 'copy-value' },
  { label: '复制 路径', key: 'copy-path' },
  { label: '复制 节点内容', key: 'copy-content' },
  { label: '复制 键名键值', key: 'copy-key-value' },
  { label: '复制 MAP式内容', key: 'copy-map' }
]

const handleNodeContextMenu = (e: MouseEvent, nodeData: any) => {
  showContextMenu.value = false
  nextTick(() => {
    contextMenuX.value = e.clientX
    contextMenuY.value = e.clientY
    currentContextNode.value = nodeData
    showContextMenu.value = true
  })
}

const handleMenuSelect = (key: string) => {
  showContextMenu.value = false
  if (!currentContextNode.value) return

  const node = currentContextNode.value
  let textToCopy = ''

  switch (key) {
    case 'copy-key':
      textToCopy = node.name || ''
      break
    case 'copy-value':
      textToCopy = typeof node.value === 'object' ? JSON.stringify(node.value) : String(node.value)
      break
    case 'copy-path':
      textToCopy = node.path
      break
    case 'copy-content':
      textToCopy = JSON.stringify(node.value, null, 2)
      break
    case 'copy-key-value':
      if (node.name) {
        const valStr = typeof node.value === 'object' ? JSON.stringify(node.value) : String(node.value)
        textToCopy = `${node.name}: ${valStr}`
      } else {
        textToCopy = typeof node.value === 'object' ? JSON.stringify(node.value) : String(node.value)
      }
      break
    case 'copy-map':
      if (node.name) {
        const valStr = JSON.stringify(node.value, null, 2)
        textToCopy = `"${node.name}": ${valStr}`
      } else {
        textToCopy = JSON.stringify(node.value, null, 2)
      }
      break
  }

  if (textToCopy) {
    navigator.clipboard.writeText(textToCopy).then(() => {
      message.success('已复制到剪贴板')
    }).catch(err => {
      message.error('复制失败: ' + err)
    })
  }
}

// JSONPath 相关状态
const jsonPath = ref('')
const isQueryVisible = ref(false)
const queryResult = ref<any>(null)
const isHelpVisible = ref(false)

const helpColumns = [
  { title: '符号', key: 'symbol', width: 80 },
  { title: '描述', key: 'desc' },
  { title: '示例', key: 'example' }
]

const helpData = [
  { symbol: '$', desc: '根节点', example: '$' },
  { symbol: '@', desc: '当前节点', example: '$.items[?(@.price > 10)]' },
  { symbol: '.', desc: '子节点', example: '$.store.book' },
  { symbol: '..', desc: '深层扫描', example: '$..author' },
  { symbol: '*', desc: '通配符', example: '$.store.book[*]' },
  { symbol: '[]', desc: '下标/切片', example: '$.items[0,1]' },
  { symbol: '[?()]', desc: '过滤表达式', example: '$.items[?(@.id < 5)]' }
]

function handleQuery() {
  if (!store.activeTab || !jsonPath.value) return
  try {
    const data = JSON.parse(store.activeTab.content)
    const result = JSONPath({ path: jsonPath.value, json: data })
    queryResult.value = result
    isQueryVisible.value = true
    message.success('查询成功')
  } catch (e: any) {
    message.error('查询失败: ' + (e.message || '路径语法错误'))
  }
}

const handleFilterChange = (val: string) => {
  if (!val) {
    matchedPaths.value = []
    currentIndex.value = -1
    return
  }
  
  // 延迟获取匹配路径，等待树节点更新
  nextTick(() => {
    if (rootNode.value) {
      matchedPaths.value = rootNode.value.getMatchedPaths(val)
      currentIndex.value = matchedPaths.value.length > 0 ? 0 : -1
      
      // 默认定位到第一个
      if (currentIndex.value !== -1) {
        const path = matchedPaths.value[currentIndex.value]
        rootNode.value.revealPath(path, false)
        emit('node-click', path)
      }
    }
  })
}

const navNext = () => {
  if (matchedPaths.value.length === 0) return
  currentIndex.value = (currentIndex.value + 1) % matchedPaths.value.length
  const path = matchedPaths.value[currentIndex.value]
  rootNode.value.revealPath(path, false)
  emit('node-click', path)
}

const navPrev = () => {
  if (matchedPaths.value.length === 0) return
  currentIndex.value = (currentIndex.value - 1 + matchedPaths.value.length) % matchedPaths.value.length
  const path = matchedPaths.value[currentIndex.value]
  rootNode.value.revealPath(path, false)
  emit('node-click', path)
}

const handleNodeClick = (path: string) => {
  emit('node-click', path)
}

defineExpose({
  revealPath: (path: string) => {
    if (rootNode.value) {
      // 如果路径是 $，直接调用 rootNode 的 revealPath
      // 如果是更深的路径，rootNode 会递归处理
      rootNode.value.revealPath(path)
    }
  }
})
</script>
