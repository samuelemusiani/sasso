export interface VM {
  id: number
  name: string
  cores: number
  ram: number
  disk: number
  status: string
  lifetime: string
  include_global_ssh_keys: boolean
  notes: string

  group_id?: number
  group_name?: string
  group_role?: string
}

export interface User {
  id: number
  username: string
  email: string
  realm: string
  role: string
  max_cores?: number
  max_ram?: number
  max_disk?: number
  max_nets?: number
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
  bind_dn: string
  password?: string
  login_filter: string
  maintainer_group_dn: string
  admin_group_dn: string
  mail_attribute: string
}

export interface Net {
  id: number
  name: string
  vlanaware: boolean
  status: string
  subnet: string
  gateway: string
  broadcast: string
  group_id?: number
  group_name?: string
  group_role?: string
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

export interface InterfaceExtended extends Interface {
  vnet_name: string
  vm_id: number
  vm_name: string
  group_id?: number
  group_name?: string
  group_role?: string
}

export interface PortForward {
  id: number
  out_port: number
  dest_port: number
  dest_ip: string
  approved: boolean
  name?: string
}

export interface AdminPortForward {
  id: number
  out_port: number
  dest_port: number
  dest_ip: string
  approved: boolean
  name: string
  is_group?: boolean
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

export interface TelegramBot {
  id: number
  name: string
  notes: string
  token: string
  chat_id: string
  enabled: boolean
}

export interface Group {
  id: number
  name: string
  description: string
  role?: string
  members?: GroupMember[]
  resources?: GroupResource[]
}

export interface GroupResource {
  user_id: number
  username: string
  cores: number
  ram: number
  disk: number
  nets: number
}

export interface GroupInvite {
  id: number
  group_id: number
  user_id: number
  role: string
  state: string
  username: string
  group_name: string
  group_description: string
}

export interface GroupMember {
  user_id: number
  username: string
  role: string
}

export interface VPNConfig {
  id: number
  vpn_config: string
}

export interface Settings {
  mail_port_forward_notification: boolean
  mail_vm_status_update_notification: boolean
  mail_global_ssh_keys_change_notification: boolean
  mail_vm_expiration_notification: boolean
  mail_vm_eliminated_notification: boolean
  mail_vm_stopped_notification: boolean
  mail_ssh_keys_changed_on_vm_notification: boolean
  mail_user_invitation_notification: boolean
  mail_user_removal_from_group_notification: boolean

  telegram_port_forward_notification: boolean
  telegram_vm_status_update_notification: boolean
  telegram_global_ssh_keys_change_notification: boolean
  telegram_vm_expiration_notification: boolean
  telegram_vm_eliminated_notification: boolean
  telegram_vm_stopped_notification: boolean
  telegram_ssh_keys_changed_on_vm_notification: boolean
  telegram_user_invitation_notification: boolean
  telegram_user_removal_from_group_notification: boolean
}
