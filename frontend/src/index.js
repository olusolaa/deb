import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter as Router } from 'react-router-dom';
// import './index.css'; // Optional: If you have global styles
import App from './App';
import { AuthProvider } from './context/AuthContext';
import { GoogleOAuthProvider } from '@react-oauth/google';

const GOOGLE_CLIENT_ID = process.env.REACT_APP_GOOGLE_CLIENT_ID;

if (!GOOGLE_CLIENT_ID) {
  console.error('FATAL: REACT_APP_GOOGLE_CLIENT_ID environment variable is not set.');
  // You could render an error message here instead of the app
}

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    {/* Ensure Client ID is provided before rendering Google provider */}
    {GOOGLE_CLIENT_ID ? (
      <GoogleOAuthProvider clientId={GOOGLE_CLIENT_ID}>
        <Router>
          <AuthProvider>
            <App />
          </AuthProvider>
        </Router>
      </GoogleOAuthProvider>
    ) : (
      <div>
        <h1>Configuration Error</h1>
        <p>Missing Google Client ID. Please check the environment configuration.</p>
      </div>
    )}
  </React.StrictMode>
);
