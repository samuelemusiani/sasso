<template>
  <div class="min-h-screen p-6">
    <div class="max-w-7xl mx-auto">
      <!-- Header -->
      <div class="mb-8">
        <h1 class="text-3xl font-bold text-base-content mb-2">Backup</h1>
        <p class="text-base-content/70">Gestisci i backup delle tue macchine virtuali</p>
      </div>

      <!-- Content -->
      <div class="text-center mb-12">
        <Icon icon="material-symbols:backup" class="text-6xl text-primary mx-auto mb-4" />
        <h2 class="text-2xl font-semibold text-base-content mb-2">Backup System</h2>
        <p class="text-base-content/70">
          Sistema di backup automatico per le tue macchine virtuali
        </p>
      </div>

      <div class="flex justify-center my-16">
        <div class="bg-warning/20 border border-warning/30 rounded-lg p-6 max-w-md text-center">
          <p class="text-warning font-medium text-lg">🚧 Funzionalità in sviluppo</p>
          <p class="text-base-content/70 text-sm mt-2">
            Il sistema di backup sarà disponibile nella prossima versione
          </p>
        </div>
      </div>

      <!-- Future features preview -->
      <div class="grid md:grid-cols-3 gap-4 mt-8">
        <div class="liquid-glass-card p-6">
          <Icon icon="material-symbols:schedule" class="text-3xl text-primary mb-3" />
          <h3 class="font-semibold mb-2">Backup Automatici</h3>
          <p class="text-sm text-base-content/70">Pianifica backup automatici delle tue VM</p>
        </div>
        <div class="liquid-glass-card p-6">
          <Icon icon="material-symbols:restore" class="text-3xl text-primary mb-3" />
          <h3 class="font-semibold mb-2">Ripristino Rapido</h3>
          <p class="text-sm text-base-content/70">Ripristina le VM da backup con un click</p>
        </div>
        <div class="liquid-glass-card p-6">
          <Icon icon="material-symbols:cloud-sync" class="text-3xl text-primary mb-3" />
          <h3 class="font-semibold mb-2">Storage Esterno</h3>
          <p class="text-sm text-base-content/70">Salva i backup su storage esterni</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Icon } from '@iconify/vue'
</script>

<style scoped>
.liquid-glass-card {
  backdrop-filter: blur(20px);
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 16px;
  box-shadow: 
    0 25px 50px -12px rgba(0, 0, 0, 0.25),
    0 0 20px rgba(255, 255, 255, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
}
</style>