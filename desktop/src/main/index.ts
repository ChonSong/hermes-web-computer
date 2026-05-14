import { app, BrowserWindow, Menu, Tray, nativeImage, shell, ipcMain, dialog, nativeTheme } from 'electron'
import path from 'path'
import log from 'electron-log'
import Store from 'electron-store'

// Configure logging
log.transports.file.level = 'info'
log.transports.console.level = 'debug'
log.info('Agent-OS starting...')

// Global exception handlers
process.on('uncaughtException', (error) => {
  log.error('Uncaught exception:', error)
  app.exit(1)
})

process.on('unhandledRejection', (reason) => {
  log.error('Unhandled rejection:', reason)
})

// Store for app settings
const store = new Store<{
  backendPort: number
  autoStart: boolean
  minimizeToTray: boolean
  launchMinimized: boolean
  windowBounds?: { x?: number; y?: number; width?: number; height?: number }
}>({
  defaults: {
    backendPort: 3113,
    autoStart: false,
    minimizeToTray: true,
    launchMinimized: false
  }
})

let mainWindow: BrowserWindow | null = null
let tray: Tray | null = null
let isQuitting = false

const isDev = !app.isPackaged

function getBackendURL(): string {
  return `http://localhost:${store.get('backendPort')}`
}

function getAssetPath(...paths: string[]): string {
  if (isDev) {
    return path.join(__dirname, '..', '..', ...paths)
  }
  return path.join(process.resourcesPath, 'assets', ...paths)
}

async function createWindow() {
  const savedBounds = store.get('windowBounds')

  mainWindow = new BrowserWindow({
    width: savedBounds?.width ?? 1400,
    height: savedBounds?.height ?? 900,
    x: savedBounds?.x,
    y: savedBounds?.y,
    minWidth: 900,
    minHeight: 600,
    title: 'Agent-OS',
    backgroundColor: '#12121a',
    show: false,
    webPreferences: {
      preload: path.join(__dirname, '../preload/index.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true
    }
  })

  // Save window bounds on move/resize
  const saveBounds = () => {
    if (mainWindow && !mainWindow.isMaximized()) {
      store.set('windowBounds', mainWindow.getBounds())
    }
  }
  mainWindow.on('resize', saveBounds)
  mainWindow.on('move', saveBounds)

  // Handle close — minimize to tray or quit
  mainWindow.on('close', (e) => {
    if (!isQuitting && store.get('minimizeToTray')) {
      e.preventDefault()
      mainWindow?.hide()
      return
    }
  })

  mainWindow.once('ready-to-show', () => {
    if (!store.get('launchMinimized')) {
      mainWindow?.show()
    }
  })

  mainWindow.on('closed', () => {
    mainWindow = null
  })

  // Open external links in default browser
  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url)
    return { action: 'deny' }
  })

  // Load the app
  const backendURL = getBackendURL()
  log.info(`Loading ${isDev ? 'dev server' : 'backend'} at ${backendURL}`)

  try {
    if (isDev) {
      // In dev, load the Vite dev server
      await mainWindow.loadURL('http://localhost:5173')
      mainWindow.webContents.openDevTools()
    } else {
      // In prod, load the Go backend (which serves embedded frontend)
      await mainWindow.loadURL(backendURL)
    }
  } catch (error) {
    log.error('Failed to load URL:', error)
    try {
      await mainWindow.loadURL(backendURL)
    } catch (fallbackError) {
      log.error('Fallback load failed:', fallbackError)
    }
  }
}

function createTray() {
  // Create a simple 16x16 tray icon (purple square)
  const iconData = 'iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAKklEQVQ4T2NkYGD4z4AEGBkZGRlRSAxDQwMDAyMjI8YAhtEQGQYGBgYGBgYGABn1AQv/QCLMAAAAAElFTkSuQmCC'
  const icon = nativeImage.createFromBuffer(Buffer.from(iconData, 'base64'))
  tray = new Tray(icon)

  const contextMenu = Menu.buildFromTemplate([
    {
      label: 'Show Agent-OS',
      click: () => {
        mainWindow?.show()
        mainWindow?.focus()
      }
    },
    { type: 'separator' },
    {
      label: `Backend: :${store.get('backendPort')}`,
      enabled: false
    },
    { type: 'separator' },
    {
      label: 'Quit',
      click: () => {
        isQuitting = true
        app.quit()
      }
    }
  ])

  tray.setToolTip('Agent-OS')
  tray.setContextMenu(contextMenu)
  tray.on('double-click', () => {
    mainWindow?.show()
    mainWindow?.focus()
  })
}

function createAppMenu() {
  const template: Electron.MenuItemConstructorOptions[] = [
    {
      label: 'Agent-OS',
      submenu: [
        { label: 'About Agent-OS', role: 'about' },
        { type: 'separator' },
        {
          label: 'Settings...',
          accelerator: 'CmdOrCtrl+,',
          click: () => {
            mainWindow?.webContents.send('open-settings')
          }
        },
        { type: 'separator' },
        { label: 'Quit', accelerator: 'CmdOrCtrl+Q', role: 'quit' }
      ]
    },
    {
      label: 'Edit',
      submenu: [
        { label: 'Undo', accelerator: 'CmdOrCtrl+Z', role: 'undo' },
        { label: 'Redo', accelerator: 'Shift+CmdOrCtrl+Z', role: 'redo' },
        { type: 'separator' },
        { label: 'Cut', accelerator: 'CmdOrCtrl+X', role: 'cut' },
        { label: 'Copy', accelerator: 'CmdOrCtrl+C', role: 'copy' },
        { label: 'Paste', accelerator: 'CmdOrCtrl+V', role: 'paste' },
        { label: 'Select All', accelerator: 'CmdOrCtrl+A', role: 'selectAll' }
      ]
    },
    {
      label: 'View',
      submenu: [
        { label: 'Reload', accelerator: 'CmdOrCtrl+R', role: 'reload' },
        { label: 'Force Reload', accelerator: 'CmdOrCtrl+Shift+R', role: 'forceReload' },
        { type: 'separator' },
        { label: 'Toggle DevTools', accelerator: 'F12', role: 'toggleDevTools' },
        { type: 'separator' },
        { label: 'Actual Size', accelerator: 'CmdOrCtrl+0', role: 'resetZoom' },
        { label: 'Zoom In', accelerator: 'CmdOrCtrl+Plus', role: 'zoomIn' },
        { label: 'Zoom Out', accelerator: 'CmdOrCtrl+-', role: 'zoomOut' },
        { type: 'separator' },
        { label: 'Toggle Fullscreen', accelerator: 'F11', role: 'togglefullscreen' }
      ]
    },
    {
      label: 'Window',
      submenu: [
        { label: 'Minimize', accelerator: 'CmdOrCtrl+M', role: 'minimize' },
        { label: 'Close', accelerator: 'CmdOrCtrl+W', role: 'close' }
      ]
    },
    {
      label: 'Help',
      submenu: [
        {
          label: 'Keyboard Shortcuts',
          click: () => {
            mainWindow?.webContents.send('open-shortcuts')
          }
        },
        {
          label: 'Documentation',
          click: () => {
            shell.openExternal('https://github.com/ChonSong/hermes-web-computer')
          }
        }
      ]
    }
  ]

  const menu = Menu.buildFromTemplate(template)
  Menu.setApplicationMenu(menu)
}

// IPC handlers
ipcMain.handle('get-app-version', () => app.getVersion())
ipcMain.handle('get-backend-url', () => getBackendURL())
ipcMain.handle('is-dev', () => isDev)

ipcMain.handle('show-message-box', async (_, options) => {
  const result = await dialog.showMessageBox(mainWindow!, options)
  return result
})

ipcMain.handle('open-external', async (_, url: string) => {
  await shell.openExternal(url)
})

ipcMain.handle('set-backend-port', (_, port: number) => {
  store.set('backendPort', port)
})

ipcMain.handle('set-auto-start', (_, enable: boolean) => {
  app.setLoginItemSettings({
    openAtLogin: enable,
    openAsHidden: store.get('launchMinimized') ?? false
  })
  store.set('autoStart', enable)
})

ipcMain.handle('set-minimize-to-tray', (_, enable: boolean) => {
  store.set('minimizeToTray', enable)
})

// Single instance lock
const gotTheLock = app.requestSingleInstanceLock()
if (!gotTheLock) {
  app.quit()
} else {
  app.on('second-instance', () => {
    if (mainWindow) {
      if (mainWindow.isMinimized()) mainWindow.restore()
      mainWindow.show()
      mainWindow.focus()
    }
  })
}

app.whenReady().then(async () => {
  log.info('App ready')
  createAppMenu()
  await createWindow()
  createTray()

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow()
    } else {
      mainWindow?.show()
    }
  })
})

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

app.on('before-quit', () => {
  isQuitting = true
})