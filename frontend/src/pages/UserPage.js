import React, { useState, useEffect, useRef, useCallback } from 'react';
import apiClient from '../api/axiosConfig'; // *** Use Axios Client ***
import { useAuth } from '../context/AuthContext'; // Import useAuth
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
    const { user } = useAuth(); // Get user info if needed
    const [dailyVerse, setDailyVerse] = useState(null);
    const [isLoadingVerse, setIsLoadingVerse] = useState(true);
    const [verseError, setVerseError] = useState(null);

    const [chatQuestion, setChatQuestion] = useState('');
    const [chatHistory, setChatHistory] = useState([]);
    const [isChatLoading, setIsChatLoading] = useState(false);

    const chatEndRef = useRef(null);

    const scrollToBottom = () => {
        chatEndRef.current?.scrollIntoView({ behavior: "smooth" });
    };

    useEffect(() => {
        scrollToBottom();
    }, [chatHistory]);

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

    return (
        <div className="page-content">
            <header className="page-header"> Verse of the Day </header>

            {/* Verse Display Area */}
            <div className="verse-container">
                {isLoadingVerse && <p className="loading">Loading today's reading...</p>}
                {verseError && <p className="error">{verseError}</p>}
                {dailyVerse && !isLoadingVerse && !verseError && (
                    <>
                        <h2 className="verse-reference">{dailyVerse.reference}</h2>
                        {dailyVerse.title && (
                            <h3 className="verse-title">{dailyVerse.title}</h3>
                        )}
                        <p className="verse-text">"{dailyVerse.text}"</p>
                        {dailyVerse.explanation && (
                            <p className="verse-explanation">
                                <strong>Quick thought:</strong> {dailyVerse.explanation}
                            </p>
                        )}
                    </>
                )}
            </div>

            {/* Chatbot Section - Show only if verse loaded and no auth error */}
            {dailyVerse && !verseError && (
                <div className="chatbot-container">
                    <div className="chat-header">
                        <h3>Ask about this reading!</h3>
                        <button
                            onClick={handleResetChat}
                            className="reset-button"
                            disabled={isChatLoading}
                            title="Start a new conversation"
                        >
                            Reset Chat
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
                            placeholder="Type your question..."
                            disabled={isChatLoading}
                        />
                        <button
                            type="submit"
                            className="send-button"
                            disabled={isChatLoading || !chatQuestion.trim()}
                        >
                            {isChatLoading ? '...' : 'Ask'}
                        </button>
                    </form>
                </div>
            )}
        </div>
    );
}

export default UserPage;
