import { create } from 'zustand'
import { InspectorWebSocket, ChannelInfo } from '../services/websocket'

// Re-export ChannelInfo for convenience
export type { ChannelInfo }

interface InspectorState {
  // Connection state
  isConnected: boolean
  ws: InspectorWebSocket | null
  
  // Channel data
  channels: ChannelInfo[]
  selectedChannelId: number | null
  channelData: Record<number, any> // channelId -> JSON data
  
  // Actions
  connect: () => void
  disconnect: () => void
  selectChannel: (channelId: number) => void
  getChannel: (channelId: number) => ChannelInfo | undefined
  fetchChannelData: (channelId: number) => Promise<void>
  updateChannelData: (channelId: number, jsonPath: string, value: string, valueType: string) => Promise<void>
  setChannels: (channels: ChannelInfo[]) => void
  addChannel: (channel: ChannelInfo) => void
  removeChannel: (channelId: number) => void
  updateChannel: (channel: ChannelInfo) => void
  setChannelData: (channelId: number, data: any) => void
}

export const useInspectorStore = create<InspectorState>((set, get) => ({
  // Initial state
  isConnected: false,
  ws: null,
  channels: [],
  selectedChannelId: null,
  channelData: {},

  // Connect to WebSocket
  connect: () => {
    const ws = new InspectorWebSocket({
      url: 'ws://localhost:80',
      onConnected: () => {
        set({ isConnected: true })
        // Request initial channel list
        ws.getChannels()
        // Subscribe to channel list updates
        ws.subscribeChannelList()
      },
      onDisconnected: () => {
        set({ isConnected: false })
      },
      onChannels: (channels) => {
        set({ channels })
      },
      onChannelAdded: (channel) => {
        set((state) => ({
          channels: [...state.channels, channel],
        }))
      },
      onChannelRemoved: (channelId) => {
        set((state) => ({
          channels: state.channels.filter(c => c.channelId !== channelId),
          selectedChannelId: state.selectedChannelId === channelId ? null : state.selectedChannelId,
        }))
      },
      onChannelUpdated: (channel) => {
        set((state) => ({
          channels: state.channels.map(c =>
            c.channelId === channel.channelId ? channel : c
          ),
        }))
      },
      onChannelData: (channelId, data) => {
        set((state) => ({
          channelData: {
            ...state.channelData,
            [channelId]: data,
          },
        }))
      },
    })
    set({ ws })
  },

  // Disconnect from WebSocket
  disconnect: () => {
    const { ws } = get()
    if (ws) {
      ws.disconnect()
      set({ ws: null, isConnected: false })
    }
  },

  // Select a channel
  selectChannel: (channelId: number) => {
    set({ selectedChannelId: channelId })
    // Fetch channel data when selected
    get().fetchChannelData(channelId)
  },

  // Get channel by ID
  getChannel: (channelId: number) => {
    return get().channels.find(c => c.channelId === channelId)
  },

  // Fetch channel data
  fetchChannelData: async (channelId: number) => {
    const { ws } = get()
    if (ws) {
      await ws.getChannelData(channelId)
    }
  },

  // Update channel data
  updateChannelData: async (channelId: number, jsonPath: string, value: string, valueType: string) => {
    const { ws } = get()
    if (ws) {
      await ws.updateChannelData(channelId, jsonPath, value, valueType)
      // Refresh channel data after update
      setTimeout(() => {
        get().fetchChannelData(channelId)
      }, 100)
    }
  },

  // Set channels (for initial load)
  setChannels: (channels: ChannelInfo[]) => {
    set({ channels })
  },

  // Add channel (for incremental updates)
  addChannel: (channel: ChannelInfo) => {
    set((state) => ({
      channels: [...state.channels, channel],
    }))
  },

  // Remove channel (for incremental updates)
  removeChannel: (channelId: number) => {
    set((state) => ({
      channels: state.channels.filter(c => c.channelId !== channelId),
      selectedChannelId: state.selectedChannelId === channelId ? null : state.selectedChannelId,
    }))
  },

  // Update channel (for incremental updates)
  updateChannel: (channel: ChannelInfo) => {
    set((state) => ({
      channels: state.channels.map(c =>
        c.channelId === channel.channelId ? channel : c
      ),
    }))
  },

  // Set channel data
  setChannelData: (channelId: number, data: any) => {
    set((state) => ({
      channelData: {
        ...state.channelData,
        [channelId]: data,
      },
    }))
  },
}))
