<template>
  <div class="min-h-screen bg-slate-50 text-slate-900 flex flex-col">
    <AppNavigation />
    <main class="flex-1 max-w-5xl mx-auto w-full p-8">
      <slot />
    </main>
    <footer class="border-t border-slate-200 bg-white py-4 text-center text-slate-500">
      <p>Built with Mizu + Nuxt</p>
    </footer>
  </div>
</template>
