<template>
  <n-config-provider :theme="theme" :locale="zhCN" :date-locale="dateZhCN">
    <n-loading-bar-provider>
      <n-message-provider>
        <n-dialog-provider>
          <main-content />
        </n-dialog-provider>
      </n-message-provider>
    </n-loading-bar-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { 
  NConfigProvider, NLoadingBarProvider, NMessageProvider, NDialogProvider,
  darkTheme, zhCN, dateZhCN
} from 'naive-ui'
import { useAppStore } from './store/app'
import MainContent from './MainContent.vue'

const store = useAppStore()
const theme = computed(() => store.isDarkMode ? darkTheme : null)
</script>

<style>
.no-scrollbar::-webkit-scrollbar {
  display: none;
}
.no-scrollbar {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
</style>
