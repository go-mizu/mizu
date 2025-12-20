/// <reference types="vite/client" />

declare module 'alpinejs' {
  interface Alpine {
    data(name: string, callback: () => object): void
    start(): void
  }
  const Alpine: Alpine
  export default Alpine
}
