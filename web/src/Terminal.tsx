import { FitAddon } from '@xterm/addon-fit'
import { Terminal as XTerm } from '@xterm/xterm'
import { useEffect, useRef } from 'react'
import '@xterm/xterm/css/xterm.css'

export type RosterUser = {
  id: string
  name: string
  avatarUrl: string
}

/** What the status strip below the terminal reports. */
export type TerminalStatus = {
  connected: boolean
  rows: number
  cols: number
}

type Props = {
  code: string
  onEnded: () => void
  onRoster: (users: RosterUser[]) => void
  onStatus: (status: TerminalStatus) => void
}

/** Renders one Session's terminal and keeps it wired to the server. */
export function Terminal({ code, onEnded, onRoster, onStatus }: Props) {
  const holder = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const node = holder.current
    if (!node) return

    const term = new XTerm({
      convertEol: false,
      cursorBlink: true,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
      fontSize: 13,
      lineHeight: 1.35,
      // Matches the tokens in index.css, so the terminal and its chrome read
      // as one surface rather than a black rectangle pasted onto the page.
      theme: {
        background: '#08080a',
        foreground: '#cfcfd6',
        cursor: '#c084fc',
        cursorAccent: '#08080a',
        selectionBackground: 'rgba(192, 132, 252, 0.28)',
      },
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(node)
    fit.fit()

    let socket: WebSocket | null = null
    let closedByCleanup = false

    /** Tells the server how big this viewport is, so the shell matches it. */
    const reportSize = () => {
      fit.fit()
      const open = socket?.readyState === WebSocket.OPEN
      if (open) {
        socket?.send(JSON.stringify({ type: 'resize', rows: term.rows, cols: term.cols }))
      }
      onStatus({ connected: open, rows: term.rows, cols: term.cols })
    }

    const connect = () => {
      const url = `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/api/sessions/${code}/ws`
      socket = new WebSocket(url)
      socket.binaryType = 'arraybuffer'

      socket.onopen = reportSize
      socket.onmessage = (event) => {
        // Terminal bytes arrive as binary frames, control messages as JSON text.
        if (typeof event.data !== 'string') {
          term.write(new Uint8Array(event.data as ArrayBuffer))
          return
        }

        const msg = JSON.parse(event.data) as { type: string; users?: RosterUser[] }
        if (msg.type === 'roster' && msg.users) onRoster(msg.users)
      }
      // A close this effect's own cleanup caused must not end the Session.
      socket.onclose = () => {
        if (closedByCleanup) return
        onStatus({ connected: false, rows: term.rows, cols: term.cols })
        onEnded()
      }
    }

    // The Session ends the moment its last User disconnects (CONTEXT.md), so
    // one connection that is opened and dropped destroys the Session. React
    // runs this effect, its cleanup, then this effect again in development,
    // which is exactly that pattern. The timer holds the socket back until
    // that remount is over, and the cleanup cancels it if the remount happens.
    const pending = setTimeout(connect, 0)

    const typing = term.onData((data) => {
      if (socket?.readyState === WebSocket.OPEN) {
        socket.send(new TextEncoder().encode(data))
      }
    })

    const observer = new ResizeObserver(reportSize)
    observer.observe(node)

    return () => {
      closedByCleanup = true
      clearTimeout(pending)
      observer.disconnect()
      typing.dispose()
      socket?.close()
      term.dispose()
    }
  }, [code, onEnded, onRoster, onStatus])

  return <div className="terminal" ref={holder} />
}
