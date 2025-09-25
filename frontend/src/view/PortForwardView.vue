<template>
  <div class="min-h-screen p-6">
    <div class="max-w-7xl mx-auto">
      <!-- Header -->
      <div class="mb-8">
        <h1 class="text-3xl font-bold text-base-content mb-2">Port Forward</h1>
        <p class="text-base-content/70">Gestisci il port forwarding per le tue applicazioni</p>
      </div>

      <!-- Content -->
      <div class="text-center mb-12">
        <Icon icon="material-symbols:router" class="text-6xl text-primary mx-auto mb-4" />
        <h2 class="text-2xl font-semibold text-base-content mb-2">Port Forwarding</h2>
        <p class="text-base-content/70">
          Configura regole di port forwarding per esporre i tuoi servizi
        </p>
      </div>

      <div class="flex justify-center my-16">
        <div class="bg-warning/20 border border-warning/30 rounded-lg p-6 max-w-md text-center">
          <p class="text-warning font-medium text-lg">🚧 Funzionalità in sviluppo</p>
          <p class="text-base-content/70 text-sm mt-2">
            Il sistema di port forwarding sarà disponibile nella prossima versione
          </p>
        </div>
      </div>

      <!-- Future features preview -->
      <div class="grid md:grid-cols-3 gap-4 mt-8">
        <div class="liquid-glass-card p-6">
          <Icon icon="material-symbols:lan" class="text-3xl text-primary mb-3" />
          <h3 class="font-semibold mb-2">Regole Dinamiche</h3>
          <p class="text-sm text-base-content/70">Crea regole di forwarding dinamiche</p>
        </div>
        <div class="liquid-glass-card p-6">
          <Icon icon="material-symbols:security" class="text-3xl text-primary mb-3" />
          <h3 class="font-semibold mb-2">Sicurezza Avanzata</h3>
          <p class="text-sm text-base-content/70">Controllo accessi e firewall integrato</p>
        </div>
        <div class="liquid-glass-card p-6">
          <Icon icon="material-symbols:monitoring" class="text-3xl text-primary mb-3" />
          <h3 class="font-semibold mb-2">Monitoraggio</h3>
          <p class="text-sm text-base-content/70">Monitora il traffico in tempo reale</p>
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