// WebSocket manager skeleton. Connects to GET /ws?subscribe=all|<id>.
// TODO(P5): real connection lifecycle — open/close handlers, exponential-backoff
// reconnect, heartbeat (ping/pong), message dispatch (hello/update/batchUpdate),
// poll fallback wiring at the store level.

/** Subscription scope: all servers or a single server id. */
export type WSScope = 'all' | string

/** Inbound message shape (refined in P5 against the WS contract). */
export interface WSMessage {
  type: string
  [key: string]: unknown
}

export type WSCallback = (msg: WSMessage) => void

export class WSManager {
  private socket: WebSocket | null = null
  private callbacks = new Set<WSCallback>()

  /** Open a WebSocket subscribed to the given scope. */
  connect(scope: WSScope = 'all'): void {
    // TODO(P5): build ws URL from base, open socket, attach listeners, reconnect.
    void scope
    console.log('[WSManager] connect: not implemented (P5)')
  }

  /** Register a callback for inbound messages. Returns an unsubscribe fn. */
  subscribe(cb: WSCallback): () => void {
    // TODO(P5): dispatch real messages to callbacks.
    this.callbacks.add(cb)
    return () => this.callbacks.delete(cb)
  }

  /** Close the socket and clear timers. */
  disconnect(): void {
    // TODO(P5): close socket, clear reconnect/heartbeat timers.
    this.socket?.close()
    this.socket = null
    this.callbacks.clear()
    console.log('[WSManager] disconnect: not implemented (P5)')
  }
}
