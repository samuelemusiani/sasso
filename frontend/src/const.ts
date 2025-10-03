export function getStatusClass(status: string) {
  switch (status) {
    case 'ready':
    case 'running':
      return 'text-success'
    case 'error':
    case 'deleting':
    case 'pre-deleting':
    case 'unknown':
      return 'text-error'
    case 'creating':
    case 'pre-creating':
	case 'configuring':
	case 'pre-configuring':
	case 'stopped':
      return 'text-warning'
    case 'pending':
	case 'suspended':
      return 'text-info'
    default:
      return 'text-info'
  }
}
