/// <reference types="vite/client" />

declare module '@app-config' {
  const config: {
    name: string
    description: string
    version: string
    branding?: {
      brandSub?: string
      welcomeTitle?: string
      welcomeSub?: string
    }
    author?: {
      name?: string
      email?: string
      blog?: string
    }
    feedback?: {
      url?: string
    }
  }
  export default config
}
