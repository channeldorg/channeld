import { useEffect, useState } from 'react'
import { useInspectorStore } from '../stores/inspectorStore'
import JsonTreeEditor from './JsonTreeEditor'
import './ChannelDetail.css'

interface ChannelDetailProps {
  channelId: number
}

export default function ChannelDetail({ channelId }: ChannelDetailProps) {
  const { getChannel, fetchChannelData, channelData } = useInspectorStore()
  const [loading, setLoading] = useState(false)
  const channel = getChannel(channelId)

  useEffect(() => {
    if (channelId !== null) {
      setLoading(true)
      fetchChannelData(channelId).finally(() => setLoading(false))
    }
  }, [channelId, fetchChannelData])

  if (!channel) {
    return (
      <div className="channel-detail">
        <div className="error">Channel not found</div>
      </div>
    )
  }

  const data = channelData[channelId]

  return (
    <div className="channel-detail">
      <div className="channel-detail-header">
        <div className="channel-info">
          <h2>Channel #{channel.channelId}</h2>
          <div className="channel-meta">
            <span className="meta-item">
              <span className="meta-label">Type:</span>
              <span className="meta-value">{channel.channelType}</span>
            </span>
            <span className="meta-item">
              <span className="meta-label">Owner:</span>
              <span className="meta-value">{channel.ownerConnId || 'None'}</span>
            </span>
            <span className="meta-item">
              <span className="meta-label">Subscribers:</span>
              <span className="meta-value">{channel.subscriberCount}</span>
            </span>
          </div>
        </div>
      </div>
      <div className="channel-detail-content">
        {loading ? (
          <div className="loading">Loading channel data...</div>
        ) : data ? (
          <JsonTreeEditor
            data={data}
            channelId={channelId}
          />
        ) : (
          <div className="no-data">No channel data</div>
        )}
      </div>
    </div>
  )
}
