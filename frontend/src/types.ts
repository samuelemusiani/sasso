export interface VM {
  id: number
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
}

export interface Net {
  id: number
  name: string
  vlanaware: boolean
  userid: number
  status: string
}

export interface SSHKey {
  id: number
  name: string
  key: string
}
