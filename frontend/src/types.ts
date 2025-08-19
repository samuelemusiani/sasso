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
