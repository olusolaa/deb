"use client"

import { useState, useEffect } from "react"
import "./ThemeToggle.css"

const ThemeToggle = ({ onToggle, currentTheme }) => {
    const [isDarkMode, setIsDarkMode] = useState(currentTheme === "dark")
    const [isAnimating, setIsAnimating] = useState(false)

    // Check for saved theme preference or system preference
    useEffect(() => {
        setIsDarkMode(currentTheme === "dark")
    }, [currentTheme])

    const toggleTheme = () => {
        // Prevent multiple rapid toggles
        if (isAnimating) return

        setIsAnimating(true)
        setIsDarkMode(!isDarkMode)

        // Apply theme to document if no external handler
        if (!onToggle) {
            const newTheme = !isDarkMode ? "dark" : "light"
            document.documentElement.classList.toggle("dark-theme", newTheme === "dark")
            localStorage.setItem("theme", newTheme)
        } else {
            // Call external handler if provided
            onToggle(!isDarkMode ? "dark" : "light")
        }

        // Reset animation lock after transition completes
        setTimeout(() => {
            setIsAnimating(false)
        }, 300)
    }

    return (
        <button
            className={`theme-toggle ${isDarkMode ? "dark" : "light"}`}
            onClick={toggleTheme}
            aria-label={isDarkMode ? "Switch to light mode" : "Switch to dark mode"}
            title={isDarkMode ? "Switch to light mode" : "Switch to dark mode"}
        >
            <div className="toggle-track">
                <div className="toggle-indicator">
                    <span className="toggle-icon"></span>
                </div>
            </div>
        </button>
    )
}

export default ThemeToggle
