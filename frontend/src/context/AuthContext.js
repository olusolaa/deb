import React, { createContext, useState, useContext, useEffect, useCallback } from 'react';
import apiClient from '../api/axiosConfig'; // Use our configured Axios instance

const AuthContext = createContext(null);

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null); // Store user info (null if not logged in)
  const [isLoading, setIsLoading] = useState(true); // Track initial loading state
  const [error, setError] = useState(null); // Store login/auth errors

  // Function to fetch user data (usually called on app load)
  const fetchUser = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    console.log('AuthProvider: Attempting to fetch current user...');
    try {
      // The auth_token cookie is sent automatically due to `withCredentials: true`
      const response = await apiClient.get('/api/me');
      setUser(response.data); // Set user data if successful
      console.log('AuthProvider: User fetched successfully:', response.data);
    } catch (err) {
      console.log('AuthProvider: Failed to fetch user (likely not logged in):', err.response?.data?.error || err.message);
      setUser(null); // Ensure user is null if fetch fails
      if (err.response?.status !== 401) { // Don't set error for expected 401s on load
        setError(err.response?.data?.error || 'Failed to verify login status');
      }
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Check login status when the AuthProvider mounts
  useEffect(() => {
    fetchUser();
  }, [fetchUser]);

  // Login function - redirects to backend Google login URL
  const login = () => {
    setIsLoading(true);
    setError(null);
    // Construct the full backend login URL
    const loginUrl = `${apiClient.defaults.baseURL}/auth/google/login`;
    console.log('AuthProvider: Redirecting to Google Login:', loginUrl);
    window.location.href = loginUrl; // Redirect the browser
    // No need to setUser here, the redirect and callback handle it.
    // fetchUser will run again upon return to the app if login is successful.
  };

  // Logout function - calls backend logout endpoint
  const logout = async () => {
    setIsLoading(true);
    setError(null);
    console.log('AuthProvider: Logging out...');
    try {
      await apiClient.post('/auth/logout');
      setUser(null); // Clear user state immediately
      console.log('AuthProvider: Logout successful');
      // Optional: redirect to home or login page after logout
      // window.location.href = '/login';
    } catch (err) {
      console.error('AuthProvider: Logout failed:', err.response?.data?.error || err.message);
      setError(err.response?.data?.error || 'Logout failed');
      // Should we keep the user logged in on frontend if backend fails? Maybe not.
      // setUser(null); // Clear user state even if backend logout fails?
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <AuthContext.Provider value={{ user, login, logout, isLoading, error, fetchUser }}>
      {children}
    </AuthContext.Provider>
  );
};

// Custom hook to easily use the AuthContext
export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
