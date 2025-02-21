import React, { useState } from 'react';
import './Modal.css';
import './AddClassModal.css';

const EMOJI_OPTIONS = ['🔬', '⚡️', '🧑‍💻', '🧪', '📚', '🔋', '🧮', '🔭', '⚗️', '🦠', '🧬', '+'];

const AddClassModal = ({ onClose, onAdd }) => {
    const [className, setClassName] = useState('');
    const [selectedEmoji, setSelectedEmoji] = useState('🔬');
    const [showCustomEmoji, setShowCustomEmoji] = useState(false);
    const [customEmoji, setCustomEmoji] = useState('');

    const handleSubmit = () => {
        if (className.trim()) {
            onAdd({
                name: className.trim(),
                emoji: customEmoji || selectedEmoji,
                glowColor: 'rgba(129, 140, 248, 0.5)'
            });
            onClose();
        }
    };

    const handleEmojiSelect = (emoji) => {
        if (emoji === '+') {
            setShowCustomEmoji(!showCustomEmoji);
        } else {
            setSelectedEmoji(emoji);
            setCustomEmoji('');
            setShowCustomEmoji(false);
        }
    };

    const handleKeyPress = (e) => {
        if (e.key === 'Enter' && className.trim()) {
            handleSubmit();
        }
    };

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal-content" onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                    <h2>Add New Class</h2>
                    <button className="modal-close" onClick={onClose}>
                        <i className="ri-close-line"></i>
                    </button>
                </div>
                <div className="modal-body">
                    <div className="add-class-form">
                        <div className="emoji-selector">
                            <label>Choose an Emoji</label>
                            <div className="emoji-grid">
                                {EMOJI_OPTIONS.map((emoji) => (
                                    <button
                                        key={emoji}
                                        className={`emoji-option ${emoji === '+' && showCustomEmoji ? 'glowing' : ''} ${emoji === selectedEmoji && !customEmoji ? 'selected' : ''}`}
                                        onClick={() => handleEmojiSelect(emoji)}
                                    >
                                        {emoji}
                                    </button>
                                ))}
                            </div>
                            {showCustomEmoji && (
                                <div className="custom-emoji-input">
                                    <input
                                        type="text"
                                        value={customEmoji}
                                        onChange={(e) => setCustomEmoji(e.target.value)}
                                        placeholder="Paste your emoji here..."
                                        maxLength={2}
                                        autoFocus
                                    />
                                </div>
                            )}
                        </div>
                        <div className="class-name-input">
                            <label>Class Name</label>
                            <div className="input-with-emoji">
                                <span className="input-emoji">{customEmoji || selectedEmoji}</span>
                                <input
                                    type="text"
                                    value={className}
                                    onChange={(e) => setClassName(e.target.value)}
                                    placeholder="Enter class name..."
                                />
                            </div>
                        </div>
                        <button 
                            className={`create-button ${!className.trim() ? 'disabled' : ''}`}
                            onClick={handleSubmit}
                            disabled={!className.trim()}
                        >
                            Create Class
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default AddClassModal; 