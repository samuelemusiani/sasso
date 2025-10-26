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
