import { createContext, useContext, useEffect, useState } from 'react'
import { configApi, getToken } from './api'

const ConfigContext = createContext({ whatsapp_enabled: false })

export function ConfigProvider({ children }) {
  const [config, setConfig] = useState({ whatsapp_enabled: false })

  useEffect(() => {
    if (!getToken()) return
    configApi.get().then(setConfig).catch(() => {})
  }, [])

  return <ConfigContext.Provider value={config}>{children}</ConfigContext.Provider>
}

export function useConfig() { return useContext(ConfigContext) }
