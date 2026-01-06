// Wails Runtime TypeScript definitions

export interface Position {
  x: number;
  y: number;
}

export interface Size {
  w: number;
  h: number;
}

export interface Screen {
  isCurrent: boolean;
  isPrimary: boolean;
  width: number;
  height: number;
}

// Events
export function EventsOn(eventName: string, callback: (...args: any[]) => void): () => void;
export function EventsOnce(eventName: string, callback: (...args: any[]) => void): () => void;
export function EventsOnMultiple(eventName: string, callback: (...args: any[]) => void, maxCallbacks: number): () => void;
export function EventsOff(eventName: string, ...additionalEventNames: string[]): void;
export function EventsEmit(eventName: string, ...data: any[]): void;

// Window
export function WindowReload(): void;
export function WindowReloadApp(): void;
export function WindowSetAlwaysOnTop(b: boolean): void;
export function WindowSetSystemDefaultTheme(): void;
export function WindowSetLightTheme(): void;
export function WindowSetDarkTheme(): void;
export function WindowCenter(): void;
export function WindowSetTitle(title: string): void;
export function WindowFullscreen(): void;
export function WindowUnfullscreen(): void;
export function WindowIsFullscreen(): Promise<boolean>;
export function WindowSetSize(width: number, height: number): void;
export function WindowGetSize(): Promise<Size>;
export function WindowSetMaxSize(width: number, height: number): void;
export function WindowSetMinSize(width: number, height: number): void;
export function WindowSetPosition(x: number, y: number): void;
export function WindowGetPosition(): Promise<Position>;
export function WindowHide(): void;
export function WindowShow(): void;
export function WindowMaximise(): void;
export function WindowToggleMaximise(): void;
export function WindowUnmaximise(): void;
export function WindowIsMaximised(): Promise<boolean>;
export function WindowMinimise(): void;
export function WindowUnminimise(): void;
export function WindowIsMinimised(): Promise<boolean>;
export function WindowIsNormal(): Promise<boolean>;
export function WindowSetBackgroundColour(R: number, G: number, B: number, A: number): void;

// Screen
export function ScreenGetAll(): Promise<Screen[]>;

// Browser
export function BrowserOpenURL(url: string): void;

// Clipboard
export function ClipboardGetText(): Promise<string>;
export function ClipboardSetText(text: string): Promise<boolean>;

// Log
export function LogPrint(message: string): void;
export function LogTrace(message: string): void;
export function LogDebug(message: string): void;
export function LogInfo(message: string): void;
export function LogWarning(message: string): void;
export function LogError(message: string): void;
export function LogFatal(message: string): void;

// Environment
export interface Environment {
  buildType: string;
  platform: string;
  arch: string;
}
export function Environment(): Promise<Environment>;

// Quit
export function Quit(): void;

// Hide
export function Hide(): void;

// Show
export function Show(): void;
