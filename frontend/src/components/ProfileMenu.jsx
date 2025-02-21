import React, { useState } from 'react';
import './ProfileMenu.css';
import mindsetIcon from '../assets/images/mindset.png';
import readingBookIcon from '../assets/images/reading-book.png';
import dropperIcon from '../assets/images/dropper.png';

const ProfileMenu = ({ profile, onClose, onUpdateName, onDelete }) => {
    const [isEditing, setIsEditing] = useState(false);
    const [editedName, setEditedName] = useState(profile.name);
    const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

    const saveChanges = () => {
        if (editedName.trim() !== '' && editedName.trim() !== profile.name) {
            onUpdateName(profile.name, editedName.trim());
        }
        setIsEditing(false);
    };

    const handleEditClick = () => {
        if (isEditing) {
            saveChanges();
        } else {
            setIsEditing(true);
        }
    };

    const handleNameChange = (e) => {
        setEditedName(e.target.value);
    };

    const handleKeyPress = (e) => {
        if (e.key === 'Enter' && editedName.trim() !== '') {
            saveChanges();
        }
    };

    const handleOverlayClick = () => {
        if (isEditing) {
            saveChanges();
        }
        onClose();
    };

    const handleDelete = () => {
        setShowDeleteConfirm(true);
    };

    const confirmDelete = () => {
        onDelete(profile.name);
        onClose();
    };

    return (
        <div className="profile-menu-overlay" onClick={handleOverlayClick}>
            <div className="profile-menu" onClick={e => e.stopPropagation()}>
                <div className="profile-menu-header">
                    <div className="profile-info">
                        <button className="delete-button" onClick={handleDelete}>
                            <i className="ri-delete-bin-line"></i>
                        </button>
                        <span className="profile-menu-emoji">{profile.emoji}</span>
                        <div className="profile-name-container">
                            {isEditing ? (
                                <input
                                    type="text"
                                    value={editedName}
                                    onChange={handleNameChange}
                                    className="profile-name-input"
                                    autoFocus
                                />
                            ) : (
                                <span className="profile-menu-name">{editedName}</span>
                            )}
                            <button className={`edit-name-button ${isEditing ? 'saving' : ''}`} onClick={handleEditClick}>
                                <i className={`ri-${isEditing ? 'check-line' : 'pencil-line'}`}></i>
                            </button>
                        </div>
                    </div>
                    <button className="label-flasher-button">
                        <img src={dropperIcon} alt="Edit Droplets" className="button-icon" />
                        <span className="label-flasher-tooltip">EDIT DROPLETS</span>
                    </button>
                </div>
                <div className="profile-menu-buttons">
                    <div className="menu-button-container">
                        <button className="menu-button learn">
                            <img src={mindsetIcon} alt="Learn" className="button-icon" />
                        </button>
                        <span className="menu-button-label learn">LEARN</span>
                    </div>
                    <div className="menu-button-container">
                        <button className="menu-button review">
                            <img src={readingBookIcon} alt="Review" className="button-icon" />
                        </button>
                        <span className="menu-button-label review">REVIEW</span>
                    </div>
                </div>

                {showDeleteConfirm && (
                    <div className="delete-confirm-dialog">
                        <div className="delete-confirm-content">
                            <p>This action is irreversible! Proceed?</p>
                            <div className="delete-confirm-buttons">
                                <button className="confirm-button delete" onClick={confirmDelete}>
                                    Delete
                                </button>
                                <button className="confirm-button cancel" onClick={() => setShowDeleteConfirm(false)}>
                                    Cancel
                                </button>
                            </div>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
};

export default ProfileMenu; 