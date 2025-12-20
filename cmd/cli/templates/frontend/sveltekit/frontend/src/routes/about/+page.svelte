<svelte:head>
	<title>About - {{.Name}}</title>
</svelte:head>

<div class="space-y-6">
	<h1 class="text-4xl font-bold">About</h1>
	<p class="text-slate-600">
		This is a SvelteKit SPA powered by Mizu, a lightweight Go web framework.
	</p>
	<ul class="list-disc list-inside space-y-2 text-slate-600">
		<li>SvelteKit 2 with Svelte 5</li>
		<li>Static adapter for SPA deployment</li>
		<li>File-based routing with layouts</li>
		<li>Tailwind CSS for styling</li>
		<li>Mizu backend with the frontend middleware</li>
	</ul>
</div>
