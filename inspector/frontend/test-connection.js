// Simple Node.js script to test WebSocket connection
const WebSocket = require('ws');

const ws = new WebSocket('ws://localhost:12108');

ws.on('open', () => {
  console.log('✓ WebSocket connected');
  
  // Create a simple test packet with CHNL header
  const testData = Buffer.from([1, 2, 3, 4, 5]);
  const packet = Buffer.alloc(testData.length + 5);
  
  // CHNL header
  packet[0] = 67;  // 'C'
  packet[1] = 72;   // 'H'
  packet[2] = (testData.length >> 8) & 0xff;  // High byte
  packet[3] = testData.length & 0xff;         // Low byte
  packet[4] = 0;   // Compression type
  testData.copy(packet, 5);
  
  console.log('Sending test packet...');
  console.log('Packet header:', Array.from(packet.slice(0, 5)).map(b => '0x' + b.toString(16).padStart(2, '0')).join(' '));
  ws.send(packet);
  
  setTimeout(() => {
    console.log('Closing connection...');
    ws.close();
    process.exit(0);
  }, 2000);
});

ws.on('error', (error) => {
  console.error('✗ WebSocket error:', error.message);
  process.exit(1);
});

ws.on('close', (code, reason) => {
  console.log('WebSocket closed:', code, reason.toString());
  process.exit(code === 1000 ? 0 : 1);
});

ws.on('message', (data) => {
  console.log('✓ Received message:', data.length, 'bytes');
  const view = new Uint8Array(data);
  console.log('First 20 bytes:', Array.from(view.slice(0, 20)).map(b => '0x' + b.toString(16).padStart(2, '0')).join(' '));
});

setTimeout(() => {
  console.error('✗ Connection timeout');
  process.exit(1);
}, 5000);
