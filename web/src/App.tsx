import { useCallback, useState } from 'react'
import { createSession, logOut } from './api'
import { Terminal } from './Terminal'
import { useMe } from './useMe'
import './App.css'

function App() {
  const { state, reload } = useMe()
  const [code, setCode] = useState<string | null>(null)
  const [problem, setProblem] = useState<string | null>(null)

  const endSession = useCallback(() => setCode(null), [])

  const startSession = async () => {
    setProblem(null)
    try {
      setCode(await createSession())
    } catch (err) {
      setProblem(String(err))
    }
  }

  if (state.status === 'loading') {
    return <main className="app"><p className="muted">Checking your login…</p></main>
  }

  if (state.status === 'error') {
    return (
      <main className="app">
        <p className="error" role="alert">Could not reach the server. {state.message}</p>
      </main>
    )
  }

  if (state.status === 'logged-out') {
    return (
      <main className="app">
        <h1>Shell</h1>
        <p className="tagline">A terminal you can share.</p>
        <a className="button" href="/api/auth/google/start">Sign in with Google</a>
      </main>
    )
  }

  return (
    <div className="shell">
      <header className="bar">
        <span className="brand">Shell</span>
        {code && <code className="code">{code}</code>}
        <span className="spacer" />
        {state.me.avatarUrl && (
          <img className="avatar" src={state.me.avatarUrl} alt="" width={28} height={28} />
        )}
        <span className="name">{state.me.name}</span>
        <button
          type="button"
          className="button"
          onClick={async () => {
            await logOut()
            setCode(null)
            await reload()
          }}
        >
          Log out
        </button>
      </header>

      {problem && <p className="error" role="alert">{problem}</p>}

      {code ? (
        <Terminal code={code} onEnded={endSession} />
      ) : (
        <main className="app">
          <h1>Shell</h1>
          <p className="tagline">A terminal you can share.</p>
          <button type="button" className="button" onClick={startSession}>
            New Session
          </button>
        </main>
      )}
    </div>
  )
}

export default App
