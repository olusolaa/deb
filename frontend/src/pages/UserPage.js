import React, { useState, useEffect } from 'react';
import  '../App.css'; // Use shared styles
import './UserPage.css';

// Backend API URL
const API_URL = process.env.REACT_APP_API_URL || window.location.hostname === 'localhost' 
  ? 'http://localhost:8080/api' 
  : `https://${window.location.hostname}/api`; // Dynamically set API URL based on environment

function UserPage() {
    const [dailyVerse, setDailyVerse] = useState(null); // Now stores DailyVerse object
    const [isLoadingVerse, setIsLoadingVerse] = useState(true);
    const [verseError, setVerseError] = useState(null);

    const [chatQuestion, setChatQuestion] = useState('');
    const [chatResponse, setChatResponse] = useState('');
    const [isChatLoading, setIsChatLoading] = useState(false);
    const [chatError, setChatError] = useState(null);

    // Fetch today's verse from the active plan
    useEffect(() => {
        const fetchVerse = async () => {
            setIsLoadingVerse(true);
            setVerseError(null);
            setDailyVerse(null); // Clear old verse
            try {
                // Use the new endpoint
                const response = await fetch(`${API_URL}/plans/today`);

                if (!response.ok) {
                    // Handle specific errors from backend
                    const errorBody = await response.json();
                    if (response.status === 404) {
                        throw new Error(errorBody.error || "No active plan or plan finished.");
                    }
                    throw new Error(`Network error (${response.status}): ${errorBody.error || 'Unknown error'}`);
                }

                const data = await response.json(); // Expects DailyVerse object
                setDailyVerse(data);
            } catch (error) {
                console.error("Failed to fetch today's verse:", error);
                // Display user-friendly message based on error type
                if (error.message.includes("No active plan") || error.message.includes("plan finished")) {
                    setVerseError("There's no reading plan active right now, or the current one has finished. Ask the admin to create one!");
                } else {
                    setVerseError(`Couldn't load today's verse. Maybe try again later? (${error.message})`);
                }
            } finally {
                setIsLoadingVerse(false);
            }
        };

        fetchVerse();
    }, []); // Fetch on mount

    // Handle chat submit (sends DailyVerse now)
    const handleChatSubmit = async (e) => {
        e.preventDefault();
        if (!chatQuestion.trim() || !dailyVerse || isChatLoading) return;

        setIsChatLoading(true);
        setChatError(null);
        setChatResponse('');

        try {
            const response = await fetch(`${API_URL}/chat`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    verse: dailyVerse, // Send the full DailyVerse object
                    question: chatQuestion,
                }),
            });

            if (!response.ok) {
                const errorBody = await response.text();
                throw new Error(`Chatbot error: ${response.status} - ${errorBody}`);
            }
            const data = await response.json();
            setChatResponse(data.answer);
        } catch (error) {
            console.error("Failed to get chat response:", error);
            setChatError(`Oops! Chatbot trouble. (${error.message})`);
            setChatResponse('');
        } finally {
            setIsChatLoading(false);
            // setChatQuestion(''); // Optional: clear input
        }
    };

    return (
        <div className="page-content"> {/* Use the new wrapper class */}
            <header className="page-header"> {/* Use the new wrapper class */}
                Verse of the Day
            </header>

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

            {/* Chatbot Section */}
            {dailyVerse && !verseError && (
                <div className="chatbot-container">
                    <h3>Ask about this reading!</h3>
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
                            {isChatLoading ? 'Thinking...' : 'Ask'}
                        </button>
                    </form>

                    {isChatLoading && <p className="chat-loading">Thinking...</p>}
                    {chatError && <p className="error">{chatError}</p>}
                    {chatResponse && (
                        <div className="chat-response">
                            <p><strong>Answer:</strong> {chatResponse}</p>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}


export default UserPage;