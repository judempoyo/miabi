<script setup lang="ts">
// Centered single-card chrome shared by the secondary auth pages (forgot /
// reset password). Mirrors the right-hand panel of Login.vue without the
// marketing hero. The form and a footer link are provided via slots; the alert
// and success notice are driven by props so their styling lives in one place.
defineProps<{
  title: string
  subtitle?: string
  error?: string
  notice?: string
}>()
</script>

<template>
  <div class="auth-shell">
    <div class="auth-card">
      <div class="auth-head">
        <img src="/brand/miabi-mark.svg" alt="Miabi" class="auth-logo" />
        <h1 class="auth-title">{{ title }}</h1>
        <p v-if="subtitle" class="auth-subtitle">{{ subtitle }}</p>
      </div>

      <Transition name="fade">
        <div v-if="error" class="auth-alert" role="alert" aria-live="assertive">
          <span class="mdi mdi-alert-circle-outline"></span>
          <span>{{ error }}</span>
        </div>
      </Transition>
      <Transition name="fade">
        <div v-if="notice" class="auth-notice" role="status" aria-live="polite">
          <span class="mdi mdi-check-circle-outline"></span>
          <span>{{ notice }}</span>
        </div>
      </Transition>

      <slot />

      <div v-if="$slots.footer" class="auth-footer">
        <slot name="footer" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.auth-shell {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 24px;
  background: var(--bg-primary);
}
.auth-card {
  width: 100%;
  max-width: 380px;
}
.auth-head {
  text-align: center;
  margin-bottom: 24px;
}
.auth-logo {
  width: 64px;
  height: 64px;
  margin-bottom: 16px;
}
.auth-title {
  font-size: 1.5rem;
  font-weight: 800;
  letter-spacing: -0.01em;
  margin: 0 0 6px;
  color: var(--text-primary);
}
.auth-subtitle {
  color: var(--text-muted);
  font-size: 14px;
  margin: 0;
}
.auth-alert,
.auth-notice {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  margin-bottom: 16px;
  border-radius: var(--radius);
  font-size: 13px;
}
.auth-alert {
  background: var(--danger-50);
  color: var(--danger-700);
  border: 1px solid var(--danger-100, var(--danger-50));
}
.auth-notice {
  background: var(--success-50);
  color: var(--success-700);
  border: 1px solid var(--success-100, var(--success-50));
}
.auth-alert .mdi,
.auth-notice .mdi {
  font-size: 18px;
  flex-shrink: 0;
}
.auth-footer {
  margin-top: 22px;
  text-align: center;
  font-size: 14px;
  color: var(--text-muted);
}
.fade-enter-active,
.fade-leave-active {
  transition: opacity 150ms ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
