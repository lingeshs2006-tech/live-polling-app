import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/useAuth'
import './Navbar.css'

export default function Navbar() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  function handleLogout() {
    logout()
    navigate('/')
  }

  return (
    <header className="nav">
      <Link to="/" className="nav-brand">
        <span className="nav-logo">◉</span> LivePoll
      </Link>
      <nav className="nav-links">
        {user ? (
          <>
            <span className="nav-user">Hi, {user.username}</span>
            <Link to="/dashboard" className="nav-link">My Polls</Link>
            <button className="nav-btn" onClick={handleLogout}>Log out</button>
          </>
        ) : (
          <>
            <Link to="/login" className="nav-link">Log in</Link>
            <Link to="/signup" className="nav-btn">Sign up</Link>
          </>
        )}
      </nav>
    </header>
  )
}