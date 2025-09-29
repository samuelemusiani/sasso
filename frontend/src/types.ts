export interface VM {
  id: number
  name?: string
  template?: string
  cores: number
  ram: number
  disk: number
  status: string
  include_global_ssh_keys: boolean
  uptime?: number // uptime in seconds
}

export interface User {
  id: number
  username: string
  email: string
  realm: string
  role: string
  max_cores: number
  max_ram: number
  max_disk: number
  max_nets: number
}

export interface Realm {
  id: number
  name: string
  description: string
  type: string
}

export interface LDAPRealm extends Realm {
  url: string
  base_dn: string
  bind_dn: string
  password: string
  filter?: string
}

export interface Net {
  id: number
  name: string
  vlanaware: boolean
  userid: number
  status: string
  subnet: string
  gateway: string
}

export interface SSHKey {
  id: number
  name: string
  key: string
}

export interface Interface {
  id: number
  vnet_id: number
  vlan_tag: number
  ip_add: string
  gateway: string
  status: string
}

export interface PortForward {
  created_at?: string
  id: number
  out_port: number
  dest_port: number
  dest_ip: string
  approved: boolean
  name?: string
  status?: string
  user_name?: string
  target_port?: number
  target_ip?: string
  source_port?: number
  description?: string
  user_id?: number
}

export interface AdminPortForward extends PortForward {
  username?: string
}

// Statistiche richieste dalla view
export interface PortForwardStats {
  totalRequests: number
  approvedRequests: number
  pendingRequests: number
  rejectedRequests?: number
  activeRequests?: number
}

// Funzione placeholder per fetchUsers (da implementare o importare dove serve)
// export async function fetchUsers() {}

export interface Backup {
  name: string // ID del backup (hash)
  ctime: string // Data/ora di creazione
  can_delete: boolean // Se può essere eliminato
}
