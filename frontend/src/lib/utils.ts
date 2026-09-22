export function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

export function isVMExpired(lifetime: string): boolean {
  // TODO: Check with timezones
  return new Date(lifetime) < new Date()
}

export interface VMExpirationInfo {
  will_expire: boolean
  possible_extend_by: number[]
}

export function vmWillExpire(lifetime: string, possibleExtendBy: number[]): VMExpirationInfo {
  let pseby: number[] = []
  for (const extendBy of possibleExtendBy) {
    const months = extendBy / 2
    const days = (extendBy % 2) * 15

    const tmp = new Date()
    tmp.setMonth(tmp.getMonth() + months)
    tmp.setDate(tmp.getDate() + days)
    if (new Date(lifetime) <= tmp) {
      console.log(`VM will expire within ${extendBy} months`)

      pseby = possibleExtendBy.filter((v) => v >= extendBy)
      return {
        will_expire: true,
        possible_extend_by: pseby,
      }
    }
  }
  return {
    will_expire: false,
    possible_extend_by: [],
  }
}

export async function copyToClipboard(text: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text)
    return
  }

  // Fallback (older browsers / some embedded webviews)
  const el = document.createElement('textarea')
  el.value = text
  el.setAttribute('readonly', '')
  el.style.position = 'absolute'
  el.style.left = '-9999px'
  document.body.appendChild(el)
  el.focus()
  el.select()

  // TS will warn: deprecated. This is expected for legacy fallback.
  const ok = document.execCommand('copy')

  document.body.removeChild(el)
  return ok
}

export function downloadTextFile(options: {
  text: string
  filename: string
  mimeType?: string // default: 'text/plain'
  charset?: string // default: 'utf-8'
}): void {
  const { text, filename, mimeType = 'text/plain', charset = 'utf-8' } = options

  const blob = new Blob([text], { type: `${mimeType};charset=${charset}` })
  const url = URL.createObjectURL(blob)

  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.style.display = 'none'
  document.body.appendChild(a)
  a.click()

  // cleanup
  a.remove()
  URL.revokeObjectURL(url)
}
