<script setup lang="ts">
const route = useRoute()

const navLinks = [
  { href: '/', label: 'Home' },
  { href: '/about', label: 'About' },
]
</script>

<template>
  <header class="bg-white border-b border-slate-200">
    <nav class="max-w-5xl mx-auto px-8 py-4">
      <ul class="flex gap-6">
        <li v-for="link in navLinks" :key="link.href">
          <NuxtLink
            :to="link.href"
            class="font-medium transition-colors"
            :class="route.path === link.href
              ? 'text-blue-600'
              : 'text-slate-600 hover:text-blue-600'"
          >
            {{"{{"}} link.label {{"}}"}}
          </NuxtLink>
        </li>
      </ul>
    </nav>
  </header>
</template>
