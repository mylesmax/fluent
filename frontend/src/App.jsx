import { useState, useEffect } from 'react';
import Header from './components/Header';
import './App.css';
import { Greet } from "../wailsjs/go/main/App";
import '@fortawesome/fontawesome-free/css/all.min.css';
import Cup from './components/Cup';
import Profile from './components/Profile';
import ProfileMenu from './components/ProfileMenu';
import AddClassModal from './components/AddClassModal';
import rd1 from './assets/gifs/rd1.gif';

function App() {
    const [selectedProfile, setSelectedProfile] = useState(null);
    const [isLoading, setIsLoading] = useState(true);
    const [isTransitioning, setIsTransitioning] = useState(false);
    const [showMainContent, setShowMainContent] = useState(false);
    const [showProfileMenu, setShowProfileMenu] = useState(false);
    const [showAddClass, setShowAddClass] = useState(false);
    const [profiles, setProfiles] = useState([
        { name: 'BIO123', emoji: '🔬', glowColor: 'rgba(129, 140, 248, 0.5)' },
        { name: 'ESE234', emoji: '⚡️', glowColor: 'rgba(52, 211, 153, 0.5)' },
        { name: 'BME456', emoji: '🧑‍💻', glowColor: 'rgba(251, 146, 60, 0.5)' },
        { name: 'SCIENCE', emoji: '🧪', glowColor: 'rgba(236, 72, 153, 0.5)' },
        { name: 'Add New', emoji: '➕', glowColor: 'rgba(52, 211, 153, 0.5)' }
    ]);

    useEffect(() => {
        document.documentElement.classList.remove('light-mode');
        
        setShowMainContent(true);
        
        const timer = setTimeout(() => {
            setIsTransitioning(true);
        }, 2000);

        return () => clearTimeout(timer);
    }, []);

    useEffect(() => {
        if (isTransitioning) {
            const timer = setTimeout(() => {
                setIsLoading(false);
            }, 600);
            
            return () => clearTimeout(timer);
        }
    }, [isTransitioning]);

    const handleProfileClick = (profile) => {
        if (profile.name === "Add New") {
            setShowAddClass(true);
        } else {
            setSelectedProfile(profile);
            setShowProfileMenu(true);
        }
    };

    const handleCloseMenu = () => {
        setShowProfileMenu(false);
        setSelectedProfile(null);
    };

    const handleUpdateProfileName = (oldName, newName) => {
        setProfiles(prevProfiles => 
            prevProfiles.map(profile => 
                profile.name === oldName 
                    ? { ...profile, name: newName }
                    : profile
            )
        );
        setSelectedProfile(prev => ({ ...prev, name: newName }));
    };

    const handleDeleteProfile = (profileName) => {
        setProfiles(prevProfiles => 
            prevProfiles.filter(profile => 
                profile.name === 'Add New' || profile.name !== profileName
            )
        );
    };

    const handleAddClass = (newClass) => {
        setProfiles(prevProfiles => [
            ...prevProfiles.slice(0, -1),
            newClass,
            prevProfiles[prevProfiles.length - 1]
        ]);
    };

    return (
        <div className="v0_3">
            <div className={`main-content ${showMainContent ? 'visible' : ''}`}>
                <div id="titlebar"></div>
                <Header />
                <div className="content">
                    <div className={`profile-list ${selectedProfile ? 'has-selected' : ''}`}>
                        {profiles.map((profile) => (
                            <Profile
                                key={profile.name}
                                name={profile.name}
                                emoji={profile.emoji}
                                isSelected={selectedProfile?.name === profile.name}
                                isAddNew={profile.name === 'Add New'}
                                onClick={() => handleProfileClick(profile)}
                                glowColor={profile.glowColor}
                            />
                        ))}
                    </div>
                </div>
            </div>

            {showProfileMenu && selectedProfile && (
                <ProfileMenu 
                    profile={selectedProfile}
                    onClose={handleCloseMenu}
                    onUpdateName={handleUpdateProfileName}
                    onDelete={handleDeleteProfile}
                />
            )}

            {showAddClass && (
                <AddClassModal
                    onClose={() => setShowAddClass(false)}
                    onAdd={handleAddClass}
                />
            )}

            {(isLoading || isTransitioning) && (
                <div className={`loading-screen ${isTransitioning ? 'fade-out' : ''}`}>
                    <img src={rd1} alt="Loading..." className="loading-animation" />
                    <div className="loading-text">Getting things started...</div>
                </div>
            )}
        </div>
    );
}

export default App
