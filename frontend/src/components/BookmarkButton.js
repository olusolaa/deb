"use client"

import { useState, useEffect } from "react"
import "./BookmarkButton.css"

const BookmarkButton = ({ verse }) => {
    const [isBookmarked, setIsBookmarked] = useState(false)
    const [showTooltip, setShowTooltip] = useState(false)

    // Check if verse is already bookmarked
    useEffect(() => {
        if (!verse) return

        const bookmarks = JSON.parse(localStorage.getItem("verse-bookmarks") || "[]")
        const isAlreadyBookmarked = bookmarks.some((bookmark) => bookmark.reference === verse.reference)

        setIsBookmarked(isAlreadyBookmarked)
    }, [verse])

    const toggleBookmark = () => {
        if (!verse) return

        const bookmarks = JSON.parse(localStorage.getItem("verse-bookmarks") || "[]")

        if (isBookmarked) {
            // Remove bookmark
            const updatedBookmarks = bookmarks.filter((bookmark) => bookmark.reference !== verse.reference)
            localStorage.setItem("verse-bookmarks", JSON.stringify(updatedBookmarks))
            setIsBookmarked(false)
            setShowTooltip(true)
            setTimeout(() => setShowTooltip(false), 2000)
        } else {
            // Add bookmark
            const newBookmark = {
                reference: verse.reference,
                text: verse.text.substring(0, 100) + (verse.text.length > 100 ? "..." : ""),
                date: new Date().toISOString(),
            }

            const updatedBookmarks = [...bookmarks, newBookmark]
            localStorage.setItem("verse-bookmarks", JSON.stringify(updatedBookmarks))
            setIsBookmarked(true)
            setShowTooltip(true)
            setTimeout(() => setShowTooltip(false), 2000)
        }
    }

    if (!verse) return null

    return (
        <div className="bookmark-container">
            <button
                className={`bookmark-button ${isBookmarked ? "bookmarked" : ""}`}
                onClick={toggleBookmark}
                aria-label={isBookmarked ? "Remove bookmark" : "Add bookmark"}
                title={isBookmarked ? "Remove bookmark" : "Add bookmark"}
            >
                <span className="bookmark-icon">{isBookmarked ? "🔖" : "🔖"}</span>
            </button>

            {showTooltip && <div className="bookmark-tooltip">{isBookmarked ? "Verse bookmarked!" : "Bookmark removed"}</div>}
        </div>
    )
}

export default BookmarkButton
