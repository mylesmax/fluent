import React from 'react';
import './Modal.css';

const version = "0.1.0";

const SettingsModal = ({ onClose }) => {
    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal-content" onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                    <h2>Settings</h2>
                    <button className="modal-close" onClick={onClose}>
                        <i className="ri-close-line"></i>
                    </button>
                </div>
                <div className="modal-body">
                    <div className="version-info">
                        <span className="label">Version</span>
                        <span className="value">{version}</span>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default SettingsModal; 