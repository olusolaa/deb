import React from 'react';
import { BrowserRouter as Router, Routes, Route, Link, Navigate } from 'react-router-dom';
import { useAuth } from './context/AuthContext';
import UserPage from './pages/UserPage';
import AdminPage from './pages/AdminPage';
import LoginPage from './pages/LoginPage'; // Import the LoginPage
import './App.css';

// A wrapper component to protect routes
function ProtectedRoute({ children }) {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    // Optional: Display a loading indicator while checking auth
    return <div className="loading-container">Checking authentication...</div>;
  }

  if (!user) {
    // If not logged in, redirect to the login page
    // Pass the current location so we can redirect back after login
    return <Navigate to="/login" replace />;
  }

  // If logged in, render the child component
  return children;
}

// Simple component for the Navbar
function Navigation() {
    const { user, logout, isLoading } = useAuth();

    return (
        <nav className="main-nav">
            <ul>
                {user && ( // Only show these links if logged in
                    <>
                        <li><Link to="/">Today's Verse</Link></li>
                        {/* Consider adding role-based access for Admin later */}
                        <li><Link to="/admin">Admin</Link></li>
                    </>
                )}
            </ul>
            {user && !isLoading && ( // Show logout button if logged in and not loading
                <button onClick={logout} className="logout-button">Logout ({user.name || user.email})</button>
            )}
        </nav>
    );
}


function App() {
    const { isLoading } = useAuth(); // Use isLoading from context

    // Display a global loading indicator if auth is still loading initially
    // This prevents flashes of content before auth state is known
    // if (isLoading) {
    //     return <div className="loading-container">Loading Application...</div>;
    // }
    // Removed the above global loader as ProtectedRoute handles loading for protected areas
    // and LoginPage handles its own loading state.

    return (
        // Router moved to index.js where AuthProvider is
        <div className="AppContainer">
            <Navigation /> {/* Use the Navigation component */}

            <main>
                <Routes>
                    {/* Public Login Route */}
                    <Route path="/login" element={<LoginPage />} />

                    {/* Protected Routes */}
                    <Route
                        path="/"
                        element={
                            <ProtectedRoute>
                                <UserPage />
                            </ProtectedRoute>
                        }
                    />
                    <Route
                        path="/admin"
                        element={
                            <ProtectedRoute>
                                {/* Add role checking here later if needed */}
                                <AdminPage />
                            </ProtectedRoute>
                        }
                    />

                    {/* Redirect root to login if not logged in, or user page if logged in */}
                    {/* This might conflict with ProtectedRoute, let's rely on ProtectedRoute */}
                    {/* <Route path="/" element={<Navigate to="/login" replace />} /> */}

                    {/* Optional: Add a 404 Not Found route */}
                    <Route path="*" element={
                        <div style={{ padding: '2rem' }}>
                            <h2>404 - Not Found</h2>
                            <p>The page you are looking for does not exist.</p>
                            <Link to="/">Go to Home</Link>
                        </div>
                    } />
                </Routes>
            </main>

            {/* Optional: Footer */}
            {/* <footer> <p>© 2024 Bible App</p> </footer> */}
        </div>
    );
}

export default App;
