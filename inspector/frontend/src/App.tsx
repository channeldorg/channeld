import { useEffect } from 'react'
import { useInspectorStore } from './stores/inspectorStore'
import TitleBar from './components/TitleBar'
import Sidebar from './components/Sidebar'
import MainContent from './components/MainContent'
import ConnectionStatus from './components/ConnectionStatus'
import './App.css'

function App() {
  const { connect, disconnect } = useInspectorStore()

  useEffect(() => {
    // Connect to WebSocket on mount
    connect()

    // Cleanup on unmount
    return () => {
      disconnect()
    }
  }, [connect, disconnect])

  return (
    <div className="app">
      <TitleBar />
      <div className="app-body">
        <Sidebar />
        <MainContent />
      </div>
      <ConnectionStatus />
    </div>
  )
}

export default App
