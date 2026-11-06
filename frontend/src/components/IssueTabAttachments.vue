<template>
  <AttachmentManager
    :issue-id="issueId"
    :project-id="projectId"
    :read-only="readOnly"
    @count-change="onCountChange"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import AttachmentManager from './AttachmentManager.vue'
import { useProjectPermissions } from '@/composables/usePermission'

const props = defineProps<{
  issueId: number
  projectId: number
}>()

const emit = defineEmits<{
  (e: 'attachmentCountChange', count: number): void
}>()

const { can } = useProjectPermissions(() => props.projectId)
const readOnly = computed(() => !can('issue:edit'))

function onCountChange(count: number) {
  emit('attachmentCountChange', count)
}
</script>
