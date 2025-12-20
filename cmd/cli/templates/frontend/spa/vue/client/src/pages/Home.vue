<script setup lang="ts">
import { ref, onMounted } from 'vue'

const message = ref('')
const loading = ref(true)

onMounted(async () => {
  try {
    const res = await fetch('/api/hello')
    const data = await res.json()
    message.value = data.message
  } catch {
    message.value = 'Failed to load message'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="page home">
    <h1>Welcome</h1>
    <p v-if="loading">Loading...</p>
    <p v-else class="api-message">{{"{{"}} message {{"}}"}}</p>
  </div>
</template>
