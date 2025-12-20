import { useState, useEffect } from 'react';
import {
  InstallState,
  getInstallState,
  onInstallStateChange,
  showInstallPrompt,
  dismissInstallPrompt,
} from '../pwa/installPrompt';

interface UseInstallPromptReturn {
  state: InstallState;
  canInstall: boolean;
  install: () => Promise<boolean>;
  dismiss: () => void;
}

export function useInstallPrompt(): UseInstallPromptReturn {
  const [state, setState] = useState<InstallState>(getInstallState());

  useEffect(() => {
    return onInstallStateChange(setState);
  }, []);

  return {
    state,
    canInstall: state === 'available',
    install: showInstallPrompt,
    dismiss: dismissInstallPrompt,
  };
}
