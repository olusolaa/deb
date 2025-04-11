import React, { useState, useEffect, useRef, useCallback } from 'react'; // Import useRef, useCallback
import '../App.css';
import './UserPage.css'; // Add page-specific styles

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';
const API_URL = `${API_BASE_URL}/api`;
// Define message type constants
const MSG_TYPE = {
    USER: 'user',
    ASSISTANT: 'assistant',
    ERROR: 'error',
    INFO: 'info', // For messages like "Chat reset"
};

function UserPage() {
    const [dailyVerse, setDailyVerse] = useState(null);
    const [isLoadingVerse, setIsLoadingVerse] = useState(true);
    const [verseError, setVerseError] = useState(null);

    const [chatQuestion, setChatQuestion] = useState('');
    // *** Store the entire chat history ***
    const [chatHistory, setChatHistory] = useState([]);
    const [isChatLoading, setIsChatLoading] = useState(false);
    // We'll display chat errors directly in the history now
    // const [chatError, setChatError] = useState(null);

    // Ref for scrolling chat to bottom
    const chatEndRef = useRef(null);

    // Function to scroll chat window
    const scrollToBottom = () => {
        chatEndRef.current?.scrollIntoView({ behavior: "smooth" });
    };

    // Scroll whenever chatHistory changes
    useEffect(() => {
        scrollToBottom();
    }, [chatHistory]);


    // Fetch today's verse
    const fetchVerse = useCallback(async () => { // Wrap in useCallback
        setIsLoadingVerse(true);
        setVerseError(null);
        setDailyVerse(null);
        try {
            const response = await fetch(`${API_URL}/plans/today`);
            if (!response.ok) {
                const errorBody = await response.json();
                if (response.status === 404) {
                    throw new Error(errorBody.error || "No active plan or plan finished.");
                }
                throw new Error(`Network error (${response.status}): ${errorBody.error || 'Unknown error'}`);
            }
            const data = await response.json();
            setDailyVerse(data);
        } catch (error) {
            console.error("Failed to fetch today's verse:", error);
            if (error.message.includes("No active plan") || error.message.includes("plan finished")) {
                setVerseError("There's no reading plan active right now, or the current one has finished. Ask the admin to create one!");
            } else {
                setVerseError(`Couldn't load today's verse. Maybe try again later? (${error.message})`);
            }
        } finally {
            setIsLoadingVerse(false);
        }
    }, []); // Empty dependency array - fetchVerse function itself doesn't change

    useEffect(() => {
        fetchVerse();
    }, [fetchVerse]); // Run fetchVerse on mount


    // --- Chat Handling ---

    // Add a message to the history state
    const addMessageToHistory = (role, content) => {
        setChatHistory(prev => [...prev, { role, content }]);
    };

    // Handle chat submit
    const handleChatSubmit = async (e) => {
        e.preventDefault();
        const question = chatQuestion.trim();
        if (!question || !dailyVerse || isChatLoading) return;

        // Add user's message to history immediately
        addMessageToHistory(MSG_TYPE.USER, question);
        setChatQuestion(''); // Clear input
        setIsChatLoading(true);

        try {
            const response = await fetch(`${API_URL}/chat`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    verse: dailyVerse, // Send current verse context (backend decides if needed)
                    question: question,
                }),
            });

            const data = await response.json(); // Try parsing JSON first

            if (!response.ok) {
                // Use error from JSON body if available
                throw new Error(data.error || `Chatbot error (${response.status})`);
            }

            // Add assistant's response to history
            addMessageToHistory(MSG_TYPE.ASSISTANT, data.answer);

        } catch (error) {
            console.error("Failed to get chat response:", error);
            // Add error message to history
            addMessageToHistory(MSG_TYPE.ERROR, `Oops! Chatbot trouble: ${error.message}`);
        } finally {
            setIsChatLoading(false);
        }
    };

    // Handle chat reset
    const handleResetChat = async () => {
        // Optional: Add a confirmation dialog
        // if (!window.confirm("Are you sure you want to start a new chat?")) return;

        setIsChatLoading(true); // Indicate activity

        try {
            const response = await fetch(`${API_URL}/chat/reset`, { method: 'POST' });
            const data = await response.json(); // Try parsing JSON

            if (!response.ok) {
                throw new Error(data.error || `Failed to reset chat (${response.status})`);
            }

            // Clear local chat history and add info message
            setChatHistory([]);
            addMessageToHistory(MSG_TYPE.INFO, "Chat history cleared. Ask a new question!");
            console.log("Chat reset successfully:", data.message);

        } catch (error) {
            console.error("Failed to reset chat:", error);
            // Add error message locally if reset fails
            addMessageToHistory(MSG_TYPE.ERROR, `Could not reset chat: ${error.message}`);
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
                        <p className="verse-text">"{dailyVerse.text}"</p>
                        {dailyVerse.explanation && (
                            <p className="verse-explanation">
                                <strong>Quick thought:</strong> {dailyVerse.explanation}
                            </p>
                        )}
                    </>
                )}
            </div>

            {/* Chatbot Section - Show only if verse loaded */}
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


                    {/* Chat History Display */}
                    <div className="chat-history">
                        {chatHistory.map((msg, index) => (
                            <div key={index} className={`chat-message ${msg.role}`}>
                                <p>{msg.content}</p>
                            </div>
                        ))}
                        {/* Add empty div to scroll to */}
                        <div ref={chatEndRef} />
                    </div>


                    {/* Loading Indicator inside chat */}
                    {isChatLoading && <p className="chat-loading">Thinking...</p>}


                    {/* Chat Input Form */}
                    <form onSubmit={handleChatSubmit} className="chat-input-area">
                        <input
                            type="text"
                            className="chat-input"
                            value={chatQuestion}
                            onChange={(e) => setChatQuestion(e.target.value)}
                            placeholder="Type your question..."
                            disabled={isChatLoading} // Disable input while loading
                        />
                        <button
                            type="submit"
                            className="send-button"
                            disabled={isChatLoading || !chatQuestion.trim()} // Disable if loading or empty
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