<script lang="ts">
  import { onMount } from 'svelte'

  let message = $state('')
  let loading = $state(true)

  onMount(async () => {
    try {
      const res = await fetch('/api/hello')
      const data = await res.json()
      message = data.message
    } catch {
      message = 'Failed to load message'
    } finally {
      loading = false
    }
  })
</script>

<div class="page home">
  <h1>Welcome</h1>
  {#if loading}
    <p>Loading...</p>
  {:else}
    <p class="api-message">{message}</p>
  {/if}
</div>
