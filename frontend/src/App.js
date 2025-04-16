"use client"
import { Routes, Route, Link, Navigate } from "react-router-dom"
import { useAuth } from "./context/AuthContext"
import UserPage from "./pages/UserPage"
import AdminPage from "./pages/AdminPage"
import LoginPage from "./pages/LoginPage"
import { useState, useEffect } from "react"
import "./App.css"

// A wrapper component to protect routes
function ProtectedRoute({ children }) {
    const { user, isLoading } = useAuth()

    if (isLoading) {
        // Display a loading indicator while checking auth
        return (
            <div className="loading-container">
                <div className="loading-spinner"></div>
                <p>Checking authentication...</p>
            </div>
        )
    }

    if (!user) {
        // If not logged in, redirect to the login page
        return <Navigate to="/login" replace />
    }

    // If logged in, render the child component
    return children
}

function App() {
    const { isLoading } = useAuth()
    const [isMobile, setIsMobile] = useState(window.innerWidth <= 768)

    // Handle window resize
    useEffect(() => {
        const handleResize = () => {
            setIsMobile(window.innerWidth <= 768)
        }

        window.addEventListener("resize", handleResize)
        return () => window.removeEventListener("resize", handleResize)
    }, [])

    // Add a class to the body for mobile devices
    useEffect(() => {
        if (isMobile) {
            document.body.classList.add("mobile-device")
        } else {
            document.body.classList.remove("mobile-device")
        }
    }, [isMobile])

    return (
        <div className={`AppContainer ${isMobile ? "mobile" : ""}`}>
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
                                <AdminPage />
                            </ProtectedRoute>
                        }
                    />

                    {/* 404 Not Found route */}
                    <Route
                        path="*"
                        element={
                            <div className="not-found-page">
                                <h2>404 - Not Found</h2>
                                <p>The page you are looking for does not exist.</p>
                                <Link to="/" className="btn btn-primary">
                                    Go to Home
                                </Link>
                            </div>
                        }
                    />
                </Routes>
            </main>
        </div>
    )
}

export default App
