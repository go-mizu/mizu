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
import Quests from './pages/Quests'
import Letters from './pages/Letters'
import Guidebook from './pages/Guidebook'
import Courses from './pages/Courses'
import Stories from './pages/Stories'
import StoryPlayer from './pages/StoryPlayer'

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
      <Route path="/quests" element={
        <ProtectedRoute>
          <Layout><Quests /></Layout>
        </ProtectedRoute>
      } />
      <Route path="/letters" element={
        <ProtectedRoute>
          <Layout><Letters /></Layout>
        </ProtectedRoute>
      } />
      <Route path="/guidebook/:unitId" element={
        <ProtectedRoute>
          <Guidebook />
        </ProtectedRoute>
      } />
      <Route path="/courses" element={
        <ProtectedRoute>
          <Layout><Courses /></Layout>
        </ProtectedRoute>
      } />
      <Route path="/stories" element={
        <ProtectedRoute>
          <Layout><Stories /></Layout>
        </ProtectedRoute>
      } />
      <Route path="/story/:id" element={
        <ProtectedRoute>
          <StoryPlayer />
        </ProtectedRoute>
      } />

      {/* Catch all */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default App
