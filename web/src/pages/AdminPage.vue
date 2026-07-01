<script setup lang="ts">
// Admin backend. Two states:
//   1. Not authed  -> login form (username default 'admin').
//   2. Authed      -> Servers management + Settings + danger zone + logout.
// The auth store mirrors the JWT in localStorage; the axios interceptor clears
// the token on any 401, so every admin call funnels through onUnauthorized(),
// which resyncs the store and drops back to the login view.
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import {
  adminListServers,
  addServer,
  editServer,
  deleteServer,
  dbRebuild,
  type Server,
} from '@/services/api'
import { dash, dateTime } from '@/utils/format'
import ServerFormModal from '@/components/admin/ServerFormModal.vue'
import SettingsSection from '@/components/admin/SettingsSection.vue'

const { t } = useI18n()
const auth = useAuthStore()
const { isAuthed } = storeToRefs(auth)

// ---- Login -----------------------------------------------------------------
const creds = reactive({ username: 'admin', password: '' })
const loginError = ref('')
const loggingIn = ref(false)

async function onLogin(): Promise<void> {
  loginError.value = ''
  loggingIn.value = true
  try {
    await auth.login(creds.username.trim(), creds.password)
    creds.password = ''
    await loadServers()
  } catch (e: unknown) {
    const status = (e as { response?: { status?: number } })?.response?.status
    loginError.value = status === 401 ? t('admin.loginFailed') : t('common.error')
  } finally {
    loggingIn.value = false
  }
}

function logout(): void {
  auth.logout()
  servers.value = []
}

/** Called by any admin action that hit a 401 — the token is already cleared. */
function onUnauthorized(): void {
  auth.sync()
  loginError.value = t('admin.sessionExpired')
}

function isAuthError(e: unknown): boolean {
  return (e as { response?: { status?: number } })?.response?.status === 401
}

// ---- Servers ---------------------------------------------------------------
const servers = ref<Server[]>([])
const loadingServers = ref(false)
const serversError = ref('')

async function loadServers(): Promise<void> {
  loadingServers.value = true
  serversError.value = ''
  try {
    servers.value = await adminListServers()
  } catch (e: unknown) {
    if (isAuthError(e)) return onUnauthorized()
    serversError.value = t('common.error')
  } finally {
    loadingServers.value = false
  }
}

// Add / edit modal state.
const formOpen = ref(false)
const editing = ref<Server | null>(null)
const formRef = ref<InstanceType<typeof ServerFormModal> | null>(null)

function openAdd(): void {
  editing.value = null
  formOpen.value = true
}
function openEdit(s: Server): void {
  editing.value = s
  formOpen.value = true
}
function closeForm(): void {
  formOpen.value = false
  editing.value = null
}

async function onFormSubmit(payload: Record<string, unknown>): Promise<void> {
  formRef.value?.setSaving(true)
  formRef.value?.setError('')
  try {
    if (editing.value) {
      await editServer(payload as { id: string })
    } else {
      const created = await addServer(payload as { name: string })
      // Reveal the install hint for the freshly-created server.
      justAddedId.value = created.id
      hintForId.value = created.id
    }
    closeForm()
    await loadServers()
  } catch (e: unknown) {
    if (isAuthError(e)) {
      closeForm()
      return onUnauthorized()
    }
    formRef.value?.setError(t('common.error'))
  } finally {
    formRef.value?.setSaving(false)
  }
}

// Inline delete confirmation (per-row).
const confirmingDelete = ref<string | null>(null)
const deletingId = ref<string | null>(null)

function askDelete(id: string): void {
  confirmingDelete.value = id
}
function cancelDelete(): void {
  confirmingDelete.value = null
}
async function doDelete(id: string): Promise<void> {
  deletingId.value = id
  try {
    await deleteServer(id)
    confirmingDelete.value = null
    if (justAddedId.value === id) justAddedId.value = null
    await loadServers()
  } catch (e: unknown) {
    if (isAuthError(e)) return onUnauthorized()
    serversError.value = t('common.error')
  } finally {
    deletingId.value = null
  }
}

// ---- Install hint ----------------------------------------------------------
// Shows a copy-paste curl for a server id. The report token is NOT exposed here
// — the server issues it; we only template the id + endpoint.
const justAddedId = ref<string | null>(null)
const hintForId = ref<string | null>(null)
const copiedId = ref<string | null>(null)

function installCommand(id: string): string {
  const base = window.location.origin
  return `curl -sSL ${base}/report | sh -s -- --server ${base} --id ${id}`
}
function toggleHint(id: string): void {
  hintForId.value = hintForId.value === id ? null : id
}
async function copyInstall(id: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(installCommand(id))
    copiedId.value = id
    setTimeout(() => {
      if (copiedId.value === id) copiedId.value = null
    }, 1500)
  } catch {
    /* clipboard unavailable — the command is visible for manual copy */
  }
}

// ---- Danger zone: DB rebuild ----------------------------------------------
const confirmingRebuild = ref(false)
const rebuilding = ref(false)
const rebuilt = ref(false)
const rebuildError = ref('')

async function doRebuild(): Promise<void> {
  rebuilding.value = true
  rebuildError.value = ''
  rebuilt.value = false
  try {
    await dbRebuild()
    confirmingRebuild.value = false
    rebuilt.value = true
    setTimeout(() => (rebuilt.value = false), 3000)
  } catch (e: unknown) {
    if (isAuthError(e)) {
      confirmingRebuild.value = false
      return onUnauthorized()
    }
    rebuildError.value = t('common.error')
  } finally {
    rebuilding.value = false
  }
}

const statusLabel = computed(() => (s: Server) => (s.online ? t('common.online') : t('common.offline')))

// If we arrive already authenticated, fetch immediately.
if (isAuthed.value) void loadServers()
</script>

<template>
  <div class="container admin">
    <!-- ============================ LOGIN ============================ -->
    <div v-if="!isAuthed" class="login-wrap">
      <form class="login-card card stack gap-4" @submit.prevent="onLogin">
        <div class="stack gap-1">
          <div class="row gap-2">
            <span class="brand-dot" aria-hidden="true"></span>
            <h1 class="login-title">{{ t('admin.login') }}</h1>
          </div>
          <p class="dim login-sub">{{ t('admin.loginSubtitle') }}</p>
        </div>

        <div class="field">
          <label for="login-user">{{ t('admin.username') }}</label>
          <input
            id="login-user"
            v-model="creds.username"
            class="input"
            type="text"
            autocomplete="username"
          />
        </div>

        <div class="field">
          <label for="login-pass">{{ t('admin.password') }}</label>
          <input
            id="login-pass"
            v-model="creds.password"
            class="input"
            type="password"
            autocomplete="current-password"
          />
        </div>

        <p v-if="loginError" class="text-offline login-err">{{ loginError }}</p>

        <button type="submit" class="btn btn-primary login-btn" :disabled="loggingIn">
          <span v-if="loggingIn" class="spinner sp-sm" aria-hidden="true"></span>
          {{ loggingIn ? t('admin.signingIn') : t('admin.signIn') }}
        </button>
      </form>
    </div>

    <!-- ============================ AUTHED ============================ -->
    <div v-else class="stack gap-8">
      <header class="row between wrap gap-3 admin-header">
        <h1 class="page-title">{{ t('nav.admin') }}</h1>
        <button type="button" class="btn btn-ghost" @click="logout">{{ t('common.logout') }}</button>
      </header>

      <!-- ---- Servers ---- -->
      <section class="stack gap-4">
        <div class="row between wrap gap-3">
          <div class="stack gap-1">
            <h2 class="section-title">{{ t('admin.servers') }}</h2>
            <p class="dim section-desc">{{ t('admin.serversDesc') }}</p>
          </div>
          <button type="button" class="btn btn-primary" @click="openAdd">+ {{ t('admin.add') }}</button>
        </div>

        <p v-if="serversError" class="text-offline">{{ serversError }}</p>

        <div v-if="loadingServers" class="row gap-2 dim">
          <span class="spinner sp-sm" aria-hidden="true"></span>{{ t('common.loading') }}
        </div>

        <p v-else-if="servers.length === 0" class="empty dim">{{ t('admin.noServers') }}</p>

        <div v-else class="table-scroll">
          <table class="data">
            <thead>
              <tr>
                <th>{{ t('admin.status') }}</th>
                <th>{{ t('admin.name') }}</th>
                <th>{{ t('admin.group') }}</th>
                <th>{{ t('admin.id') }}</th>
                <th>{{ t('admin.hidden') }}</th>
                <th>{{ t('admin.expireDate') }}</th>
                <th class="col-actions">{{ t('admin.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <template v-for="s in servers" :key="s.id">
                <tr>
                  <td>
                    <span class="row gap-2">
                      <span class="status-dot" :class="s.online ? 'is-online' : 'is-offline'"></span>
                      <span class="faint status-txt">{{ statusLabel(s) }}</span>
                    </span>
                  </td>
                  <td class="name-cell">{{ s.name }}</td>
                  <td class="dim">{{ dash(s.server_group) }}</td>
                  <td>
                    <code class="id-chip mono">{{ s.id }}</code>
                  </td>
                  <td>
                    <span class="badge" :class="s.is_hidden ? '' : 'is-online'">
                      {{ s.is_hidden ? t('admin.hidden') : t('admin.visible') }}
                    </span>
                  </td>
                  <td class="dim mono">{{ dash(s.expire_date) }}</td>
                  <td class="col-actions">
                    <div class="row gap-1">
                      <button type="button" class="btn btn-sm" @click="toggleHint(s.id)">
                        {{ t('admin.installHint') }}
                      </button>
                      <button type="button" class="btn btn-sm" @click="openEdit(s)">
                        {{ t('admin.edit') }}
                      </button>
                      <button
                        type="button"
                        class="btn btn-sm btn-danger"
                        @click="askDelete(s.id)"
                      >
                        {{ t('admin.delete') }}
                      </button>
                    </div>
                  </td>
                </tr>

                <!-- Inline delete confirmation -->
                <tr v-if="confirmingDelete === s.id" class="inline-row">
                  <td colspan="7">
                    <div class="row between wrap gap-3 confirm-bar">
                      <span class="text-offline">{{ t('admin.confirmDelete') }}</span>
                      <div class="row gap-2">
                        <button type="button" class="btn btn-sm" @click="cancelDelete">
                          {{ t('common.cancel') }}
                        </button>
                        <button
                          type="button"
                          class="btn btn-sm btn-danger"
                          :disabled="deletingId === s.id"
                          @click="doDelete(s.id)"
                        >
                          <span v-if="deletingId === s.id" class="spinner sp-sm" aria-hidden="true"></span>
                          {{ t('admin.delete') }}
                        </button>
                      </div>
                    </div>
                  </td>
                </tr>

                <!-- Install hint -->
                <tr v-if="hintForId === s.id" class="inline-row">
                  <td colspan="7">
                    <div class="stack gap-2 install-box">
                      <div class="row between wrap gap-2">
                        <span class="eyebrow">{{ t('admin.installHint') }}</span>
                        <button type="button" class="btn btn-sm" @click="copyInstall(s.id)">
                          {{ copiedId === s.id ? t('admin.copied') : t('admin.copy') }}
                        </button>
                      </div>
                      <p class="faint install-desc">{{ t('admin.installDesc') }}</p>
                      <pre class="install-cmd mono"><code>{{ installCommand(s.id) }}</code></pre>
                      <p v-if="justAddedId === s.id" class="text-online created-flag">
                        ✓ {{ t('admin.serverAdded') }}
                      </p>
                    </div>
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
      </section>

      <!-- ---- Settings ---- -->
      <section class="stack gap-4">
        <div class="stack gap-1">
          <h2 class="section-title">{{ t('admin.settings') }}</h2>
          <p class="dim section-desc">{{ t('admin.settingsDesc') }}</p>
        </div>
        <div class="card">
          <SettingsSection @unauthorized="onUnauthorized" />
        </div>
      </section>

      <!-- ---- Danger zone ---- -->
      <section class="stack gap-4">
        <div class="stack gap-1">
          <h2 class="section-title text-offline">{{ t('admin.dangerZone') }}</h2>
          <p class="dim section-desc">{{ t('admin.dangerDesc') }}</p>
        </div>
        <div class="card danger-card">
          <div class="row between wrap gap-3">
            <div class="stack gap-1">
              <strong>{{ t('admin.rebuild') }}</strong>
              <span class="faint">{{ t('admin.confirmRebuild') }}</span>
            </div>
            <div class="row gap-2">
              <span v-if="rebuilt" class="text-online">✓ {{ t('admin.rebuilt') }}</span>
              <span v-if="rebuildError" class="text-offline">{{ rebuildError }}</span>

              <template v-if="!confirmingRebuild">
                <button type="button" class="btn btn-danger" @click="confirmingRebuild = true">
                  {{ t('admin.rebuild') }}
                </button>
              </template>
              <template v-else>
                <button type="button" class="btn btn-sm" :disabled="rebuilding" @click="confirmingRebuild = false">
                  {{ t('common.cancel') }}
                </button>
                <button type="button" class="btn btn-sm btn-danger" :disabled="rebuilding" @click="doRebuild">
                  <span v-if="rebuilding" class="spinner sp-sm" aria-hidden="true"></span>
                  {{ rebuilding ? t('admin.rebuilding') : t('admin.confirm') }}
                </button>
              </template>
            </div>
          </div>
        </div>
      </section>
    </div>

    <!-- ---- Add / Edit modal ---- -->
    <ServerFormModal
      v-if="formOpen"
      ref="formRef"
      :server="editing"
      @close="closeForm"
      @submit="onFormSubmit"
    />
  </div>
</template>

<style scoped>
.admin {
  padding-block: var(--sp-5) var(--sp-8);
}

/* ---- Login ---- */
.login-wrap {
  display: flex;
  justify-content: center;
  padding-top: var(--sp-8);
}
.login-card {
  width: 100%;
  max-width: 380px;
  box-shadow: var(--shadow);
}
.brand-dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 25%, transparent);
}
.login-title {
  font-size: var(--fs-xl);
  letter-spacing: -0.01em;
}
.login-sub {
  font-size: var(--fs-sm);
}
.login-err {
  font-size: var(--fs-sm);
}
.login-btn {
  margin-top: var(--sp-1);
}

/* ---- Authed layout ---- */
.admin-header {
  border-bottom: 1px solid var(--border);
  padding-bottom: var(--sp-4);
}
.section-title {
  font-size: var(--fs-lg);
}
.section-desc {
  font-size: var(--fs-sm);
}
.empty {
  padding: var(--sp-6);
  text-align: center;
  border: 1px dashed var(--border);
  border-radius: var(--radius);
  font-size: var(--fs-sm);
}

/* ---- Table cells ---- */
.name-cell {
  font-weight: 600;
}
.status-txt {
  font-size: var(--fs-xs);
}
.id-chip {
  font-size: var(--fs-xs);
  padding: 2px var(--sp-2);
  border-radius: var(--radius-sm);
  background: var(--surface-2);
  border: 1px solid var(--border);
  color: var(--text-dim);
}
.col-actions {
  text-align: right;
}
.col-actions .row {
  justify-content: flex-end;
}

/* ---- Inline expansion rows ---- */
.inline-row td {
  background: var(--surface-2);
  white-space: normal;
}
.confirm-bar {
  font-size: var(--fs-sm);
}
.install-box {
  padding-block: var(--sp-1);
}
.install-desc {
  font-size: var(--fs-xs);
}
.install-cmd {
  margin: 0;
  padding: var(--sp-3);
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: var(--fs-xs);
  overflow-x: auto;
  white-space: pre;
  color: var(--text);
}
.created-flag {
  font-size: var(--fs-xs);
  font-weight: 600;
}

/* ---- Danger zone ---- */
.danger-card {
  border-color: color-mix(in srgb, var(--offline) 35%, var(--border));
}

.sp-sm {
  width: 14px;
  height: 14px;
  border-width: 2px;
}
</style>
