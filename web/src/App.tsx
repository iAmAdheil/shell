import * as Sentry from '@sentry/react'
import { useCallback, useEffect, useState } from 'react'
import { checkSession, createSession, logOut, type Me } from './api'
import { Replay } from './Replay'
import { Terminal, type RosterUser, type TerminalStatus } from './Terminal'
import { useMe } from './useMe'
import './App.css'

/**
 * A Session Code rides in the URL hash, so a shareable link needs nothing from
 * the server. The server answers /api only and never serves this app, so a
 * path-shaped link would 404 anywhere but the dev server.
 */
function codeFromHash(): string | null {
  return /^#\/s\/([\w-]+)$/.exec(location.hash)?.[1] ?? null
}

function writeHash(code: string | null) {
  // replaceState keeps a shared link out of the back stack and, unlike
  // assigning location.hash, does not fire hashchange back at us.
  history.replaceState(null, '', code ? `#/s/${code}` : location.pathname + location.search)
}

function shareUrl(code: string): string {
  return `${location.origin}${location.pathname}#/s/${code}`
}

function App() {
  const { state, reload } = useMe()
  const [code, setCode] = useState<string | null>(null)
  const [typedCode, setTypedCode] = useState('')
  const [problem, setProblem] = useState<string | null>(null)
  const [roster, setRoster] = useState<RosterUser[]>([])
  const [status, setStatus] = useState<TerminalStatus>({ connected: false, rows: 0, cols: 0 })

  const loggedIn = state.status === 'logged-in'

  const leaveSession = useCallback(() => {
    setCode(null)
    setRoster([])
    writeHash(null)
  }, [])

  const enterSession = useCallback((wanted: string) => {
    setProblem(null)
    setCode(wanted)
    writeHash(wanted)
  }, [])

  // A Session Code in the link is only worth opening a terminal on once the
  // server says it still runs. This also covers a link pasted into the tab
  // that is already open, which changes the hash without reloading.
  useEffect(() => {
    if (!loggedIn) return
    let cancelled = false

    const open = async () => {
      const wanted = codeFromHash()
      if (!wanted) return
      try {
        await checkSession(wanted)
        if (!cancelled) setCode(wanted)
      } catch (err) {
        if (cancelled) return
        setProblem(message(err))
        writeHash(null)
      }
    }

    void open()
    window.addEventListener('hashchange', open)
    return () => {
      cancelled = true
      window.removeEventListener('hashchange', open)
    }
  }, [loggedIn])

  const startSession = async () => {
    setProblem(null)
    try {
      enterSession(await createSession())
    } catch (err) {
      // A Session that will not start is a server fault, not a typo, so it is
      // worth reporting. A Session Code that will not open usually is a typo.
      Sentry.captureException(err)
      setProblem(message(err))
    }
  }

  /** Opens an existing Session, but only after the Code is known to be live. */
  const joinSession = async (event: React.FormEvent) => {
    event.preventDefault()
    setProblem(null)

    const wanted = typedCode.trim().toLowerCase()
    if (!wanted) return

    try {
      await checkSession(wanted)
      enterSession(wanted)
      setTypedCode('')
    } catch (err) {
      setProblem(message(err))
    }
  }

  if (state.status === 'loading') {
    return (
      <>
        <TopBar />
        <p className="centered">Checking your login…</p>
      </>
    )
  }

  if (state.status === 'error') {
    return (
      <>
        <TopBar />
        <p className="centered" role="alert">
          Could not reach the server. {state.message}
        </p>
      </>
    )
  }

  if (state.status === 'logged-out') {
    return <Landing />
  }

  if (code) {
    return (
      <Session
        code={code}
        me={state.me}
        roster={roster}
        status={status}
        problem={problem}
        onRoster={setRoster}
        onStatus={setStatus}
        onLeave={leaveSession}
        onEnded={leaveSession}
      />
    )
  }

  return (
    <>
      <TopBar>
        <span className="spacer" />
        <Who me={state.me} />
        <button
          type="button"
          className="btn btn-ghost btn-sm"
          onClick={async () => {
            await logOut()
            setProblem(null)
            leaveSession()
            await reload()
          }}
        >
          Log out
        </button>
      </TopBar>
      <Lobby
        problem={problem}
        typedCode={typedCode}
        onTypedCode={setTypedCode}
        onStart={startSession}
        onJoin={joinSession}
      />
    </>
  )
}

function TopBar({ children }: { children?: React.ReactNode }) {
  return (
    <header className="topbar">
      <span className="brand">
        shell
        <span className="caret" aria-hidden="true" />
      </span>
      {children}
    </header>
  )
}

/**
 * The logged-out page. It replays a recorded Session instead of describing
 * one, because the product is legible on sight and a tagline is not.
 */
function Landing() {
  return (
    <>
      <TopBar>
        <span className="spacer" />
        <span className="badge">read-only demo</span>
        <a className="btn btn-primary btn-sm" href="/api/auth/google/start">
          Sign in with Google
        </a>
      </TopBar>

      <main className="page">
        <div className="lede">
          <h1>
            One shell.
            <br />
            <span className="accent">Many keyboards.</span>
          </h1>
          <p className="sub">
            A real terminal in the browser that two people type into at once. Share a link and
            they land in the same shell, with every line of scrollback already there.
          </p>
        </div>

        <div className="term-window">
          <div className="term-head">
            <span className="dots" aria-hidden="true">
              <i />
              <i />
              <i />
            </span>
            <span className="term-title">a recorded Session</span>
          </div>
          <Replay />
          <div className="term-foot">
            <span className="badge replaying">
              <span className="pulse" aria-hidden="true" />
              replaying
            </span>
            <span className="msg">Sign in to open a Session of your own.</span>
            <span className="spacer" />
            <a className="btn btn-primary btn-sm" href="/api/auth/google/start">
              Take the keyboard →
            </a>
          </div>
        </div>

        <div className="facts">
          <div className="fact">
            <b>share a link</b>
            <span>
              A Session Code is a link. Send it and they are in, with no install and no setup
              past one Google click.
            </span>
          </div>
          <div className="fact">
            <b>full scrollback</b>
            <span>
              Whoever joins sees the Session from its first command, not from the moment they
              arrived.
            </span>
          </div>
          <div className="fact">
            <b>one input stream</b>
            <span>
              Every keystroke from every User merges into the same shell. The bar shows who is
              in it.
            </span>
          </div>
        </div>
      </main>
    </>
  )
}

type LobbyProps = {
  problem: string | null
  typedCode: string
  onTypedCode: (value: string) => void
  onStart: () => void
  onJoin: (event: React.FormEvent) => void
}

function Lobby({ problem, typedCode, onTypedCode, onStart, onJoin }: LobbyProps) {
  return (
    <main className="page">
      {problem && <Notice text={problem} page />}

      <div className="lede">
        <h1>
          Start a shell.
          <br />
          <span className="accent">Bring someone into it.</span>
        </h1>
        <p className="sub">
          A Session is a real shell in its own container. It runs until the last person leaves.
        </p>
      </div>

      <div className="start">
        <div className="start-card">
          <h2>New Session</h2>
          <p>Opens a fresh shell and gives you a link to share.</p>
          <button type="button" className="btn btn-primary" onClick={onStart}>
            New Session
          </button>
        </div>

        <div className="start-card">
          <h2>Join a Session</h2>
          <p>Paste a Session Code. It looks like amber-koi-7412.</p>
          <form className="join" onSubmit={onJoin}>
            <input
              className="input"
              value={typedCode}
              onChange={(event) => onTypedCode(event.target.value)}
              placeholder="session-code"
              aria-label="Session Code"
              autoCapitalize="off"
              autoCorrect="off"
              spellCheck={false}
            />
            <button type="submit" className="btn">
              Join
            </button>
          </form>
        </div>
      </div>
    </main>
  )
}

type SessionProps = {
  code: string
  me: Me
  roster: RosterUser[]
  status: TerminalStatus
  problem: string | null
  onRoster: (users: RosterUser[]) => void
  onStatus: (status: TerminalStatus) => void
  onLeave: () => void
  onEnded: () => void
}

function Session({
  code,
  me,
  roster,
  status,
  problem,
  onRoster,
  onStatus,
  onLeave,
  onEnded,
}: SessionProps) {
  const alone = roster.length < 2

  return (
    <div className="session">
      <TopBar>
        <div className="codepill">
          <code>{code}</code>
          <CopyButton value={shareUrl(code)} label="Copy link" />
        </div>
        <span className="spacer" />
        {roster.length > 0 && <Roster roster={roster} meId={me.id} />}
        <button type="button" className="btn btn-danger btn-sm" onClick={onLeave}>
          Leave Session
        </button>
      </TopBar>

      {problem && <Notice text={problem} />}

      {/* The invite band earns its space only while there is nobody to talk
          to. Once a second User arrives it goes, and the bar's Code pill is
          enough to invite a third. */}
      {alone && (
        <div className="invite">
          <span className="txt">
            <b>Nobody else is here yet.</b> Send this link and they join this shell.
          </span>
          <span className="spacer" />
          <div className="linkbox">
            <code>{shareUrl(code)}</code>
            <CopyButton value={shareUrl(code)} label="Copy link" />
          </div>
        </div>
      )}

      <Terminal code={code} onEnded={onEnded} onRoster={onRoster} onStatus={onStatus} />

      <div className="status">
        <span>
          <span className={`dot${status.connected ? '' : ' off'}`} aria-hidden="true" />
          {status.connected ? 'connected' : 'disconnected'}
        </span>
        <span>
          {roster.length} {roster.length === 1 ? 'keyboard' : 'keyboards'} live
        </span>
        {status.cols > 0 && (
          <span>
            {status.cols} × {status.rows}
          </span>
        )}
        <span className="spacer" />
        <span>scrollback from the first command</span>
      </div>
    </div>
  )
}

function Roster({ roster, meId }: { roster: RosterUser[]; meId: string }) {
  return (
    <ul className="roster" aria-label="Connected Users">
      {roster.map((user) => (
        <li key={user.id} className={`who${user.id === meId ? ' you' : ''}`} title={user.name}>
          <Avatar user={user} />
          <span className="who-name">{user.id === meId ? 'you' : user.name}</span>
        </li>
      ))}
    </ul>
  )
}

function Who({ me }: { me: Me }) {
  return (
    <span className="who you">
      <Avatar user={me} />
      <span className="who-name">{me.name}</span>
    </span>
  )
}

function Avatar({ user }: { user: RosterUser | Me }) {
  if (user.avatarUrl) {
    return <img className="av" src={user.avatarUrl} alt="" width={22} height={22} />
  }
  return (
    <span className="av" aria-hidden="true">
      {user.name.charAt(0)}
    </span>
  )
}

/** Copies a value and says so, because a copy with no feedback reads as a dead button. */
function CopyButton({ value, label }: { value: string; label: string }) {
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!copied) return
    const timer = setTimeout(() => setCopied(false), 1500)
    return () => clearTimeout(timer)
  }, [copied])

  return (
    <button
      type="button"
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(value)
          setCopied(true)
        } catch {
          // Clipboard access can be refused. Saying "Copied" then would lie.
        }
      }}
    >
      {copied ? 'Copied' : label}
    </button>
  )
}

function Notice({ text, page }: { text: string; page?: boolean }) {
  return (
    <p className={`notice${page ? ' page-notice' : ''}`} role="alert">
      {text}
    </p>
  )
}

/** Pulls the readable text out of whatever was thrown. */
function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

export default App
