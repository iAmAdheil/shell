import { logOut } from './api'
import { useMe } from './useMe'
import './App.css'

function App() {
  const { state, reload } = useMe()

  return (
    <main className="app">
      <h1>Shell</h1>
      <p className="tagline">A terminal you can share.</p>

      {state.status === 'loading' && <p className="muted">Checking your login…</p>}

      {state.status === 'error' && (
        <p className="error" role="alert">
          Could not reach the server. {state.message}
        </p>
      )}

      {state.status === 'logged-out' && (
        <a className="button" href="/api/auth/google/start">
          Sign in with Google
        </a>
      )}

      {state.status === 'logged-in' && (
        <div className="who">
          {state.me.avatarUrl && (
            <img className="avatar" src={state.me.avatarUrl} alt="" width={40} height={40} />
          )}
          <span className="name">{state.me.name}</span>
          <button
            type="button"
            className="button"
            onClick={async () => {
              await logOut()
              await reload()
            }}
          >
            Log out
          </button>
        </div>
      )}
    </main>
  )
}

export default App
