import { Sun, Moon, Wifi, WifiOff, AlertCircle } from 'lucide-react'

export default function Header({ theme, onToggleTheme, connected, error }) {
  return (
    <header className="header">
      <div className="header-left">
        <div className="header-logo">E</div>
        <div>
          <div className="header-title">Eino Chat</div>
          <div className="header-status">
            {error ? (
              <>
                <AlertCircle size={12} color="var(--error)" />
                <span style={{ color: 'var(--error)' }}>连接错误</span>
              </>
            ) : connected ? (
              <>
                <span className="status-dot" />
                已连接
              </>
            ) : (
              <>
                <span className="status-dot disconnected" />
                未连接
              </>
            )}
          </div>
        </div>
      </div>

      <div className="header-right">
        <button className="btn-icon" onClick={onToggleTheme} title={theme === 'light' ? '切换暗色主题' : '切换亮色主题'}>
          {theme === 'light' ? <Moon size={18} /> : <Sun size={18} />}
        </button>
      </div>
    </header>
  )
}
