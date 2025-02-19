import React from 'react';
import './ProfileMenu.css';
import mindsetIcon from '../assets/images/mindset.png';
import readingBookIcon from '../assets/images/reading-book.png';
import dropperIcon from '../assets/images/dropper.png';

const ProfileMenu = ({ profile, onClose }) => {
    return (
        <div className="profile-menu-overlay" onClick={onClose}>
            <div className="profile-menu" onClick={e => e.stopPropagation()}>
                <div className="profile-menu-header">
                    <span className="profile-menu-emoji">{profile.emoji}</span>
                    <span className="profile-menu-name">{profile.name}</span>
                </div>
                <div className="profile-menu-buttons">
                    <div className="menu-button-container">
                        <button className="menu-button learn">
                            <img src={mindsetIcon} alt="Learn" className="button-icon" />
                        </button>
                        <span className="menu-button-label">LEARN</span>
                    </div>
                    <div className="menu-button-container">
                        <button className="menu-button review">
                            <img src={readingBookIcon} alt="Review" className="button-icon" />
                        </button>
                        <span className="menu-button-label">REVIEW</span>
                    </div>
                    <div className="menu-button-container">
                        <button className="menu-button edit">
                            <img src={dropperIcon} alt="Edit Droplets" className="button-icon" />
                        </button>
                        <span className="menu-button-label">EDIT DROPLETS</span>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default ProfileMenu; 