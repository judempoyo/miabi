// copyText copies text to the clipboard, resolving to whether it succeeded.
//
// It prefers the async Clipboard API (secure contexts) and falls back to a
// hidden-textarea execCommand copy for insecure origins — plain-HTTP LAN and
// homelab deployments where `navigator.clipboard` is undefined. It never throws,
// so callers can branch on the boolean to show success/failure feedback instead
// of leaking an unhandled promise rejection (the old `clipboard?.writeText(...)
// .then(...)` pattern threw a TypeError whenever the API was unavailable).
export async function copyText(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    // Secure-context write can still be rejected (denied permission, focus
    // loss); fall through to the legacy path before giving up.
  }
  return legacyCopy(text)
}

function legacyCopy(text: string): boolean {
  try {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.setAttribute('readonly', '')
    ta.style.position = 'fixed'
    ta.style.top = '-9999px'
    document.body.appendChild(ta)
    ta.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    return ok
  } catch {
    return false
  }
}
