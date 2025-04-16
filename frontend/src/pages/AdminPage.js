"use client"

import { useState, useEffect, useCallback } from "react"
import apiClient from "../api/axiosConfig"
import { useAuth } from "../context/AuthContext"
import { Link } from "react-router-dom"
import "../App.css"
import "./AdminPage.css"
import ThemeToggle from "../components/ThemeToggle"

function AdminPage() {
    const { user } = useAuth()
    const [topic, setTopic] = useState("")
    const [duration, setDuration] = useState(7)
    const [isLoading, setIsLoading] = useState(false)
    const [error, setError] = useState(null)
    const [successMessage, setSuccessMessage] = useState("")
    const [existingPlans, setExistingPlans] = useState([])
    const [isLoadingPlans, setIsLoadingPlans] = useState(false)
    const [activePlanId, setActivePlanId] = useState(null)
    const [confirmDelete, setConfirmDelete] = useState(null)
    const [windowHeight, setWindowHeight] = useState(window.innerHeight)

    // Handle window resize events for responsiveness
    useEffect(() => {
        const handleResize = () => {
            setWindowHeight(window.innerHeight)
        }

        window.addEventListener("resize", handleResize)
        return () => window.removeEventListener("resize", handleResize)
    }, [])

    // Fetch existing plans using apiClient
    const fetchPlans = useCallback(async () => {
        setIsLoadingPlans(true)
        setError(null)
        try {
            const response = await apiClient.get("/api/plans")
            const plans = response.data || []
            setExistingPlans(plans)

            // Find active plan
            const active = plans.find(plan => plan.is_active)
            if (active) {
                setActivePlanId(active.id)
            }
        } catch (err) {
            console.error("Failed to fetch plans:", err)
            const errorMsg = err.response?.data?.error || err.message || "Unknown error"
            setError(`Could not load existing plans: ${errorMsg}`)
            setExistingPlans([])
        } finally {
            setIsLoadingPlans(false)
        }
    }, [])

    useEffect(() => {
        fetchPlans()
    }, [fetchPlans])

    // Handle plan creation using apiClient
    const handleCreatePlan = async (e) => {
        e.preventDefault()
        if (!topic || duration <= 0) {
            setError("Please enter a valid topic and duration (days).")
            return
        }

        setIsLoading(true)
        setError(null)
        setSuccessMessage("")

        try {
            const response = await apiClient.post("/api/plans", {
                topic: topic,
                duration_days: Number.parseInt(duration, 10),
            })

            const responseData = response.data
            setSuccessMessage(
                `Successfully created plan for "${responseData.topic}" (ID: ${responseData.id}). This is now the active plan.`
            )
            setTopic("")
            setDuration(7)
            fetchPlans() // Refresh the list of plans

            // Auto-dismiss success message after 5 seconds
            setTimeout(() => {
                setSuccessMessage("")
            }, 5000)
        } catch (err) {
            console.error("Failed to create plan:", err)
            const errorMsg = err.response?.data?.error || err.message || "Unknown error"
            setError(`Plan creation failed: ${errorMsg}`)
        } finally {
            setIsLoading(false)
        }
    }

    // Handle plan activation
    const handleActivatePlan = async (planId) => {
        setIsLoading(true)
        setError(null)
        setSuccessMessage("")

        try {
            await apiClient.post(`/api/plans/${planId}/activate`)
            setSuccessMessage("Plan activated successfully!")
            setActivePlanId(planId)
            fetchPlans() // Refresh the list of plans

            // Auto-dismiss success message after 5 seconds
            setTimeout(() => {
                setSuccessMessage("")
            }, 5000)
        } catch (err) {
            console.error("Failed to activate plan:", err)
            const errorMsg = err.response?.data?.error || err.message || "Unknown error"
            setError(`Plan activation failed: ${errorMsg}`)
        } finally {
            setIsLoading(false)
        }
    }

    // Handle plan deletion
    const handleDeletePlan = async (planId) => {
        if (confirmDelete !== planId) {
            setConfirmDelete(planId)
            return
        }

        setIsLoading(true)
        setError(null)
        setSuccessMessage("")

        try {
            await apiClient.delete(`/api/plans/${planId}`)
            setSuccessMessage("Plan deleted successfully!")
            fetchPlans() // Refresh the list of plans
            setConfirmDelete(null)

            // Auto-dismiss success message after 5 seconds
            setTimeout(() => {
                setSuccessMessage("")
            }, 5000)
        } catch (err) {
            console.error("Failed to delete plan:", err)
            const errorMsg = err.response?.data?.error || err.message || "Unknown error"
            setError(`Plan deletion failed: ${errorMsg}`)
        } finally {
            setIsLoading(false)
        }
    }

    // Cancel delete confirmation
    const cancelDelete = () => {
        setConfirmDelete(null)
    }

    return (
        <>
            <ThemeToggle />

            {/* Left Navigation with Icons - Now outside page-content */}
            <nav className="left-nav" aria-label="Main Navigation">
                <Link to="/" className="nav-icon-container" title="Today's Reading">
          <span className="nav-icon verse-icon" aria-hidden="true">
            📘
          </span>
                    <span className="nav-label">Today's Reading</span>
                </Link>
                <div className="nav-icon-container active" title="Admin">
          <span className="nav-icon admin-icon" aria-hidden="true">
            ⚙️
          </span>
                    <span className="nav-label">Admin</span>
                    <span className="sr-only">Admin (Current Page)</span>
                </div>
                <div className="nav-icon-container" title="Search">
          <span className="nav-icon message-icon" aria-hidden="true">
            🔍
          </span>
                    <span className="nav-label">Search</span>
                </div>
                <div className="nav-icon-container" title="Notes">
          <span className="nav-icon tools-icon" aria-hidden="true">
            📝
          </span>
                    <span className="nav-label">Notes</span>
                </div>
                <div className="nav-icon-container" title="Bookmarks">
          <span className="nav-icon save-icon" aria-hidden="true">
            🔖
          </span>
                    <span className="nav-label">Bookmarks</span>
                </div>
                <div className="nav-icon-container" title="Share">
          <span className="nav-icon share-icon" aria-hidden="true">
            🔄
          </span>
                    <span className="nav-label">Share</span>
                </div>
                <div className="nav-icon-container" title="Favorites">
          <span className="nav-icon like-icon" aria-hidden="true">
            ❤️
          </span>
                    <span className="nav-label">Favorites</span>
                </div>
            </nav>

            <div className="page-content" style={{ minHeight: `${windowHeight - 60}px` }}>
                <div className="main-content-area">
                    <div className="admin-container">
                        <div className="admin-header">
                            <h1 className="admin-title">Reading Plan Management</h1>
                            <p className="admin-subtitle">Create and manage Bible study reading plans</p>
                        </div>

                        {error && (
                            <div className="admin-alert error" role="alert">
                                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                    <circle cx="12" cy="12" r="10"></circle>
                                    <line x1="12" y1="8" x2="12" y2="12"></line>
                                    <line x1="12" y1="16" x2="12.01" y2="16"></line>
                                </svg>
                                <span>{error}</span>
                                <button className="close-alert" onClick={() => setError(null)} aria-label="Dismiss error">
                                    ×
                                </button>
                            </div>
                        )}

                        {successMessage && (
                            <div className="admin-alert success" role="alert">
                                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                    <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
                                    <polyline points="22 4 12 14.01 9 11.01"></polyline>
                                </svg>
                                <span>{successMessage}</span>
                                <button className="close-alert" onClick={() => setSuccessMessage("")} aria-label="Dismiss message">
                                    ×
                                </button>
                            </div>
                        )}

                        <div className="admin-content">
                            <div className="admin-card create-plan-card">
                                <div className="admin-card-header">
                                    <h2 className="admin-card-title">Create New Reading Plan</h2>
                                    <p className="admin-card-description">
                                        Create a new reading plan based on a topic or theme. The system will generate daily verses for the specified duration.
                                    </p>
                                </div>

                                <form onSubmit={handleCreatePlan} className="admin-form">
                                    <div className="form-group">
                                        <label htmlFor="topic">Topic / Theme / Name:</label>
                                        <input
                                            type="text"
                                            id="topic"
                                            value={topic}
                                            onChange={(e) => setTopic(e.target.value)}
                                            placeholder="e.g., The Story of David, Love, Parables of Jesus"
                                            required
                                            disabled={isLoading}
                                            className="admin-input"
                                        />
                                        <p className="input-help">Choose a specific theme, character, or concept from the Bible</p>
                                    </div>

                                    <div className="form-group">
                                        <label htmlFor="duration">Duration (days):</label>
                                        <div className="duration-input-group">
                                            <button
                                                type="button"
                                                className="duration-adjust"
                                                onClick={() => setDuration(Math.max(1, duration - 1))}
                                                disabled={duration <= 1 || isLoading}
                                                aria-label="Decrease duration"
                                            >
                                                −
                                            </button>
                                            <input
                                                type="number"
                                                id="duration"
                                                value={duration}
                                                onChange={(e) => setDuration(Math.max(1, Math.min(90, parseInt(e.target.value) || 7)))}
                                                min="1"
                                                max="90"
                                                required
                                                disabled={isLoading}
                                                className="admin-input duration-input"
                                            />
                                            <button
                                                type="button"
                                                className="duration-adjust"
                                                onClick={() => setDuration(Math.min(90, duration + 1))}
                                                disabled={duration >= 90 || isLoading}
                                                aria-label="Increase duration"
                                            >
                                                +
                                            </button>
                                        </div>
                                        <p className="input-help">Recommended: 7-30 days (maximum 90 days)</p>
                                    </div>

                                    <button type="submit" className="admin-button create-button" disabled={isLoading}>
                                        {isLoading ? (
                                            <>
                                                <span className="button-spinner"></span>
                                                <span>Generating Plan...</span>
                                            </>
                                        ) : (
                                            <>
                                                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                                    <line x1="12" y1="5" x2="12" y2="19"></line>
                                                    <line x1="5" y1="12" x2="19" y2="12"></line>
                                                </svg>
                                                <span>Create Reading Plan</span>
                                            </>
                                        )}
                                    </button>
                                </form>
                            </div>

                            <div className="admin-card plans-list-card">
                                <div className="admin-card-header">
                                    <h2 className="admin-card-title">Existing Reading Plans</h2>
                                    <p className="admin-card-description">
                                        Manage your created reading plans. Only one plan can be active at a time.
                                    </p>
                                </div>

                                <div className="plans-list-container">
                                    {isLoadingPlans && (
                                        <div className="plans-loading">
                                            <div className="loading-spinner"></div>
                                            <p>Loading reading plans...</p>
                                        </div>
                                    )}

                                    {!isLoadingPlans && existingPlans.length === 0 && (
                                        <div className="empty-plans">
                                            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
                                                <line x1="12" y1="11" x2="12" y2="17"></line>
                                                <line x1="9" y1="14" x2="15" y2="14"></line>
                                            </svg>
                                            <p>No reading plans created yet</p>
                                            <p className="empty-plans-subtext">Create your first reading plan above</p>
                                        </div>
                                    )}

                                    {!isLoadingPlans && existingPlans.length > 0 && (
                                        <div className="plans-list">
                                            {existingPlans
                                                .sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
                                                .map((plan) => (
                                                    <div key={plan.id} className={`plan-item ${plan.id === activePlanId ? 'active-plan' : ''}`}>
                                                        <div className="plan-info">
                                                            <div className="plan-header">
                                                                <h3 className="plan-title">{plan.topic}</h3>
                                                                {plan.id === activePlanId && (
                                                                    <span className="active-badge">Active</span>
                                                                )}
                                                            </div>
                                                            <div className="plan-details">
                                <span className="plan-duration">
                                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                    <circle cx="12" cy="12" r="10"></circle>
                                    <polyline points="12 6 12 12 16 14"></polyline>
                                  </svg>
                                    {plan.duration_days} days
                                </span>
                                                                <span className="plan-date">
                                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                    <rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect>
                                    <line x1="16" y1="2" x2="16" y2="6"></line>
                                    <line x1="8" y1="2" x2="8" y2="6"></line>
                                    <line x1="3" y1="10" x2="21" y2="10"></line>
                                  </svg>
                                  Created: {new Date(plan.created_at).toLocaleDateString()}
                                </span>
                                                            </div>
                                                        </div>
                                                        <div className="plan-actions">
                                                            {plan.id !== activePlanId && (
                                                                <button
                                                                    className="plan-action-button activate-button"
                                                                    onClick={() => handleActivatePlan(plan.id)}
                                                                    disabled={isLoading}
                                                                    aria-label={`Activate ${plan.topic} plan`}
                                                                >
                                                                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                                                        <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
                                                                        <polyline points="22 4 12 14.01 9 11.01"></polyline>
                                                                    </svg>
                                                                    <span>Activate</span>
                                                                </button>
                                                            )}

                                                            {confirmDelete === plan.id ? (
                                                                <div className="delete-confirmation">
                                                                    <button
                                                                        className="confirm-delete-button"
                                                                        onClick={() => handleDeletePlan(plan.id)}
                                                                        disabled={isLoading}
                                                                        aria-label={`Confirm delete ${plan.topic} plan`}
                                                                    >
                                                                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                                                            <polyline points="3 6 5 6 21 6"></polyline>
                                                                            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                                                                            <line x1="10" y1="11" x2="10" y2="17"></line>
                                                                            <line x1="14" y1="11" x2="14" y2="17"></line>
                                                                        </svg>
                                                                        <span>Confirm</span>
                                                                    </button>
                                                                    <button
                                                                        className="cancel-delete-button"
                                                                        onClick={cancelDelete}
                                                                        aria-label="Cancel delete"
                                                                    >
                                                                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                                                            <line x1="18" y1="6" x2="6" y2="18"></line>
                                                                            <line x1="6" y1="6" x2="18" y2="18"></line>
                                                                        </svg>
                                                                        <span>Cancel</span>
                                                                    </button>
                                                                </div>
                                                            ) : (
                                                                <button
                                                                    className="plan-action-button delete-button"
                                                                    onClick={() => handleDeletePlan(plan.id)}
                                                                    disabled={isLoading || plan.id === activePlanId}
                                                                    aria-label={`Delete ${plan.topic} plan`}
                                                                    title={plan.id === activePlanId ? "Cannot delete active plan" : "Delete plan"}
                                                                >
                                                                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                                                        <polyline points="3 6 5 6 21 6"></polyline>
                                                                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                                                                        <line x1="10" y1="11" x2="10" y2="17"></line>
                                                                        <line x1="14" y1="11" x2="14" y2="17"></line>
                                                                    </svg>
                                                                    <span>Delete</span>
                                                                </button>
                                                            )}
                                                        </div>
                                                    </div>
                                                ))}
                                        </div>
                                    )}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </>
    )
}

export default AdminPage
