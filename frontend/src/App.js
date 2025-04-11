import React, { useState, useEffect } from 'react';
import './App.css';

// Backend API URL (adjust if your backend runs elsewhere)
const API_URL = 'http://localhost:8080/api';

function App() {
    const [verseData, setVerseData] = useState(null);
    const [isLoadingVerse, setIsLoadingVerse] = useState(true);
    const [verseError, setVerseError] = useState(null);

    const [chatQuestion, setChatQuestion] = useState('');
    const [chatResponse, setChatResponse] = useState('');
    const [isChatLoading, setIsChatLoading] = useState(false);
    const [chatError, setChatError] = useState(null);

    // Fetch daily verse on component mount
    useEffect(() => {
        const fetchVerse = async () => {
            setIsLoadingVerse(true);
            setVerseError(null);
            try {
                const response = await fetch(`${API_URL}/verse/today`);
                if (!response.ok) {
                    throw new Error(`Network response was not ok (${response.status})`);
                }
                const data = await response.json();
                setVerseData(data);
            } catch (error) {
                console.error("Failed to fetch verse:", error);
                setVerseError(`Couldn't load the verse today. Maybe try again later? (${error.message})`);
            } finally {
                setIsLoadingVerse(false);
            }
        };

        fetchVerse();
    }, []); // Empty dependency array means run once on mount

    // Handle sending chat question
    const handleChatSubmit = async (e) => {
        e.preventDefault(); // Prevent default form submission
        if (!chatQuestion.trim() || !verseData || isChatLoading) {
            return; // Don't submit if empty, no verse, or already loading
        }

        setIsChatLoading(true);
        setChatError(null);
        setChatResponse(''); // Clear previous response

        try {
            const response = await fetch(`${API_URL}/chat`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    verse: verseData, // Send the full verse object
                    question: chatQuestion,
                }),
            });

            if (!response.ok) {
                const errorBody = await response.text(); // Get error details from backend
                throw new Error(`Chatbot error: ${response.status} - ${errorBody}`);
            }

            const data = await response.json();
            setChatResponse(data.answer);

        } catch (error) {
            console.error("Failed to get chat response:", error);
            setChatError(`Oops! The chatbot couldn't answer right now. (${error.message})`);
            setChatResponse(''); // Clear any potential partial response
        } finally {
            setIsChatLoading(false);
            // Optional: Clear input after sending
            // setChatQuestion('');
        }
    };

    return (
        <div className="App">
            <header className="App-header">
                Verse of the Day
            </header>

            <div className="verse-container">
                {isLoadingVerse && <p className="loading">Finding today's verse...</p>}
                {verseError && <p className="error">{verseError}</p>}
                {verseData && !isLoadingVerse && !verseError && (
                    <>
                        <h2 className="verse-reference">{verseData.reference}</h2>
                        <p className="verse-text">"{verseData.text}"</p>
                    </>
                )}
            </div>

            {verseData && !verseError && ( // Only show chatbot if verse loaded successfully
                <div className="chatbot-container">
                    <h3>Ask about this verse!</h3>
                    <form onSubmit={handleChatSubmit} className="chat-input-area">
                        <input
                            type="text"
                            className="chat-input"
                            value={chatQuestion}
                            onChange={(e) => setChatQuestion(e.target.value)}
                            placeholder="Type your question here..."
                            aria-label="Ask a question about the verse"
                            disabled={isChatLoading}
                        />
                        <button
                            type="submit"
                            className="send-button"
                            disabled={isChatLoading || !chatQuestion.trim()}
                            aria-label="Send question"
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

export default App;