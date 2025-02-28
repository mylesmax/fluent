import React, { useState, useEffect, useRef } from 'react';
import './LearnMode.css';
import dropperIcon from '../assets/images/dropper.png';
import rd3Gif from '../assets/gifs/rd3.gif';
import rd1Gif from '../assets/gifs/rd1.gif';
import splashGif from '../assets/gifs/splash.gif';
import pourLoadGif from '../assets/gifs/pour-load.gif';
import SimpleCup from './SimpleCup';

const LearnMode = ({ profile, onClose }) => {
    const [currentScreen, setCurrentScreen] = useState('welcome');
    const [message, setMessage] = useState('');
    const [conversation, setConversation] = useState([]);
    const [uploadText, setUploadText] = useState('');
    const [isAITyping, setIsAITyping] = useState(false);
    
    const [currentDrops, setCurrentDrops] = useState(30);
    const [maxDrops, setMaxDrops] = useState(50);
    const [isAnimatingDrop, setIsAnimatingDrop] = useState(false);
    const [dropAnimation, setDropAnimation] = useState(null);
    const [splashes, setSplashes] = useState([]);
    const [triggerShake, setTriggerShake] = useState(0);
    
    const cupRef = useRef(null);
    const messagesContainerRef = useRef(null);
    
    const handleCupClick = () => {
    };
    
    const addDropsFromChat = (n) => {
        if (isAnimatingDrop) return;
        
        const messagesContainer = messagesContainerRef.current;
        const cup = cupRef.current;
        
        if (!messagesContainer || !cup) return;
        
        const messages = messagesContainer.querySelectorAll('.message.user-message');
        if (messages.length === 0) {
            const allMessages = messagesContainer.querySelectorAll('.message');
            let lastUserMessage = null;
            for (let i = allMessages.length - 1; i >= 0; i--) {
                if (allMessages[i].querySelector('.message-content')) {
                    lastUserMessage = allMessages[i];
                    break;
                }
            }

            if (!lastUserMessage) {
                const newDropsValue = Math.min(currentDrops + n, maxDrops);
                setCurrentDrops(newDropsValue);
                return;
            }
            
            setTimeout(() => {
                animateMultipleDrops(lastUserMessage, cup, n);
            }, 300);
        } else {
            const lastUserMessage = messages[messages.length - 1];
            
            setTimeout(() => {
                animateMultipleDrops(lastUserMessage, cup, n);
            }, 300);
        }
    };
    
    const animateMultipleDrops = (sourceElement, targetElement, totalDrops) => {
        if (!sourceElement || !targetElement) return;
        
        setIsAnimatingDrop(true);
        
        const cupRect = targetElement.getBoundingClientRect();
        const cupWidth = cupRect.width;
        const cupLeft = cupRect.left;
        const cupTop = cupRect.top;
        
        const waterHeightPercentage = currentDrops / maxDrops;
        const waterHeight = 70 * waterHeightPercentage;
        const waterY = cupTop + 78 - waterHeight;
        
        const dropletPaths = [];
        const messageRect = sourceElement.getBoundingClientRect();
        
        for (let i = 0; i < totalDrops; i++) {
            const startX = messageRect.left + (Math.random() * 10 - 5) + (i * 2);
            const startY = messageRect.top + messageRect.height / 2 + (Math.random() * 10 - 5);
            
            const endXOffset = (Math.random() * (cupWidth * 0.6) - (cupWidth * 0.3));
            const endX = cupLeft + cupWidth/2 + endXOffset;
            
            const arcHeight = 50 + (Math.random() * 30);
            const duration = 800 + (Math.random() * 200) + (arcHeight * 2);
            const delay = i * (30 + Math.random() * 70);
            
            dropletPaths.push({
                startX, startY, endX, endY: waterY, arcHeight, duration, delay,
                splash: {
                    size: 30 + Math.random() * 10,
                    opacity: 0.8 + Math.random() * 0.2,
                    offset: {
                        x: 0,
                        y: 0
                    }
                }
            });
        }
        
        let completedDrops = 0;
        
        const naturalEase = (t) => {
            if (t < 0.4) {
                return 2.5 * t * t;
            } else if (t < 0.8) {
                return 0.4 + (t - 0.4) * 1.25;
            } else {
                return 0.9 + (t - 0.8) * 0.5;
            }
        };
        
        const animateDroplet = (path) => {
            const droplet = document.createElement('img');
            droplet.src = rd3Gif;
            droplet.alt = 'Droplet';
            droplet.style.position = 'fixed';
            droplet.style.left = `${path.startX}px`;
            droplet.style.top = `${path.startY}px`;
            droplet.style.width = '25px';
            droplet.style.height = '25px';
            droplet.style.transform = 'scale(0.9)';
            droplet.style.zIndex = '1000';
            droplet.style.pointerEvents = 'none';
            document.body.appendChild(droplet);
            
            const startTime = performance.now();
            
            const animate = (timestamp) => {
                if (!document.body.contains(droplet)) {
                    completedDrops++;
                    checkAllComplete();
                    return;
                }
                
                let progress = Math.min((timestamp - startTime) / path.duration, 1);
                progress = naturalEase(progress);
                
                const x = path.startX + (path.endX - path.startX) * progress;
                
                const y = path.startY + 
                    (path.endY - path.startY) * progress - 
                    Math.sin(Math.PI * progress) * path.arcHeight;
                
                droplet.style.left = `${x}px`;
                droplet.style.top = `${y}px`;
                
                const scale = 0.9 + (progress * 0.2);
                droplet.style.transform = `scale(${scale})`;
                
                if (progress < 1) {
                    requestAnimationFrame(animate);
                } else {
                    const splash = document.createElement('img');
                    splash.src = splashGif;
                    splash.alt = 'Splash';
                    splash.style.position = 'fixed';
                    splash.style.width = `${path.splash.size}px`;
                    splash.style.height = `${path.splash.size}px`;
                    
                    splash.style.left = `${path.endX - (path.splash.size / 2) + 3}px`;
                    splash.style.top = `${path.endY - (path.splash.size / 2) + 2}px`;
                    
                    splash.style.opacity = `${path.splash.opacity}`;
                    splash.style.zIndex = '1001';
                    splash.style.pointerEvents = 'none';
                    
                    document.body.appendChild(splash);
                    document.body.removeChild(droplet);
                    
                    let splashOpacity = path.splash.opacity;
                    const splashFade = setInterval(() => {
                        splashOpacity -= 0.1;
                        if (splashOpacity <= 0 || !document.body.contains(splash)) {
                            clearInterval(splashFade);
                            if (document.body.contains(splash)) {
                                document.body.removeChild(splash);
                            }
                        } else {
                            splash.style.opacity = splashOpacity;
                        }
                    }, 50);
                    
                    setTimeout(() => {
                        if (document.body.contains(splash)) {
                            document.body.removeChild(splash);
                        }
                    }, 250);
                    
                    completedDrops++;
                    checkAllComplete();
                }
            };
            
            setTimeout(() => {
                requestAnimationFrame(animate);
            }, path.delay);
        };
        
        const checkAllComplete = () => {
            if (completedDrops === totalDrops) {
                const newDropsValue = Math.min(currentDrops + totalDrops, maxDrops);
                setCurrentDrops(newDropsValue);
                
                setTriggerShake(prev => prev + 1);
                
                setIsAnimatingDrop(false);
            }
        };
        
        dropletPaths.forEach(animateDroplet);
    };
    
    const welcomeAnimationDone = useRef(false);
    const messagesEndRef = useRef(null);
    const inputRef = useRef(null);
    const welcomeTextRef = useRef('');
    const welcomeFullText = `Let's get Fluent in ${profile.name}.`;
    const typingSpeedMs = 30;
    const animationTimer = useRef(null);
    const [welcomeSubtext, setWelcomeSubtext] = useState('');
    const animationDuration = 5000;
    const animationTimeoutRef = useRef(null);

    useEffect(() => {
        if (currentScreen === 'welcome' && !welcomeAnimationDone.current) {
            let currentCharIndex = 0;
            
            animationTimer.current = setInterval(() => {
                if (currentCharIndex <= welcomeFullText.length) {
                    welcomeTextRef.current = welcomeFullText.substring(0, currentCharIndex);
                    setWelcomeSubtext(welcomeTextRef.current);
                    currentCharIndex++;
                } else {
                    clearInterval(animationTimer.current);
                    welcomeAnimationDone.current = true;
                    
                    animationTimeoutRef.current = setTimeout(() => {
                        setCurrentScreen('upload');
                    }, animationDuration);
                }
            }, typingSpeedMs);
        }
        
        return () => {
            if (animationTimer.current) {
                clearInterval(animationTimer.current);
            }
            if (animationTimeoutRef.current) {
                clearTimeout(animationTimeoutRef.current);
            }
        };
    }, [currentScreen]);

    useEffect(() => {
        const handleKeyDown = (e) => {
            if (e.key === 'Escape' && currentScreen === 'welcome') {
                if (animationTimer.current) {
                    clearInterval(animationTimer.current);
                }
                if (animationTimeoutRef.current) {
                    clearTimeout(animationTimeoutRef.current);
                }
                welcomeAnimationDone.current = true;
                setWelcomeSubtext(welcomeFullText);
                setCurrentScreen('upload');
            }
        };

        window.addEventListener('keydown', handleKeyDown);
        return () => {
            window.removeEventListener('keydown', handleKeyDown);
        };
    }, [currentScreen, welcomeFullText]);

    useEffect(() => {
        return () => {
            if (dropAnimation && document.body.contains(dropAnimation)) {
                try {
                    document.body.removeChild(dropAnimation);
                } catch (error) {
                    console.error("Error cleaning up animation:", error);
                }
            }
        };
    }, [dropAnimation]);

    useEffect(() => {
        if (currentScreen === 'chat') {
            scrollToBottom();
            
            if (inputRef.current) {
                inputRef.current.focus();
            }
        }
    }, [conversation, currentScreen]);

    const scrollToBottom = () => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    };

    const handleSendMessage = () => {
        if (message.trim() === '') return;

        setConversation(prev => [...prev, { role: 'user', content: message }]);
        
        const addDropsMatch = message.match(/^add drops (\d+)$/i);
        if (addDropsMatch) {
            const dropsToAdd = parseInt(addDropsMatch[1], 10);
            
            if (!isNaN(dropsToAdd) && dropsToAdd > 0) {
                setMessage('');
                
                const willExceedMax = currentDrops + dropsToAdd > maxDrops;
                const actualDropsAdded = willExceedMax ? maxDrops - currentDrops : dropsToAdd;
                const finalDropCount = currentDrops + actualDropsAdded;
                
                if (actualDropsAdded <= 0) {
                    setConversation(prev => [
                        ...prev,
                        { 
                            role: 'ai', 
                            content: `Your cup is already full! (${currentDrops}/${maxDrops})` 
                        }
                    ]);
                    return;
                }
                
                setTimeout(() => {
                    addDropsFromChat(actualDropsAdded);
                    
                    let responseContent = `Added ${actualDropsAdded} ${profile.name} ${actualDropsAdded === 1 ? 'droplet' : 'droplets'}. `;
                    
                    if (willExceedMax) {
                        responseContent += `(That's all that would fit! Cup is now full: ${finalDropCount}/${maxDrops})`;
                    } else {
                        responseContent += `Current count: ${finalDropCount}/${maxDrops}`;
                    }
                    
                    setConversation(prev => [
                        ...prev,
                        { 
                            role: 'ai', 
                            content: responseContent
                        }
                    ]);
                }, 200);
                
                return;
            }
        }
        
        setMessage('');
        
        setIsAITyping(true);
        
        setTimeout(() => {
            setIsAITyping(false);
            setConversation(prev => [
                ...prev, 
                { 
                    role: 'ai', 
                    content: `I'm analyzing the content about ${profile.name}. This is a key concept we can explore: [Concept Example]. Would you like me to explain more about this concept or explore something else?` 
                }
            ]);
        }, 1500);
    };

    const handleKeyPress = (e) => {
        if (e.key === 'Enter') {
            if (currentScreen === 'upload') {
                handleProcessText();
            } else if (currentScreen === 'chat') {
                handleSendMessage();
            }
        }
    };

    const handleProcessText = () => {
        if (uploadText.trim() === '') return;
        
        setCurrentScreen('processing');
        
        setTimeout(() => {
            setCurrentScreen('chat');
            setConversation([
                { role: 'ai', content: `I've analyzed the content you provided about ${profile.name}. I've extracted some key concepts that we can explore together.` },
                { role: 'ai', content: `Let's dive into what you'd like to learn. You can ask me specific questions about ${profile.name} or ask for an overview of the main topics.` }
            ]);
        }, 3000);
    };

    const renderScreen = () => {
        switch (currentScreen) {
            case 'welcome':
                return (
                    <div className="welcome-screen">
                        <div className="welcome-content">
                            <div className="liquid-cup-large">
                                <img src={rd3Gif} alt="Liquid animation" className="rd3-animation" />
                            </div>
                            <p className="typing-animation">{welcomeSubtext}</p>
                            <p className="skip-text">ESC to skip</p>
                        </div>
                    </div>
                );
                
            case 'upload':
                return (
                    <div className="upload-screen">
                        <div className="upload-container">
                            <textarea 
                                className="upload-textarea"
                                placeholder={`Paste your ${profile.name} content here...`}
                                value={uploadText}
                                onChange={(e) => setUploadText(e.target.value)}
                                onKeyPress={handleKeyPress}
                                autoFocus
                            />
                            
                            <div 
                                className={`continue-button ${uploadText.trim() === '' ? 'disabled' : ''}`}
                                onClick={uploadText.trim() !== '' ? handleProcessText : undefined}
                            >
                                <span className="continue-text">Continue</span>
                                <div className="continue-icon">
                                    <img src={rd1Gif} alt="Continue" />
                                </div>
                            </div>
                        </div>
                    </div>
                );
                
            case 'processing':
                return (
                    <div className="processing-screen">
                        <div className="processing-content">
                            <img src={pourLoadGif} alt="Processing" className="pour-load-animation" />
                            <h2 className="processing-title">Processing...</h2>
                        </div>
                    </div>
                );
                
            case 'chat':
                return (
                    <div className="chat-screen">
                        <div className="chat-container">
                            <div className="chat-messages" ref={messagesContainerRef}>
                                {conversation.map((msg, index) => (
                                    <div key={index} className={`message ${msg.role}-message`}>
                                        {msg.role !== 'system' && <strong>{msg.role === 'user' ? 'You' : 'AI'}</strong>}
                                        <div className="message-content">{msg.content}</div>
                                    </div>
                                ))}
                                {isAITyping && (
                                    <div className="message ai-message typing">
                                        <strong>AI</strong>
                                        <div className="typing-indicator">
                                            <span></span>
                                            <span></span>
                                            <span></span>
                                        </div>
                                    </div>
                                )}
                                <div ref={messagesEndRef} />
                            </div>

                            <div className="chat-input">
                                <input
                                    type="text"
                                    value={message}
                                    onChange={(e) => setMessage(e.target.value)}
                                    onKeyPress={handleKeyPress}
                                    placeholder="Ask about the content or concepts..."
                                    ref={inputRef}
                                    disabled={isAITyping}
                                />
                                <button 
                                    onClick={handleSendMessage} 
                                    className="send-button"
                                    disabled={isAITyping || message.trim() === ''}
                                >
                                    <i className="ri-send-plane-fill"></i>
                                </button>
                            </div>
                        </div>

                        <div className="chat-sidebar">
                            <div className="cup-container">
                                <div ref={cupRef}>
                                <SimpleCup 
                                    currentDrops={currentDrops} 
                                    maxDrops={maxDrops} 
                                    onClick={handleCupClick} 
                                        triggerShake={triggerShake}
                                />
                                </div>
                                <div className="cup-info">
                                    <span className="droplet-count">{profile.name} Droplets: {currentDrops}/{maxDrops}</span>
                                </div>
                            </div>
                            <div className="factoid-container">
                                <h3>Learning Progress</h3>
                                <div className="factoid-list">
                                    <div className="factoid-item">
                                        <span className="factoid-bullet">•</span>
                                        <p>concept</p>
                                    </div>
                                    <div className="factoid-item">
                                        <span className="factoid-bullet">•</span>
                                        <p>concept</p>
                                    </div>
                                    <div className="factoid-item">
                                        <span className="factoid-bullet">•</span>
                                        <p>concept</p>
                                    </div>
                                </div>
                            </div>
                        </div>
                        
                        {splashes.map((splash, index) => (
                            <img 
                                key={index}
                                src={splashGif} 
                                alt="Splash" 
                                style={{
                                    position: 'fixed',
                                    top: `${splash.top}px`,
                                    left: `${splash.left}px`,
                                    width: `${splash.size}px`,
                                    height: `${splash.size}px`,
                                    zIndex: 1001,
                                    pointerEvents: 'none',
                                    transform: `scale(${splash.scale})`,
                                    opacity: splash.opacity
                                }}
                            />
                        ))}
                    </div>
                );
                
            default:
                return null;
        }
    };

    return (
        <div className={`learn-mode-window ${currentScreen === 'welcome' ? 'fullscreen' : ''}`}>
            {currentScreen !== 'welcome' && (
                <>
                    <div id="titlebar"></div>
                    <div className="learn-mode-header">
                        <button className="back-button-learn" onClick={onClose}>
                            <i className="ri-arrow-left-line"></i>
                        </button>
                        {currentScreen === 'upload' ? (
                            <div className="integrated-header">
                                <h2 className="learn-mode-title">Learn: {profile.name}</h2>
                                <p className="header-instructions">Paste your lecture notes, textbook excerpts, articles, or any educational material</p>
                            </div>
                        ) : (
                            <h2 className="learn-mode-title">Learn: {profile.name}</h2>
                        )}
                        <button className="edit-droplets-button">
                            <img src={dropperIcon} alt="Edit droplets" className="button-icon" />
                            <span className="edit-droplets-tooltip">Edit Droplets</span>
                        </button>
                    </div>
                </>
            )}
            {renderScreen()}
        </div>
    );
};

export default LearnMode; 