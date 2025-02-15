import React, { useEffect } from 'react';
import ThemeToggle from './ThemeToggle';
import Cup from './Cup';
import flntLogo from '../assets/images/flntribbon.png';

const Header = () => { 
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
    <header className="header">
      <div className="menu-section">
        <button className="menu-button">
          <i className="ri-settings-3-line"></i>
        </button>
        <button className="user-button">
          <i className="ri-user-line"></i>
        </button>
      </div>
      <div className="logo-container">
        <img src={flntLogo} alt="flnt" className="flnt-logo" />
        <div className="logo">
          <span>fl</span>
          <Cup />
          <span>ent</span>
        </div>
      </div>
      <div className="theme-section">
        <ThemeToggle />
      </div>
    </header>
  );
};

export default Header; 