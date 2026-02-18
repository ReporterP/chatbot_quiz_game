import { useEffect, useRef } from 'react';

export default function useRoomWebSocket(roomCode, onMessage) {
  const wsRef = useRef(null);

  useEffect(() => {
    if (!roomCode) return;

    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const url = `${proto}//${window.location.host}/ws/room/${roomCode}`;

    const connect = () => {
      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onmessage = (e) => {
        try {
          const msg = JSON.parse(e.data);
          onMessage(msg);
        } catch { /* ignore */ }
      };

      ws.onclose = () => {
        setTimeout(connect, 3000);
      };
    };

    connect();

    return () => {
      if (wsRef.current) {
        wsRef.current.onclose = null;
        wsRef.current.close();
      }
    };
  }, [roomCode]);
}
