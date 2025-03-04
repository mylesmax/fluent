import React, { useState, useEffect } from 'react';
import ReactDOM from 'react-dom';
import './Modal.css';
import './DropletExplorerModal.css';
import { GetDatabaseExplorerData } from '../../wailsjs/go/main/App';

const DropletExplorerModal = ({ onClose, profile }) => {
    const [loading, setLoading] = useState(true);
    const [activeSection, setActiveSection] = useState('overview');
    const [selectedEntity, setSelectedEntity] = useState(null);
    const [dbData, setDbData] = useState({
        classInfo: null,
        sessions: [],
        aiHistory: {
            uploads: [],
            chats: [],
            parser: []
        }
    });
    const [expandedSession, setExpandedSession] = useState(null);
    const [error, setError] = useState(null);
    const [secondaryView, setSecondaryView] = useState(null);
    const [sidebarVisible, setSidebarVisible] = useState(true);

    useEffect(() => {
        document.querySelector('.droplet-explorer-modal')?.classList.add('dark-mode');
        
        loadDatabaseData();
        
        document.body.style.overflow = 'hidden';
        
        return () => {
            document.body.style.overflow = '';
        };
    }, [profile]);

    const loadDatabaseData = async () => {
        setLoading(true);
        setError(null);
        
        try {
            const data = await GetDatabaseExplorerData(profile.classUUID);
            
            const safeData = {
                ...data,
                aiHistory: {
                    uploads: data.aiHistory?.uploads || [],
                    chats: data.aiHistory?.chats || [],
                    parser: data.aiHistory?.parser || []
                }
            };
            
            setDbData(safeData);
        } catch (err) {
            console.error('err loading database data:', err);
            setError(`failed to load database data: ${err.message || 'unknown error'}`);
        } finally {
            setLoading(false);
        }
    };

    const handleSectionChange = (section) => {
        setActiveSection(section);
        setSelectedEntity(null);
        setExpandedSession(null);
        setSecondaryView(null);
    };

    const handleEntityClick = (entity, type) => {
        const prevType = selectedEntity?.type;
        
        setSelectedEntity({ data: entity, type });
        
        if (type === 'session') {
            setExpandedSession(entity.id);
            setSecondaryView(null);
        } else if (type === 'ai-entry') {
            const parentType = prevType || 'uploads';
            if (['uploads', 'chats', 'parser'].includes(parentType)) {
                const entries = dbData.aiHistory[parentType] || [];
                setSelectedEntity({
                    data: entity,
                    type: type,
                    parentType: parentType,
                    parentData: entries
                });
            }
        }
    };

    const switchToApiView = (sessionId, type = 'uploads') => {
        setActiveSection('ai-history');
        const relatedEntries = getRelatedEntities(sessionId, 'session');
        setSelectedEntity({ 
            data: relatedEntries[type], 
            type,
            sessionFilter: sessionId
        });
    };

    const getRelatedEntities = (entityId, entityType) => {
        if (entityType === 'session') {
            const uploads = dbData.aiHistory?.uploads?.filter(entry => entry?.session_id === entityId) || [];
            const chats = dbData.aiHistory?.chats?.filter(entry => entry?.session_id === entityId) || [];
            const parser = dbData.aiHistory?.parser?.filter(entry => entry?.session_id === entityId) || [];
            
            return { uploads, chats, parser };
        }
        
        return { uploads: [], chats: [], parser: [] };
    };

    const formatTimestamp = (timestamp) => {
        if (!timestamp) return 'In progress';
        const date = new Date(timestamp);
        return date.toLocaleString();
    };

    const formatChatHistory = (chatHistoryJson) => {
        try {
            let chatHistory;
            if (typeof chatHistoryJson === 'object') {
                chatHistory = chatHistoryJson;
            } else {
                chatHistory = JSON.parse(chatHistoryJson);
            }
            
            return Array.isArray(chatHistory) ? chatHistory.map((msg, index) => (
                <div key={index} className={`chat-message ${msg.role}`}>
                    <div className="message-header">{msg.role === 'user' ? '👤 You' : '🤖 Fluent'}</div>
                    <div className="message-content">{msg.content}</div>
                </div>
            )) : <div className="error">Invalid chat history format</div>;
        } catch (e) {
            return <div className="error">Error parsing chat history: {e.message}</div>;
        }
    };

    const toggleSidebar = () => {
        setSidebarVisible(!sidebarVisible);
    };

    const safeSubstring = (value, start, end) => {
        if (value === undefined || value === null) return 'Unknown';
        const str = String(value);
        return str.substring(start, end) + '...';
    };

    const renderOverviewSection = () => {
        if (loading) {
            return <div className="loading-message">Loading database information...</div>;
        }
        
        if (error) {
            return <div className="error-message">{error}</div>;
        }

        if (!dbData.classInfo) {
            return <div className="no-data-message">No class information available.</div>;
        }

        const sessionCount = dbData.sessions?.length || 0;
        const uploadCount = dbData.aiHistory?.uploads?.length || 0;
        const chatCount = dbData.aiHistory?.chats?.length || 0;
        const parserCount = dbData.aiHistory?.parser?.length || 0;

        return (
            <div className="overview-section">
                <div className="profile-summary">
                    <div className="profile-header">
                        <h3>{profile.name}</h3>
                        <div className="profile-meta">
                            <div className="meta-item">
                                <span className="meta-label">Created</span>
                                <span className="meta-value">{formatTimestamp(dbData.classInfo.created_at)}</span>
                            </div>
                            <div className="meta-item">
                                <span className="meta-label">Drops</span>
                                <span className="meta-value">{profile.currentDrops !== undefined ? profile.currentDrops : 'Unknown'}</span>
                            </div>
                        </div>
                    </div>
                </div>

                <div className="activity-overview">
                    <div className="activity-header">
                        <h3>Activity Overview</h3>
                    </div>

                    <div className="stat-cards">
                        <div className="stat-card" onClick={() => handleSectionChange('sessions')}>
                            <div className="stat-icon"><i className="ri-time-line"></i></div>
                            <div className="stat-number">{sessionCount}</div>
                            <div className="stat-label">Sessions</div>
                        </div>
                        
                        <div className="stat-card" onClick={() => handleSectionChange('ai-history')}>
                            <div className="stat-icon"><i className="ri-upload-2-line"></i></div>
                            <div className="stat-number">{uploadCount}</div>
                            <div className="stat-label">Uploads</div>
                        </div>
                        
                        <div className="stat-card" onClick={() => handleSectionChange('ai-history')}>
                            <div className="stat-icon"><i className="ri-message-3-line"></i></div>
                            <div className="stat-number">{chatCount}</div>
                            <div className="stat-label">Chats</div>
                        </div>
                        
                        <div className="stat-card" onClick={() => handleSectionChange('ai-history')}>
                            <div className="stat-icon"><i className="ri-code-line"></i></div>
                            <div className="stat-number">{parserCount}</div>
                            <div className="stat-label">Parser Calls</div>
                        </div>
                    </div>
                </div>

                <div className="recent-sessions">
                    <div className="section-header">
                        <h3>Recent Sessions</h3>
                        <button className="view-all" onClick={() => handleSectionChange('sessions')}>
                            View All
                        </button>
                    </div>
                    <div className="session-list">
                        {dbData.sessions.slice(0, 3).map(session => (
                            <div 
                                key={session.id} 
                                className="session-card"
                                onClick={() => handleSectionChange('sessions')}
                            >
                                <div className="session-title">
                                    {session.topic || session.upload_prompt?.substring(0, 30) || `Session: ${session.id.substring(0, 8)}`}
                                    {session.topic?.length > 30 || (session.upload_prompt && !session.topic && session.upload_prompt.length > 30) ? '...' : ''}
                                </div>
                                <div className="session-time">{formatTimestamp(session.start_timestamp)}</div>
                            </div>
                        ))}
                    </div>
                </div>
            </div>
        );
    };

    const renderSessionsSection = () => {
        if (loading) {
            return <div className="loading-message">Loading sessions...</div>;
        }
        
        if (error) {
            return <div className="error-message">{error}</div>;
        }

        if (dbData.sessions.length === 0) {
            return <div className="no-data-message">No sessions found for this class.</div>;
        }

        if (selectedEntity && selectedEntity.type === 'session') {
            return (
                <div className="session-detail-view">
                    <div className="detail-header">
                        <button className="back-button" onClick={() => setSelectedEntity(null)}>
                            <i className="ri-arrow-left-line"></i> Back to Sessions
                        </button>
                        <h2>{selectedEntity.data.topic || 'Session Details'}</h2>
                    </div>
                    
                    <div className="session-info-cards">
                        <div className="info-card session-info">
                            <h3>Session Information</h3>
                            <div className="info-grid">
                                <div className="info-row">
                                    <span className="info-label">Started:</span>
                                    <span className="info-value">{formatTimestamp(selectedEntity.data.start_timestamp)}</span>
                                </div>
                                <div className="info-row">
                                    <span className="info-label">Ended:</span>
                                    <span className="info-value">
                                        {selectedEntity.data.end_timestamp ? formatTimestamp(selectedEntity.data.end_timestamp) : 'In progress'}
                                    </span>
                                </div>
                                {selectedEntity.data.droplet_count && (
                                    <div className="info-row">
                                        <span className="info-label">Droplets:</span>
                                        <span className="info-value">{selectedEntity.data.droplet_count}</span>
                                    </div>
                                )}
                            </div>
                        </div>
                        
                        <div className="related-api-calls">
                            <h3>API Calls</h3>
                            <div className="api-call-stats">
                                {Object.entries(getRelatedEntities(selectedEntity.data.id, 'session')).map(([type, entries]) => (
                                    <div 
                                        key={type} 
                                        className="api-stat-card" 
                                        onClick={() => switchToApiView(selectedEntity.data.id, type)}
                                    >
                                        <div className="api-stat-count">{entries.length}</div>
                                        <div className="api-stat-label">{type.charAt(0).toUpperCase() + type.slice(1)}</div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    </div>
                    
                    {selectedEntity.data.upload_prompt && (
                        <div className="upload-prompt-card">
                            <h3>Upload Prompt</h3>
                            <div className="prompt-content">{selectedEntity.data.upload_prompt}</div>
                        </div>
                    )}
                    
                    {selectedEntity.data.chat_history && (
                        <div className="chat-history-card">
                            <h3>Chat History</h3>
                            <div className="chat-history-container">
                                {formatChatHistory(selectedEntity.data.chat_history)}
                            </div>
                        </div>
                    )}
                </div>
            );
        }
        
        return (
            <div className="sessions-list-view">
                <div className="list-header">
                    <h2>Sessions</h2>
                    <div className="sessions-count">{dbData.sessions.length} total</div>
                </div>
                
                <div className="sessions-info">
                    <p>Select a session from the sidebar to view detailed information.</p>
                </div>
            </div>
        );
    };

    const renderAIHistorySection = () => {
        if (loading) {
            return <div className="loading-message">Loading API call history...</div>;
        }
        
        if (error) {
            return <div className="error-message">{error}</div>;
        }

        const aiHistory = dbData.aiHistory || { uploads: [], chats: [], parser: [] };
        
        const aiTypes = [
            { key: 'uploads', label: 'Uploads', data: aiHistory.uploads || [], icon: 'ri-upload-2-line' },
            { key: 'chats', label: 'Chats', data: aiHistory.chats || [], icon: 'ri-message-3-line' },
            { key: 'parser', label: 'Parser', data: aiHistory.parser || [], icon: 'ri-code-line' }
        ];
        
        const parentType = selectedEntity?.parentType || selectedEntity?.type || 'uploads';
        
        const getFilteredData = (data, sessionFilter) => {
            if (!data) return [];
            if (!sessionFilter) return data;
            return data.filter(entry => entry?.session_id === sessionFilter);
        };

        const handleApiTabClick = (type) => {
            const filteredData = getFilteredData(aiHistory[type], selectedEntity?.sessionFilter);
            setSelectedEntity({
                data: filteredData,
                type: type,
                sessionFilter: selectedEntity?.sessionFilter
            });
        };

        //may wanto to rework this at some point
        if (selectedEntity?.type === 'ai-entry') {
            const entry = selectedEntity.data;
            if (!entry) return <div className="error-message">Entry data is missing</div>;
            
            return (
                <div className="api-entry-detail-view">
                    <div className="detail-header">
                        <button className="back-button" onClick={() => setSelectedEntity({ 
                            data: selectedEntity.parentData, 
                            type: selectedEntity.parentType || 'uploads'
                        })}>
                            <i className="ri-arrow-left-line"></i> Back to {
                                selectedEntity.parentType
                                    ? selectedEntity.parentType.charAt(0).toUpperCase() + selectedEntity.parentType.slice(1)
                                    : 'List'
                            }
                        </button>
                        <h2>API Call Details</h2>
                    </div>
                    
                    <div className="api-entry-info">
                        <div className="info-card">
                            <h3>Request Information</h3>
                            <div className="info-grid">
                                <div className="info-row">
                                    <span className="info-label">Timestamp:</span>
                                    <span className="info-value">{formatTimestamp(entry.timestamp)}</span>
                                </div>
                                <div className="info-row">
                                    <span className="info-label">Tokens:</span>
                                    <span className="info-value">{entry.tokens || 0}</span>
                                </div>
                                <div className="info-row">
                                    <span className="info-label">Cost:</span>
                                    <span className="info-value">${entry.cost || 0}</span>
                                </div>
                                <div className="info-row">
                                    <span className="info-label">Session:</span>
                                    <span className="info-value session-link" onClick={() => {
                                        if (!entry.session_id) return;
                                        
                                        const session = dbData.sessions.find(s => s.id === entry.session_id);
                                        if (session) {
                                            setActiveSection('sessions');
                                            handleEntityClick(session, 'session');
                                        }
                                    }}>
                                        <i className="ri-link"></i> {entry.session_id ? safeSubstring(entry.session_id, 0, 8) : 'Unknown'}
                                    </span>
                                </div>
                            </div>
                        </div>
                    </div>
                    
                    {entry.prompt && (
                        <div className="content-card">
                            <h3>Prompt</h3>
                            <div className="content-box">{entry.prompt}</div>
                        </div>
                    )}
                    
                    {entry.response && (
                        <div className="content-card">
                            <h3>Response</h3>
                            <div className="content-box">{entry.response}</div>
                        </div>
                    )}
                </div>
            );
        }
        
        return (
            <div className="api-history-section">
                <div className="api-header">
                    <h2>API Call History</h2>
                    {selectedEntity?.sessionFilter && (
                        <div className="filter-badge">
                            Filtered by Session: 
                            <span className="session-id">{safeSubstring(selectedEntity.sessionFilter, 0, 8)}</span>
                            <button className="clear-filter" onClick={() => setSelectedEntity({ 
                                data: selectedEntity.data, 
                                type: selectedEntity.type || 'uploads',
                                sessionFilter: null 
                            })}>
                                <i className="ri-close-line"></i>
                            </button>
                        </div>
                    )}
                </div>
                
                <div className="api-tabs">
                    {aiTypes.map(type => (
                        <button 
                            key={type.key}
                            className={`api-tab ${parentType === type.key ? 'active' : ''}`}
                            onClick={() => handleApiTabClick(type.key)}
                        >
                            <i className={type.icon}></i>
                            {type.label} 
                            <span className="count">
                                ({getFilteredData(type.data, selectedEntity?.sessionFilter).length})
                            </span>
                        </button>
                    ))}
                </div>
                
                <div className="api-info">
                    <p>Select an API call from the sidebar to view detailed information.</p>
                </div>
            </div>
        );
    };

    const renderSessionList = () => {
        if (!dbData.sessions || !Array.isArray(dbData.sessions)) {
            return <div className="no-data-message sidebar-message">No sessions available</div>;
        }
        
        return dbData.sessions.map(session => (
            <div 
                key={session.id} 
                className={`entity-item ${selectedEntity?.type === 'session' && selectedEntity.data.id === session.id ? 'selected' : ''}`}
                onClick={() => handleEntityClick(session, 'session')}
            >
                <div className="entity-header">
                    <div className="entity-title">
                        {session.topic || session.upload_prompt?.substring(0, 25) || `Session ${safeSubstring(session.id, 0, 8)}`}
                        {(session.topic?.length > 25 || (session.upload_prompt?.length > 25 && !session.topic)) ? '...' : ''}
                    </div>
                    <div className="entity-meta">
                        {formatTimestamp(session.start_timestamp)}
                    </div>
                </div>
                <div className="entity-summary">
                    {session.droplet_count && (
                        <div className="entity-stat">
                            <i className="ri-drop-line"></i> {session.droplet_count}
                        </div>
                    )}
                    <div className="entity-stat">
                        <i className="ri-api-line"></i> {
                            getRelatedEntities(session.id, 'session').uploads.length + 
                            getRelatedEntities(session.id, 'session').chats.length + 
                            getRelatedEntities(session.id, 'session').parser.length
                        }
                    </div>
                </div>
            </div>
        ));
    };

    const renderAPIEntryList = () => {
        const parentType = selectedEntity?.parentType || selectedEntity?.type || 'uploads';
        
        const aiHistory = dbData.aiHistory || {};
        const typeData = aiHistory[parentType] || [];
        
        const getFilteredData = (data, sessionFilter) => {
            if (!data || !Array.isArray(data)) return [];
            if (!sessionFilter) return data;
            return data.filter(entry => entry?.session_id === sessionFilter);
        };

        const displayData = getFilteredData(
            selectedEntity?.data || typeData, 
            selectedEntity?.sessionFilter
        );
        
        if (!displayData || displayData.length === 0) {
            return (
                <div className="no-data-message sidebar-message">
                    No {parentType} entries found
                    {selectedEntity?.sessionFilter ? ' for this session' : ''}
                </div>
            );
        }
        
        return displayData.map(entry => (
            <div 
                key={entry.id || Math.random().toString()} 
                className={`entity-item ${selectedEntity?.type === 'ai-entry' && selectedEntity.data.id === entry.id ? 'selected' : ''}`}
                onClick={() => handleEntityClick(entry, 'ai-entry')}
            >
                <div className="entity-header">
                    <div className="entity-title">
                        {parentType.charAt(0).toUpperCase() + parentType.slice(1, -1)} {safeSubstring(entry.id, 0, 8)}
                    </div>
                    <div className="entity-meta">
                        {formatTimestamp(entry.timestamp)}
                    </div>
                </div>
                <div className="entity-summary">
                    <div className="entity-stat">
                        <i className="ri-token-line"></i> {entry.tokens || 0}
                    </div>
                    <div className="entity-stat cost">
                        ${entry.cost || 0}
                    </div>
                </div>
            </div>
        ));
    };

    // we have to use a portal to the document body because the modal is not part of the react component tree
    return ReactDOM.createPortal(
        <div className="modal-overlay">
            <div className="modal-container">
                <div className={`droplet-explorer-modal db-explorer ${sidebarVisible ? '' : 'sidebar-hidden'}`}>
                    {/* titlebar - use this to copy and paste next time im making a modal */}
                    <div id="titlebar" data-wails-drag></div>
                    
                    <div className="draggable-header">
                        <div className="draggable-header-title">
                            {profile.name} Database Explorer
                        </div>
                        <div className="header-actions">
                            <button className="sidebar-toggle" onClick={toggleSidebar}>
                                <i className={`ri-${sidebarVisible ? 'menu-fold-line' : 'menu-unfold-line'}`}></i>
                            </button>
                            <button className="modal-close" onClick={onClose}>
                                <i className="ri-close-line"></i>
                            </button>
                        </div>
                    </div>
                    
                    <div className="db-explorer-tabs">
                        <button 
                            className={`db-tab ${activeSection === 'overview' ? 'active' : ''}`}
                            onClick={() => handleSectionChange('overview')}
                        >
                            <i className="ri-dashboard-line"></i> Overview
                        </button>
                        <button 
                            className={`db-tab ${activeSection === 'sessions' ? 'active' : ''}`}
                            onClick={() => handleSectionChange('sessions')}
                        >
                            <i className="ri-time-line"></i> Sessions
                        </button>
                        <button 
                            className={`db-tab ${activeSection === 'ai-history' ? 'active' : ''}`}
                            onClick={() => handleSectionChange('ai-history')}
                        >
                            <i className="ri-api-line"></i> API Call History
                        </button>
                    </div>
                    
                    <div className="db-explorer-content">
                        {activeSection !== 'overview' && (
                            <div className="droplet-explorer-sidebar">
                                {activeSection === 'sessions' && (
                                    <>
                                        <div className="sidebar-header">
                                            <h3>Sessions</h3>
                                        </div>
                                        <div className="sidebar-content">
                                            {renderSessionList()}
                                        </div>
                                    </>
                                )}
                                
                                {activeSection === 'ai-history' && (
                                    <>
                                        <div className="sidebar-header">
                                            <h3>API Calls</h3>
                                            <div className="api-tabs sidebar-tabs">
                                                {[
                                                    { key: 'uploads', label: 'Uploads', icon: 'ri-upload-2-line' },
                                                    { key: 'chats', label: 'Chats', icon: 'ri-message-3-line' },
                                                    { key: 'parser', label: 'Parser', icon: 'ri-code-line' }
                                                ].map(type => (
                                                    <button 
                                                        key={type.key}
                                                        className={`sidebar-tab ${(selectedEntity?.parentType || selectedEntity?.type) === type.key ? 'active' : ''}`}
                                                        onClick={() => {
                                                            const aiHistory = dbData.aiHistory || {};
                                                            const filteredData = (aiHistory[type.key] || []).filter(entry => 
                                                                selectedEntity?.sessionFilter ? entry?.session_id === selectedEntity.sessionFilter : true
                                                            );
                                                            setSelectedEntity({
                                                                data: filteredData,
                                                                type: type.key,
                                                                sessionFilter: selectedEntity?.sessionFilter
                                                            });
                                                        }}
                                                    >
                                                        <i className={type.icon}></i>
                                                    </button>
                                                ))}
                                            </div>
                                        </div>
                                        <div className="sidebar-content">
                                            {renderAPIEntryList()}
                                        </div>
                                    </>
                                )}
                            </div>
                        )}
                        
                        <div className="droplet-explorer-main">
                            {activeSection === 'overview' && renderOverviewSection()}
                            {activeSection === 'sessions' && renderSessionsSection()}
                            {activeSection === 'ai-history' && renderAIHistorySection()}
                        </div>
                    </div>
                </div>
            </div>
        </div>,
        document.body
    );
};

export default DropletExplorerModal;