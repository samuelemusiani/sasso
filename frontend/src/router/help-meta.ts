import type { Component } from 'vue'

export function withHelp(helpComponent: Component) {
  return { meta: { helpComponent } }
}
