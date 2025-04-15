import React, { useState, useEffect, useCallback } from 'react';
import apiClient from '../api/axiosConfig'; // *** Use Axios Client ***
import { useAuth } from '../context/AuthContext'; // Import useAuth
import '../App.css';
import './AdminPage.css';

function AdminPage() {
    const { user } = useAuth(); // Get user info if needed (e.g., for role checks later)
    const [topic, setTopic] = useState('');
    const [duration, setDuration] = useState(7); // Default duration
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState(null);
    const [successMessage, setSuccessMessage] = useState('');

    const [existingPlans, setExistingPlans] = useState([]);
    const [isLoadingPlans, setIsLoadingPlans] = useState(false);

    // Fetch existing plans using apiClient
    const fetchPlans = useCallback(async () => {
        setIsLoadingPlans(true);
        setError(null);
        try {
            // *** Use apiClient.get ***
            // Note: The backend endpoint was /api/plans, not /api/admin/plans
            const response = await apiClient.get('/api/plans');
            setExistingPlans(response.data || []); // Ensure it's an array
        } catch (err) {
            console.error("Failed to fetch plans:", err);
            const errorMsg = err.response?.data?.error || err.message || "Unknown error";
            setError(`Could not load existing plans: ${errorMsg}`);
            setExistingPlans([]);
             if (err.response?.status === 401) {
                // This shouldn't happen if ProtectedRoute works
                setError("Authentication error loading plans. Please try logging out and back in.");
            }
        } finally {
            setIsLoadingPlans(false);
        }
    }, []); // Dependency array is empty

    useEffect(() => {
        fetchPlans();
    }, [fetchPlans]);

    // Handle plan creation using apiClient
    const handleCreatePlan = async (e) => {
        e.preventDefault();
        if (!topic || duration <= 0) {
            setError("Please enter a valid topic and duration (days).");
            return;
        }

        setIsLoading(true);
        setError(null);
        setSuccessMessage('');

        try {
            // *** Use apiClient.post ***
            // Note: The backend endpoint was /api/plans, not /api/admin/plans
            const response = await apiClient.post('/api/plans', {
                topic: topic,
                duration_days: parseInt(duration, 10),
            });

            const responseData = response.data;
            setSuccessMessage(`Successfully created plan for "${responseData.topic}" (ID: ${responseData.id}). This is now the active plan.`);
            setTopic('');
            setDuration(7);
            fetchPlans(); // Refresh the list of plans
        } catch (err) {
            console.error("Failed to create plan:", err);
            const errorMsg = err.response?.data?.error || err.message || "Unknown error";
            setError(`Plan creation failed: ${errorMsg}`);
             if (err.response?.status === 401) {
                // This shouldn't happen if ProtectedRoute works
                setError("Authentication error creating plan. Please try logging out and back in.");
            }
        } finally {
            setIsLoading(false);
        }
    };

    // Simple Admin check (can be expanded with roles later)
    // if (!user) { // Should be handled by ProtectedRoute
    //     return <p>Loading user...</p>;
    // }
    // Add role check here if backend provides roles in /api/me
    // if (user && !user.isAdmin) {
    //     return <p>Access Denied. You must be an administrator to view this page.</p>;
    // }

    return (
        <div className="page-content admin-page">
            <header className="page-header">Admin - Create Reading Plan</header>

            {error && <p className="error admin-error">{error}</p>}
            {successMessage && <p className="success-message">{successMessage}</p>}

            <form onSubmit={handleCreatePlan} className="admin-form">
                <div className="form-group">
                    <label htmlFor="topic">Topic / Story / Name:</label>
                    <input
                        type="text"
                        id="topic"
                        value={topic}
                        onChange={(e) => setTopic(e.target.value)}
                        placeholder="e.g., The Story of David, Love, Parables of Jesus"
                        required
                        disabled={isLoading}
                    />
                </div>
                <div className="form-group">
                    <label htmlFor="duration">Duration (days):</label>
                    <input
                        type="number"
                        id="duration"
                        value={duration}
                        onChange={(e) => setDuration(e.target.value)}
                        min="1"
                        max="90" // Set a reasonable max
                        required
                        disabled={isLoading}
                    />
                </div>
                <button type="submit" className="admin-button" disabled={isLoading}>
                    {isLoading ? 'Generating Plan (this can take a minute)...' : 'Create Reading Plan'}
                </button>
            </form>

            <div className="existing-plans">
                <h3>Existing Plans (Newest First)</h3>
                {isLoadingPlans && <p className="loading">Loading plans...</p>}
                {!isLoadingPlans && existingPlans.length === 0 && (
                    <p>No plans created yet.</p>
                )}
                {!isLoadingPlans && existingPlans.length > 0 && (
                    <ul>
                        {existingPlans
                            // Sort plans by created_at date, newest first
                            .sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
                            .map((plan) => (
                                <li key={plan.id}>
                                    <strong>{plan.topic}</strong> ({plan.duration_days} days)
                                    <br/>
                                    <small>Created: {new Date(plan.created_at).toLocaleString()}</small>
                                    {/* TODO: Add more plan management features - activate, delete? */}
                                </li>
                            ))}
                    </ul>
                )}
            </div>
        </div>
    );
}

export default AdminPage;
