import { useInspectorStore } from '../stores/inspectorStore'
import './Sidebar.css'

export default function Sidebar() {
  const { channels, selectedChannelId, selectChannel } = useInspectorStore()

  return (
    <div className="sidebar">
      <div className="sidebar-header">
        <h2>Channels</h2>
        <span className="channel-count">{channels.length}</span>
      </div>
      <div className="channel-list">
        {channels.map((channel) => (
          <div
            key={channel.channelId}
            className={`channel-item ${selectedChannelId === channel.channelId ? 'selected' : ''}`}
            onClick={() => selectChannel(channel.channelId)}
          >
            <div className="channel-item-header">
              <span className="channel-id">#{channel.channelId}</span>
              <span className="channel-type">{channel.channelType}</span>
            </div>
            <div className="channel-item-info">
              <span className="subscriber-count">{channel.subscriberCount} subscribers</span>
            </div>
          </div>
        ))}
        {channels.length === 0 && (
          <div className="empty-state">No channels</div>
        )}
      </div>
    </div>
  )
}
