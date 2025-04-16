"use client"
import { useAuth } from "../context/AuthContext"
import { Navigate } from "react-router-dom"
import { useState, useEffect } from "react"
import "./LoginPage.css"

function LoginPage() {
  const { login, user, isLoading, error } = useAuth()
  const [animationComplete, setAnimationComplete] = useState(false)

  // Set animation complete after component mounts
  useEffect(() => {
    const timer = setTimeout(() => {
      setAnimationComplete(true)
    }, 500)
    return () => clearTimeout(timer)
  }, [])

  if (isLoading) {
    return (
        <div className="login-container loading">
          <div className="login-loading">
            <div className="loading-spinner"></div>
            <div className="login-loading-text">Preparing your experience...</div>
          </div>
        </div>
    )
  }

  // If user is already logged in, redirect to home page
  if (user) {
    return <Navigate to="/" replace />
  }

  const handleLoginClick = () => {
    login() // This will redirect to the backend /auth/google/login
  }

  return (
      <div className="login-container">
        {/* Decorative Bible SVGs */}
        <div className="bible-decoration top-left">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 2C6.486 2 2 6.486 2 12s4.486 10 10 10 10-4.486 10-10S17.514 2 12 2zm0 18c-4.411 0-8-3.589-8-8s3.589-8 8-8 8 3.589 8 8-3.589 8-8 8z" />
            <path d="M13 7h-2v6h6v-2h-4z" />
          </svg>
        </div>
        <div className="bible-decoration bottom-right">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
            <path d="M6 22h12a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2zm0-2V4h12v16H6z" />
            <path d="M12 5.5c-1.654 0-3 1.346-3 3s1.346 3 3 3 3-1.346 3-3-1.346-3-3-3zm0 4c-.551 0-1-.449-1-1s.449-1 1-1 1 .449 1 1-.449 1-1 1zM12 13.5c-2.757 0-5 2.243-5 5H7c0-2.757 2.243-5 5-5s5 2.243 5 5h-2c0-2.757-2.243-5-5-5z" />
          </svg>
        </div>

        <div className={`login-box ${animationComplete ? 'animate-fade-in' : ''}`}>
          <div className="login-logo">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"></path>
              <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"></path>
            </svg>
          </div>

          <div className="login-header">
            <h1 className="login-title">Bible Study App</h1>
            <p className="login-subtitle">Sign in to continue your spiritual journey</p>
          </div>

          {error && (
              <div className="error login-error">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="10"></circle>
                  <line x1="12" y1="8" x2="12" y2="12"></line>
                  <line x1="12" y1="16" x2="12.01" y2="16"></line>
                </svg>
                {error}
              </div>
          )}

          <div className="login-form">
            <button onClick={handleLoginClick} className="login-button google" disabled={isLoading}>
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
                <path d="M12.545 10.239v3.821h5.445c-.712 2.315-2.647 3.972-5.445 3.972a6.033 6.033 0 1 1 0-12.064c1.498 0 2.866.549 3.921 1.453l2.814-2.814A9.969 9.969 0 0 0 12.545 2C7.021 2 2.543 6.477 2.543 12s4.478 10 10.002 10c8.396 0 10.249-7.85 9.426-11.748l-9.426-.013z" fill="currentColor"/>
              </svg>
              {isLoading ? "Signing in..." : "Sign in with Google"}
            </button>
          </div>

          <div className="login-divider">or continue as guest</div>

          <button className="secondary">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
              <circle cx="9" cy="7" r="4"></circle>
              <path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
              <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
            </svg>
            Continue as Guest
          </button>

          <div className="login-footer">
            <p>By signing in, you agree to our <a href="#">Terms of Service</a> and <a href="#">Privacy Policy</a></p>
          </div>
        </div>
      </div>
  )
}

export default LoginPage
