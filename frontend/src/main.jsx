import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { SchoolProvider } from './services/SchoolContext'
import { ThemeProvider } from './services/ThemeContext'
import { ConfigProvider } from './services/ConfigContext'
import App from './App.jsx'
import './styles/global.css'

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <BrowserRouter>
      <ThemeProvider>
        <ConfigProvider>
          <SchoolProvider>
            <App />
          </SchoolProvider>
        </ConfigProvider>
      </ThemeProvider>
    </BrowserRouter>
  </StrictMode>,
)
