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

// Page transition component
function PageTransition({ children }) {
    const [isVisible, setIsVisible] = useState(false)

    useEffect(() => {
        // Small delay to ensure the animation triggers
        const timer = setTimeout(() => {
            setIsVisible(true)
        }, 50)

        return () => clearTimeout(timer)
    }, [])

    return <div className={`page-transition ${isVisible ? "visible" : ""}`}>{children}</div>
}

function App() {
    const { isLoading } = useAuth()
    const [isMobile, setIsMobile] = useState(window.innerWidth <= 768)
    const [theme, setTheme] = useState(() => {
        // Check for saved theme preference
        const savedTheme = localStorage.getItem("theme")
        return (
            savedTheme || (window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light")
        )
    })

    // Handle window resize
    useEffect(() => {
        const handleResize = () => {
            setIsMobile(window.innerWidth <= 768)
        }

        window.addEventListener("resize", handleResize)
        return () => window.removeEventListener("resize", handleResize)
    }, [])

    // Apply theme class to body
    useEffect(() => {
        if (theme === "dark") {
            document.body.classList.add("dark-theme")
        } else {
            document.body.classList.remove("dark-theme")
        }
        localStorage.setItem("theme", theme)
    }, [theme])

    // Add a class to the body for mobile devices
    useEffect(() => {
        if (isMobile) {
            document.body.classList.add("mobile-device")
        } else {
            document.body.classList.remove("mobile-device")
        }
    }, [isMobile])

    // Theme toggle function
    const toggleTheme = () => {
        setTheme((prevTheme) => (prevTheme === "light" ? "dark" : "light"))
    }

    return (
        <div className={`AppContainer ${isMobile ? "mobile" : ""}`}>
            <main>
                <Routes>
                    {/* Public Login Route */}
                    <Route
                        path="/login"
                        element={
                            <PageTransition>
                                <LoginPage />
                            </PageTransition>
                        }
                    />

                    {/* Protected Routes */}
                    <Route
                        path="/"
                        element={
                            <ProtectedRoute>
                                <PageTransition>
                                    <UserPage toggleTheme={toggleTheme} theme={theme} />
                                </PageTransition>
                            </ProtectedRoute>
                        }
                    />
                    <Route
                        path="/admin"
                        element={
                            <ProtectedRoute>
                                <PageTransition>
                                    <AdminPage toggleTheme={toggleTheme} theme={theme} />
                                </PageTransition>
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
