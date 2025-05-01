import React, { useState, useEffect, useContext } from 'react';
import ThemeToggle from './ThemeToggle';
import Cup from './Cup';
import SettingsModal from './SettingsModal';
import UserProfileModal from './UserProfileModal';
import flntLogo from '../assets/images/flntribbon.png';
import { GetProfiles } from '../../wailsjs/go/main/App';

const Header = () => { 
  const [showSettings, setShowSettings] = useState(false);
  const [showUserProfile, setShowUserProfile] = useState(false);
  const [totalDrops, setTotalDrops] = useState(0);
  const [maxPossibleDrops, setMaxPossibleDrops] = useState(1); // Prevent division by zero

  useEffect(() => {
    const loadDropStats = async () => {
      try {
        const profiles = await GetProfiles();
        
        let currentSum = 0;
        let possibleSum = 0;
        
        profiles.forEach(profile => {
          if (profile.name !== 'Add New') {
            currentSum += profile.currentDrops;
            possibleSum += (profile.totalPossibleDrops || 0);
          }
        });
        
        //possibleSum is at least equal to currentSum
        possibleSum = Math.max(possibleSum, currentSum);
        
        console.log("Master cup calculation:", {
          totalDrops: currentSum,
          totalPossibleDrops: possibleSum,
          profiles: profiles.map(p => ({ 
            name: p.name, 
            drops: p.currentDrops, 
            possible: p.totalPossibleDrops 
          }))
        });
        
        setTotalDrops(currentSum);
        setMaxPossibleDrops(possibleSum > 0 ? possibleSum : 1); // Prevent division by zero
      } catch (err) {
        console.error('Failed to load profile drop stats:', err);
      }
    };
    
    loadDropStats();
    const interval = setInterval(loadDropStats, 2000);
    
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    const header = document.querySelector('.header');
    const handleMouseMove = (e) => {
      const rect = header.getBoundingClientRect();
      const x = ((e.clientX - rect.left) / rect.width) * 100;
      const y = ((e.clientY - rect.top) / rect.height) * 100;
      header.style.setProperty('--mouse-x', `${x}%`);
      header.style.setProperty('--mouse-y', `${y}%`);
    };

    header.addEventListener('mousemove', handleMouseMove);
    return () => header.removeEventListener('mousemove', handleMouseMove);
  }, []);

  return (
    <>
      <header className="header">
        <div className="menu-section">
          <button className="user-button" onClick={() => setShowSettings(true)}>
            <i className="ri-settings-3-line"></i>
          </button>
          <button className="user-button" onClick={() => setShowUserProfile(true)}>
            <i className="ri-user-line"></i>
          </button>
        </div>
        <div className="logo-container">
          <img src={flntLogo} alt="flnt" className="flnt-logo" />
          <div className="logo">
            <span>fl</span>
            <Cup currentDrops={totalDrops} maxDrops={maxPossibleDrops} />
            <span>ent</span>
          </div>
        </div>
        <div className="theme-section">
          <ThemeToggle />
        </div>
      </header>

      {showSettings && <SettingsModal onClose={() => setShowSettings(false)} />}
      {showUserProfile && <UserProfileModal onClose={() => setShowUserProfile(false)} />}
    </>
  );
};

export default Header; 