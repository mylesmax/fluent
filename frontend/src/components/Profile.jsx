import React, { useState, useRef, useEffect } from 'react';
import ReactDOM from 'react-dom';
import './Profile.css';

const Profile = ({ name, emoji, description, isSelected, isAddNew, onClick, glowColor = 'rgba(244, 114, 182, 0.5)', onGlowColorChange }) => {
    const [showColorPicker, setShowColorPicker] = useState(false);
    const [showColorWheel, setShowColorWheel] = useState(false);
    const [customColor, setCustomColor] = useState('#f472b6');
    const colorWheelRef = useRef(null);
    const buttonRef = useRef(null);
    const [pickerPosition, setPickerPosition] = useState({ top: 0, left: 0 });
    
    
    const colorOptions = [
        'rgba(244, 114, 182, 0.5)', 
        'rgba(239, 68, 68, 0.5)',   
        'rgba(59, 130, 246, 0.5)',  
        'rgba(245, 158, 11, 0.5)',  
        'rgba(139, 92, 246, 0.5)',  
        'rgba(5, 150, 105, 0.5)',   
        'rgba(14, 165, 233, 0.5)',  
        'custom'                    
    ];
    
    const style = {
        '--glow-color': glowColor,
        '--glow-color-alpha': glowColor.replace(/[\d.]+\)$/g, '0.15)'),
        '--glow-color-dim': glowColor.replace(/[\d.]+\)$/g, '0.3)')
    };
    
    
    const updatePickerPosition = () => {
        if (buttonRef.current) {
            const rect = buttonRef.current.getBoundingClientRect();
            const windowHeight = window.innerHeight;
            const windowWidth = window.innerWidth;
            
            
            const spaceBelow = windowHeight - rect.bottom;
            const spaceAbove = rect.top;
            const openBelow = spaceBelow >= 200 || spaceBelow > spaceAbove;
            
            setPickerPosition({
                top: openBelow ? rect.bottom + 10 : rect.top - 180,
                left: Math.min(rect.left, windowWidth - 180) 
            });
        }
    };
    
    const handleColorClick = (color, e) => {
        e.stopPropagation(); 
        
        if (color === 'custom') {
            setShowColorWheel(true);
            return;
        }
        
        if (onGlowColorChange) {
            onGlowColorChange(color);
        }
        setShowColorPicker(false);
    };
    
    const toggleColorPicker = (e) => {
        e.stopPropagation(); 
        updatePickerPosition();
        setShowColorPicker(!showColorPicker);
        setShowColorWheel(false);
    };
    
    const handleColorWheelChange = (e) => {
        const color = e.target.value;
        setCustomColor(color);
    };
    
    const applyCustomColor = (e) => {
        e.stopPropagation();
        
        const hex = customColor;
        const r = parseInt(hex.substr(1, 2), 16);
        const g = parseInt(hex.substr(3, 2), 16);
        const b = parseInt(hex.substr(5, 2), 16);
        const rgba = `rgba(${r}, ${g}, ${b}, 0.5)`;
        
        if (onGlowColorChange) {
            onGlowColorChange(rgba);
        }
        setShowColorWheel(false);
        setShowColorPicker(false);
    };
    
    
    useEffect(() => {
        const handleClickOutside = (event) => {
            if (buttonRef.current && !buttonRef.current.contains(event.target)) {
                if (showColorPicker && 
                    (!event.target.closest('.color-picker-dropdown') && 
                     !event.target.closest('.color-wheel-container'))) {
                    setShowColorPicker(false);
                    setShowColorWheel(false);
                }
            }
        };
        
        document.addEventListener('mousedown', handleClickOutside);
        return () => {
            document.removeEventListener('mousedown', handleClickOutside);
        };
    }, [showColorPicker]);
    
    
    useEffect(() => {
        if (showColorPicker) {
            const handleScroll = () => updatePickerPosition();
            const handleResize = () => updatePickerPosition();
            
            window.addEventListener('scroll', handleScroll);
            window.addEventListener('resize', handleResize);
            
            return () => {
                window.removeEventListener('scroll', handleScroll);
                window.removeEventListener('resize', handleResize);
            };
        }
    }, [showColorPicker]);

    
    const renderColorPickerPortal = () => {
        if (!showColorPicker) return null;
        
        const content = (
            <div 
                className="color-picker-dropdown fixed"
                onClick={e => e.stopPropagation()}
                style={{
                    position: 'fixed',
                    top: `${pickerPosition.top}px`,
                    left: `${pickerPosition.left}px`,
                }}
            >
                {colorOptions.map((color, index) => (
                    <button
                        key={index}
                        className={`color-option ${color === 'custom' ? 'color-option-plus' : ''}`}
                        style={color !== 'custom' ? { backgroundColor: color } : {}}
                        onClick={(e) => handleColorClick(color, e)}
                    >
                        {color === 'custom' && '+'}
                    </button>
                ))}
            </div>
        );
        
        
        if (typeof document !== 'undefined') {
            return ReactDOM.createPortal(content, document.body);
        }
        
        return null;
    };
    
    const renderColorWheelPortal = () => {
        if (!showColorWheel) return null;
        
        const content = (
            <div 
                className="color-wheel-container" 
                onClick={e => e.stopPropagation()} 
                ref={colorWheelRef}
                style={{
                    position: 'fixed',
                    top: `${pickerPosition.top}px`,
                    left: `${pickerPosition.left}px`,
                }}
            >
                <input 
                    type="color" 
                    className="color-wheel"
                    value={customColor}
                    onChange={handleColorWheelChange}
                />
                <button className="apply-color-button" onClick={applyCustomColor}>
                    Apply
                </button>
            </div>
        );
        
        
        if (typeof document !== 'undefined') {
            return ReactDOM.createPortal(content, document.body);
        }
        
        return null;
    };

    
    useEffect(() => {
        return () => {
            setShowColorPicker(false);
            setShowColorWheel(false);
        };
    }, []);

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
                
                {!isAddNew && (
                    <div className="color-switcher-container">
                        <button 
                            className="color-switcher-button"
                            onClick={toggleColorPicker}
                            style={{ backgroundColor: glowColor }}
                            title="Change glow color"
                            ref={buttonRef}
                        >
                            <span className="color-dot" style={{ backgroundColor: glowColor }}></span>
                        </button>
                        
                        {renderColorPickerPortal()}
                        {renderColorWheelPortal()}
                    </div>
                )}
            </div>
        </div>
    );
};

export default Profile; 