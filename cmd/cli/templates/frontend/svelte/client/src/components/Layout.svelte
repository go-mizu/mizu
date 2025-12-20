<script lang="ts">
  import { Route, Link } from 'svelte-routing'
  import Home from '../pages/Home.svelte'
  import About from '../pages/About.svelte'
</script>

<div class="app">
  <header class="header">
    <nav>
      <Link to="/">Home</Link>
      <Link to="/about">About</Link>
    </nav>
  </header>
  <main class="main">
    <Route path="/" component={Home} />
    <Route path="/about" component={About} />
  </main>
  <footer class="footer">
    <p>Built with Mizu + Svelte</p>
  </footer>
</div>
