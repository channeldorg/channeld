import { useInspectorStore } from '../stores/inspectorStore'
import ChannelDetail from './ChannelDetail'
import './MainContent.css'

export default function MainContent() {
  const { selectedChannelId } = useInspectorStore()

  return (
    <div className="main-content">
      {selectedChannelId !== null ? (
        <ChannelDetail channelId={selectedChannelId} />
      ) : (
        <div className="empty-selection">
          <p>Select a channel to view details</p>
        </div>
      )}
    </div>
  )
}
