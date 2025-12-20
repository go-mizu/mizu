<script lang="ts">
	import '../app.css';

	let { children } = $props();

	const navLinks = [
		{ href: '/', label: 'Home' },
		{ href: '/about', label: 'About' }
	];
</script>

<svelte:head>
	<title>{{.Name}}</title>
</svelte:head>

<div class="min-h-screen bg-slate-50 text-slate-900 flex flex-col">
	<header class="bg-white border-b border-slate-200">
		<nav class="max-w-5xl mx-auto px-8 py-4">
			<ul class="flex gap-6">
				{#each navLinks as link}
					<li>
						<a
							href={link.href}
							class="font-medium text-slate-600 hover:text-blue-600 transition-colors"
						>
							{link.label}
						</a>
					</li>
				{/each}
			</ul>
		</nav>
	</header>

	<main class="flex-1 max-w-5xl mx-auto w-full p-8">
		{@render children()}
	</main>

	<footer class="border-t border-slate-200 bg-white py-4 text-center text-slate-500">
		<p>Built with Mizu + SvelteKit</p>
	</footer>
</div>
