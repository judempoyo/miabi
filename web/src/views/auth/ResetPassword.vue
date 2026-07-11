<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { authApi } from '@/api/auth'
import { apiErrorMessage } from '@/api/client'
import AuthShell from './AuthShell.vue'

const router = useRouter()
const route = useRoute()

const token = (route.query.token as string | undefined) ?? ''
const password = ref('')
const confirm = ref('')
const showPassword = ref(false)
const capsOn = ref(false)
const error = ref('')
const loading = ref(false)
const done = ref(false)
const passwordInput = ref<HTMLInputElement | null>(null)

const MIN_LEN = 8
// Mirror the backend rules (minLength 8) plus a local match check so the user
// gets immediate feedback before submitting.
const tooShort = computed(() => password.value.length > 0 && password.value.length < MIN_LEN)
const mismatch = computed(() => confirm.value.length > 0 && confirm.value !== password.value)
const canSubmit = computed(
  () => !!token && password.value.length >= MIN_LEN && password.value === confirm.value,
)

onMounted(() => {
  if (!token) {
    error.value = 'This reset link is missing its token. Request a new one.'
    return
  }
  passwordInput.value?.focus()
})

function onCaps(e: KeyboardEvent) {
  capsOn.value = e.getModifierState?.('CapsLock') ?? false
}

async function submit() {
  if (!canSubmit.value) return
  error.value = ''
  loading.value = true
  try {
    await authApi.resetPassword(token, password.value)
    done.value = true
    // Brief confirmation, then send them to sign in with the new password.
    setTimeout(() => router.replace({ name: 'login' }), 2500)
  } catch (e) {
    error.value = apiErrorMessage(e, 'This reset link is invalid or has expired.')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AuthShell
    :title="done ? 'Password updated' : 'Choose a new password'"
    :subtitle="done ? '' : 'Enter a new password for your account.'"
    :error="error"
    :notice="done ? 'Your password has been reset. Redirecting to sign in…' : ''"
  >
    <template v-if="done">
      <RouterLink :to="{ name: 'login' }" class="btn btn-primary auth-submit">
        Continue to sign in
      </RouterLink>
    </template>

    <template v-else-if="!token">
      <RouterLink :to="{ name: 'forgot-password' }" class="btn btn-primary auth-submit">
        Request a new link
      </RouterLink>
    </template>

    <form v-else class="auth-form" @submit.prevent="submit">
      <div class="form-group">
        <label class="form-label">New password</label>
        <div class="password-wrap">
          <input
            ref="passwordInput"
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            class="form-input"
            placeholder="At least 8 characters"
            autocomplete="new-password"
            aria-label="New password"
            :disabled="loading"
            required
            @keyup="onCaps"
            @keydown="onCaps"
          />
          <button
            type="button"
            class="password-toggle"
            :aria-label="showPassword ? 'Hide password' : 'Show password'"
            :aria-pressed="showPassword"
            :title="showPassword ? 'Hide password' : 'Show password'"
            @click="showPassword = !showPassword"
          >
            <span class="mdi" :class="showPassword ? 'mdi-eye-off-outline' : 'mdi-eye-outline'"></span>
          </button>
        </div>
        <p v-if="tooShort" class="form-hint hint-warn">Use at least {{ MIN_LEN }} characters.</p>
        <p v-else-if="capsOn" class="form-hint hint-warn">
          <span class="mdi mdi-apple-keyboard-caps"></span> Caps Lock is on
        </p>
      </div>

      <div class="form-group">
        <label class="form-label">Confirm password</label>
        <input
          v-model="confirm"
          :type="showPassword ? 'text' : 'password'"
          class="form-input"
          placeholder="Re-enter your new password"
          autocomplete="new-password"
          aria-label="Confirm password"
          :disabled="loading"
          required
        />
        <p v-if="mismatch" class="form-hint hint-warn">Passwords don't match.</p>
      </div>

      <button class="btn btn-primary auth-submit" :disabled="loading || !canSubmit">
        <span v-if="loading" class="mdi mdi-loading mdi-spin"></span>
        {{ loading ? 'Updating…' : 'Update password' }}
      </button>
    </form>

    <template #footer>
      <RouterLink :to="{ name: 'login' }" class="auth-back">
        <span class="mdi mdi-arrow-left"></span> Back to sign in
      </RouterLink>
    </template>
  </AuthShell>
</template>

<style scoped>
.auth-submit {
  width: 100%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 4px;
  text-decoration: none;
}
.password-wrap {
  position: relative;
}
.password-wrap .form-input {
  padding-right: 40px;
}
.password-toggle {
  position: absolute;
  top: 50%;
  right: 8px;
  transform: translateY(-50%);
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-muted);
  font-size: 18px;
  display: flex;
  align-items: center;
  padding: 4px;
}
.password-toggle:hover {
  color: var(--text-primary);
}
.hint-warn {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 6px;
  color: var(--warning-700, var(--text-muted));
}
.hint-warn .mdi {
  font-size: 15px;
}
.auth-back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--text-muted);
  text-decoration: none;
}
.auth-back:hover {
  color: var(--text-primary);
}
</style>
