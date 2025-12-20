import { useEffect, useState } from 'react';
import { MizuRuntime } from './runtime/MizuRuntime';
import { AppConfig } from './config/config';
import { useAuthStore } from './store/authStore';
import { useOnlineStatus } from './hooks/useOnlineStatus';
import HomeScreen from './screens/HomeScreen';
import WelcomeScreen from './screens/WelcomeScreen';
import LoadingView from './components/LoadingView';
import OfflineBanner from './components/OfflineBanner';
import InstallPrompt from './components/InstallPrompt';

export default function App() {
  const [isReady, setIsReady] = useState(false);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const setIsAuthenticated = useAuthStore((state) => state.setIsAuthenticated);
  const isOnline = useOnlineStatus();

  useEffect(() => {
    async function initialize() {
      // Initialize Mizu runtime
      await MizuRuntime.initialize({
        baseURL: AppConfig.baseURL,
        timeout: AppConfig.timeout,
        enableOffline: true,
      });

      // Subscribe to auth state changes
      const unsubscribe = MizuRuntime.shared.onAuthStateChange((isAuth) => {
        setIsAuthenticated(isAuth);
      });

      // Check initial auth state
      setIsAuthenticated(MizuRuntime.shared.isAuthenticated);
      setIsReady(true);

      return unsubscribe;
    }

    initialize();
  }, [setIsAuthenticated]);

  if (!isReady) {
    return <LoadingView message="Loading..." />;
  }

  return (
    <div className="app">
      {!isOnline && <OfflineBanner />}
      {isAuthenticated ? <HomeScreen /> : <WelcomeScreen />}
      <InstallPrompt />
    </div>
  );
}
