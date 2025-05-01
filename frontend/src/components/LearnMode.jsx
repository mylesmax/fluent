import React, { useState, useEffect, useRef } from 'react';
import './LearnMode.css';
import dropperIcon from '../assets/images/dropper.png';
import rd3Gif from '../assets/gifs/rd3.gif';
import rd1Gif from '../assets/gifs/rd1.gif';
import splashGif from '../assets/gifs/splash.gif';
import pourLoadGif from '../assets/gifs/pour-load.gif';
import SimpleCup from './SimpleCup';
import DropletExplorerModal from './DropletExplorerModal';
import { 
    CreateLearnSession, 
    RecordAIUploadHistory, 
    UpdateSessionChatHistory, 
    EndSession,
    ProcessUserMessage,
    GetSessionStatistics,
    ListSessionsByActivity,
    ResumeSession,
    StartSessionConversation,
    UpdateDrops,
    UpdateProfileTotalPossibleDrops
} from '../../wailsjs/go/main/App';

const LearnMode = ({ profile, onClose }) => {
    const [currentScreen, setCurrentScreen] = useState('welcome');
    const [message, setMessage] = useState('');
    const [conversation, setConversation] = useState([]);
    const [uploadText, setUploadText] = useState('');
    const [isAITyping, setIsAITyping] = useState(false);
    const [sessionId, setSessionId] = useState(null);
    const [showDropletExplorer, setShowDropletExplorer] = useState(false);
    const [showSessionSelector, setShowSessionSelector] = useState(false);
    const [availableSessions, setAvailableSessions] = useState([]);
    const [sessionStats, setSessionStats] = useState(null);
    const [factoids, setFactoids] = useState([]);
    const [isPostponeResponsePending, setIsPostponeResponsePending] = useState(false);
    
    const [lastKnownDropsAwarded, setLastKnownDropsAwarded] = useState(0);
    const [dropsBeingAnimated, setDropsBeingAnimated] = useState(0);
    const [currentDrops, setCurrentDrops] = useState(profile?.currentDrops || 0);
    const [maxDrops, setMaxDrops] = useState(50);
    const [isAnimatingDrop, setIsAnimatingDrop] = useState(false);
    const [dropAnimation, setDropAnimation] = useState(null);
    const [splashes, setSplashes] = useState([]);
    const [triggerShake, setTriggerShake] = useState(0);
    
    const cupRef = useRef(null);
    const messagesContainerRef = useRef(null);
    const messagesEndRef = useRef(null);
    const inputRef = useRef(null);
    const welcomeAnimationDone = useRef(false);
    const welcomeTextRef = useRef('');
    const animationTimer = useRef(null);
    const animationTimeoutRef = useRef(null);
    
    const welcomeFullText = `Let's get Fluent in ${profile.name}.`;
    const typingSpeedMs = 30;
    const [welcomeSubtext, setWelcomeSubtext] = useState('');
    const animationDuration = 5000;
    
    const [isIntentionalClose, setIsIntentionalClose] = useState(false);
    const [isSessionEnding, setIsSessionEnding] = useState(false);
    
    const handleCupClick = () => {
        setTriggerShake(prev => prev + 1);
    };
    
    useEffect(() => {
        if (!profile || !profile.classUUID) return;

        setCurrentScreen(prev => (prev === 'welcome' || prev === 'upload') ? 'welcome' : prev);

        ListSessionsByActivity(profile.classUUID)
            .then(sessions => {
                if (sessions && sessions.length > 0) {
                    setAvailableSessions(sessions);
                }
            })
            .catch(err => console.error("failed:", err));
    }, [profile]);
    
    useEffect(() => {
        if (profile && profile.currentDrops !== undefined) {
            //todo
        }
    }, [profile]);
    
    const handleSessionResume = (selectedSessionId) => {
        if (!selectedSessionId) return;
        
        console.log("Resuming session:", selectedSessionId);
        setSessionId(selectedSessionId);
        setShowDropletExplorer(false);
        
        setIsAITyping(true);
        
        GetSessionStatistics(profile.classUUID, selectedSessionId)
            .then(stats => {
                console.log("got the session stats:", {
                    sessionId: selectedSessionId,
                    factoids: stats.factoids?.length || 0,
                    status: stats.status
                });
                
                setSessionStats(stats);
                
                console.log("resume session with:", { 
                    profileDrops: profile.currentDrops || 0,
                    sessionDropsAwarded: stats.drops_awarded || 0
                });
                
                //drops restored
                setCurrentDrops(stats.drops_awarded || 0);
                setLastKnownDropsAwarded(stats.drops_awarded || 0);
                
                if (stats.total_factoids) {
                    setMaxDrops(stats.total_factoids);
                }
                
                const sessionHistory = stats.chat_history || [];
                const formattedConversation = sessionHistory.map(exchange => [
                    { role: 'user', content: exchange.user_message },
                    { role: 'ai', content: exchange.system_message }
                ]).flat();
                
                if (formattedConversation.length === 0) {
                    const welcomeMessage = { 
                        role: 'ai', 
                        content: `Welcome back to your ${profile.name} learning session!` 
                    };
                    //todo: fix this later
                    setConversation([welcomeMessage]);
                    setCurrentScreen('chat');
                    ResumeSession(profile.classUUID, selectedSessionId)
                        .then(() => {
                            console.log("Session resumed successfully");
                            return StartSessionConversation(profile.classUUID);
                        })
                        .then(response => {
                            const initialConversation = [
                                welcomeMessage,
                                { role: 'ai', content: response }
                            ];
                            setConversation(initialConversation);
                            setIsAITyping(false);
                            
                            UpdateSessionChatHistory(profile.classUUID, selectedSessionId, JSON.stringify(initialConversation))
                                .catch(err => console.error("failed to update chat history:", err));
                        })
                        .catch(err => {
                            console.error("failed to start session conversation (trying to resume it):", err);
                            setIsAITyping(false);
                            setConversation([welcomeMessage]);//just default to basic
                        });
                } else {
                    setConversation(formattedConversation);
                    setIsAITyping(false);
                    
                    setCurrentScreen('chat');
                }
            })
            .catch(err => {
                console.error("failed to get session stats:", err);
                setIsAITyping(false);
                setCurrentScreen('upload');
                alert("failed to resume session");
            });
    };
    
    const handleCreateNewSession = () => {
        setShowSessionSelector(false);
        setCurrentScreen('upload');
    };
    
    const addDropsFromChat = (n) => {
        if (isAnimatingDrop) return;
        setDropsBeingAnimated(n);
        //todo: potentially we should just do one drop per factoid, and there's no way to get multiple?
        //this is awfully complicated
        
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
                console.log("upd. direcrly:", n);
                const newDropsValue = Math.min(currentDrops + n, maxDrops);
                setCurrentDrops(newDropsValue);
                
                if (profile && profile.name && n > 0) {
                    UpdateDrops(profile.name, newDropsValue)
                        .catch(err => console.error("failed to update profile drops:", err));
                }
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
                const dropsToAdd = totalDrops;
                const beforeDrops = currentDrops;
                //reset
                setIsAnimatingDrop(false);
                setDropsBeingAnimated(0);
                
                const newDropsValue = Math.min(beforeDrops + dropsToAdd, maxDrops);//pray
                
                console.log("Cup update - ANIMATION COMPLETE:", { 
                    currentDropsBefore: beforeDrops,
                    dropsToAdd: dropsToAdd,
                    newDropsValue: newDropsValue,
                    maxDrops: maxDrops,
                    completedDrops: completedDrops,
                    totalDrops: totalDrops
                });
                
                setCurrentDrops(newDropsValue);
                setTimeout(() => {
                    setTriggerShake(prev => prev + 1);
                    
                    console.log("drops:", newDropsValue);
                }, 10);
                
                if (profile && profile.name && dropsToAdd > 0) {
                    //master update with local
                    const newMasterDrops = profile.currentDrops + dropsToAdd;
                    const newMasterTotalPossible = Math.max(profile.totalPossibleDrops, newMasterDrops);
                    
                    console.log("Updating master cup numerator:", {
                        currentMasterDrops: profile.currentDrops,
                        dropsToAdd: dropsToAdd,
                        newMasterDrops: newMasterDrops,
                        totalPossibleDrops: newMasterTotalPossible
                    });
                    
                    //db
                    UpdateDrops(profile.name, newMasterDrops, newMasterTotalPossible)
                        .then(() => {
                            return UpdateProfileTotalPossibleDrops(profile.classUUID);
                        })
                        .catch(err => console.error("Failed to update master cup denominator:", err));
                }
            }
        };
        
        dropletPaths.forEach(animateDroplet);
    };
    
    useEffect(() => {
        if (currentScreen === 'welcome' && !welcomeAnimationDone.current) {
            let currentIndex = 0;
            welcomeTextRef.current = '';
            
            animationTimer.current = setInterval(() => {
                if (currentIndex < welcomeFullText.length) {
                    welcomeTextRef.current += welcomeFullText[currentIndex];
                    setWelcomeSubtext(welcomeTextRef.current);
                    currentIndex++;
                } else {
                    clearInterval(animationTimer.current);
                    welcomeAnimationDone.current = true;
                    
                    animationTimeoutRef.current = setTimeout(() => {
                        if (availableSessions.length > 0) {
                            setShowSessionSelector(true);
                        } else {
                            setCurrentScreen('upload');
                        }
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
    }, [currentScreen, availableSessions, welcomeFullText, animationDuration]);

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
                setShowSessionSelector(false);//thjis is needed to prevent esc errors
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

    useEffect(() => {
        if (sessionStats && sessionStats.factoids) {
            setFactoids(sessionStats.factoids);
            
            if (sessionStats.total_factoids) {
                setMaxDrops(sessionStats.total_factoids);
            }
        }
    }, [sessionStats]);

    const scrollToBottom = () => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    };

    const handleProcessText = () => {
        if (uploadText.trim() === '') return;
        
        console.log("processing");
        setCurrentScreen('processing');
        
        const emptyConversation = [];
        const conversationJSON = JSON.stringify(emptyConversation);
        
        const safetyTimeout = setTimeout(() => {
            console.log("safety timeout – reverting to upload");
            setCurrentScreen('upload');
            alert("Timed out while processing. Please try again later.");
        }, 120000);
        
        let isTransitioning = false;//added a flag here becauase it broke soo many times
        
        CreateLearnSession(profile.classUUID, uploadText, conversationJSON)
            .then(newSessionId => {
                console.log("successful creation of session:", newSessionId);
                setSessionId(newSessionId);
                
                RecordAIUploadHistory(profile.classUUID, uploadText)
                    .catch(err => console.error("failed to record upload history:", err));
                
                let checkAttempts = 0;
                const pollInterval = 1000;
                const maxCheckAttempts = 30; // up to 30 s total
                
                const checkForFactoids = () => {
                    if (isTransitioning) return;
                    
                    checkAttempts++;
                    console.log(`Checking for factoids (attempt ${checkAttempts}/${maxCheckAttempts})...`);
                    
                    GetSessionStatistics(profile.classUUID, newSessionId)
                        .then(stats => {
                            if (isTransitioning) return;
                            
                            if (stats && stats.factoids && stats.factoids.length > 0) {
                                console.log("Factoids ready! Proceeding to chat screen.");
                                
                                isTransitioning = true;
                                
                                clearTimeout(safetyTimeout);
                                setSessionStats(stats);
                                setCurrentDrops(0);
                                
                                const totalFactoids = stats.factoids.length;
                                const masterTotalPossibleDrops = Math.max(
                                    (profile.totalPossibleDrops || 0) + totalFactoids,
                                    profile.currentDrops
                                );
                                
                                console.log("Updating master cup denominator:", {
                                    currentPossibleDrops: profile.totalPossibleDrops || 0,
                                    sessionFactoids: totalFactoids,
                                    profileCurrentDrops: profile.currentDrops,
                                    newTotalPossibleDrops: masterTotalPossibleDrops
                                });
                                
                                UpdateDrops(profile.name, profile.currentDrops, masterTotalPossibleDrops)
                                    .then(() => {
                                        return UpdateProfileTotalPossibleDrops(profile.classUUID);
                                    })
                                    .catch(err => console.error("Failed to update master cup denominator:", err));
                                
                                setLastKnownDropsAwarded(stats.drops_awarded || 0);
                                
                                setIsAITyping(true);
                                
                                const welcomeMessage = { 
                                    role: 'ai', 
                                    content: `Welcome to ${profile.name}! Let's get Fluent!` 
                                };//make more professional for presentation
                                setConversation([welcomeMessage]);
                                
                                setCurrentScreen('chat');
                                
                                UpdateSessionChatHistory(profile.classUUID, newSessionId, JSON.stringify([welcomeMessage]))
                                    .catch(err => console.error("failed to update chat history:", err));
                                
                                StartSessionConversation(profile.classUUID)
                                    .then(response => {
                                        setIsAITyping(false);
                                        
                                        if (typeof response === 'string' && (response.trim().startsWith('<') || response.includes('<!DOCTYPE'))) {
                                            console.error("received html instead of valid response from API:", response.substring(0, 100));
                                            
                                            const fallbackConversation = [
                                                welcomeMessage,
                                                { role: 'ai', content: `analyzed the content you provided about ${profile.name}. extracted some key concepts that we can explore together.` }
                                            ];
                                            setConversation(fallbackConversation);
                                            
                                            UpdateSessionChatHistory(profile.classUUID, newSessionId, JSON.stringify(fallbackConversation))
                                                .catch(err => console.error("Failed to update chat history with fallback:", err));
                                            return;
                                        }
                                        
                                        const initialConversation = [
                                            welcomeMessage,
                                            { role: 'ai', content: response }
                                        ];
                                        
                                        setConversation(initialConversation);
                                        
                                        UpdateSessionChatHistory(profile.classUUID, newSessionId, JSON.stringify(initialConversation))
                                            .catch(err => console.error("Failed to update chat history:", err));
                                    })
                                    .catch(err => {
                                        console.error("Failed to start conversation:", err);
                                        setIsAITyping(false);
                                    });
                            } else if (checkAttempts < maxCheckAttempts) {
                                setTimeout(checkForFactoids, pollInterval);
                            } else {
                                console.log("Max check attempts reached, proceeding with fallback...");
                                isTransitioning = true;
                                clearTimeout(safetyTimeout);
                                
                                setSessionStats(stats || { factoids: [], drops_awarded: 0 });
                                setCurrentDrops(0);
                                
                                setIsAITyping(true);
                                
                                const fallbackMessage = { 
                                    role: 'ai', 
                                    content: `analyzing the content you provided about ${profile.name}. what topic would you like to explore first?` 
                                };
                                setConversation([fallbackMessage]);
                                setCurrentScreen('chat');
                                
                                UpdateSessionChatHistory(profile.classUUID, newSessionId, JSON.stringify([fallbackMessage]))
                                    .catch(err => console.error("Failed to update chat history:", err));
                                
                                StartSessionConversation(profile.classUUID)
                                    .then(response => {
                                        setIsAITyping(false);
                                        const initialConversation = [
                                            fallbackMessage,
                                            { role: 'ai', content: response }
                                        ];
                                        setConversation(initialConversation);
                                        
                                        UpdateSessionChatHistory(profile.classUUID, newSessionId, JSON.stringify(initialConversation))
                                            .catch(err => console.error("Failed to update chat history:", err));
                                    })
                                    .catch(err => {
                                        console.error("Failed to start conversation:", err);
                                        setIsAITyping(false);
                                    });
                            }
                        })
                        .catch(err => {
                            console.error("Failed to get session statistics:", err);
                            if (checkAttempts < maxCheckAttempts) {
                                setTimeout(checkForFactoids, pollInterval);
                            } else {
                                clearTimeout(safetyTimeout);
                                alert("Failed to process content after multiple attempts. Please try again or use a different text.");
                            }
                        });
                };
                
                checkForFactoids();
            })
            .catch(err => {
                console.error("failed to create session:", err);
                clearTimeout(safetyTimeout);
                alert("failed to create session. Please try again.");
            });
    };

    const handleSendMessage = () => {
        if (message.trim() === '' || isAITyping) return;
        
        const userMessage = { role: 'user', content: message };
        setConversation(prev => [...prev, userMessage]);
        
        const handleFactoidResponse = (factoidResponse, previousConversation, successCallback, errorCallback) => {
            if (typeof factoidResponse === 'string' && 
                (factoidResponse.trim().startsWith('<') || 
                factoidResponse.includes('<!DOCTYPE'))) {
                console.error("Received HTML instead of valid response from API:", 
                             factoidResponse.substring(0, 100));
                
                const fallbackMessage = { 
                    role: 'ai', 
                    content: `great job! Let's move on to another concept about ${profile.name}. do you have any questions about this?` 
                };
                const nextConversation = [...previousConversation, fallbackMessage];
                setConversation(nextConversation);
                
                setIsAITyping(false);
                
                if (sessionId) {
                    UpdateSessionChatHistory(profile.classUUID, sessionId, JSON.stringify(nextConversation))
                        .catch(err => console.error("Failed to update chat history for fallback:", err));
                }
                
                if (errorCallback) errorCallback(nextConversation);
                return;
            }
            
            const nextFactoidMessage = { role: 'ai', content: factoidResponse };
            const nextConversation = [...previousConversation, nextFactoidMessage];
            setConversation(nextConversation);
            
            setIsAITyping(false);
            
            if (sessionId) {
                UpdateSessionChatHistory(profile.classUUID, sessionId, JSON.stringify(nextConversation))
                    .catch(err => console.error("failed to update chat history for next factoid:", err));
            }
            
            if (successCallback) successCallback(nextConversation);
        };
        
        const addDropsMatch = message.match(/^add drops (\d+)$/i);
        if (addDropsMatch) {
            const dropsToAdd = parseInt(addDropsMatch[1], 10);
            
            if (!isNaN(dropsToAdd) && dropsToAdd > 0) {
                setMessage('');
                
                console.log("add drops command detected:", { dropsToAdd, currentDrops, maxDrops });
                
                const willExceedMax = currentDrops + dropsToAdd > maxDrops;
                const actualDropsAdded = willExceedMax ? maxDrops - currentDrops : dropsToAdd;
                const finalDropCount = currentDrops + actualDropsAdded;
                
                if (actualDropsAdded <= 0) {
                    const aiResponse = { 
                        role: 'ai', 
                        content: `Your cup is already full! (${currentDrops}/${maxDrops})` 
                    };
                    const updatedConversation = [...conversation, userMessage, aiResponse];
                    setConversation(updatedConversation);
                    
                    if (sessionId) {
                        UpdateSessionChatHistory(profile.classUUID, sessionId, JSON.stringify(updatedConversation))
                            .catch(err => console.error("Failed to update chat history:", err));
                    }
                    return;
                }
                
                setTimeout(() => {
                    const beforeDrops = currentDrops;
                    
                    setCurrentDrops(finalDropCount);
                    console.log("Manual drops update:", {
                        before: beforeDrops,
                        added: actualDropsAdded,
                        after: finalDropCount
                    });
                    
                    setTimeout(() => {
                        setTriggerShake(prev => prev + 1);
                    }, 50);
                    
                    if (profile && profile.name) {
                        const newMasterTotalPossible = Math.max(profile.totalPossibleDrops, finalDropCount);
                        
                        UpdateDrops(profile.name, finalDropCount, newMasterTotalPossible)
                            .then(() => {
                                if (profile && profile.classUUID) {
                                    return UpdateProfileTotalPossibleDrops(profile.classUUID);
                                }
                            })
                            .catch(err => console.error("failed to update profile drops:", err));
                    }
                    
                    let responseContent = `Added ${actualDropsAdded} ${profile.name} ${actualDropsAdded === 1 ? 'droplet' : 'droplets'}. `;
                    
                    if (willExceedMax) {
                        responseContent += `(Cup is now full: ${finalDropCount}/${maxDrops})`;
                    } else {
                        responseContent += `Current count: ${finalDropCount}/${maxDrops}`;
                    }
                    
                    const aiResponse = { 
                        role: 'ai', 
                        content: responseContent
                    };
                    const updatedConversation = [...conversation, userMessage, aiResponse];
                    setConversation(updatedConversation);
                    
                    if (sessionId) {
                        UpdateSessionChatHistory(profile.classUUID, sessionId, JSON.stringify(updatedConversation))
                            .catch(err => console.error("Failed to update chat history:", err));
                    }
                }, 200);
                
                return;
            }
        }
        
        setMessage('');
        setIsAITyping(true);
        
        if (sessionId) {
            const updatedConversation = [...conversation, userMessage];
            UpdateSessionChatHistory(profile.classUUID, sessionId, JSON.stringify(updatedConversation))
                .catch(err => console.error("Failed to update chat history:", err));
        }
        
        ProcessUserMessage(profile.classUUID, message)
            .then(response => {
                setIsAITyping(false);
                
                const { content, outcome, dropsAwarded } = response;
                
                const aiResponse = { role: 'ai', content: content };
                const updatedConversation = [...conversation, userMessage, aiResponse];
                setConversation(updatedConversation);
                
                if (sessionId) {
                    UpdateSessionChatHistory(profile.classUUID, sessionId, JSON.stringify(updatedConversation))
                        .catch(err => console.error("failed to update chat history:", err));
                }
                
                if (dropsAwarded > lastKnownDropsAwarded) {
                    const newDropsAwarded = dropsAwarded - lastKnownDropsAwarded;
                    
                    console.log("New drops awarded:", {
                        previous: lastKnownDropsAwarded,
                        current: dropsAwarded,
                        added: newDropsAwarded,
                        currentDrops: currentDrops
                    });
                    
                    setLastKnownDropsAwarded(dropsAwarded);
                    
                    setTimeout(() => {
                        setDropsBeingAnimated(newDropsAwarded);
                        
                        console.log("Starting drops animation:", {
                            before: currentDrops,
                            adding: newDropsAwarded,
                            expected: Math.min(currentDrops + newDropsAwarded, maxDrops)
                        });
                        
                        addDropsFromChat(newDropsAwarded);
                    }, 800);
                }
                
                if (outcome === 'success') {
                    let transitionDelay = 3000; // Default 3s delay
                    
                    const quickMasteryIndicators = [
                        "correct", "that's right", "well done", "good job", "excellent", 
                        "perfectly", "exactly", "you got it right", "you're right", "great job"
                    ];
                    
                    const isQuickMastery = quickMasteryIndicators.some(
                        indicator => content.toLowerCase().includes(indicator)
                    );
                    
                    if (content.length < 150 || isQuickMastery) {
                        transitionDelay = 1500; // proficient users
                        console.log("using shorter transition delay due to quick mastery detection");
                    }
                    
                    setTimeout(() => {
                        setIsAITyping(true);
                        
                        StartSessionConversation(profile.classUUID)
                            .then(nextFactoidResponse => {
                                handleFactoidResponse(nextFactoidResponse, updatedConversation);
                            })
                            .catch(err => {
                                console.error("Failed to start conversation for next factoid:", err);
                                setIsAITyping(false);
                                
                                const fallbackMessage = { 
                                    role: 'ai', 
                                    content: `Great job! Let's move on to another concept about ${profile.name}. What would you like to learn about next?` 
                                };
                                const nextConversation = [...updatedConversation, fallbackMessage];
                                setConversation(nextConversation);
                                
                                if (sessionId) {
                                    UpdateSessionChatHistory(profile.classUUID, sessionId, JSON.stringify(nextConversation))
                                        .catch(e => console.error("Failed to update chat history with fallback:", e));
                                }
                            });
                    }, transitionDelay);
                } else if (outcome === 'postpone') {
                    setIsPostponeResponsePending(true);
                } else if (isPostponeResponsePending) {
                    setIsPostponeResponsePending(false);
                    
                    setTimeout(() => {
                        setIsAITyping(true);
                        
                        StartSessionConversation(profile.classUUID)
                            .then(nextFactoidResponse => {
                                handleFactoidResponse(nextFactoidResponse, updatedConversation);
                            })
                            .catch(err => {
                                console.error("Failed to start conversation for next factoid:", err);
                                setIsAITyping(false);
                                
                                const fallbackMessage = { 
                                    role: 'ai', 
                                    content: `nice job! Let's move on to another concept about ${profile.name}. What would you like to learn about next?` 
                                };
                                const nextConversation = [...updatedConversation, fallbackMessage];
                                setConversation(nextConversation);
                                
                                if (sessionId) {
                                    UpdateSessionChatHistory(profile.classUUID, sessionId, JSON.stringify(nextConversation))
                                        .catch(e => console.error("Failed to update chat history with fallback:", e));
                                }
                            });
                    }, 1500); // 1.5 second delay
                }
                
                if (sessionId) {
                    GetSessionStatistics(profile.classUUID, sessionId)
                        .then(stats => {
                            setSessionStats(stats);
                            if (stats.drops_awarded !== undefined) {
                                setLastKnownDropsAwarded(stats.drops_awarded);
                            }
                        })
                        .catch(err => console.error("Failed to get session statistics:", err));
                }
            })
            .catch(err => {
                console.error("Failed to process message:", err);
                
                setIsAITyping(false);
                const aiResponse = { 
                    role: 'ai', 
                    content: `Please try again.`
                };
                const updatedConversation = [...conversation, userMessage, aiResponse];
                setConversation(updatedConversation);
                
                if (sessionId) {
                    UpdateSessionChatHistory(profile.classUUID, sessionId, JSON.stringify(updatedConversation))
                        .catch(err => console.error("Failed to update chat history:", err));
                }
            });
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

    const handleEditDropletsClick = () => {
        setShowDropletExplorer(true);
        document.body.style.overflow = 'hidden';
    };
    
    const handleDropletExplorerClose = () => {
        setShowDropletExplorer(false);
        document.body.style.overflow = '';
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
                            <p className="processing-subtitle">Extracting key concepts and generating learning material</p>
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
                                    {sessionStats && sessionStats.completed_factoids > 0 ? (
                                        <div className="factoid-progress">
                                            <div className="progress-bar">
                                                <div 
                                                    className="progress-fill" 
                                                    style={{
                                                        width: `${(sessionStats.completed_factoids / sessionStats.total_factoids) * 100}%`
                                                    }}
                                                ></div>
                                            </div>
                                            <div className="progress-text">
                                                {sessionStats.completed_factoids} / {sessionStats.total_factoids} concepts
                                            </div>
                                        </div>
                                    ) : (
                                        <p className="no-factoids">Start chatting to learn concepts!</p>
                                    )}
                                    
                                    {}
                                    {factoids && factoids.slice(0, 5).map((factoid, index) => (
                                        <div key={index} className="factoid-item">
                                            <span className="factoid-bullet">•</span>
                                            <p>{factoid.question || factoid.concept || "concept"}</p>
                                        </div>
                                    ))}
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

    const handleBackClick = () => {
        setIsIntentionalClose(true);
        
        if (sessionId && !isSessionEnding) {
            setIsSessionEnding(true);
            console.log("ending session on back click:", sessionId);
            EndSession(profile.classUUID, sessionId)
                .then(() => {
                    if (profile && profile.classUUID) {
                        return UpdateProfileTotalPossibleDrops(profile.classUUID);
                    }
                })
                .catch(err => console.error("failed to end session:", err));
        }
        
        if (profile && profile.classUUID) {
            UpdateProfileTotalPossibleDrops(profile.classUUID)
                .catch(err => console.error("Failed to update total possible drops on exit:", err));
        }
        
        setTimeout(() => {
            onClose();
        }, 100);
    };

    useEffect(() => {
        return () => {
            if (sessionId && !isSessionEnding && (isIntentionalClose || currentScreen === 'chat')) {
                console.log("ending session on component unmount:", sessionId);
                setIsSessionEnding(true);
                EndSession(profile.classUUID, sessionId)
                    .then(() => {
                        if (profile && profile.classUUID) {
                            return UpdateProfileTotalPossibleDrops(profile.classUUID);
                        }
                    })
                    .catch(err => console.error("failed to end session:", err));
            }
        };
    }, [sessionId, profile.classUUID, isIntentionalClose, currentScreen, isSessionEnding]);

    return (
        <div className={`learn-mode-window ${currentScreen === 'welcome' ? 'fullscreen' : ''}`}>
            {currentScreen !== 'welcome' && (
                <>
                    <div id="titlebar"></div>
                    <div className="learn-mode-header">
                        <button className="back-button-learn" onClick={handleBackClick}>
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
                        <button className="edit-droplets-button" onClick={handleEditDropletsClick}>
                            <img src={dropperIcon} alt="Edit droplets" className="button-icon" />
                            <span className="edit-droplets-tooltip">Edit Droplets</span>
                        </button>
                    </div>
                </>
            )}
            {renderScreen()}
            
            {showDropletExplorer && (
                <DropletExplorerModal 
                    profile={profile} 
                    onClose={handleDropletExplorerClose} 
                    onSessionResumed={handleSessionResume}
                />
            )}
        </div>
    );
};

export default LearnMode; 