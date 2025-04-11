import React, { useState, useEffect } from 'react';
import '../App.css'; // Use shared styles
import './AdminPage.css'; // Add specific admin styles

// Backend API URL from environment variable
const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';
const API_URL = `${API_BASE_URL}/api/admin`;

function AdminPage() {
    const [topic, setTopic] = useState('');
    const [duration, setDuration] = useState(7); // Default duration
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState(null);
    const [successMessage, setSuccessMessage] = useState('');

    const [existingPlans, setExistingPlans] = useState([]);
    const [isLoadingPlans, setIsLoadingPlans] = useState(false);

    // Fetch existing plans on mount
    const fetchPlans = async () => {
        setIsLoadingPlans(true);
        setError(null); // Clear previous errors
        try {
            const response = await fetch(`${API_URL}/plans`);
            if (!response.ok) {
                throw new Error(`Failed to fetch plans (${response.status})`);
            }
            const data = await response.json();
            setExistingPlans(data || []); // Ensure it's an array
        } catch (err) {
            console.error("Failed to fetch plans:", err);
            setError("Could not load existing plans.");
            setExistingPlans([]); // Clear plans on error
        } finally {
            setIsLoadingPlans(false);
        }
    };

    useEffect(() => {
        fetchPlans();
    }, []); // Fetch on mount

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
            const response = await fetch(`${API_URL}/plans`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    topic: topic,
                    duration_days: parseInt(duration, 10), // Ensure integer
                }),
            });

            const responseData = await response.json(); // Try to parse JSON regardless of status

            if (!response.ok) {
                throw new Error(responseData.error || `Failed to create plan (${response.status})`);
            }


            setSuccessMessage(`Successfully created plan for "${responseData.topic}" (ID: ${responseData.id}). This is now the active plan.`);
            setTopic(''); // Clear form
            setDuration(7);
            fetchPlans(); // Refresh the list of plans
        } catch (err) {
            console.error("Failed to create plan:", err);
            setError(`Plan creation failed: ${err.message}`);
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <div className="page-content admin-page"> {/* Use wrapper + specific admin class */}
            <header className="page-header">Admin - Create Plan</header>

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
                        {existingPlans.map((plan) => (
                            <li key={plan.id}>
                                <strong>{plan.topic}</strong> ({plan.duration_days} days)
                                <br/>
                                <small>Created: {new Date(plan.created_at).toLocaleString()}</small>
                                {/* Optional: Add button to view details or delete */}
                            </li>
                        ))}
                    </ul>
                )}
            </div>
        </div>
    );
}

export default AdminPage;