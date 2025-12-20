<script lang="ts">
	import { onMount } from 'svelte';

	let message = $state('');
	let loading = $state(true);

	onMount(async () => {
		try {
			const res = await fetch('/api/hello');
			const data = await res.json();
			message = data.message;
		} catch {
			message = 'Failed to load message';
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>Home - {{.Name}}</title>
</svelte:head>

<div class="space-y-6">
	<h1 class="text-4xl font-bold">Welcome to {{.Name}}</h1>
	{#if loading}
		<p class="text-slate-500">Loading...</p>
	{:else}
		<div class="bg-white rounded-lg border border-slate-200 p-4">
			<p class="text-lg">{message}</p>
		</div>
	{/if}
</div>
