import type { Interface as SassoInterface } from '@/types'
import type { AxiosInstance } from 'axios'
import { ref } from 'vue'

type Missing = { sshKeys: boolean; interfaces: boolean }
// Taken from @/src/stores/loading.ts
type KeySeg = string | number | boolean | null | undefined
type LoadingLike = {
  start: (...segments: KeySeg[]) => void
  stop: (...segments: KeySeg[]) => void
}

export function useVmStartWithChecks(opts: {
  api: AxiosInstance
  loading: LoadingLike
  onStarted?: () => void
}) {
  const showModal = ref(false)
  const modalMissing = ref<Missing>({ sshKeys: false, interfaces: false })
  const pendingVmId = ref<number | null>(null)

  async function fetchInterfaces(vmid: number) {
    const res = await opts.api.get(`/vm/${vmid}/interface`)
    return (res.data ?? []).filter((i: SassoInterface) => i.status === 'ready')
  }

  async function fetchSSHKeys() {
    const res = await opts.api.get(`/ssh-keys`)
    return res.data ?? []
  }

  async function preStartVM(vmid: number) {
    opts.loading.start('vm', vmid, 'start')
    pendingVmId.value = vmid
    modalMissing.value = { sshKeys: false, interfaces: false }

    const [ifaces, keys] = await Promise.all([fetchInterfaces(vmid), fetchSSHKeys()])

    const needsModal = ifaces.length === 0 || keys.length === 0
    modalMissing.value.interfaces = ifaces.length === 0
    modalMissing.value.sshKeys = keys.length === 0

    if (needsModal) {
      showModal.value = true
      return
    }
    await confirmStart()
  }

  async function confirmStart() {
    const vmid = pendingVmId.value
    if (!vmid) return
    try {
      await opts.api.post(`/vm/${vmid}/start`)
      opts.onStarted?.()
    } finally {
      showModal.value = false
      opts.loading.stop('vm', vmid, 'start')
      pendingVmId.value = null
    }
  }

  function cancelStart() {
    const vmid = pendingVmId.value
    showModal.value = false
    if (vmid) opts.loading.stop('vm', vmid, 'start')
    pendingVmId.value = null
  }

  return { showModal, modalMissing, preStartVM, confirmStart, cancelStart }
}
