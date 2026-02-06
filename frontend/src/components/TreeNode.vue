<template>
  <div v-if="isVisible" class="tree-node">
    <n-tooltip 
      trigger="hover" 
      placement="top-start" 
      :show-arrow="false" 
      :follow-mouse="true"
      :delay="500"
    >
      <template #trigger>
        <div 
          ref="nodeRef"
          class="flex items-center py-1 hover:bg-gray-100 dark:hover:bg-[#2a2d2e] cursor-pointer group transition-all duration-300"
          :class="{ 
            'bg-[#ffd33d] z-10 rounded-sm shadow-sm text-black': isHighlighted 
          }"
          :style="{ paddingLeft: `${depth * 16 + 4}px` }"
          @click="handleNodeClick"
          @contextmenu.prevent="handleContextMenu"
        >
          <!-- 展开/收起图标 -->
          <div class="w-4 h-4 flex items-center justify-center mr-1">
            <n-icon v-if="isExpandable" :class="{ 'rotate-90': expanded }" class="transition-transform duration-200">
              <chevron-forward />
            </n-icon>
          </div>

          <!-- 类型图标 -->
          <n-icon :color="typeColor" class="text-base" style="margin-right: 8px;">
            <component :is="typeIcon" />
          </n-icon>

          <!-- 键名 -->
          <span v-if="name" class="text-blue-600 dark:text-[#9cdcfe] mr-1">{{ name }}:</span>

          <!-- 预览值/统计信息 -->
          <span v-if="!expanded || !isExpandable" class="truncate flex-1">
            <span :class="valueClass">{{ displayValue }}</span>
            <span v-if="isExpandable" class="text-gray-400 text-xs ml-2">
              ({{ childCount }} items)
            </span>
          </span>
        </div>
      </template>
      <div class="text-[10px] font-mono">{{ path }}</div>
    </n-tooltip>

    <!-- 子节点 -->
    <div v-if="expanded && isExpandable">
      <tree-node 
        v-for="(val, key) in value" 
        :key="key"
        ref="childRefs"
        :name="String(key)"
        :value="val"
        :depth="depth + 1"
        :path="`${path}${Array.isArray(value) ? '[' + key + ']' : '.' + key}`"
        :filter="filter"
        @node-click="handleChildClick"
        @node-context-menu="handleChildContextMenu"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { NIcon, NTooltip } from 'naive-ui'
import { 
  ChevronForwardOutline as ChevronForward,
  CubeOutline as ObjectIcon,
  ListOutline as ArrayIcon,
  TextOutline as StringIcon,
  StatsChartOutline as NumberIcon,
  CheckmarkCircleOutline as BooleanIcon,
  HelpCircleOutline as NullIcon
} from '@vicons/ionicons5'

const props = defineProps<{
  name?: string
  value: any
  depth: number
  path: string
  filter?: string
}>()

const emit = defineEmits<{
  (e: 'node-click', path: string): void
  (e: 'node-context-menu', event: MouseEvent, nodeData: { name?: string, value: any, path: string }): void
}>()

const nodeRef = ref<HTMLElement | null>(null)
const expanded = ref(props.depth < 2)
const isHighlighted = ref(false)
const forceVisible = ref(false)

// 当过滤文本改变时，如果节点包含过滤文本，则自动展开
watch(() => props.filter, (newFilter) => {
  forceVisible.value = false // 重置强制显示状态
  if (newFilter && isVisible.value && isExpandable.value) {
    expanded.value = true
  }
})

const isVisible = computed(() => {
  if (forceVisible.value) return true
  if (!props.filter) return true
  const f = props.filter.toLowerCase()
  
  // 检查当前节点是否匹配（键名或值）
  const nameMatch = props.name?.toLowerCase().includes(f)
  const valueMatch = (props.value !== null && typeof props.value !== 'object') 
    ? String(props.value).toLowerCase().includes(f) 
    : false
  
  if (nameMatch || valueMatch) return true

  // 如果当前节点不匹配，检查子节点
  if (isExpandable.value) {
    if (Array.isArray(props.value)) {
      return props.value.some((v, i) => checkMatch(v, String(i), f))
    } else {
      return Object.entries(props.value).some(([k, v]) => checkMatch(v, k, f))
    }
  }

  return false
})

const checkMatch = (val: any, name: string, filter: string): boolean => {
  if (name.toLowerCase().includes(filter)) return true
  if (val !== null && typeof val !== 'object' && String(val).toLowerCase().includes(filter)) return true
  if (val && typeof val === 'object') {
    if (Array.isArray(val)) {
      return val.some((v, i) => checkMatch(v, String(i), filter))
    } else {
      return Object.entries(val).some(([k, v]) => checkMatch(v, k, filter))
    }
  }
  return false
}

// 辅助函数：递归获取匹配路径
const getMatchedPathsForData = (val: any, name: string, path: string, filter: string): string[] => {
  const paths: string[] = []
  const nameMatch = name.toLowerCase().includes(filter)
  const valueMatch = (val !== null && typeof val !== 'object') 
    ? String(val).toLowerCase().includes(filter) 
    : false
    
  if (nameMatch || valueMatch) {
    paths.push(path)
  }
  
  if (val && typeof val === 'object') {
    if (Array.isArray(val)) {
      val.forEach((v, i) => {
        paths.push(...getMatchedPathsForData(v, String(i), `${path}[${i}]`, filter))
      })
    } else {
      Object.entries(val).forEach(([k, v]) => {
        paths.push(...getMatchedPathsForData(v, k, `${path}.${k}`, filter))
      })
    }
  }
  return paths
}

const handleNodeClick = (e: MouseEvent) => {
  // 阻止冒泡，避免触发父节点的点击
  e.stopPropagation()
  
  // 发送点击事件
  emit('node-click', props.path)
  
  // 如果是展开/收起图标区域，则不切换展开状态
  if (isExpandable.value) {
    expanded.value = !expanded.value
  }
}

const handleContextMenu = (e: MouseEvent) => {
  e.stopPropagation()
  emit('node-context-menu', e, {
    name: props.name,
    value: props.value,
    path: props.path
  })
}

const handleChildClick = (path: string) => {
  emit('node-click', path)
}

const handleChildContextMenu = (e: MouseEvent, nodeData: any) => {
  emit('node-context-menu', e, nodeData)
}

// 暴露给外部的方法
const childRefs = ref<any[]>([])
defineExpose({
  // 获取所有匹配过滤条件的节点路径
  getMatchedPaths: (filter: string): string[] => {
    const paths: string[] = []
    const f = filter.toLowerCase()
    
    // 检查当前节点
    const nameMatch = props.name?.toLowerCase().includes(f)
    const valueMatch = (props.value !== null && typeof props.value !== 'object') 
      ? String(props.value).toLowerCase().includes(f) 
      : false
      
    if (nameMatch || valueMatch) {
      paths.push(props.path)
    }
    
    // 递归检查子节点数据（不依赖组件是否挂载）
    if (isExpandable.value && props.value) {
      if (Array.isArray(props.value)) {
        props.value.forEach((v, i) => {
          const childPath = `${props.path}[${i}]`
          const childPaths = getMatchedPathsForData(v, String(i), childPath, f)
          paths.push(...childPaths)
        })
      } else {
        Object.entries(props.value).forEach(([k, v]) => {
          const childPath = `${props.path}.${k}`
          const childPaths = getMatchedPathsForData(v, k, childPath, f)
          paths.push(...childPaths)
        })
      }
    }
    return paths
  },
  revealPath: (targetPath: string, force: boolean = true) => {
    // 只有在需要强制显示（如从编辑器同步）时才设置 forceVisible
    // 如果是搜索定位，节点本身应该是可见的
    if (force) {
      forceVisible.value = true
    }

    // 1. 如果目标路径就是当前路径
    if (targetPath === props.path) {
      isHighlighted.value = true
      if (nodeRef.value) {
        nodeRef.value.scrollIntoView({ behavior: 'smooth', block: 'center' })
      }
      setTimeout(() => {
        isHighlighted.value = false
      }, 600)
      return true
    }

    // 2. 如果目标路径包含当前路径，说明在子节点中
    const isParentPath = props.path === '$' ? targetPath.startsWith('$') : (
      targetPath.startsWith(props.path) && 
      (targetPath[props.path.length] === '.' || targetPath[props.path.length] === '[')
    )

    if (isParentPath) {
      // 必须展开才能看到子节点
      if (!expanded.value) {
        expanded.value = true
      }
      
      // 递归寻找
      const tryRevealInChildren = (attempts = 0) => {
        if (attempts > 15) return

        const activeChildren = childRefs.value.filter(c => c !== null)
        
        if (activeChildren.length === 0 && isExpandable.value && childCount.value > 0) {
          setTimeout(() => tryRevealInChildren(attempts + 1), 50)
          return
        }

        let found = false
        for (const child of activeChildren) {
          if (child && child.revealPath(targetPath, force)) {
            found = true
            break
          }
        }

        if (!found) {
          setTimeout(() => tryRevealInChildren(attempts + 1), 50)
        }
      }

      setTimeout(() => tryRevealInChildren(), 50)
      return true
    }
    return false
  }
})

const type = computed(() => {
  if (props.value === null) return 'null'
  if (Array.isArray(props.value)) return 'array'
  return typeof props.value
})

const isExpandable = computed(() => {
  return type.value === 'object' || type.value === 'array'
})

const childCount = computed(() => {
  if (type.value === 'array') return props.value.length
  if (type.value === 'object') return Object.keys(props.value).length
  return 0
})

const typeIcon = computed(() => {
  switch (type.value) {
    case 'object': return ObjectIcon
    case 'array': return ArrayIcon
    case 'string': return StringIcon
    case 'number': return NumberIcon
    case 'boolean': return BooleanIcon
    default: return NullIcon
  }
})

const typeColor = computed(() => {
  switch (type.value) {
    case 'object': return '#3498db'
    case 'array': return '#e67e22'
    case 'string': return '#ce9178'
    case 'number': return '#b5cea8'
    case 'boolean': return '#569cd6'
    default: return '#808080'
  }
})

const valueClass = computed(() => {
  switch (type.value) {
    case 'string': return 'text-orange-600 dark:text-[#ce9178]'
    case 'number': return 'text-green-600 dark:text-[#b5cea8]'
    case 'boolean': return 'text-blue-600 dark:text-[#569cd6]'
    case 'null': return 'text-gray-500 italic'
    default: return ''
  }
})

const displayValue = computed(() => {
  if (type.value === 'object') return '{...}'
  if (type.value === 'array') return '[...]'
  if (type.value === 'string') return `"${props.value}"`
  return String(props.value)
})
</script>
