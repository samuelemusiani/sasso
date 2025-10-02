export interface VM {
  id: number
  name: string
  cores: number
  ram: number
  disk: number
  status: string
  include_global_ssh_keys: boolean
  notes: string
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
  user_base_dn: string
  group_base_dn: string
  bind_dn: string
  password: string
  admin_group: string
  maintainer_group: string
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
  out_port: number
  dest_port: number
  dest_ip: string
  approved: boolean
}

export interface AdminPortForward {
  id: number
  out_port: number
  dest_port: number
  dest_ip: string
  approved: boolean
  username: string
}

export interface Backup {
  id: string
  ctime: string
  can_delete: boolean
  name: string
  notes: string
  protected: boolean
}

export interface BackupRequest {
  id: number
  created_at: string

  type: string
  status: string
  vmid: number
}

export interface Stat {
  max_cores: number
  max_ram: number
  max_disk: number
  max_nets: number
  allocated_cores: number
  allocated_ram: number
  allocated_disk: number
  allocated_nets: number
  active_vms_cores: number
  active_vms_ram: number
  active_vms_disk: number
}
