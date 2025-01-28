import React from 'react';
import './Profile.css';

const Profile = ({ name, emoji, description, isSelected, isAddNew, onClick, glowColor = 'rgba(244, 114, 182, 0.5)' }) => {
    const style = {
        '--glow-color': glowColor,
        '--glow-color-alpha': glowColor.replace(/[\d.]+\)$/g, '0.15)'),
        '--glow-color-dim': glowColor.replace(/[\d.]+\)$/g, '0.3)')
    };

    return (
        <div 
            className="profile loaded"
            data-name={isAddNew ? "Add New" : name}
            onClick={onClick}
            style={style}
        >
            <div className="card-front">
                <span className="profile-name">{name}</span>
                <span className="profile-emoji">{emoji}</span>
                {description && <p className="profile-description">{description}</p>}
            </div>
        </div>
    );
};

export default Profile; 