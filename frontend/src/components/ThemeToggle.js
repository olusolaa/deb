"use client"

import { useState, useEffect } from "react"
import "./ThemeToggle.css"

const ThemeToggle = () => {
    const [isDarkMode, setIsDarkMode] = useState(false)

    // Check for saved theme preference or system preference
    useEffect(() => {
        const savedTheme = localStorage.getItem("theme")
        if (savedTheme === "dark") {
            setIsDarkMode(true)
            document.documentElement.classList.add("dark-theme")
        } else if (savedTheme === "light") {
            setIsDarkMode(false)
            document.documentElement.classList.remove("dark-theme")
        } else {
            // Check system preference
            const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches
            setIsDarkMode(prefersDark)
            if (prefersDark) {
                document.documentElement.classList.add("dark-theme")
            }
        }
    }, [])

    const toggleTheme = () => {
        setIsDarkMode(!isDarkMode)
        if (!isDarkMode) {
            document.documentElement.classList.add("dark-theme")
            localStorage.setItem("theme", "dark")
        } else {
            document.documentElement.classList.remove("dark-theme")
            localStorage.setItem("theme", "light")
        }
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
                    <span className="toggle-icon">{isDarkMode ? "🌙" : "☀️"}</span>
                </div>
            </div>
            <span className="sr-only">{isDarkMode ? "Switch to light mode" : "Switch to dark mode"}</span>
        </button>
    )
}

export default ThemeToggle
