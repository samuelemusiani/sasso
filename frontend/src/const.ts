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

export function getPageIcon(page: string) {
  switch (page) {
    case 'home':
      return 'material-symbols:home-rounded'
    case 'vm':
    case 'vms':
      return 'mi:computer'
    case 'net':
    case 'nets':
      return 'ph:network'
    case 'interface':
    case 'interfaces':
      return 'ph:path'
    case 'ssh-key':
    case 'ssh-keys':
      return 'material-symbols:key'
    case 'vpn':
      return 'cib:wireguard'
    case 'port-forward':
    case 'port-forwards':
      return 'material-symbols:router'
    case 'telegram':
      return 'mdi:telegram'
    case 'group':
    case 'groups':
      return 'material-symbols:group-rounded'
    case 'admin':
      return 'material-symbols:admin-panel-settings'
    case 'help':
      return 'material-symbols:help'
    case 'settings':
      return 'material-symbols:settings'
    case 'backup':
    case 'backups':
      return 'material-symbols:backup'
    default:
      return ''
  }
}
