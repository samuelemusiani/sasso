export function getStatusClass(status: string) {
  switch (status) {
    case 'ready':
    case 'running':
    case 'true':
      return 'text-success'
    case 'error':
    case 'deleting':
    case 'pre-deleting':
    case 'unknown':
    case 'false':
      return 'text-error'
    case 'creating':
    case 'pre-creating':
    case 'configuring':
    case 'pre-configuring':
    case 'stopped':
      return 'text-warning'
    case 'pending':
    case 'paused':
      return 'text-info'
    default:
      return 'text-info'
  }
}
