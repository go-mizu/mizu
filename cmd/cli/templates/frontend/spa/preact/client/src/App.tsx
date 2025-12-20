import Router from 'preact-router'
import Layout from './components/Layout'
import Home from './pages/Home'
import About from './pages/About'

function App() {
  return (
    <Layout>
      <Router>
        <Home path="/" />
        <About path="/about" />
      </Router>
    </Layout>
  )
}

export default App
