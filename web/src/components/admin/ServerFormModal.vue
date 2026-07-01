<script setup lang="ts">
// Add / edit dialog for a single server. In "add" mode it collects the minimal
// fields the backend needs (name + optional group / expiry); in "edit" mode it
// exposes the mutable metadata (name, group, expiry, report interval, hidden,
// notify). The parent owns the API call — this component only gathers input and
// emits the payload.
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Server } from '@/services/api'
import AdminModal from './AdminModal.vue'

const props = defineProps<{ server: Server | null }>()
const emit = defineEmits<{
  close: []
  submit: [payload: Record<string, unknown>]
}>()

const { t } = useI18n()

const isEdit = computed(() => props.server !== null)
const saving = ref(false)
const error = ref('')

const form = reactive({
  name: props.server?.name ?? '',
  server_group: props.server?.server_group ?? '',
  expire_date: props.server?.expire_date ?? '',
  report_interval: props.server?.report_interval ?? 60,
  is_hidden: props.server?.is_hidden ?? false,
  notify: props.server?.notify ?? true,
})

/** Expose saving/error to the parent so it can surface failures and stop spin. */
defineExpose({
  setSaving: (v: boolean) => (saving.value = v),
  setError: (msg: string) => (error.value = msg),
})

function onSubmit(): void {
  error.value = ''
  const name = form.name.trim()
  if (!name) {
    error.value = t('admin.nameRequired')
    return
  }
  if (isEdit.value) {
    emit('submit', {
      id: props.server!.id,
      name,
      server_group: form.server_group.trim(),
      expire_date: form.expire_date.trim(),
      report_interval: Number(form.report_interval) || 0,
      is_hidden: form.is_hidden,
      notify: form.notify,
    })
  } else {
    emit('submit', {
      name,
      server_group: form.server_group.trim() || undefined,
      expire_date: form.expire_date.trim() || undefined,
    })
  }
}
</script>

<template>
  <AdminModal :title="isEdit ? t('admin.editServer') : t('admin.addServer')" @close="emit('close')">
    <form class="stack gap-4" @submit.prevent="onSubmit">
      <div class="field">
        <label for="sf-name">{{ t('admin.name') }}</label>
        <input
          id="sf-name"
          v-model="form.name"
          class="input"
          type="text"
          autocomplete="off"
          :placeholder="t('admin.newServer')"
        />
      </div>

      <div class="field">
        <label for="sf-group">{{ t('admin.group') }}</label>
        <input id="sf-group" v-model="form.server_group" class="input" type="text" autocomplete="off" />
      </div>

      <div class="field">
        <label for="sf-expire">{{ t('admin.expireDate') }}</label>
        <input
          id="sf-expire"
          v-model="form.expire_date"
          class="input"
          type="text"
          placeholder="YYYY-MM-DD"
          autocomplete="off"
        />
      </div>

      <template v-if="isEdit">
        <div class="field">
          <label for="sf-report">{{ t('admin.reportInterval') }}</label>
          <input id="sf-report" v-model.number="form.report_interval" class="input mono" type="number" min="0" />
        </div>

        <div class="row gap-4 wrap">
          <label class="check row gap-2">
            <input v-model="form.is_hidden" type="checkbox" />
            <span>{{ t('admin.hidden') }}</span>
          </label>
          <label class="check row gap-2">
            <input v-model="form.notify" type="checkbox" />
            <span>{{ t('admin.notify') }}</span>
          </label>
        </div>
      </template>

      <p v-if="error" class="form-error text-offline">{{ error }}</p>
    </form>

    <template #footer>
      <button type="button" class="btn btn-ghost" :disabled="saving" @click="emit('close')">
        {{ t('common.cancel') }}
      </button>
      <button type="button" class="btn btn-primary" :disabled="saving" @click="onSubmit">
        <span v-if="saving" class="spinner" aria-hidden="true"></span>
        {{ saving ? t('admin.saving') : t('admin.save') }}
      </button>
    </template>
  </AdminModal>
</template>

<style scoped>
.check {
  font-size: var(--fs-sm);
  cursor: pointer;
  user-select: none;
}
.check input {
  width: 15px;
  height: 15px;
  accent-color: var(--accent);
  cursor: pointer;
}
.form-error {
  font-size: var(--fs-sm);
}
.spinner {
  width: 14px;
  height: 14px;
  border-width: 2px;
}
</style>
