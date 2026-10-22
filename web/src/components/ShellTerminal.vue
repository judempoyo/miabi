<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { useI18n } from 'vue-i18n'
import { wsUrl } from '@/api/client'

// ShellTerminal opens a live interactive shell into an application's running
// container over a WebSocket. The connection uses the session cookie and
// authorized server-side (Admin+ and the plan's shell-exec capability).
const props = defineProps<{
  // base is the app's API path, e.g. /workspaces/3/apps/12
  base: string
  appName: string
}>()
const emit = defineEmits<{ (e: 'close'): void }>()

const { t } = useI18n()
const connectionError = ref('')
const host = ref<HTMLDivElement | null>(null)
const status = ref<'connecting' | 'open' | 'closed'>('connecting')

const panel = ref<HTMLElement | null>(null)
const dock = ref<HTMLButtonElement | null>(null)
const minimized = ref(false)
const expanded = ref(false)
const position = reactive({ x: 24, y: 80 })
const shell = ref(localStorage.getItem('miabi.shell.command') === '/bin/bash' ? '/bin/bash' : '/bin/sh')
const windowStyle = computed(() => expanded.value
  ? { left: '8px', top: '8px' }
  : { left: `${position.x}px`, top: `${position.y}px` })
let previousFocus: HTMLElement | null = null
let drag: { x: number; y: number; pointerX: number; pointerY: number } | null = null

function clampPosition() {
  if (minimized.value || expanded.value) return
  const rect = panel.value?.getBoundingClientRect()
  position.x = Math.max(8, Math.min(position.x, window.innerWidth - (rect?.width ?? 320) - 8))
  position.y = Math.max(8, Math.min(position.y, window.innerHeight - (rect?.height ?? 400) - 8))
}
function startDrag(event: PointerEvent) {
  if (expanded.value || event.button !== 0 || (event.target as HTMLElement).closest('.shell-controls')) return
  drag = { ...position, pointerX: event.clientX, pointerY: event.clientY }
    ; (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}
function moveDrag(event: PointerEvent) {
  if (!drag) return
  position.x = drag.x + event.clientX - drag.pointerX
  position.y = drag.y + event.clientY - drag.pointerY
  clampPosition()
}
function moveWithKeyboard(event: KeyboardEvent) {
  if (expanded.value || !event.key.startsWith('Arrow')) return
  event.preventDefault()
  const step = event.shiftKey ? 48 : 16
  if (event.key === 'ArrowLeft') position.x -= step
  if (event.key === 'ArrowRight') position.x += step
  if (event.key === 'ArrowUp') position.y -= step
  if (event.key === 'ArrowDown') position.y += step
  clampPosition()
}
async function restore() {
  minimized.value = false
  await nextTick()
  clampPosition()
  sendResize()
  term?.focus()
}
async function minimize() {
  minimized.value = true
  await nextTick()
  dock.value?.focus()
}
async function toggleExpanded() {
  expanded.value = !expanded.value
  await restore()
}
function close() {
  emit('close')
  if (previousFocus?.isConnected) previousFocus.focus()
}

let term: Terminal | null = null
let fit: FitAddon | null = null
let socket: WebSocket | null = null
let resizeObserver: ResizeObserver | null = null
let connectionTimer: ReturnType<typeof setTimeout> | undefined

function clearConnectionTimer() {
  clearTimeout(connectionTimer)
  connectionTimer = undefined
}

function send(obj: Record<string, unknown>) {
  if (socket && socket.readyState === WebSocket.OPEN) socket.send(JSON.stringify(obj))
}

function sendResize() {
  if (!fit || !term || minimized.value || !host.value?.clientHeight) return
  try {
    fit.fit()
  } catch {
    /* container not measurable yet */
  }
  send({ type: 'resize', cols: term.cols, rows: term.rows })
}

onMounted(() => {
  previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
  position.x = Math.max(8, window.innerWidth - 924)
  clampPosition()
  window.addEventListener('resize', clampPosition)
  if (!host.value) return
  term = new Terminal({
    cursorBlink: true,
    fontSize: 13,
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace',
    theme: { background: '#0b0f17' },
  })
  fit = new FitAddon()
  term.loadAddon(fit)
  term.open(host.value)
  sendResize()
  void nextTick(() => term?.focus())

  // xterm owns Tab, arrows and Escape; Shift+Tab returns to the window controls.
  term.attachCustomKeyEventHandler((event) => {
    if (event.key === 'Tab' && event.shiftKey && event.type === 'keydown') {
      event.preventDefault()
      panel.value?.querySelector<HTMLButtonElement>('.shell-controls button')?.focus()
      return false
    }
    return true
  })
  connect()

  term.onData((d) => send({ type: 'stdin', data: d }))
  term.onResize(() => send({ type: 'resize', cols: term!.cols, rows: term!.rows }))

  resizeObserver = new ResizeObserver(() => sendResize())
  resizeObserver.observe(host.value)
})

function connect() {
  clearConnectionTimer()
  connectionError.value = ''
  if (socket) {
    socket.onopen = socket.onmessage = socket.onclose = socket.onerror = null
    socket.close()
  }
  status.value = 'connecting'
  try { localStorage.setItem('miabi.shell.command', shell.value) } catch { /* storage unavailable */ }
  term?.reset()
  try {
    socket = new WebSocket(wsUrl(`${props.base}/exec?shell=${encodeURIComponent(shell.value)}`))
  } catch {
    status.value = 'closed'
    connectionError.value = t('shellTerminal.connectionFailed')
    return
  }
  const currentSocket = socket
  connectionTimer = setTimeout(() => {
    if (currentSocket !== socket || status.value !== 'connecting') return
    connectionError.value = t('shellTerminal.connectionTimeout')
    status.value = 'closed'
    currentSocket.close()
  }, 15000)
  socket.binaryType = 'arraybuffer'

  socket.onopen = () => {
    clearConnectionTimer()
    connectionError.value = ''
    status.value = 'open'
    sendResize()
  }
  socket.onmessage = (ev) => {
    if (typeof ev.data === 'string') term?.write(ev.data)
    else term?.write(new Uint8Array(ev.data as ArrayBuffer))
  }
  socket.onclose = () => {
    clearConnectionTimer()
    if (status.value === 'connecting' && !connectionError.value) connectionError.value = t('shellTerminal.connectionFailed')
    status.value = 'closed'
  }
  socket.onerror = () => {
    clearConnectionTimer()
    status.value = 'closed'
    if (!connectionError.value) connectionError.value = t('shellTerminal.connectionFailed')
    term?.write('\r\n\x1b[31mconnection error\x1b[0m\r\n')
  }

  void nextTick(() => term?.focus())
}

onBeforeUnmount(() => {
  clearConnectionTimer()
  window.removeEventListener('resize', clampPosition)
  resizeObserver?.disconnect()
  socket?.close()
  term?.dispose()
})
</script>

<template>
  <Teleport to="body">
    <section v-show="!minimized" ref="panel" class="shell-window" :class="{ 'is-expanded': expanded }"
      :style="windowStyle" role="dialog" aria-modal="false" :aria-label="`Shell — ${appName}`">
      <header class="shell-bar" @pointerdown="startDrag" @pointermove="moveDrag" @pointerup="drag = null"
        @pointercancel="drag = null" @lostpointercapture="drag = null">
        <button type="button" class="btn-icon btn-icon-muted shell-grip" :aria-label="$t('shellTerminal.move')"
          :title="$t('shellTerminal.move')" @keydown="moveWithKeyboard">⠿</button>
        <strong class="shell-title"><span class="mdi mdi-console-line"></span> {{ appName }}</strong>
        <span class="shell-status" :class="status">{{ status }}</span>
        <div class="shell-controls">
          <button type="button" class="btn-icon btn-icon-muted" :title="$t('shellTerminal.minimize')"
            :aria-label="$t('shellTerminal.minimize')" @click="minimize"><span
              class="mdi mdi-window-minimize"></span></button>
          <button type="button" class="btn-icon btn-icon-muted"
            :title="$t(expanded ? 'shellTerminal.restore' : 'shellTerminal.maximize')"
            :aria-label="$t(expanded ? 'shellTerminal.restore' : 'shellTerminal.maximize')" @click="toggleExpanded">
            <span class="mdi" :class="expanded ? 'mdi-window-restore' : 'mdi-window-maximize'"></span></button>
          <button type="button" class="btn-icon btn-icon-muted shell-close" :title="$t('shell.close')"
            :aria-label="$t('shell.close')" @click="close"><span class="mdi mdi-close"></span></button>
        </div>
      </header>
      <div ref="host" class="shell-host" data-modal-tab-through data-modal-escape-through @keydown.stop></div>
      <div v-if="connectionError" class="shell-error" role="alert">{{ connectionError }}</div>
      <footer class="shell-footer">
        <span>{{ $t('shellTerminal.keyboardHint') }}</span>
        <label>{{ $t('shellTerminal.newSession') }}
          <select v-model="shell" :aria-label="$t('shellTerminal.newSession')" @change="connect" class="form-select"
            style="width: 140px;">
            <option value="/bin/sh">sh</option>
            <option value="/bin/bash">bash</option>
          </select>
        </label>
        <button v-if="status === 'closed'" type="button" class="btn-icon btn-icon-muted"
          :title="$t('shellTerminal.reconnect')" :aria-label="$t('shellTerminal.reconnect')" @click="connect"><span
            class="mdi mdi-refresh"></span></button>
      </footer>
    </section>
    <button v-if="minimized" ref="dock" type="button" class="shell-dock" :title="$t('shellTerminal.restore')"
      @click="restore">
      <span class="mdi mdi-console-line"></span> {{ appName }} <span class="shell-status" :class="status">{{ status
        }}</span>
      <span class="mdi mdi-window-restore"></span>
    </button>
  </Teleport>
</template>

<style scoped>
.shell-window {
  position: fixed; z-index: 900; width: min(900px, calc(100vw - 16px));
  height: min(560px, calc(100dvh - 16px)); min-width: min(340px, calc(100vw - 16px));
  min-height: min(240px, calc(100dvh - 16px)); max-width: calc(100vw - 16px); max-height: calc(100dvh - 16px);
  display: flex; flex-direction: column; overflow: hidden; resize: both;
  border: 1px solid var(--border-color, #334155); border-radius: 12px;
  background: #0b0f17; box-shadow: 0 20px 70px #0006;
}
.shell-window.is-expanded { width: calc(100vw - 16px); height: calc(100dvh - 16px); resize: none; }
.shell-bar { display: flex; align-items: center; gap: 8px; padding: 8px; background: var(--bg-secondary); cursor: move; touch-action: none; user-select: none; }
.shell-title { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 14px; }
.shell-grip { cursor: move; }
.shell-controls { display: flex; flex-shrink: 0; gap: 2px; }
.shell-close:hover { color: #f87171; }
.shell-host { flex: 1; min-height: 0; min-width: 0; padding: 8px; overflow: hidden; }
.shell-error { padding: 8px 12px; background: #3a1a1a; color: #f87171; font-size: 12px; }
.shell-footer { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px; padding: 6px 16px 6px 10px; background: var(--bg-secondary); color: var(--text-muted); font-size: 11px; }
.shell-footer label { display: flex; align-items: center; gap: 6px; }
.shell-footer select { background: var(--bg-secondary); color: var(--text-primary); border: 1px solid var(--border-color, #334155); border-radius: 4px; }
.shell-dock { position: fixed; right: 16px; bottom: 16px; z-index: 900; display: flex; align-items: center; gap: 10px; max-width: calc(100vw - 32px); padding: 12px 16px; background: var(--bg-secondary); color: var(--text-primary); border: 1px solid var(--border-color, #334155); border-radius: 10px; box-shadow: 0 8px 30px #0004; cursor: pointer; }
.shell-status { font-size: 12px; padding: 2px 6px; border-radius: 999px; }
.shell-status.connecting { background: #4a3b1a; color: #e0b341; }
.shell-status.open { background: #14361f; color: #4ade80; }
.shell-status.closed { background: #3a1a1a; color: #f87171; }
@media (max-width: 480px) { .shell-bar { gap: 3px; } .shell-bar .shell-status { display: none; } }
</style>

