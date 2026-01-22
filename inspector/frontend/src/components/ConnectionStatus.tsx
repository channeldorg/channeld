import { useInspectorStore } from '../stores/inspectorStore'
import './ConnectionStatus.css'

export default function ConnectionStatus() {
  const { isConnected, ws } = useInspectorStore()

  if (isConnected) {
    return (
      <div className="connection-status connected">
        <span className="status-indicator connected"></span>
        Connected
      </div>
    )
  }

  return (
    <div className="connection-status disconnected">
      <span className="status-indicator disconnected"></span>
      {ws ? 'Connecting...' : 'Disconnected'}
    </div>
  )
}
