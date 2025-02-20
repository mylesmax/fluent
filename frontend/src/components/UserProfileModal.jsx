import React from 'react';
import './Modal.css';

const UserProfileModal = ({ onClose }) => {
    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal-content" onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                    <h2>User Profile</h2>
                    <button className="modal-close" onClick={onClose}>
                        <i className="ri-close-line"></i>
                    </button>
                </div>
                <div className="modal-body">
                    {/* later */}
                </div>
            </div>
        </div>
    );
};

export default UserProfileModal; 