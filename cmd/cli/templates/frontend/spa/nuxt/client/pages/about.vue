<script setup lang="ts">
useHead({
  title: 'About - {{.Name}}',
})
</script>

<template>
  <div class="space-y-6">
    <h1 class="text-4xl font-bold">About</h1>
    <p class="text-slate-600">
      This is a Nuxt SPA powered by Mizu, a lightweight Go web framework.
    </p>
    <ul class="list-disc list-inside space-y-2 text-slate-600">
      <li>Nuxt 3 with TypeScript</li>
      <li>Static generation for SPA deployment</li>
      <li>File-based routing with auto-imports</li>
      <li>Tailwind CSS for styling</li>
      <li>Mizu backend with the frontend middleware</li>
    </ul>
  </div>
</template>
