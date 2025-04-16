"use client"

import { useState, useEffect } from "react"

// This component allows users to highlight and save important parts of verses
const VerseHighlighter = ({ verseText, verseReference }) => {
    const [selectedText, setSelectedText] = useState("")
    const [highlights, setHighlights] = useState([])
    const [showTooltip, setShowTooltip] = useState(false)
    const [tooltipPosition, setTooltipPosition] = useState({ x: 0, y: 0 })

    // Load saved highlights from localStorage on component mount
    useEffect(() => {
        const savedHighlights = localStorage.getItem(`highlights-${verseReference}`)
        if (savedHighlights) {
            setHighlights(JSON.parse(savedHighlights))
        }
    }, [verseReference])

    // Save highlights to localStorage when they change
    useEffect(() => {
        if (highlights.length > 0) {
            localStorage.setItem(`highlights-${verseReference}`, JSON.stringify(highlights))
        }
    }, [highlights, verseReference])

    // Handle text selection
    const handleTextSelection = () => {
        const selection = window.getSelection()
        const text = selection.toString().trim()

        if (text && text.length > 0) {
            setSelectedText(text)

            // Calculate position for the tooltip
            if (selection.rangeCount > 0) {
                const range = selection.getRangeAt(0)
                const rect = range.getBoundingClientRect()
                setTooltipPosition({
                    x: rect.left + rect.width / 2,
                    y: rect.top - 40,
                })
                setShowTooltip(true)
            }
        } else {
            setShowTooltip(false)
        }
    }

    // Add a highlight
    const addHighlight = (color = "yellow") => {
        if (selectedText) {
            setHighlights([...highlights, { text: selectedText, color, date: new Date().toISOString() }])
            setSelectedText("")
            setShowTooltip(false)
        }
    }

    // Remove a highlight
    const removeHighlight = (index) => {
        const newHighlights = [...highlights]
        newHighlights.splice(index, 1)
        setHighlights(newHighlights)

        // If no highlights left, remove from localStorage
        if (newHighlights.length === 0) {
            localStorage.removeItem(`highlights-${verseReference}`)
        }
    }

    // Render the verse text with highlights
    const renderHighlightedText = () => {
        let displayText = verseText

        // Simple highlighting implementation
        // Note: A more robust implementation would use ranges and offsets
        highlights.forEach((highlight) => {
            const parts = displayText.split(highlight.text)
            if (parts.length > 1) {
                displayText = parts.join(`<span class="highlight highlight-${highlight.color}">${highlight.text}</span>`)
            }
        })

        return (
            <div
                className="highlighted-text"
                onMouseUp={handleTextSelection}
                dangerouslySetInnerHTML={{ __html: displayText }}
            />
        )
    }

    return (
        <div className="verse-highlighter">
            {renderHighlightedText()}

            {/* Highlight tooltip */}
            {showTooltip && (
                <div
                    className="highlight-tooltip"
                    style={{
                        left: `${tooltipPosition.x}px`,
                        top: `${tooltipPosition.y}px`,
                    }}
                >
                    <button
                        className="highlight-btn highlight-yellow"
                        onClick={() => addHighlight("yellow")}
                        aria-label="Highlight in yellow"
                    />
                    <button
                        className="highlight-btn highlight-green"
                        onClick={() => addHighlight("green")}
                        aria-label="Highlight in green"
                    />
                    <button
                        className="highlight-btn highlight-blue"
                        onClick={() => addHighlight("blue")}
                        aria-label="Highlight in blue"
                    />
                    <button
                        className="highlight-btn highlight-cancel"
                        onClick={() => setShowTooltip(false)}
                        aria-label="Cancel highlighting"
                    >
                        ✕
                    </button>
                </div>
            )}

            {/* Highlights list */}
            {highlights.length > 0 && (
                <div className="highlights-list">
                    <h4>Your Highlights</h4>
                    <ul>
                        {highlights.map((highlight, index) => (
                            <li key={index} className={`highlight-item highlight-${highlight.color}`}>
                                <span className="highlight-text">"{highlight.text}"</span>
                                <button
                                    className="remove-highlight"
                                    onClick={() => removeHighlight(index)}
                                    aria-label="Remove highlight"
                                >
                                    ✕
                                </button>
                            </li>
                        ))}
                    </ul>
                </div>
            )}
        </div>
    )
}

export default VerseHighlighter
