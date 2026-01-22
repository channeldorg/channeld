import protobuf from 'protobufjs'

// Message type constants
export const MessageType = {
  // Debug event messages (81-86)
  DEBUG_CHANNEL_LIST_SNAPSHOT: 81,
  DEBUG_CONNECTION_UPDATED: 82,
  DEBUG_CHANNEL_DATA_UPDATED: 83,
  DEBUG_CHANNEL_REMOVED: 84,
  DEBUG_CHANNEL_ADDED: 85,
  DEBUG_CHANNEL_UPDATED: 86,
  // Debug response messages (87-91)
  DEBUG_ERROR: 87,
  DEBUG_UPDATE_CHANNEL_DATA_RESULT: 88,
  DEBUG_GET_CHANNEL_DATA_RESULT: 89,
  DEBUG_GET_CHANNEL_RESULT: 90,
  DEBUG_GET_CHANNELS_RESULT: 91,
  // Debug request messages (92-98)
  DEBUG_UNSUBSCRIBE_CHANNEL_LIST: 92,
  DEBUG_SUBSCRIBE_CHANNEL_LIST: 93,
  DEBUG_SEARCH_CHANNELS: 94,
  DEBUG_UPDATE_CHANNEL_DATA: 95,
  DEBUG_GET_CHANNEL_DATA: 96,
  DEBUG_GET_CHANNEL: 97,
  DEBUG_GET_CHANNELS: 98,
}

export interface ChannelInfo {
  channelId: number
  channelType: string
  ownerConnId: number
  subscriberCount: number
  metadata: string
  createdTime: number
}

export interface InspectorWebSocketOptions {
  url: string
  onConnected?: () => void
  onDisconnected?: () => void
  onChannels?: (channels: ChannelInfo[]) => void
  onChannelAdded?: (channel: ChannelInfo) => void
  onChannelRemoved?: (channelId: number) => void
  onChannelUpdated?: (channel: ChannelInfo) => void
  onChannelData?: (channelId: number, data: any) => void
  onError?: (error: Error) => void
}

export class InspectorWebSocket {
  private ws: WebSocket | null = null
  private root: protobuf.Root | null = null
  private Packet: protobuf.Type | null = null
  private MessagePack: protobuf.Type | null = null
  private stubIdCounter = 1
  private stubCallbacks: Map<number, (msg: any) => void> = new Map()
  private options: InspectorWebSocketOptions
  private compressionType = 0 // NO_COMPRESSION

  constructor(options: InspectorWebSocketOptions) {
    this.options = options
    this.initProtobuf()
  }

  private async initProtobuf() {
    try {
      // Load protobuf definitions
      // Try multiple paths for proto file
      let loaded = false
      const protoPaths = [
        '/proto/channeld.proto',
        '/proto',
        'http://localhost:12108/proto',
      ]
      
      for (const path of protoPaths) {
        try {
          this.root = await protobuf.load(path)
          loaded = true
          console.log(`Loaded protobuf from: ${path}`)
          break
        } catch (err) {
          console.warn(`Failed to load from ${path}:`, err)
        }
      }
      
      if (!loaded || !this.root) {
        throw new Error('Failed to load protobuf from all paths. Please ensure proto files are available.')
      }
      
      // Verify proto file contains Debug messages
      const debugRequestType = this.root.lookupType('channeldpb.DebugGetChannelsRequest')
      if (!debugRequestType) {
        console.warn('Proto file may be outdated - DebugGetChannelsRequest not found')
        console.warn('Please ensure inspector/frontend/public/proto/channeld.proto is up to date')
      } else {
        console.log('✅ Proto file verified - Debug messages found')
      }
      
      const PacketType = this.root.lookupType('channeldpb.Packet')
      const MessagePackType = this.root.lookupType('channeldpb.MessagePack')
      
      if (!PacketType || !MessagePackType) {
        throw new Error('Failed to lookup Packet or MessagePack types')
      }
      
      this.Packet = PacketType as protobuf.Type
      this.MessagePack = MessagePackType as protobuf.Type
      
      // Load Inspector message types
      this.loadInspectorTypes()
      
      // Connect after protobuf is loaded
      this.connect()
    } catch (error) {
      console.error('Failed to load protobuf definitions:', error)
      this.options.onError?.(error as Error)
      // Still try to connect even if protobuf fails (for debugging)
      this.connect()
    }
  }

  private loadInspectorTypes() {
    if (!this.root) return

    // Load Inspector message types
    // These will be available after loading the proto file
    // For now, we'll define them dynamically or load from the proto file
  }

  private connect() {
    if (this.ws) {
      this.disconnect()
    }

    console.log(`Connecting to ${this.options.url}...`)
    this.ws = new WebSocket(this.options.url)
    this.ws.binaryType = 'arraybuffer'

    this.ws.onopen = () => {
      console.log('WebSocket connected, readyState:', this.ws?.readyState)
      
      // Verify WebSocket is in OPEN state
      if (this.ws?.readyState !== WebSocket.OPEN) {
        console.error('WebSocket not in OPEN state:', this.ws?.readyState)
        return
      }
      
      // Wait a bit to ensure connection is fully established and WebSocket upgrade is complete
      // WebSocket upgrade from HTTP might take a moment
      setTimeout(() => {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
          console.error('WebSocket not ready after delay')
          return
        }
        
        if (this.root && this.Packet && this.MessagePack) {
          console.log('Protobuf ready, sending AUTH...')
          this.sendAuth()
        } else {
          console.warn('Protobuf not ready yet, will retry...')
          // Retry after a short delay
          setTimeout(() => {
            if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
              console.error('WebSocket not ready on retry')
              return
            }
            
            if (this.root && this.Packet && this.MessagePack) {
              console.log('Protobuf ready (retry), sending AUTH...')
              this.sendAuth()
            } else {
              console.error('Protobuf failed to load, cannot send AUTH')
              this.options.onError?.(new Error('Protobuf not loaded'))
            }
          }, 500)
        }
      }, 200) // Increased delay to ensure WebSocket upgrade is complete
    }

    this.ws.onclose = (event) => {
      console.log('WebSocket disconnected', event.code, event.reason)
      if (event.code === 1006) {
        // Abnormal closure - connection was closed before handshake completed
        const errorMsg = 'Connection closed before handshake. Make sure channeld is running with WebSocket support: channeld.exe -cn ws -ca :12108'
        console.error(errorMsg)
        this.options.onError?.(new Error(errorMsg))
      }
      this.options.onDisconnected?.()
      this.ws = null
    }

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error)
      const errorMsg = 'WebSocket connection error. Make sure channeld is running with WebSocket support: channeld.exe -cn ws -ca :12108'
      console.error(errorMsg)
      this.options.onError?.(new Error(errorMsg))
    }

    this.ws.onmessage = (event) => {
      console.log('WebSocket message received, size:', event.data.byteLength, 'type:', event.data.constructor.name)
      
      // Log first few bytes for debugging
      if (event.data instanceof ArrayBuffer) {
        const arr = new Uint8Array(event.data)
        const preview = Array.from(arr.slice(0, 10)).map(b => '0x' + b.toString(16).padStart(2, '0')).join(' ')
        console.log('First 10 bytes:', preview)
      }
      
      this.handleMessage(event.data as ArrayBuffer)
    }
  }

  private sendAuth() {
    if (!this.root || !this.ws) {
      console.warn('Cannot send AUTH: protobuf not loaded or WebSocket not connected')
      if (!this.root) {
        console.error('Protobuf root is null - proto file may not be loaded correctly')
      }
      if (!this.ws) {
        console.error('WebSocket is null')
      }
      return
    }

    if (!this.Packet || !this.MessagePack) {
      console.error('Packet or MessagePack types not loaded')
      return
    }

    if (this.ws.readyState !== WebSocket.OPEN) {
      console.error('WebSocket is not in OPEN state:', this.ws.readyState)
      return
    }

    try {
      const AuthMessage = this.root.lookupType('channeldpb.AuthMessage')
      if (!AuthMessage) {
        console.error('AuthMessage type not found in proto')
        return
      }

      console.log('Sending AUTH message...')
      this.sendPacket(
        0, // GLOBAL channel
        1, // MessageType.AUTH
        AuthMessage,
        {
          playerIdentifierToken: 'inspector',
          loginToken: 'inspector',
        },
        (msg: any) => {
          console.log('Auth result:', msg)
          if (msg.result === 0) { // SUCCESSFUL
            console.log('Authentication successful, connId:', msg.connId)
            this.compressionType = msg.compressionType || 0
            // Now we're authenticated, trigger onConnected
            this.options.onConnected?.()
            // Request initial channel list
            this.getChannels()
            // Subscribe to channel list updates
            this.subscribeChannelList()
          } else {
            console.error('Authentication failed:', msg.result)
            this.options.onError?.(new Error(`Authentication failed: ${msg.result}`))
          }
        }
      )
    } catch (error) {
      console.error('Failed to send AUTH:', error)
      this.options.onError?.(error as Error)
    }
  }

  private handleMessage(data: ArrayBuffer) {
    if (!this.Packet) {
      console.warn('Packet type not loaded, cannot handle message')
      return
    }

    try {
      const uint8Arr = new Uint8Array(data)
      const packet = this.readPacket(uint8Arr)

      if (!packet.messages || packet.messages.length === 0) {
        console.warn('Received empty packet')
        return
      }

      for (const mp of packet.messages) {
        this.handleMessagePack(mp)
      }
    } catch (error) {
      console.error('Failed to handle message:', error)
      console.error('Message data length:', data.byteLength)
    }
  }

  private readPacket(uint8Arr: Uint8Array): any {
    if (!this.Packet) {
      throw new Error('Packet type not loaded')
    }

    // Validate packet header (CH)
    // Format: [67('C'), 72('H'), size_high, size_low, compressionType] + packet data
    if (uint8Arr.length < 5) {
      throw new Error('Packet too short')
    }

    // Check for 'CH' prefix (bytes 0-1)
    if (uint8Arr[0] !== 67 || uint8Arr[1] !== 72) {
      console.error('Invalid packet header:', {
        byte0: uint8Arr[0],
        byte1: uint8Arr[1],
        expected0: 67, // 'C'
        expected1: 72, // 'H'
        firstBytes: Array.from(uint8Arr.slice(0, 10)).map(b => '0x' + b.toString(16).padStart(2, '0')).join(' ')
      })
      throw new Error('Invalid packet header (not CH)')
    }

    // Parse packet size from bytes 2-3 (little-endian)
    // size = (tag[3]) | (tag[2] << 8)
    const packetSize = (uint8Arr[3] & 0xff) | ((uint8Arr[2] & 0xff) << 8)

    if (packetSize === 0) {
      throw new Error('Invalid packet size (0)')
    }

    if (packetSize > 0xffff) {
      throw new Error(`Packet size too large: ${packetSize}`)
    }

    const compressionType = uint8Arr[4]
    const fullSize = 5 + packetSize

    if (uint8Arr.length < fullSize) {
      throw new Error(`Packet incomplete: expected ${fullSize} bytes, got ${uint8Arr.length}`)
    }

    const bytes = uint8Arr.subarray(5, fullSize)

    // Handle compression if needed
    if (compressionType === 1) { // SNAPPY
      // Would need snappyjs library for decompression
      console.warn('Snappy compression not yet supported, trying to decode anyway')
    }

    try {
      return this.Packet.decode(bytes)
    } catch (error) {
      console.error('Failed to decode packet:', error)
      console.error('Packet size:', packetSize, 'Compression:', compressionType, 'Data length:', uint8Arr.length)
      throw error
    }
  }

  private handleMessagePack(mp: any) {
    const msgType = mp.msgType

    // Handle RPC responses (with stubId)
    if (mp.stubId !== 0) {
      const callback = this.stubCallbacks.get(mp.stubId)
      if (callback) {
        console.log(`Handling RPC response: msgType=${msgType}, stubId=${mp.stubId}`)
        try {
          // First try to get message type for Inspector messages
          let msgClass = this.getMessageType(msgType)
          
          // If not found, try to get from root (for AUTH, etc.)
          if (!msgClass && this.root) {
            const typeMap: Record<number, string> = {
              1: 'channeldpb.AuthResultMessage', // AUTH
            }
            const typeName = typeMap[msgType]
            if (typeName) {
              try {
                msgClass = this.root.lookupType(typeName)
              } catch {
                // Ignore
              }
            }
          }
          
          if (msgClass) {
            const msg = msgClass.decode(mp.msgBody)
            console.log(`Decoded message type ${msgType}:`, msg)
            callback(msg)
          } else {
            console.warn(`Message type ${msgType} not found for stubId ${mp.stubId}`)
          }
        } catch (error) {
          console.error(`Failed to decode message type ${msgType}:`, error)
          const hexBytes = Array.from(mp.msgBody.slice(0, 50)) as number[]
          console.error('Message body (hex):', hexBytes.map(b => '0x' + b.toString(16).padStart(2, '0')).join(' '))
        }
        this.stubCallbacks.delete(mp.stubId)
      } else {
        console.warn(`No callback found for stubId ${mp.stubId}, msgType=${msgType}`)
        console.warn('Active callbacks:', Array.from(this.stubCallbacks.keys()))
      }
      return
    }

    // Handle events (no stubId)
    this.handleEvent(msgType, mp.msgBody)
  }

  private getMessageType(msgType: number): protobuf.Type | null {
    if (!this.root) return null

    const typeMap: Record<number, string> = {
      [MessageType.DEBUG_GET_CHANNELS_RESULT]: 'channeldpb.DebugGetChannelsResponse',
      [MessageType.DEBUG_GET_CHANNEL_RESULT]: 'channeldpb.DebugGetChannelResponse',
      [MessageType.DEBUG_GET_CHANNEL_DATA_RESULT]: 'channeldpb.DebugGetChannelDataResponse',
      [MessageType.DEBUG_UPDATE_CHANNEL_DATA_RESULT]: 'channeldpb.DebugUpdateChannelDataResponse',
      [MessageType.DEBUG_ERROR]: 'channeldpb.DebugErrorResponse',
      [MessageType.DEBUG_CHANNEL_ADDED]: 'channeldpb.DebugChannelAddedEvent',
      [MessageType.DEBUG_CHANNEL_REMOVED]: 'channeldpb.DebugChannelRemovedEvent',
      [MessageType.DEBUG_CHANNEL_UPDATED]: 'channeldpb.DebugChannelUpdatedEvent',
      [MessageType.DEBUG_CHANNEL_DATA_UPDATED]: 'channeldpb.DebugChannelDataUpdatedEvent',
      [MessageType.DEBUG_CHANNEL_LIST_SNAPSHOT]: 'channeldpb.DebugChannelListSnapshotEvent',
    }

    const typeName = typeMap[msgType]
    if (!typeName) return null

    try {
      return this.root.lookupType(typeName)
    } catch {
      return null
    }
  }

  private handleEvent(msgType: number, msgBody: Uint8Array) {
    const msgClass = this.getMessageType(msgType)
    if (!msgClass) {
      console.warn(`Unknown message type: ${msgType}`)
      return
    }

    try {
      const msg = msgClass.decode(msgBody)

      switch (msgType) {
        case MessageType.DEBUG_CHANNEL_ADDED:
          if ((msg as any).channel) {
            this.options.onChannelAdded?.(this.convertChannelInfo((msg as any).channel))
          }
          break
        case MessageType.DEBUG_CHANNEL_REMOVED:
          if ((msg as any).channelId !== undefined) {
            this.options.onChannelRemoved?.((msg as any).channelId)
          }
          break
        case MessageType.DEBUG_CHANNEL_UPDATED:
          this.options.onChannelUpdated?.(this.convertChannelInfo(msg as any))
          break
        case MessageType.DEBUG_CHANNEL_DATA_UPDATED:
          // Handle channel data update event
          if ((msg as any).channelId !== undefined) {
            console.log('Channel data updated:', (msg as any).channelId)
          }
          break
        case MessageType.DEBUG_CHANNEL_LIST_SNAPSHOT:
          if ((msg as any).channels && Array.isArray((msg as any).channels)) {
            this.options.onChannels?.((msg as any).channels.map((c: any) => this.convertChannelInfo(c)))
          }
          break
      }
    } catch (error) {
      console.error(`Failed to handle event ${msgType}:`, error)
    }
  }

  private convertChannelInfo(pbChannel: any): ChannelInfo {
    if (!pbChannel) {
      throw new Error('pbChannel is null or undefined')
    }

    return {
      channelId: pbChannel.channelId || 0,
      channelType: this.getChannelTypeName(pbChannel.channelType || 0),
      ownerConnId: pbChannel.ownerConnId || 0,
      subscriberCount: pbChannel.subscriberCount || 0,
      metadata: pbChannel.metadata || '',
      createdTime: pbChannel.createdTime || 0,
    }
  }

  private getChannelTypeName(type: number): string {
    const types: Record<number, string> = {
      0: 'UNKNOWN',
      1: 'GLOBAL',
      2: 'PRIVATE',
      3: 'SUBWORLD',
      4: 'SPATIAL',
      5: 'ENTITY',
    }
    return types[type] || 'UNKNOWN'
  }

  private sendPacket(channelId: number, msgType: number, msgClass: protobuf.Type, msgData: any, callback?: (msg: any) => void): void {
    if (!this.ws) {
      console.error('WebSocket not connected')
      return
    }

    if (this.ws.readyState !== WebSocket.OPEN) {
      console.error('WebSocket not in OPEN state:', this.ws.readyState)
      return
    }

    if (!this.Packet || !this.MessagePack) {
      console.error('Protobuf types not initialized')
      return
    }

    if (!msgClass) {
      console.error('Message class is null')
      return
    }

    let stubId = 0
    try {
      const msg = msgClass.create(msgData)
      const msgBody = msgClass.encode(msg).finish()

      if (callback) {
        stubId = this.stubIdCounter++
        if (stubId >= 4294967296) stubId = 1
        this.stubCallbacks.set(stubId, callback)
      }

      const mp = this.MessagePack.create({
        channelId,
        broadcast: 0,
        stubId,
        msgType,
        msgBody,
      })

      const packet = this.Packet.create({
        messages: [mp],
      })

      const bytes = this.Packet.encode(packet).finish()
      const data = this.createPacketBuffer(bytes)

      // Verify packet header before sending
      if (data[0] !== 67 || data[1] !== 72) {
        throw new Error('Invalid packet header created')
      }

      console.log(`Sending message type ${msgType}, stubId ${stubId}, packet size ${data.length} bytes (payload: ${bytes.length} bytes)`)
      console.log('Packet header:', Array.from(data.slice(0, 5)).map(b => '0x' + b.toString(16).padStart(2, '0')).join(' '))
      
      this.ws.send(data)
    } catch (error) {
      console.error('Failed to send packet:', error)
      console.error('Message type:', msgType, 'Data:', msgData)
      if (callback && stubId) {
        this.stubCallbacks.delete(stubId)
      }
    }
  }

  private createPacketBuffer(bytes: Uint8Array): Uint8Array {
    // Create packet header: [67, 72, 78, 76, compressionType] + packet data
    // Format matches channeld's packet format (see pkg/channeld/connection.go):
    // - tag[0] = 67 ('C')
    // - tag[1] = 72 ('H')
    // - tag[2] = high byte of size (len >> 8)
    // - tag[3] = low byte of size (len & 0xff)
    // - tag[4] = compression type
    // 
    // channeld's readSize reads: size = int(tag[3]) | int(tag[2])<<8
    const data = new Uint8Array(bytes.length + 5)
    const tag = new Uint8Array(5)
    
    const len = bytes.length
    
    tag[0] = 67  // 'C'
    tag[1] = 72  // 'H'
    tag[2] = (len >> 8) & 0xff // High byte (must be set even if 0)
    tag[3] = len & 0xff        // Low byte
    tag[4] = this.compressionType

    data.set(tag, 0)
    data.set(bytes, 5)

    // Verify the packet format
    if (tag[0] !== 67 || tag[1] !== 72) {
      throw new Error('Invalid packet tag created')
    }

    return data
  }

  // Public API methods
  public getChannels(page: number = 1, pageSize: number = 100) {
    if (!this.root) {
      console.warn('Protobuf root not loaded, cannot get channels')
      return
    }

    console.log('Requesting channels, page:', page, 'pageSize:', pageSize)
    
    // Try to lookup the message type
    let RequestType: protobuf.Type | null = null
    try {
      RequestType = this.root.lookupType('channeldpb.DebugGetChannelsRequest')
    } catch (error) {
      console.error('Failed to lookup DebugGetChannelsRequest:', error)
      // Try alternative names in case proto file is outdated
      try {
        RequestType = this.root.lookupType('channeldpb.InspectorGetChannelsRequest')
        console.warn('Using deprecated InspectorGetChannelsRequest, please update proto file')
      } catch (e) {
        console.error('Failed to lookup InspectorGetChannelsRequest:', e)
      }
    }
    
    if (!RequestType) {
      console.error('DebugGetChannelsRequest type not found in proto file')
      console.error('Available types:', this.root ? Object.keys(this.root.nested || {}).join(', ') : 'root is null')
      return
    }

    this.sendPacket(
      0, // GLOBAL channel
      MessageType.DEBUG_GET_CHANNELS,
      RequestType,
      { page, pageSize },
      (msg: any) => {
        console.log('Received channels response:', msg)
        if (msg.channels && Array.isArray(msg.channels)) {
          console.log(`Received ${msg.channels.length} channels, total: ${msg.total}`)
          const convertedChannels = msg.channels.map((c: any) => this.convertChannelInfo(c))
          console.log('Converted channels:', convertedChannels)
          this.options.onChannels?.(convertedChannels)
        } else {
          console.warn('Response does not contain channels array:', msg)
        }
      }
    )
  }

  public getChannel(channelId: number) {
    if (!this.root) return

    const RequestType = this.root.lookupType('channeldpb.DebugGetChannelRequest')
    this.sendPacket(
      0,
      MessageType.DEBUG_GET_CHANNEL,
      RequestType,
      { channelId },
      (msg: any) => {
        // Handle response
        console.log('Channel details:', msg)
      }
    )
  }

  public getChannelData(channelId: number): Promise<any> {
    return new Promise((resolve, reject) => {
      if (!this.root) {
        reject(new Error('Protobuf not initialized'))
        return
      }

      const RequestType = this.root.lookupType('channeldpb.DebugGetChannelDataRequest')
      this.sendPacket(
        0,
        MessageType.DEBUG_GET_CHANNEL_DATA,
        RequestType,
        { channelId },
        (msg: any) => {
          if (msg.hasData && msg.jsonData) {
            try {
              const data = JSON.parse(msg.jsonData)
              this.options.onChannelData?.(channelId, data)
              resolve(data)
            } catch (error) {
              reject(error)
            }
          } else {
            resolve(null)
          }
        }
      )
    })
  }

  public updateChannelData(channelId: number, jsonPath: string, value: string, valueType: string): Promise<void> {
    return new Promise((resolve, reject) => {
      if (!this.root) {
        reject(new Error('Protobuf not initialized'))
        return
      }

      const RequestType = this.root.lookupType('channeldpb.DebugUpdateChannelDataRequest')
      this.sendPacket(
        0,
        MessageType.DEBUG_UPDATE_CHANNEL_DATA,
        RequestType,
        { channelId, jsonPath, value, valueType },
        (msg: any) => {
          if (msg.success) {
            resolve()
          } else {
            reject(new Error(msg.error || 'Update failed'))
          }
        }
      )
    })
  }

  public subscribeChannelList() {
    if (!this.root) {
      console.warn('Protobuf not loaded, cannot subscribe to channel list')
      return
    }

    try {
      const RequestType = this.root.lookupType('channeldpb.DebugSubscribeChannelListRequest')
      if (!RequestType) {
        console.error('DebugSubscribeChannelListRequest type not found')
        return
      }

      this.sendPacket(
        0,
        MessageType.DEBUG_SUBSCRIBE_CHANNEL_LIST,
        RequestType,
        { enableFullSync: true, fullSyncIntervalSec: 300 }
      )
    } catch (error) {
      console.error('Failed to subscribe to channel list:', error)
    }
  }

  public unsubscribeChannelList() {
    if (!this.root) return

    const RequestType = this.root.lookupType('channeldpb.DebugUnsubscribeChannelListRequest')
    this.sendPacket(
      0,
      MessageType.DEBUG_UNSUBSCRIBE_CHANNEL_LIST,
      RequestType,
      {}
    )
  }

  public disconnect() {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    this.stubCallbacks.clear()
  }
}
