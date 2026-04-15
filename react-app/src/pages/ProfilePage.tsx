import { useState, useEffect } from 'react'
import { api, ApiError } from '../lib/api'

function SaveButton({ onClick, disabled, saving, label }: {
  onClick: () => void
  disabled: boolean
  saving: boolean
  label: string
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className="pressable btn btn-primary self-start disabled:opacity-50"
    >
      {saving ? `${label}ing…` : label}
    </button>
  )
}

export default function ProfilePage() {
  const [username, setUsername] = useState('')
  const [bggUsername, setBggUsername] = useState('')
  const [bggInput, setBggInput] = useState('')
  const [bggSaving, setBggSaving] = useState(false)
  const [bggMsg, setBggMsg] = useState<{ ok: boolean; text: string } | null>(null)

  const [currentPw, setCurrentPw] = useState('')
  const [newPw, setNewPw] = useState('')
  const [pwSaving, setPwSaving] = useState(false)
  const [pwMsg, setPwMsg] = useState<{ ok: boolean; text: string } | null>(null)

  useEffect(() => {
    api.getProfile().then(p => {
      setUsername(p.username)
      setBggUsername(p.bgg_username ?? '')
      setBggInput(p.bgg_username ?? '')
    }).catch(() => {})
  }, [])

  async function saveBGG() {
    setBggSaving(true)
    setBggMsg(null)
    try {
      const r = await api.setBGGUsername(bggInput.trim())
      setBggUsername(r.bgg_username)
      setBggMsg({ ok: true, text: 'Saved' })
    } catch (e) {
      setBggMsg({ ok: false, text: e instanceof ApiError ? e.message : 'Failed to save' })
    } finally {
      setBggSaving(false)
    }
  }

  async function changePassword() {
    setPwSaving(true)
    setPwMsg(null)
    try {
      await api.changePassword(currentPw, newPw)
      setPwMsg({ ok: true, text: 'Password changed' })
      setCurrentPw('')
      setNewPw('')
    } catch (e) {
      setPwMsg({ ok: false, text: e instanceof ApiError ? e.message : 'Failed to change password' })
    } finally {
      setPwSaving(false)
    }
  }

  return (
    <div className="flex flex-col gap-5 pt-1">
      <h1 className="font-heading text-[1.6rem] font-bold text-ink">Profile</h1>

      {/* Account */}
      <section className="card p-5 flex flex-col gap-3">
        <div className="text-[0.78rem] font-semibold text-muted uppercase tracking-wider">Account</div>
        <div>
          <div className="text-[0.72rem] font-semibold text-muted uppercase tracking-wider mb-1">Username</div>
          <div className="text-[0.95rem] font-medium text-ink">{username || '—'}</div>
        </div>
      </section>

      {/* BoardGameGeek */}
      <section className="card p-5 flex flex-col gap-3">
        <div className="text-[0.78rem] font-semibold text-muted uppercase tracking-wider">BoardGameGeek</div>
        <div>
          <label className="block text-[0.72rem] font-semibold text-muted uppercase tracking-wider mb-1" htmlFor="bgg-username">
            BGG Username
          </label>
          <input
            id="bgg-username"
            className="w-full px-3 py-[0.55rem] border border-edge rounded-lg text-[0.9rem] bg-parchment text-ink font-sans focus:outline-none focus:border-accent"
            value={bggInput}
            onChange={e => { setBggInput(e.target.value); setBggMsg(null) }}
            placeholder="your BGG username"
            autoCapitalize="none"
            autoCorrect="off"
            spellCheck={false}
          />
        </div>
        {bggMsg && (
          <div className={`text-[0.82rem] ${bggMsg.ok ? 'text-[#059669]' : 'text-[#b91c1c]'}`}>
            {bggMsg.text}
          </div>
        )}
        <SaveButton
          onClick={saveBGG}
          disabled={bggSaving || bggInput.trim() === bggUsername}
          saving={bggSaving}
          label="Save"
        />
      </section>

      {/* Change Password */}
      <section className="card p-5 flex flex-col gap-3">
        <div className="text-[0.78rem] font-semibold text-muted uppercase tracking-wider">Change Password</div>
        <div>
          <label className="block text-[0.72rem] font-semibold text-muted uppercase tracking-wider mb-1" htmlFor="current-pw">
            Current password
          </label>
          <input
            id="current-pw"
            type="password"
            className="w-full px-3 py-[0.55rem] border border-edge rounded-lg text-[0.9rem] bg-parchment text-ink font-sans focus:outline-none focus:border-accent"
            value={currentPw}
            onChange={e => { setCurrentPw(e.target.value); setPwMsg(null) }}
            autoComplete="current-password"
          />
        </div>
        <div>
          <label className="block text-[0.72rem] font-semibold text-muted uppercase tracking-wider mb-1" htmlFor="new-pw">
            New password
          </label>
          <input
            id="new-pw"
            type="password"
            className="w-full px-3 py-[0.55rem] border border-edge rounded-lg text-[0.9rem] bg-parchment text-ink font-sans focus:outline-none focus:border-accent"
            value={newPw}
            onChange={e => { setNewPw(e.target.value); setPwMsg(null) }}
            autoComplete="new-password"
          />
        </div>
        {pwMsg && (
          <div className={`text-[0.82rem] ${pwMsg.ok ? 'text-[#059669]' : 'text-[#b91c1c]'}`}>
            {pwMsg.text}
          </div>
        )}
        <SaveButton
          onClick={changePassword}
          disabled={pwSaving || !currentPw || !newPw}
          saving={pwSaving}
          label="Change Password"
        />
      </section>
    </div>
  )
}
