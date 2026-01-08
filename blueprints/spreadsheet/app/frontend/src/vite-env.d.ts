/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_DEV_AUTO_LOGIN: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
