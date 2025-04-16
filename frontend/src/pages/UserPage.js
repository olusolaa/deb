"use client"

import { useState, useEffect, useRef, useCallback } from "react"
import apiClient from "../api/axiosConfig"
import { useAuth } from "../context/AuthContext"
import { Link } from "react-router-dom"
import "../App.css"
import "./UserPage.css"
import VerseHighlighter from "../components/VerseHighlighter"
import ThemeToggle from "../components/ThemeToggle"
import BookmarkButton from "../components/BookmarkButton"

// Define message type constants
const MSG_TYPE = {
    USER: "user",
    ASSISTANT: "assistant",
    ERROR: "error",
    INFO: "info",
}

function UserPage() {
    const { user, logout } = useAuth()
    const [dailyVerse, setDailyVerse] = useState(null)
    const [isLoadingVerse, setIsLoadingVerse] = useState(true)
    const [verseError, setVerseError] = useState(null)
    const [chatQuestion, setChatQuestion] = useState("")
    const [chatHistory, setChatHistory] = useState([])
    const [isChatLoading, setIsChatLoading] = useState(false)
    const [isDarkMode, setIsDarkMode] = useState(() => {
        // Check user preference or system preference
        return window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches
    })

    // State for pagination and animations
    const [currentPage, setCurrentPage] = useState(0)
    const [versePages, setVersePages] = useState([])
    const [windowHeight, setWindowHeight] = useState(window.innerHeight)
    const [windowWidth, setWindowWidth] = useState(window.innerWidth)
    const [isBookmarked, setIsBookmarked] = useState(false)

    // Refs for scrolling
    const chatEndRef = useRef(null)

    const scrollToBottom = () => {
        chatEndRef.current?.scrollIntoView({ behavior: "smooth" })
    }

    useEffect(() => {
        scrollToBottom()
    }, [chatHistory])

    // Handle window resize events for responsiveness
    useEffect(() => {
        const handleResize = () => {
            setWindowHeight(window.innerHeight)
            setWindowWidth(window.innerWidth)
        }

        window.addEventListener("resize", handleResize)
        return () => window.removeEventListener("resize", handleResize)
    }, [])

    // Toggle dark mode
    const toggleTheme = () => {
        setIsDarkMode((prev) => !prev)
        // You would also apply the theme to the document here
        document.documentElement.classList.toggle("dark-theme")
    }

    // Toggle bookmark
    const toggleBookmark = () => {
        setIsBookmarked((prev) => !prev)
        // In a real app, you would save this to the user's profile
    }

    // Split verse content into pages when it changes
    useEffect(() => {
        if (dailyVerse && dailyVerse.text) {
            // Roughly split content into pages if it's long enough
            const text = dailyVerse.text
            const avgCharsPerPage = windowWidth < 768 ? 300 : 600 // Adjust based on screen size

            if (text.length <= avgCharsPerPage) {
                setVersePages([text])
            } else {
                // Find reasonable break points (periods followed by space)
                const pages = []
                let startIdx = 0
                let currentPageLength = 0
                let lastBreakIdx = 0

                for (let i = 0; i < text.length; i++) {
                    currentPageLength++

                    // Consider a period followed by space as a good break point
                    if (text[i] === "." && i + 1 < text.length && text[i + 1] === " ") {
                        lastBreakIdx = i + 1
                    }

                    if (currentPageLength >= avgCharsPerPage && lastBreakIdx > startIdx) {
                        pages.push(text.substring(startIdx, lastBreakIdx))
                        startIdx = lastBreakIdx
                        currentPageLength = i - lastBreakIdx
                    }
                }

                // Add the last page
                if (startIdx < text.length) {
                    pages.push(text.substring(startIdx))
                }

                setVersePages(pages)
            }

            setCurrentPage(0)
        }
    }, [dailyVerse, windowWidth])

    // Fetch today's verse using apiClient
    const fetchVerse = useCallback(async () => {
        setIsLoadingVerse(true)
        setVerseError(null)
        setDailyVerse(null)
        try {
            const response = await apiClient.get("/api/plans/today")
            setDailyVerse(response.data)
        } catch (error) {
            console.error("Failed to fetch today's verse:", error)
            const errorMsg = error.response?.data?.error || error.message || "Unknown error"
            if (error.response?.status === 404 || errorMsg.includes("No active plan") || errorMsg.includes("plan finished")) {
                setVerseError(
                    "There's no reading plan active right now, or the current one has finished. An admin might need to create one!",
                )
            } else if (error.response?.status === 401) {
                setVerseError("Authentication error. Please try logging out and back in.")
            } else {
                setVerseError(`Couldn't load today's verse. Maybe try again later? (${errorMsg})`)
            }
        } finally {
            setIsLoadingVerse(false)
        }
    }, [])

    useEffect(() => {
        fetchVerse()
    }, [fetchVerse])

    // Format timestamp for chat messages
    const formatTime = (date) => {
        return new Date(date).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
    }

    // Add a message to the history state
    const addMessageToHistory = (role, content) => {
        setChatHistory((prev) => [...prev, { role, content, timestamp: new Date() }])
    }

    // Handle chat submit using apiClient
    const handleChatSubmit = async (e) => {
        e.preventDefault()
        const question = chatQuestion.trim()
        if (!question || !dailyVerse || isChatLoading) return

        addMessageToHistory(MSG_TYPE.USER, question)
        setChatQuestion("")
        setIsChatLoading(true)

        try {
            const response = await apiClient.post("/api/chat", {
                verse: dailyVerse,
                question: question,
            })
            addMessageToHistory(MSG_TYPE.ASSISTANT, response.data.answer)
        } catch (error) {
            console.error("Failed to get chat response:", error)
            const errorMsg = error.response?.data?.error || error.message || "Unknown error"
            addMessageToHistory(MSG_TYPE.ERROR, `Oops! Chatbot trouble: ${errorMsg}`)
        } finally {
            setIsChatLoading(false)
        }
    }

    // Navigate through verse pages
    const nextPage = () => {
        if (currentPage < versePages.length - 1) {
            setCurrentPage((prev) => prev + 1)
        }
    }

    const prevPage = () => {
        if (currentPage > 0) {
            setCurrentPage((prev) => prev - 1)
        }
    }

    // Handle chat reset using apiClient
    const handleResetChat = async () => {
        setIsChatLoading(true)
        try {
            const response = await apiClient.post("/api/chat/reset")
            setChatHistory([])
            addMessageToHistory(MSG_TYPE.INFO, response.data.message || "Chat history cleared. Ask a new question!")
            console.log("Chat reset successfully")
        } catch (error) {
            console.error("Failed to reset chat:", error)
            const errorMsg = error.response?.data?.error || error.message || "Unknown error"
            addMessageToHistory(MSG_TYPE.ERROR, `Could not reset chat: ${errorMsg}`)
        } finally {
            setIsChatLoading(false)
        }
    }

    // Share verse functionality
    const shareVerse = () => {
        if (navigator.share && dailyVerse) {
            navigator
                .share({
                    title: `Bible Verse: ${dailyVerse.reference}`,
                    text: `"${dailyVerse.text}" - ${dailyVerse.reference}`,
                    url: window.location.href,
                })
                .then(() => console.log("Successful share"))
                .catch((error) => console.log("Error sharing", error))
        } else {
            // Fallback for browsers that don't support the Web Share API
            alert(`"${dailyVerse?.text}" - ${dailyVerse?.reference}`)
        }
    }

    return (
        <>
            <ThemeToggle />

            {/* Left Navigation with Icons - Now outside page-content */}
            <nav className="left-nav" aria-label="Main Navigation">
                <div className="nav-icon-container active" title="Today's Reading">
          <span className="nav-icon verse-icon" aria-hidden="true">
            📘
          </span>
                    <span className="nav-label">Today's Reading</span>
                    <span className="sr-only">Today's Reading (Current Page)</span>
                </div>
                <Link to="/admin" className="nav-icon-container" title="Admin">
          <span className="nav-icon admin-icon" aria-hidden="true">
            ⚙️
          </span>
                    <span className="nav-label">Admin</span>
                </Link>
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
                {user && (
                    <button className="nav-icon-container logout-icon" onClick={logout} title="Logout" aria-label="Logout">
            <span className="nav-icon" aria-hidden="true">
              🚪
            </span>
                        <span className="nav-label">Logout</span>
                    </button>
                )}
            </nav>

            <div className="page-content" style={{ minHeight: `${windowHeight - 60}px` }}>
                <div className="main-content-area">
                    {/* Main container wrapper */}
                    <div className="container-wrapper">
                        {/* Verse Display Area */}
                        <div className="verse-container">
                            {isLoadingVerse && (
                                <div className="loading">
                                    <div className="loading-spinner"></div>
                                    <p>Loading today's reading...</p>
                                </div>
                            )}
                            {verseError && <p className="error">{verseError}</p>}
                            {dailyVerse && !isLoadingVerse && !verseError && (
                                <>
                                    <div className="verse-header">
                                        <h2 className="verse-reference">{dailyVerse.reference}</h2>
                                        {dailyVerse.title && <h3 className="verse-title">{dailyVerse.title}</h3>}
                                    </div>

                                    <BookmarkButton verse={dailyVerse} />

                                    {/* Verse action buttons */}
                                    <div className="verse-actions">
                                        <button
                                            className="verse-action-button tooltip"
                                            onClick={toggleBookmark}
                                            data-tooltip={isBookmarked ? "Remove bookmark" : "Bookmark this verse"}
                                            aria-label={isBookmarked ? "Remove bookmark" : "Bookmark this verse"}
                                        >
                                            <svg
                                                xmlns="http://www.w3.org/2000/svg"
                                                viewBox="0 0 24 24"
                                                fill={isBookmarked ? "currentColor" : "none"}
                                                stroke="currentColor"
                                                strokeWidth="2"
                                                strokeLinecap="round"
                                                strokeLinejoin="round"
                                            >
                                                <path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"></path>
                                            </svg>
                                        </button>
                                        <button
                                            className="verse-action-button tooltip"
                                            onClick={shareVerse}
                                            data-tooltip="Share this verse"
                                            aria-label="Share this verse"
                                        >
                                            <svg
                                                xmlns="http://www.w3.org/2000/svg"
                                                viewBox="0 0 24 24"
                                                fill="none"
                                                stroke="currentColor"
                                                strokeWidth="2"
                                                strokeLinecap="round"
                                                strokeLinejoin="round"
                                            >
                                                <circle cx="18" cy="5" r="3"></circle>
                                                <circle cx="6" cy="12" r="3"></circle>
                                                <circle cx="18" cy="19" r="3"></circle>
                                                <line x1="8.59" y1="13.51" x2="15.42" y2="17.49"></line>
                                                <line x1="15.41" y1="6.51" x2="8.59" y2="10.49"></line>
                                            </svg>
                                        </button>
                                    </div>

                                    {/* Paginated verse text with animation */}
                                    <div className="verse-pages-container">
                                        <VerseHighlighter
                                            verseText={versePages[currentPage]}
                                            verseReference={dailyVerse.reference}
                                            key={`${dailyVerse.reference}-${currentPage}`}
                                            className={`verse-text page-animate-${currentPage}`}
                                        />

                                        {/* Pagination controls - only show if multiple pages */}
                                        {versePages.length > 1 && (
                                            <div className="pagination-controls">
                                                <button
                                                    onClick={(e) => {
                                                        e.stopPropagation()
                                                        prevPage()
                                                    }}
                                                    className={`page-button prev ${currentPage === 0 ? "disabled" : ""}`}
                                                    disabled={currentPage === 0}
                                                    aria-label="Previous page"
                                                >
                                                    <svg
                                                        xmlns="http://www.w3.org/2000/svg"
                                                        viewBox="0 0 24 24"
                                                        fill="none"
                                                        stroke="currentColor"
                                                        strokeWidth="2"
                                                        strokeLinecap="round"
                                                        strokeLinejoin="round"
                                                        width="16"
                                                        height="16"
                                                    >
                                                        <polyline points="15 18 9 12 15 6"></polyline>
                                                    </svg>
                                                </button>
                                                <span className="page-indicator" aria-live="polite">
                          {currentPage + 1} / {versePages.length}
                        </span>
                                                <button
                                                    onClick={(e) => {
                                                        e.stopPropagation()
                                                        nextPage()
                                                    }}
                                                    className={`page-button next ${currentPage === versePages.length - 1 ? "disabled" : ""}`}
                                                    disabled={currentPage === versePages.length - 1}
                                                    aria-label="Next page"
                                                >
                                                    <svg
                                                        xmlns="http://www.w3.org/2000/svg"
                                                        viewBox="0 0 24 24"
                                                        fill="none"
                                                        stroke="currentColor"
                                                        strokeWidth="2"
                                                        strokeLinecap="round"
                                                        strokeLinejoin="round"
                                                        width="16"
                                                        height="16"
                                                    >
                                                        <polyline points="9 18 15 12 9 6"></polyline>
                                                    </svg>
                                                </button>
                                            </div>
                                        )}
                                    </div>

                                    {dailyVerse.explanation && (
                                        <div className="verse-explanation">
                                            <strong>Quick thought:</strong> {dailyVerse.explanation}
                                        </div>
                                    )}
                                </>
                            )}
                        </div>

                        {/* Chat Section */}
                        {dailyVerse && !verseError && (
                            <div className="chatbot-container">
                                <div className="chat-header">
                                    <h2 className="chat-title">Bible Study Assistant</h2>
                                    <div className="chat-actions">
                                        <button
                                            onClick={handleResetChat}
                                            className="reset-button"
                                            disabled={isChatLoading}
                                            aria-label="Reset conversation"
                                        >
                                            <svg
                                                xmlns="http://www.w3.org/2000/svg"
                                                viewBox="0 0 24 24"
                                                fill="none"
                                                stroke="currentColor"
                                                strokeWidth="2"
                                                strokeLinecap="round"
                                                strokeLinejoin="round"
                                            >
                                                <path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38"></path>
                                            </svg>
                                            New Conversation
                                        </button>
                                    </div>
                                </div>

                                <div className="chat-history" aria-live="polite">
                                    {chatHistory.length === 0 && (
                                        <div className="empty-state">
                                            <svg
                                                xmlns="http://www.w3.org/2000/svg"
                                                viewBox="0 0 24 24"
                                                fill="none"
                                                stroke="currentColor"
                                                strokeWidth="2"
                                                strokeLinecap="round"
                                                strokeLinejoin="round"
                                            >
                                                <circle cx="12" cy="12" r="10"></circle>
                                                <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path>
                                                <line x1="12" y1="17" x2="12.01" y2="17"></line>
                                            </svg>
                                            <h3 className="empty-state-title">Ask a Question</h3>
                                            <p className="empty-state-description">
                                                Ask about today's reading to deepen your understanding of the scripture.
                                            </p>
                                        </div>
                                    )}
                                    {chatHistory.map((msg, index) => (
                                        <div
                                            key={index}
                                            className={`chat-message ${msg.role}`}
                                            role={msg.role === MSG_TYPE.ASSISTANT ? "status" : ""}
                                        >
                                            <p>{msg.content}</p>
                                            <div className="chat-message-time">{formatTime(msg.timestamp)}</div>
                                        </div>
                                    ))}
                                    <div ref={chatEndRef} />
                                </div>

                                {isChatLoading && (
                                    <div className="chat-loading" aria-live="polite">
                                        <span>Thinking</span>
                                        <div className="chat-loading-dots">
                                            <div className="chat-loading-dot"></div>
                                            <div className="chat-loading-dot"></div>
                                            <div className="chat-loading-dot"></div>
                                        </div>
                                    </div>
                                )}

                                <form onSubmit={handleChatSubmit} className="chat-input-area">
                                    <input
                                        type="text"
                                        className="chat-input"
                                        value={chatQuestion}
                                        onChange={(e) => setChatQuestion(e.target.value)}
                                        placeholder="Ask about this passage or seek deeper understanding..."
                                        disabled={isChatLoading}
                                        aria-label="Your question"
                                    />
                                    <button
                                        type="submit"
                                        className="send-button"
                                        disabled={isChatLoading || !chatQuestion.trim()}
                                        aria-label="Send message"
                                    >
                                        <svg
                                            xmlns="http://www.w3.org/2000/svg"
                                            viewBox="0 0 24 24"
                                            fill="none"
                                            stroke="currentColor"
                                            strokeWidth="2"
                                            strokeLinecap="round"
                                            strokeLinejoin="round"
                                        >
                                            <line x1="22" y1="2" x2="11" y2="13"></line>
                                            <polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
                                        </svg>
                                    </button>
                                </form>
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </>
    )
}

export default UserPage
