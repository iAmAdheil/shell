import { FitAddon } from '@xterm/addon-fit'
import { Terminal as XTerm } from '@xterm/xterm'
import { useEffect, useRef } from 'react'
import '@xterm/xterm/css/xterm.css'

type Props = {
  code: string
  onEnded: () => void
}

/** Renders one Session's terminal and keeps it wired to the server. */
export function Terminal({ code, onEnded }: Props) {
  const holder = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const node = holder.current
    if (!node) return

    const term = new XTerm({
      convertEol: false,
      cursorBlink: true,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
      fontSize: 13,
      theme: { background: '#0d0d0d' },
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(node)
    fit.fit()

    const url = `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/api/sessions/${code}/ws`
    const socket = new WebSocket(url)
    socket.binaryType = 'arraybuffer'

    /** Tells the server how big this viewport is, so the shell matches it. */
    const reportSize = () => {
      fit.fit()
      if (socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({ type: 'resize', rows: term.rows, cols: term.cols }))
      }
    }

    socket.onopen = reportSize
    socket.onmessage = (event) => {
      term.write(new Uint8Array(event.data as ArrayBuffer))
    }
    socket.onclose = onEnded

    const typing = term.onData((data) => {
      if (socket.readyState === WebSocket.OPEN) {
        socket.send(new TextEncoder().encode(data))
      }
    })

    const observer = new ResizeObserver(reportSize)
    observer.observe(node)

    return () => {
      observer.disconnect()
      typing.dispose()
      socket.close()
      term.dispose()
    }
  }, [code, onEnded])

  return <div className="terminal" ref={holder} />
}
