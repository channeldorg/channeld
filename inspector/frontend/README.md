# Channeld Inspector Frontend

Web-based management interface for Channeld server.

## Tech Stack

- **React 18** + **TypeScript**
- **Vite** - Build tool
- **protobufjs** - Protobuf support
- **Zustand** - State management
- **WebSocket** - Real-time communication

## Getting Started

### Prerequisites

- Node.js 18+ 
- npm or yarn

### Installation

```bash
cd inspector/frontend
npm install
```

### Development

```bash
npm run dev
```

The app will be available at `http://localhost:5173`

### Build

```bash
npm run build
```

### Protobuf Setup

The frontend needs access to the Protobuf definitions. You have two options:

1. **Serve proto files from backend** (Recommended)
   - The backend should serve the proto files at `/proto` endpoint
   - The WebSocket service will load them automatically

2. **Copy proto files to public directory**
   - Copy `pkg/channeldpb/channeld.proto` to `public/proto/`
   - Update the load path in `src/services/websocket.ts`

## Project Structure

```
inspector/frontend/
├── src/
│   ├── components/          # React components
│   │   ├── TitleBar.tsx     # Top title bar
│   │   ├── Sidebar.tsx      # Channel list sidebar
│   │   ├── MainContent.tsx  # Main content area
│   │   ├── ChannelDetail.tsx # Channel detail view
│   │   └── JsonTreeEditor.tsx # Editable JSON tree
│   ├── services/
│   │   └── websocket.ts     # WebSocket & Protobuf handling
│   ├── stores/
│   │   └── inspectorStore.ts # Zustand state store
│   ├── App.tsx              # Main app component
│   └── main.tsx             # Entry point
├── public/                  # Static files
├── package.json
└── vite.config.ts
```

## Features

- ✅ Channel list display
- ✅ Channel detail view
- ✅ JSON tree editor (editable leaf properties)
- ✅ Real-time channel updates
- ✅ WebSocket connection management

## Connecting to Server

By default, the frontend connects to `ws://localhost:80` (Inspector connection type).

**IMPORTANT**: Inspector connections use a dedicated `ConnectionType_INSPECTOR` on port 80. The server automatically starts the Inspector listener when started normally.

To start the server:
```bash
# Windows
channeld.exe
# or use the provided script
start-channeld-inspector.bat

# Linux/Mac
./channeld
```

The Inspector listener uses WebSocket by default on port 80, separate from client connections (port 12108, TCP).

To change the server URL, update the `url` in `src/stores/inspectorStore.ts`:

```typescript
const ws = new InspectorWebSocket({
  url: 'ws://your-server:12108',
  // ...
})
```

## Development Notes

- The WebSocket service uses protobufjs to serialize/deserialize messages
- Message types are defined in `src/services/websocket.ts`
- State is managed using Zustand in `src/stores/inspectorStore.ts`
- The JSON tree editor supports inline editing of leaf properties
