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
