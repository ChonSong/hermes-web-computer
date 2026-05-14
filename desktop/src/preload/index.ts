import { contextBridge, ipcRenderer } from 'electron';

// Expose protected methods to the renderer process
contextBridge.exposeInMainWorld('electronAPI', {
  getBackendURL: (): Promise<string> => ipcRenderer.invoke('get-backend-url'),
  getAppVersion: (): Promise<string> => ipcRenderer.invoke('get-app-version'),
  isDev: (): Promise<boolean> => ipcRenderer.invoke('is-dev'),
  
  // Platform info
  platform: process.platform,
  
  // Window controls
  minimizeWindow: () => ipcRenderer.send('window-minimize'),
  maximizeWindow: () => ipcRenderer.send('window-maximize'),
  closeWindow: () => ipcRenderer.send('window-close')
});

// Type declarations for the exposed API
declare global {
  interface Window {
    electronAPI: {
      getBackendURL: () => Promise<string>;
      getAppVersion: () => Promise<string>;
      isDev: () => Promise<boolean>;
      platform: string;
      minimizeWindow: () => void;
      maximizeWindow: () => void;
      closeWindow: () => void;
    };
  }
}