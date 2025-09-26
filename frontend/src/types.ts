export interface VM {
  id: number
  name?: string
  template?: string
  cores: number
  ram: number
  disk: number
  status: string
  include_global_ssh_keys: boolean
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
  id: number
  user_id: number
  user_name: string
  name: string
  description: string
  source_port: number
  target_ip: string
  target_port: number
  status: 'pending' | 'approved' | 'rejected' | 'active'
  created_at: string
  approved_at?: string
  approved_by?: string
}
