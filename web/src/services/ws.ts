// WebSocket manager for the /ws realtime feed. Builds a same-origin ws/wss URL,
// replies to server heartbeats, and reconnects with capped exponential backoff.

import type { WsFrame } from '@/types'

/** Subscription scope: all servers or a single server id. */
export type WSScope = 'all' | string

export type WSFrameHandler = (frame: WsFrame) => void

const MIN_BACKOFF = 1000
const MAX_BACKOFF = 30000

export class WSManager {
  private socket: WebSocket | null = null
  private onFrame: WSFrameHandler | null = null
  private scope: WSScope = 'all'
  private backoff = MIN_BACKOFF
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private closedByUser = false

  /** Open a socket for `scope` and stream frames to `onFrame`. */
  connect(scope: WSScope, onFrame: WSFrameHandler): void {
    this.disconnect()
    this.closedByUser = false
    this.scope = scope
    this.onFrame = onFrame
    this.open()
  }

  private url(): string {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${window.location.host}/ws?subscribe=${encodeURIComponent(this.scope)}`
  }

  private open(): void {
    let socket: WebSocket
    try {
      socket = new WebSocket(this.url())
    } catch {
      this.scheduleReconnect()
      return
    }
    this.socket = socket

    socket.onopen = () => {
      this.backoff = MIN_BACKOFF
    }

    socket.onmessage = (ev) => {
      let frame: WsFrame
      try {
        frame = JSON.parse(ev.data as string) as WsFrame
      } catch {
        return
      }
      // Reply to heartbeats to keep the connection alive.
      if (frame.type === 'ping') {
        this.send({ type: 'pong' })
        return
      }
      this.onFrame?.(frame)
    }

    socket.onclose = () => {
      this.socket = null
      if (!this.closedByUser) this.scheduleReconnect()
    }

    socket.onerror = () => {
      // Let onclose drive reconnection; close proactively if still open.
      try {
        socket.close()
      } catch {
        /* ignore */
      }
    }
  }

  private send(payload: unknown): void {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      try {
        this.socket.send(JSON.stringify(payload))
      } catch {
        /* ignore */
      }
    }
  }

  private scheduleReconnect(): void {
    if (this.closedByUser || this.reconnectTimer) return
    const delay = this.backoff
    this.backoff = Math.min(this.backoff * 2, MAX_BACKOFF)
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      if (!this.closedByUser) this.open()
    }, delay)
  }

  /** True while a socket is open. */
  get isOpen(): boolean {
    return this.socket?.readyState === WebSocket.OPEN
  }

  /** Close the socket and cancel any pending reconnect. */
  disconnect(): void {
    this.closedByUser = true
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.socket) {
      const s = this.socket
      this.socket = null
      s.onopen = s.onmessage = s.onclose = s.onerror = null
      try {
        s.close()
      } catch {
        /* ignore */
      }
    }
    this.backoff = MIN_BACKOFF
  }
}
