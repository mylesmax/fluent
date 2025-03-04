import { useState, useEffect } from 'react';
import Header from './components/Header';
import './App.css';
import { GetProfiles, AddProfile, UpdateProfileName, DeleteProfile, UpdateProfileGlowColor } from "../wailsjs/go/main/App";
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
    const [profiles, setProfiles] = useState([]);

    useEffect(() => {
        document.documentElement.classList.add('dark-mode');
        
        loadProfiles();
        setShowMainContent(true);
        
        const timer = setTimeout(() => {
            setIsTransitioning(true);
        }, 2000);

        return () => clearTimeout(timer);
    }, []);

    const loadProfiles = async () => {
        try {
            const loadedProfiles = await GetProfiles();
            setProfiles(loadedProfiles);
            if (selectedProfile) {
                const updatedProfile = loadedProfiles.find(p => p.classUUID === selectedProfile.classUUID);
                if (updatedProfile) {
                    setSelectedProfile(updatedProfile);
                }
            }
        } catch (err) {
            console.error('fail to load profiles:', err);
        }
    };

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

    const handleUpdateProfileName = async (oldName, newName) => {
        try {
            await UpdateProfileName(oldName, newName);
            await loadProfiles();
        } catch (err) {
            console.error('faild to update profile name:', err);
        }
    };

    const handleDeleteProfile = async (profileName) => {
        try {
            await DeleteProfile(profileName);
            await loadProfiles();
        } catch (err) {
            console.error('fail to delete profile:', err);
        }
    };

    const handleAddClass = async (newClass) => {
        try {
            await AddProfile({
                name: newClass.name,
                emoji: newClass.emoji,
                glowColor: newClass.glowColor,
                currentDrops: 0,
                isAddNew: false
            });
            await loadProfiles();
            setShowAddClass(false);
        } catch (err) {
            console.error('fail to add profile:', err);
        }
    };

    const handleUpdateGlowColor = async (profileName, newColor) => {
        try {
            await UpdateProfileGlowColor(profileName, newColor);
            await loadProfiles();
        } catch (err) {
            console.error('failed to update glow color:', err);
        }
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
                                key={profile.classUUID || profile.name}
                                name={profile.name}
                                emoji={profile.emoji}
                                isSelected={selectedProfile?.classUUID === profile.classUUID}
                                isAddNew={profile.name === 'Add New'}
                                onClick={() => handleProfileClick(profile)}
                                glowColor={profile.glowColor}
                                onGlowColorChange={(newColor) => handleUpdateGlowColor(profile.name, newColor)}
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
