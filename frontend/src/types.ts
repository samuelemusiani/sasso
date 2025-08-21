export interface VM {
  id: number
}

export interface User {
  id: number
  username: string
  email: string
  realm: string
  role: string
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
