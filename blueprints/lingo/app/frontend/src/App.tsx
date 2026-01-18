import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './stores/auth'
import Layout from './components/Layout'
import Landing from './pages/Landing'
import Login from './pages/Login'
import Signup from './pages/Signup'
import Home from './pages/Home'
import Lesson from './pages/Lesson'
import Leaderboards from './pages/Leaderboards'
import Profile from './pages/Profile'
import Shop from './pages/Shop'
import Achievements from './pages/Achievements'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuthStore()
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

function App() {
  const { isAuthenticated } = useAuthStore()

  return (
    <Routes>
      {/* Public routes */}
      <Route path="/" element={isAuthenticated ? <Navigate to="/learn" replace /> : <Landing />} />
      <Route path="/login" element={isAuthenticated ? <Navigate to="/learn" replace /> : <Login />} />
      <Route path="/signup" element={isAuthenticated ? <Navigate to="/learn" replace /> : <Signup />} />

      {/* Protected routes */}
      <Route path="/learn" element={
        <ProtectedRoute>
          <Layout><Home /></Layout>
        </ProtectedRoute>
      } />
      <Route path="/lesson/:id" element={
        <ProtectedRoute>
          <Lesson />
        </ProtectedRoute>
      } />
      <Route path="/leaderboards" element={
        <ProtectedRoute>
          <Layout><Leaderboards /></Layout>
        </ProtectedRoute>
      } />
      <Route path="/profile" element={
        <ProtectedRoute>
          <Layout><Profile /></Layout>
        </ProtectedRoute>
      } />
      <Route path="/shop" element={
        <ProtectedRoute>
          <Layout><Shop /></Layout>
        </ProtectedRoute>
      } />
      <Route path="/achievements" element={
        <ProtectedRoute>
          <Layout><Achievements /></Layout>
        </ProtectedRoute>
      } />

      {/* Catch all */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default App
