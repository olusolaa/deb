import React, { useState, useEffect, useRef, useCallback } from 'react';
import apiClient from '../api/axiosConfig'; // *** Use Axios Client ***
import { useAuth } from '../context/AuthContext'; // Import useAuth
import { Link } from 'react-router-dom';
import '../App.css';
import './UserPage.css';

// Define message type constants
const MSG_TYPE = {
    USER: 'user',
    ASSISTANT: 'assistant',
    ERROR: 'error',
    INFO: 'info',
};

function UserPage() {
    const { user, logout } = useAuth(); // Get user info and logout function
    const [dailyVerse, setDailyVerse] = useState(null);
    const [isLoadingVerse, setIsLoadingVerse] = useState(true);
    const [verseError, setVerseError] = useState(null);

    const [chatQuestion, setChatQuestion] = useState('');
    const [chatHistory, setChatHistory] = useState([]);
    const [isChatLoading, setIsChatLoading] = useState(false);
    
    // State for pagination and animations
    const [currentPage, setCurrentPage] = useState(0);
    const [versePages, setVersePages] = useState([]);
    const [windowHeight, setWindowHeight] = useState(window.innerHeight);
    const [windowWidth, setWindowWidth] = useState(window.innerWidth);
    
    // Refs for scrolling
    const chatEndRef = useRef(null);

    const scrollToBottom = () => {
        chatEndRef.current?.scrollIntoView({ behavior: "smooth" });
    };

    useEffect(() => {
        scrollToBottom();
    }, [chatHistory]);
    
    // Handle window resize events for responsiveness
    useEffect(() => {
        const handleResize = () => {
            setWindowHeight(window.innerHeight);
            setWindowWidth(window.innerWidth);
        };
        
        window.addEventListener('resize', handleResize);
        return () => window.removeEventListener('resize', handleResize);
    }, []);
    
    // Split verse content into pages when it changes
    useEffect(() => {
        if (dailyVerse && dailyVerse.text) {
            // Roughly split content into pages if it's long enough
            const text = dailyVerse.text;
            const avgCharsPerPage = windowWidth < 768 ? 300 : 600; // Adjust based on screen size
            
            if (text.length <= avgCharsPerPage) {
                setVersePages([text]);
            } else {
                // Find reasonable break points (periods followed by space)
                const pages = [];
                let startIdx = 0;
                let currentPageLength = 0;
                let lastBreakIdx = 0;
                
                for (let i = 0; i < text.length; i++) {
                    currentPageLength++;
                    
                    // Consider a period followed by space as a good break point
                    if (text[i] === '.' && (i + 1 < text.length && text[i + 1] === ' ')) {
                        lastBreakIdx = i + 1;
                    }
                    
                    if (currentPageLength >= avgCharsPerPage && lastBreakIdx > startIdx) {
                        pages.push(text.substring(startIdx, lastBreakIdx));
                        startIdx = lastBreakIdx;
                        currentPageLength = i - lastBreakIdx;
                    }
                }
                
                // Add the last page
                if (startIdx < text.length) {
                    pages.push(text.substring(startIdx));
                }
                
                setVersePages(pages);
            }
            
            setCurrentPage(0);
        }
    }, [dailyVerse, windowWidth]);

    // Fetch today's verse using apiClient
    const fetchVerse = useCallback(async () => {
        setIsLoadingVerse(true);
        setVerseError(null);
        setDailyVerse(null);
        // Clear chat history when fetching a new verse?
        // setChatHistory([]);
        try {
            // *** Use apiClient.get ***
            const response = await apiClient.get('/api/plans/today');
            setDailyVerse(response.data);
        } catch (error) {
            console.error("Failed to fetch today's verse:", error);
            const errorMsg = error.response?.data?.error || error.message || "Unknown error";
            if (error.response?.status === 404 || errorMsg.includes("No active plan") || errorMsg.includes("plan finished")) {
                setVerseError("There's no reading plan active right now, or the current one has finished. An admin might need to create one!");
            } else if (error.response?.status === 401) {
                // This shouldn't happen if ProtectedRoute works, but handle defensively
                setVerseError("Authentication error. Please try logging out and back in.");
                // Potentially call logout() from useAuth here?
            } else {
                setVerseError(`Couldn't load today's verse. Maybe try again later? (${errorMsg})`);
            }
        } finally {
            setIsLoadingVerse(false);
        }
    }, []); // No dependencies needed for this version

    useEffect(() => {
        fetchVerse();
    }, [fetchVerse]);

    // Add a message to the history state
    const addMessageToHistory = (role, content) => {
        setChatHistory(prev => [...prev, { role, content }]);
    };

    // Handle chat submit using apiClient
    const handleChatSubmit = async (e) => {
        e.preventDefault();
        const question = chatQuestion.trim();
        if (!question || !dailyVerse || isChatLoading) return;

        // We no longer need to mark chat interaction since both are always visible

        addMessageToHistory(MSG_TYPE.USER, question);
        setChatQuestion('');
        setIsChatLoading(true);

        try {
            // *** Use apiClient.post ***
            const response = await apiClient.post('/api/chat', {
                verse: dailyVerse,
                question: question,
            });
            addMessageToHistory(MSG_TYPE.ASSISTANT, response.data.answer);
        } catch (error) {
            console.error("Failed to get chat response:", error);
            const errorMsg = error.response?.data?.error || error.message || "Unknown error";
            addMessageToHistory(MSG_TYPE.ERROR, `Oops! Chatbot trouble: ${errorMsg}`);
        } finally {
            setIsChatLoading(false);
        }
    };
    
    // We no longer need the toggle view function as we're displaying both side-by-side
    
    // Navigate through verse pages
    const nextPage = () => {
        if (currentPage < versePages.length - 1) {
            setCurrentPage(prev => prev + 1);
        }
    };
    
    const prevPage = () => {
        if (currentPage > 0) {
            setCurrentPage(prev => prev - 1);
        }
    };

    // Handle chat reset using apiClient
    const handleResetChat = async () => {
        setIsChatLoading(true);
        try {
            // *** Use apiClient.post ***
            const response = await apiClient.post('/api/chat/reset');
            setChatHistory([]);
            addMessageToHistory(MSG_TYPE.INFO, response.data.message || "Chat history cleared. Ask a new question!");
            console.log("Chat reset successfully");
        } catch (error) {
            console.error("Failed to reset chat:", error);
            const errorMsg = error.response?.data?.error || error.message || "Unknown error";
            addMessageToHistory(MSG_TYPE.ERROR, `Could not reset chat: ${errorMsg}`);
        } finally {
            setIsChatLoading(false);
        }
    };

    // No need to redeclare useAuth since we already have it at the top

    return (
        <>
            {/* Left Navigation with Icons - Now outside page-content */}
            <div className="left-nav">
                <div className="nav-icon-container active">
                    <span className="nav-icon verse-icon">📘</span>
                    <span className="nav-label">Today's Reading</span>
                </div>
                <Link to="/admin" className="nav-icon-container">
                    <span className="nav-icon admin-icon">⚙️</span>
                    <span className="nav-label">Admin</span>
                </Link>
                <div className="nav-icon-container">
                    <span className="nav-icon message-icon">🔍</span>
                    <span className="nav-label">Search</span>
                </div>
                <div className="nav-icon-container">
                    <span className="nav-icon tools-icon">📝</span>
                    <span className="nav-label">Notes</span>
                </div>
                <div className="nav-icon-container">
                    <span className="nav-icon save-icon">🔖</span>
                    <span className="nav-label">Bookmarks</span>
                </div>
                <div className="nav-icon-container">
                    <span className="nav-icon share-icon">🔄</span>
                    <span className="nav-label">Share</span>
                </div>
                <div className="nav-icon-container">
                    <span className="nav-icon like-icon">❤️</span>
                    <span className="nav-label">Favorites</span>
                </div>
                {user && (
                    <div className="nav-icon-container logout-icon" onClick={logout}>
                        <span className="nav-icon">🚪</span>
                        <span className="nav-label">Logout</span>
                    </div>
                )}
            </div>

            <div className="page-content" style={{ minHeight: `${windowHeight - 60}px` }}>
                <div className="main-content-area">

                {/* Main container wrapper */}
                <div className="container-wrapper">
                    {/* Verse Display Area - Now on the left */}
                    <div className="verse-container">

                    {isLoadingVerse && <p className="loading">Loading today's reading...</p>}
                    {verseError && <p className="error">{verseError}</p>}
                    {dailyVerse && !isLoadingVerse && !verseError && (
                        <>
                            <div className="verse-header">
                                <h2 className="verse-reference">{dailyVerse.reference}</h2>
                                {dailyVerse.title && (
                                    <h3 className="verse-title">{dailyVerse.title}</h3>
                                )}
                            </div>
                            
                            {/* Paginated verse text with animation */}
                            <div className="verse-pages-container">
                                <p className={`verse-text page-animate-${currentPage}`} key={currentPage}>
                                    "{versePages[currentPage]}"
                                </p>
                                
                                {/* Pagination controls - only show if multiple pages */}
                                {versePages.length > 1 && (
                                    <div className="pagination-controls">
                                        <button 
                                            onClick={(e) => { e.stopPropagation(); prevPage(); }}
                                            className={`page-button prev ${currentPage === 0 ? 'disabled' : ''}`}
                                            disabled={currentPage === 0}
                                        >
                                            ◀
                                        </button>
                                        <span className="page-indicator">{currentPage + 1}/{versePages.length}</span>
                                        <button 
                                            onClick={(e) => { e.stopPropagation(); nextPage(); }}
                                            className={`page-button next ${currentPage === versePages.length - 1 ? 'disabled' : ''}`}
                                            disabled={currentPage === versePages.length - 1}
                                        >
                                            ▶
                                        </button>
                                    </div>
                                )}
                            </div>
                            
                            {dailyVerse.explanation && (
                                <p className="verse-explanation">
                                    <strong>Quick thought:</strong> {dailyVerse.explanation}
                                </p>
                            )}
                            

                        </>
                    )}
                    </div>

                    {/* Chat Section - Now on the right */}
                    {dailyVerse && !verseError && (
                        <div className="chatbot-container">
                            <div className="chat-actions">
                                <button
                                    onClick={(e) => { handleResetChat(); }}
                                    className="reset-button"
                                    disabled={isChatLoading}
                                    title="Start a new conversation"
                                >
                                    <span className="reset-icon">🔄</span> New Conversation
                                </button>
                            </div>

                            <div className="chat-history">
                                {chatHistory.map((msg, index) => (
                                    <div key={index} className={`chat-message ${msg.role}`}>
                                        <p>{msg.content}</p>
                                    </div>
                                ))}
                                <div ref={chatEndRef} />
                            </div>

                            {isChatLoading && <p className="chat-loading">Thinking...</p>}

                            <form onSubmit={handleChatSubmit} className="chat-input-area">
                                <input
                                    type="text"
                                    className="chat-input"
                                    value={chatQuestion}
                                    onChange={(e) => setChatQuestion(e.target.value)}
                                    placeholder="Ask about this passage or seek deeper understanding..."
                                    disabled={isChatLoading}
                                />
                                <button
                                    type="submit"
                                    className="send-button"
                                    disabled={isChatLoading || !chatQuestion.trim()}
                                    title="Send message"
                                >
                                    {isChatLoading ? '...' : '➤'}
                                </button>
                            </form>
                        </div>
                    )}
                </div>
                </div>
            </div>
        </>
    );
}

export default UserPage;
