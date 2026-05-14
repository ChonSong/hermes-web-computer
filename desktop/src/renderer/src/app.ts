// Hermes Desktop - Renderer Entry Point
// Loads the web UI from the Go backend

interface ElectronAPI {
  getBackendURL: () => Promise<string>;
  getAppVersion: () => Promise<string>;
  isDev: () => Promise<boolean>;
  platform: string;
  minimizeWindow: () => void;
  maximizeWindow: () => void;
  closeWindow: () => void;
}

declare global {
  interface Window {
    electronAPI?: ElectronAPI;
  }
}

async function getBackendURL(): Promise<string> {
  if (window.electronAPI) {
    return await window.electronAPI.getBackendURL();
  }
  // Fallback for browser testing
  return 'http://localhost:3113';
}

async function init(): Promise<void> {
  const app = document.getElementById('app');
  if (!app) return;

  try {
    const backendURL = await getBackendURL();
    console.log('Hermes Desktop connecting to:', backendURL);

    // For Electron: redirect to the backend URL
    // The backend serves the SPA at its root
    if (window.electronAPI) {
      // In Electron, we iframe the backend or redirect
      app.innerHTML = `
        <iframe 
          src="${backendURL}" 
          style="width: 100%; height: 100%; border: none;"
          allow="cross-origin-isolated"
        ></iframe>
      `;
    } else {
      // Browser fallback: direct navigation
      window.location.href = backendURL;
    }
  } catch (error) {
    console.error('Failed to initialize:', error);
    app.innerHTML = `
      <div class="loading">
        <div style="color: #ff6b6b;">Failed to connect to backend</div>
        <div style="color: #888; font-size: 14px;">${error}</div>
      </div>
    `;
  }
}

// Initialize when DOM is ready
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

export {};