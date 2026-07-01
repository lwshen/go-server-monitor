<script setup lang="ts">
// Settings panel. Loads GET /api/admin/settings which returns a flat map of
// non-secret string values plus "<key>_set":boolean flags for write-only
// secrets. We render a curated set of known fields (typed controls) and submit
// only the values the operator actually changed. Secrets show a set/not-set
// badge and an overwrite input that is sent only when non-empty.
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getSettings, saveSettings } from '@/services/api'

const emit = defineEmits<{ unauthorized: [] }>()
const { t } = useI18n()

type FieldKind = 'text' | 'number' | 'bool' | 'theme' | 'lang'
interface FieldDef {
  key: string
  kind: FieldKind
  section: 'general' | 'retention' | 'notifications'
}
interface SecretDef {
  key: string
  section: 'notifications'
}

// Non-secret settings, grouped for the form. field.<key> supplies the label.
const FIELDS: FieldDef[] = [
  { key: 'site_title', kind: 'text', section: 'general' },
  { key: 'theme_default', kind: 'theme', section: 'general' },
  { key: 'lang_default', kind: 'lang', section: 'general' },
  { key: 'is_public', kind: 'bool', section: 'general' },
  { key: 'retention_days', kind: 'number', section: 'retention' },
  { key: 'offline_factor', kind: 'number', section: 'retention' },
  { key: 'notify_enabled', kind: 'bool', section: 'notifications' },
  { key: 'telegram_chat_id', kind: 'text', section: 'notifications' },
  { key: 'webhook_url', kind: 'text', section: 'notifications' },
]
// Write-only secrets — never echoed by the API; only "<key>_set" is returned.
const SECRETS: SecretDef[] = [{ key: 'telegram_bot_token', section: 'notifications' }]

const SECTIONS: Array<'general' | 'retention' | 'notifications'> = [
  'general',
  'retention',
  'notifications',
]

const loading = ref(true)
const saving = ref(false)
const saved = ref(false)
const error = ref('')

// Reactive string-backed model (checkboxes map to "true"/"false" strings so the
// backend receives the same string-value shape it emits).
const model = reactive<Record<string, string>>({})
const secretSet = reactive<Record<string, boolean>>({})
const secretInput = reactive<Record<string, string>>({})

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const data = await getSettings()
    for (const f of FIELDS) {
      const raw = data[f.key]
      model[f.key] = raw === undefined || raw === null ? '' : String(raw)
    }
    for (const s of SECRETS) {
      secretSet[s.key] = data[`${s.key}_set`] === true || data[`${s.key}_set`] === 'true'
      secretInput[s.key] = ''
    }
  } catch (e: unknown) {
    handleErr(e)
  } finally {
    loading.value = false
  }
}

function handleErr(e: unknown): void {
  const status = (e as { response?: { status?: number } })?.response?.status
  if (status === 401) {
    emit('unauthorized')
    return
  }
  error.value = t('common.error')
}

function fieldsFor(section: string): FieldDef[] {
  return FIELDS.filter((f) => f.section === section)
}
function secretsFor(section: string): SecretDef[] {
  return SECRETS.filter((s) => s.section === section)
}
function label(key: string): string {
  return t(`admin.field.${key}`)
}

const hasNotificationSection = computed(
  () => fieldsFor('notifications').length > 0 || secretsFor('notifications').length > 0,
)

async function save(): Promise<void> {
  saving.value = true
  saved.value = false
  error.value = ''
  try {
    const payload: Record<string, string> = {}
    for (const f of FIELDS) payload[f.key] = model[f.key] ?? ''
    // Only send a secret when the operator typed a replacement value.
    for (const s of SECRETS) {
      const v = secretInput[s.key]?.trim()
      if (v) payload[s.key] = v
    }
    await saveSettings(payload)
    saved.value = true
    // Reflect any newly-set secrets and clear the inputs.
    for (const s of SECRETS) {
      if (secretInput[s.key]?.trim()) {
        secretSet[s.key] = true
        secretInput[s.key] = ''
      }
    }
    setTimeout(() => (saved.value = false), 2000)
  } catch (e: unknown) {
    handleErr(e)
  } finally {
    saving.value = false
  }
}

watch(model, () => (saved.value = false), { deep: true })

void load()
</script>

<template>
  <div class="settings">
    <div v-if="loading" class="row gap-2 dim">
      <span class="spinner" aria-hidden="true"></span>
      {{ t('common.loading') }}
    </div>

    <form v-else class="stack gap-6" @submit.prevent="save">
      <template v-for="section in SECTIONS" :key="section">
        <fieldset
          v-if="section !== 'notifications' || hasNotificationSection"
          class="settings-group stack gap-4"
        >
          <legend class="eyebrow">{{ t(`admin.section.${section}`) }}</legend>

          <div class="settings-grid">
            <div v-for="f in fieldsFor(section)" :key="f.key" class="field">
              <label :for="`set-${f.key}`">{{ label(f.key) }}</label>

              <select v-if="f.kind === 'bool'" :id="`set-${f.key}`" v-model="model[f.key]" class="input">
                <option value="true">{{ t('admin.yes') }}</option>
                <option value="false">{{ t('admin.no') }}</option>
              </select>

              <select v-else-if="f.kind === 'theme'" :id="`set-${f.key}`" v-model="model[f.key]" class="input">
                <option value="dark">{{ t('admin.theme.dark') }}</option>
                <option value="light">{{ t('admin.theme.light') }}</option>
              </select>

              <select v-else-if="f.kind === 'lang'" :id="`set-${f.key}`" v-model="model[f.key]" class="input">
                <option value="zh">{{ t('admin.lang.zh') }}</option>
                <option value="en">{{ t('admin.lang.en') }}</option>
              </select>

              <input
                v-else-if="f.kind === 'number'"
                :id="`set-${f.key}`"
                v-model="model[f.key]"
                class="input mono"
                type="number"
                inputmode="numeric"
              />

              <input v-else :id="`set-${f.key}`" v-model="model[f.key]" class="input" type="text" autocomplete="off" />
            </div>

            <div v-for="s in secretsFor(section)" :key="s.key" class="field">
              <label :for="`set-${s.key}`">
                {{ label(s.key) }}
                <span class="badge" :class="secretSet[s.key] ? 'is-online' : ''">
                  {{ secretSet[s.key] ? t('admin.secretSet') : t('admin.secretUnset') }}
                </span>
              </label>
              <input
                :id="`set-${s.key}`"
                v-model="secretInput[s.key]"
                class="input mono"
                type="password"
                autocomplete="new-password"
                :placeholder="secretSet[s.key] ? '••••••••' : ''"
              />
              <small class="faint">{{ t('admin.overwriteSecret') }}</small>
            </div>
          </div>
        </fieldset>
      </template>

      <div class="row gap-3">
        <button type="submit" class="btn btn-primary" :disabled="saving">
          <span v-if="saving" class="spinner sp-sm" aria-hidden="true"></span>
          {{ saving ? t('admin.saving') : t('admin.save') }}
        </button>
        <span v-if="saved" class="text-online saved-flag">✓ {{ t('admin.saved') }}</span>
        <span v-if="error" class="text-offline">{{ error }}</span>
      </div>
    </form>
  </div>
</template>

<style scoped>
.settings-group {
  border: none;
  padding: 0;
}
.settings-group legend {
  padding: 0;
  margin-bottom: var(--sp-2);
}
.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: var(--sp-4);
}
.field label {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-2);
}
.field small {
  font-size: var(--fs-xs);
}
.sp-sm {
  width: 14px;
  height: 14px;
  border-width: 2px;
}
.saved-flag {
  font-size: var(--fs-sm);
  font-weight: 600;
}
</style>
