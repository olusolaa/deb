"use client"

import { useState } from "react"
import "./ShareVerse.css"

const ShareVerse = ({ verse }) => {
    const [isOpen, setIsOpen] = useState(false)
    const [shareStatus, setShareStatus] = useState("")

    if (!verse) return null

    const toggleShareMenu = () => {
        setIsOpen(!isOpen)
        setShareStatus("")
    }

    const shareText = `"${verse.text.substring(0, 150)}${verse.text.length > 150 ? "..." : ""}" - ${verse.reference}`

    const copyToClipboard = () => {
        navigator.clipboard
            .writeText(shareText)
            .then(() => {
                setShareStatus("Copied to clipboard!")
                setTimeout(() => setShareStatus(""), 2000)
            })
            .catch((err) => {
                setShareStatus("Failed to copy")
                console.error("Could not copy text: ", err)
            })
    }

    const shareViaTwitter = () => {
        const twitterUrl = `https://twitter.com/intent/tweet?text=${encodeURIComponent(shareText)}`
        window.open(twitterUrl, "_blank")
        setIsOpen(false)
    }

    const shareViaEmail = () => {
        const emailSubject = `Bible Verse: ${verse.reference}`
        const emailBody = `I wanted to share this verse with you:\n\n${shareText}`
        const mailtoUrl = `mailto:?subject=${encodeURIComponent(emailSubject)}&body=${encodeURIComponent(emailBody)}`
        window.location.href = mailtoUrl
        setIsOpen(false)
    }

    const shareViaWhatsApp = () => {
        const whatsappUrl = `https://wa.me/?text=${encodeURIComponent(shareText)}`
        window.open(whatsappUrl, "_blank")
        setIsOpen(false)
    }

    return (
        <div className="share-verse-container">
            <button
                className="share-button"
                onClick={toggleShareMenu}
                aria-label="Share verse"
                aria-expanded={isOpen}
                title="Share verse"
            >
                <span className="share-icon">🔗</span>
            </button>

            {isOpen && (
                <div className="share-menu">
                    <div className="share-menu-header">
                        <h4>Share this verse</h4>
                        <button className="close-share-menu" onClick={toggleShareMenu} aria-label="Close share menu">
                            ✕
                        </button>
                    </div>

                    <div className="share-options">
                        <button className="share-option" onClick={copyToClipboard} aria-label="Copy to clipboard">
                            <span className="share-option-icon">📋</span>
                            <span className="share-option-text">Copy</span>
                        </button>

                        <button className="share-option" onClick={shareViaTwitter} aria-label="Share on Twitter">
                            <span className="share-option-icon">🐦</span>
                            <span className="share-option-text">Twitter</span>
                        </button>

                        <button className="share-option" onClick={shareViaEmail} aria-label="Share via Email">
                            <span className="share-option-icon">✉️</span>
                            <span className="share-option-text">Email</span>
                        </button>

                        <button className="share-option" onClick={shareViaWhatsApp} aria-label="Share via WhatsApp">
                            <span className="share-option-icon">📱</span>
                            <span className="share-option-text">WhatsApp</span>
                        </button>
                    </div>

                    {shareStatus && <div className="share-status">{shareStatus}</div>}
                </div>
            )}
        </div>
    )
}

export default ShareVerse
