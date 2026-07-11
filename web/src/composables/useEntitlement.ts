import { computed, type ComputedRef } from 'vue'
import { useLicenseStore } from '@/stores/license'
import type { LicenseEdition } from '@/api/types'

export interface Entitlement {
  // has: the feature is licensed and usable at runtime (false in Community).
  has: ComputedRef<boolean>
  // mutable: the feature's config may be changed (false once the license has
  // degraded past its grace period — paid config goes read-only on expiry).
  mutable: ComputedRef<boolean>
  edition: ComputedRef<LicenseEdition>
}

// useEntitlement gates a UI affordance on a license flag. Render a lock/upgrade
// badge when !has, and a read-only state when has && !mutable. Example:
//   const sso = useEntitlement('multi_sso')
//   <button :disabled="!sso.mutable.value">Add provider</button>
export function useEntitlement(flag: string): Entitlement {
  const store = useLicenseStore()
  return {
    has: computed(() => store.has(flag)),
    mutable: computed(() => store.mutable(flag)),
    edition: computed(() => store.edition),
  }
}
