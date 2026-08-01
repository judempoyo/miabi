// Workspace-wide SSE feed of application events. While the dashboard is open,
// activity and health update in place — no page refresh.
//
// Two things happen per event: it is prepended to the timeline immediately (so
// the feed feels live), and a debounced reconcile re-reads the overview, because
// counts and per-app health are derived server-side and a burst of events from
// one deploy should cost one refetch, not six.
import { onUnmounted, ref } from 'vue'
import { eventsApi } from '@/api/events'
import type { AppEvent } from '@/api/types'

// REFRESH_DEBOUNCE_MS collapses a deploy's burst of events into one reconcile.
const REFRESH_DEBOUNCE_MS = 1200
// FEED_CAP bounds the in-page timeline; the full history lives on the app page.
const FEED_CAP = 12

export function useWorkspaceStream(onReconcile: (id: number) => void) {
  const connected = ref(false)
  // id of the most recently arrived event, briefly highlighted in the timeline.
  const freshEventId = ref<number | null>(null)

  let es: EventSource | null = null
  let timer: ReturnType<typeof setTimeout> | null = null

  function close() {
    es?.close()
    es = null
    if (timer) clearTimeout(timer)
    timer = null
    connected.value = false
  }

  function scheduleReconcile(id: number) {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => onReconcile(id), REFRESH_DEBOUNCE_MS)
  }

  // open connects the stream. `onEvent` receives each event so the caller can
  // splice it into whatever it renders the timeline from.
  function open(id: number | null, onEvent: (e: AppEvent) => void) {
    close()
    if (!id) return
    es = new EventSource(eventsApi.workspaceStreamUrl(id))
    es.onopen = () => {
      connected.value = true
    }
    es.onmessage = (m) => {
      let msg: { type?: string; data?: AppEvent }
      try {
        msg = JSON.parse(m.data)
      } catch {
        return // keep-alive / non-JSON frame
      }
      if (msg.type !== 'event' || !msg.data) return
      freshEventId.value = msg.data.id
      onEvent(msg.data)
      scheduleReconcile(id)
    }
    es.onerror = () => {
      // EventSource auto-reconnects on transient errors; reflect the gap in the UI.
      connected.value = false
    }
  }

  onUnmounted(close)

  return { connected, freshEventId, open, close, FEED_CAP }
}
