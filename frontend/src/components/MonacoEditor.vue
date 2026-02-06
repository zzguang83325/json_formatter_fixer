<template>
  <div class="w-full h-full flex flex-col relative" @contextmenu.prevent="handleContextMenu">
    <div ref="editorContainer" class="flex-1"></div>
    
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

    <!-- 注入全局样式以覆盖 Monaco 高亮和背景 -->
    <component is="style">
      /* 这里的样式会注入到全局，尝试覆盖 Monaco 内部元素 */
      .search-highlight-bg {
        background-color: #ffd33d !important; /* 使用与右侧一致的黄色 */
        opacity: 0.6; /* 使用透明度确保文字可见 */
        border-radius: 2px;
      }
      .search-highlight-text {
        font-weight: bold !important;
      }
      .dark .search-highlight-bg {
        background-color: #ffd33d !important;
        opacity: 0.4;
        border-radius: 2px;
      }
      .dark .search-highlight-text {
        font-weight: bold !important;
      }
    </component>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, nextTick, computed } from 'vue'
import * as monaco from 'monaco-editor'
import { NDropdown, useMessage } from 'naive-ui'
import { useAppStore } from '../store/app'
import { GetPathOffset, GetPathByOffset } from '../../wailsjs/go/main/App'

// 防抖函数
function debounce(fn: Function, delay: number) {
  let timer: any = null
  return function(this: any, ...args: any[]) {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => {
      fn.apply(this, args)
    }, delay)
  }
}

const props = defineProps<{
  value: string
  readOnly?: boolean
  theme?: string
}>()

const store = useAppStore()
const message = useMessage()
const emit = defineEmits<{
  (e: 'update:value', val: string): void
  (e: 'cursor-path', path: string): void
  (e: 'paste'): void
}>()

const editorContainer = ref<HTMLElement | null>(null)
let editor: monaco.editor.IStandaloneCodeEditor | null = null
let decorations = ref<string[]>([])
let isRevealing = false

// 右键菜单相关
const showContextMenu = ref(false)
const contextMenuX = ref(0)
const contextMenuY = ref(0)

const menuOptions = computed(() => {
  const options = [
    { label: '复制', key: 'copy' },
    { label: '全选', key: 'select-all' }
  ]
  if (!props.readOnly) {
    options.splice(1, 0, { label: '粘贴', key: 'paste' })
    options.push({ label: '清空', key: 'clear' })
  }
  return options
})

const handleContextMenu = (e: MouseEvent) => {
  showContextMenu.value = false
  nextTick(() => {
    contextMenuX.value = e.clientX
    contextMenuY.value = e.clientY
    showContextMenu.value = true
  })
}

const handleMenuSelect = async (key: string) => {
  showContextMenu.value = false
  if (!editor) return

  switch (key) {
    case 'copy':
      editor.focus()
      document.execCommand('copy')
      // Monaco 的 trigger 也可以，但 execCommand 对选中内容更通用
      // editor.trigger('source', 'editor.action.clipboardCopyAction')
      break
    case 'paste':
      editor.focus()
      try {
        const text = await navigator.clipboard.readText()
        if (text) {
          const selection = editor.getSelection()
          if (selection) {
            editor.executeEdits('paste', [
              {
                range: selection,
                text: text,
                forceMoveMarkers: true
              }
            ])
          }
        }
      } catch (err) {
        // 如果 navigator.clipboard 不可用，尝试触发 monaco 粘贴
        editor.trigger('source', 'editor.action.clipboardPasteAction', null)
      }
      break
    case 'select-all':
      editor.focus()
      editor.trigger('source', 'editor.action.selectAll', null)
      break
    case 'clear':
      editor.setValue('')
      emit('update:value', '')
      break
  }
}

onMounted(() => {
  if (editorContainer.value) {
    // 定义自定义主题以支持背景色同步
    monaco.editor.defineTheme('custom-yellow', {
      base: 'vs',
      inherit: true,
      rules: [],
      colors: { 'editor.background': '#fdf6e3' }
    })
    monaco.editor.defineTheme('custom-green', {
      base: 'vs',
      inherit: true,
      rules: [],
      colors: { 'editor.background': '#e8f5e9' }
    })
    monaco.editor.defineTheme('custom-blue', {
      base: 'vs',
      inherit: true,
      rules: [],
      colors: { 'editor.background': '#e3f2fd' }
    })

    const getInitialTheme = () => {
      if (store.themeColor === 'yellow') return 'custom-yellow'
      if (store.themeColor === 'green') return 'custom-green'
      if (store.themeColor === 'blue') return 'custom-blue'
      return store.isDarkMode ? 'vs-dark' : 'vs'
    }

    editor = monaco.editor.create(editorContainer.value, {
      value: props.value,
      language: 'json',
      theme: props.theme || getInitialTheme(),
      automaticLayout: true,
      tabSize: 4,
      fontSize: 14,
      scrollBeyondLastLine: false,
      minimap: { 
        enabled: false,
      },
      folding: true,
      bracketPairColorization: { enabled: true },
      formatOnPaste: false,
      formatOnType: false,
      readOnly: props.readOnly || false,
      contextmenu: false,
      renderLineHighlight: 'none', // 禁用行高亮以减少背景色冲突
    })

    editor.onDidChangeModelContent(() => {
      const val = editor?.getValue() || ''
      if (val !== props.value) {
        emit('update:value', val)
      }
    })

    editor.onDidPaste(() => {
      emit('paste')
    })

    // 监听光标位置变化（增加防抖）
    const debouncedCursorChange = debounce(async (position: monaco.IPosition) => {
      if (!editor || isRevealing) return
      
      // 检查 Wails 绑定是否就绪
      // @ts-ignore
      if (!window.go || !window.go.main) return

      const model = editor.getModel()
      if (!model) return
      
      const offset = model.getOffsetAt(position)
      const content = model.getValue()
      
      try {
        const path = await GetPathByOffset(content, offset)
        if (path) {
          emit('cursor-path', path)
        }
      } catch (err) {
        console.error('Failed to get path by offset:', err)
      }
    }, 100)

    editor.onDidChangeCursorPosition((e) => {
      debouncedCursorChange(e.position)
    })
  }
})

watch(() => props.theme, (newTheme) => {
  if (editor && newTheme) {
    monaco.editor.setTheme(newTheme)
  }
})

watch(() => store.themeColor, (newTheme) => {
  if (editor) {
    if (newTheme === 'yellow') monaco.editor.setTheme('custom-yellow')
    else if (newTheme === 'green') monaco.editor.setTheme('custom-green')
    else if (newTheme === 'blue') monaco.editor.setTheme('custom-blue')
    else if (newTheme === 'dark') monaco.editor.setTheme('vs-dark')
    else monaco.editor.setTheme('vs')
    
    // 强制触发一次重绘，确保背景色更新
    setTimeout(() => {
      editor?.layout()
    }, 0)
  }
})

watch(() => props.value, (newVal) => {
  if (editor && newVal !== editor.getValue()) {
    editor.setValue(newVal)
    // 内容更新后（如美化格式），重置横向滚动条到最左侧
    editor.setScrollLeft(0)
  }
})

watch(() => store.globalFilter, (newFilter) => {
  if (editor) {
    const model = editor.getModel()
    if (!model) return

    if (!newFilter) {
      // 清除高亮
      decorations.value = editor.deltaDecorations(decorations.value, [])
      return
    }

    // 查找匹配项
    const matches = model.findMatches(newFilter, false, false, false, null, true)
    
    // 使用 Decorations (装饰器) 而不是 Selections (选中项) 来实现更显眼的高亮
    const newDecorations = matches.map(m => ({
      range: m.range,
      options: {
        inlineClassName: 'search-highlight',
        isWholeLine: false,
        stickiness: monaco.editor.TrackedRangeStickiness.NeverGrowsWhenTypingAtEdges
      }
    }))

    decorations.value = editor.deltaDecorations(decorations.value, newDecorations)

    if (matches.length > 0) {
      // 滚动到第一个匹配项
      editor.revealRangeInCenterIfOutsideViewport(matches[0].range)
    }
  }
}, { immediate: true })

onBeforeUnmount(() => {
  if (editor) {
    editor.dispose()
  }
})

// 暴露方法给父组件
defineExpose({
  revealPath: async (path: string) => {
    if (!editor) return

    // 标记正在定位，防止触发回传给树
    isRevealing = true

    // 检查 Wails 绑定是否就绪
    // @ts-ignore
    if (!window.go || !window.go.main) {
      console.warn('Wails bindings not ready yet')
      isRevealing = false
      return
    }

    const content = editor.getValue()
    if (!content) {
      isRevealing = false
      return
    }

    try {
        const pathInfo = await GetPathOffset(content, path)
        if (pathInfo && pathInfo.offset >= 0) {
          const model = editor.getModel()
          if (!model) {
            isRevealing = false
            return
          }
          
          // 将字节偏移转换为行列
          const startPos = model.getPositionAt(pathInfo.offset)
          const endPos = model.getPositionAt(pathInfo.offset + pathInfo.length)
          
          // 精确高亮匹配范围
          const range = new monaco.Range(
            startPos.lineNumber, 
            startPos.column, 
            endPos.lineNumber, 
            endPos.column
          )
          
          // 滚动并高亮
          editor.revealRangeInCenterIfOutsideViewport(range)
          editor.setPosition(startPos)

          const tempDecorations = editor.deltaDecorations([], [{
            range: range,
            options: {
              className: 'search-highlight-bg',
              inlineClassName: 'search-highlight-text',
              isWholeLine: false,
              stickiness: monaco.editor.TrackedRangeStickiness.NeverGrowsWhenTypingAtEdges
            }
          }])
          
          // 2秒后清除临时高亮
          setTimeout(() => {
            if (editor) {
              editor.deltaDecorations(tempDecorations, [])
            }
          }, 2000)
      }
    } catch (e) {
      console.error('Failed to get path offset:', e)
    } finally {
      // 延迟重置，确保 debounce 的事件被忽略
      setTimeout(() => {
        isRevealing = false
      }, 200)
    }
  }
})
</script>
