<script setup lang="ts">
const message = ref('')
const loading = ref(true)

onMounted(async () => {
  try {
    const data = await $fetch<{ message: string }>('/api/hello')
    message.value = data.message
  } catch {
    message.value = 'Failed to load message'
  } finally {
    loading.value = false
  }
})

useHead({
  title: '{{.Name}}',
})
</script>

<template>
  <div class="space-y-6">
    <h1 class="text-4xl font-bold">Welcome to {{.Name}}</h1>
    <p v-if="loading" class="text-slate-500">Loading...</p>
    <div v-else class="bg-white rounded-lg border border-slate-200 p-4">
      <p class="text-lg">{{"{{"}} message {{"}}"}}</p>
    </div>
  </div>
</template>
