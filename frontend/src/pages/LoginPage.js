// frontend/src/pages/LoginPage.js
import React from 'react';
import { useAuth } from '../context/AuthContext';
import { Navigate } from 'react-router-dom';
import './LoginPage.css'; // We'll create this for styling

function LoginPage() {
  const { login, user, isLoading, error } = useAuth();

  if (isLoading) {
    return <div className="login-container loading">Loading...</div>; // Basic loading indicator
  }

  // If user is already logged in, redirect to home page
  if (user) {
    return <Navigate to="/" replace />;
  }

  const handleLoginClick = () => {
    login(); // This will redirect to the backend /auth/google/login
  };

  return (
    <div className="login-container">
      <div className="login-box">
        <h1>Bible App</h1>
        <p>Sign in to continue</p>
        {error && <p className="error-message">Error: {error}</p>}
        {/* 
          We don't use the GoogleLogin component from @react-oauth/google directly here 
          because our backend handles the entire OAuth flow starting from the redirect.
          We just need a button to initiate that redirect.
        */}
        <button onClick={handleLoginClick} className="login-button" disabled={isLoading}>
          {isLoading ? 'Processing...' : 'Sign in with Google'}
        </button>
        {/* You could add a loading spinner inside the button */} 
      </div>
    </div>
  );
}

export default LoginPage;
