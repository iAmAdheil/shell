import { useEffect, useState } from 'react'

/** One coloured run of the transcript. `c` is a class from App.css's .replay. */
type Token = { t: string; c?: string }

// A recorded Session, replayed on the landing page. It is the only thing a
// logged-out visitor can see, so it shows the product doing the one thing the
// tagline used to only claim: two people typing into the same shell.
const TRANSCRIPT: Token[] = [
  { t: '➜', c: 'p' },
  { t: '  ' },
  { t: 'shell', c: 'b' },
  { t: ' git:(', c: 'g' },
  { t: 'main', c: 'm' },
  { t: ') ', c: 'g' },
  { t: 'npm run build\n\n' },
  { t: '> web@0.0.0 build\n> tsc -b && vite build\n\n', c: 'g' },
  { t: 'vite v8.2.2', c: 'm' },
  { t: ' building for production...\n' },
  { t: '✓', c: 'p' },
  { t: ' 84 modules transformed.\n' },
  { t: 'dist/index.html                  ' },
  { t: '0.46 kB\n', c: 'g' },
  { t: 'dist/assets/index-C1t9wQ2p.css   ' },
  { t: '3.71 kB\n', c: 'g' },
  { t: 'dist/assets/index-DkPq8xLm.js  ' },
  { t: '412.88 kB\n', c: 'g' },
  { t: '✓ built in 1.24s\n\n', c: 'p' },
  { t: '➜', c: 'p' },
  { t: '  ' },
  { t: 'shell', c: 'b' },
  { t: ' git:(', c: 'g' },
  { t: 'main', c: 'm' },
  { t: ') ', c: 'g' },
  { t: 'docker ps --format "table {{.Names}}\\t{{.Status}}"\n' },
  { t: 'NAMES               STATUS\nshell-postgres-1    Up 4 minutes (healthy)\n\n' },
  { t: '➜', c: 'p' },
  { t: '  ' },
  { t: 'shell', c: 'b' },
  { t: ' git:(', c: 'g' },
  { t: 'main', c: 'm' },
  { t: ') ', c: 'g' },
  { t: 'tail -f logs/server.log' },
]

const TOTAL = TRANSCRIPT.reduce((n, token) => n + token.t.length, 0)

/** Where each token starts in the transcript, so rendering needs no running total. */
const OFFSETS: number[] = []
for (let i = 0, start = 0; i < TRANSCRIPT.length; i++) {
  OFFSETS.push(start)
  start += TRANSCRIPT[i].t.length
}

/** Characters revealed per tick. A whole line lands about every 40ms. */
const PER_TICK = 3
const TICK_MS = 16

export function Replay() {
  // Anyone who asked for less motion gets the finished transcript at once.
  const [shown, setShown] = useState(() =>
    window.matchMedia('(prefers-reduced-motion: reduce)').matches ? TOTAL : 0,
  )

  // Watching only whether anything is left to reveal keeps this effect from
  // tearing down and rebuilding the interval on every tick.
  const done = shown >= TOTAL

  useEffect(() => {
    if (done) return
    const timer = setInterval(() => setShown((n) => Math.min(n + PER_TICK, TOTAL)), TICK_MS)
    return () => clearInterval(timer)
  }, [done])

  return (
    <div className="replay" aria-label="A recorded Session replaying">
      {TRANSCRIPT.map((token, i) => {
        const text = token.t.slice(0, Math.max(0, shown - OFFSETS[i]))
        if (!text) return null
        return (
          <span key={i} className={token.c}>
            {text}
          </span>
        )
      })}
      <span className="cur" />
    </div>
  )
}
