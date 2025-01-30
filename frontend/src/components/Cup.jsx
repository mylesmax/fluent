import { useEffect, useRef, useState } from 'react';
import './Cup.css';

const Cup = () => {
    const canvasRef = useRef(null);
    const [isGolden, setIsGolden] = useState(false);
    const [currentDrops, setCurrentDrops] = useState(30);
    const [maxDrops, setMaxDrops] = useState(50);
    const [ripples, setRipples] = useState([]);
    const [goldParticles, setGoldParticles] = useState([]);
    const [isShaking, setIsShaking] = useState(false);
    const animationFrameRef = useRef();
    const shakeStartTimeRef = useRef(0);

    useEffect(() => {
        const canvas = canvasRef.current;
        const ctx = canvas.getContext('2d');
        const dpr = window.devicePixelRatio || 1;
        const cupWidth = 120;
        const cupHeight = 140;

        // Setup canvas
        canvas.width = cupWidth * dpr;
        canvas.height = cupHeight * dpr;
        canvas.style.width = `${cupWidth}px`;
        canvas.style.height = `${cupHeight}px`;
        ctx.scale(dpr, dpr);

        const drawShimmeringWater = (ctx, x, y, width, height) => {
            const time = Date.now() / 1000;
            const gradient = ctx.createLinearGradient(x, y + height, x + width, y);
            const colors = [
                'rgba(129, 188, 239, 0.8)',
                'rgba(85, 155, 232, 0.7)',
                'rgba(50, 85, 122, 0.6)'
            ];

            for (let i = 0; i < 15; i++) {
                const stop = i / 14;
                const wave = Math.sin(time * 1.5 + i * 0.2) * 0.05;
                const mix = stop + wave;
                gradient.addColorStop(stop, lerpColor(
                    mix < 0.5 ? colors[0] : colors[1],
                    mix < 0.5 ? colors[1] : colors[2],
                    mix < 0.5 ? mix * 2 : (mix - 0.5) * 2
                ));
            }

            ctx.fillStyle = gradient;
            if (width > 0 && height > 0) {
                ctx.fillRect(x, y, width, height);
            }

            //wavy wavy wavuy
            ctx.beginPath();
            ctx.moveTo(x, y + height);
            ctx.lineTo(x, y);
            
            for (let i = 0; i <= width; i++) {
                let baseY = y;
                
                const normalizedX = i / width;
                const baseFreq = isShaking ? 35 : 30;
                const basePhase = time * baseFreq;
                
                const centerPoint = width / 2;
                const distanceFromCenter = Math.abs(i - centerPoint) / centerPoint;
                const edgeFactor = Math.pow(1 - distanceFromCenter, 2);
                
                const primaryWave = Math.sin(normalizedX * Math.PI * 2 + basePhase * 0.1) 
                    * (1.5 + edgeFactor * 0.5);
                const secondaryWave = Math.sin(normalizedX * Math.PI * 3 + basePhase * 0.07) 
                    * 0.8 * edgeFactor;
                const tertiaryWave = Math.cos(normalizedX * Math.PI * 4 + basePhase * 0.05) 
                    * 0.4 * edgeFactor;
                
                let totalOffset = primaryWave + secondaryWave + tertiaryWave;
                
                if (isShaking) {
                    const elapsed = Date.now() - shakeStartTimeRef.current;
                    const shakeProgress = Math.min(1, elapsed / 1200);
                    
                    const impactProgress = Math.min(1, elapsed / 300);
                    const impactEase = Math.pow(Math.cos(impactProgress * Math.PI), 3);
                    const initialImpact = impactEase * 4.5;
                    
                    const easeOutElastic = (x) => {
                        const c4 = (2 * Math.PI) / 3;
                        return x === 0 ? 0 : x === 1 ? 1
                            : Math.pow(2, -10 * x) * Math.sin((x * 10 - 0.75) * c4) + 1;
                    };
                    
                    const smoothDecay = 1 - easeOutElastic(shakeProgress);
                    
                    const splashPhase = normalizedX * Math.PI * 2;
                    const splashFreq = 0.01 + Math.pow(shakeProgress, 2) * 0.006;
                    const initialSplash = Math.sin(splashPhase + elapsed * splashFreq) * initialImpact;
                    
                    const sloshPhase = elapsed * (0.005 - shakeProgress * 0.002);
                    const horizontalFactor = Math.sin(normalizedX * Math.PI);
                    const sloshAmplitude = 6.5 * smoothDecay;
                    const sloshOffset = Math.sin(sloshPhase) * sloshAmplitude;
                    
                    const momentumFactor = Math.pow(1 - shakeProgress, 1.5);
                    const dampedOffset = (sloshOffset + initialSplash) * horizontalFactor * momentumFactor;
                    totalOffset += dampedOffset;
                    
                    const ripplePhase = elapsed * (0.012 - shakeProgress * 0.006);
                    const rippleCenter = (Math.sin(ripplePhase) * 0.5 + 0.5) * width;
                    const distanceToRipple = Math.abs(i - rippleCenter);
                    const normalizedDistance = distanceToRipple / (width * 0.3);
                    const rippleIntensity = Math.max(0, 1 - Math.pow(normalizedDistance, 1.5));
                    
                    const shakeRipple = Math.sin(normalizedDistance * Math.PI * 2 - elapsed * 0.006)
                        * rippleIntensity 
                        * smoothDecay 
                        * 1.5;
                    
                    totalOffset += shakeRipple * momentumFactor;
                }
                
                if (ripples.length > 0) {
                    for (const ripple of ripples) {
                        const dx = x + i - ripple.x;
                        const dy = baseY - ripple.y;
                        const distance = Math.sqrt(dx * dx + dy * dy);
                        
                        const normalizedDistance = distance / (ripple.wavelength * 4);
                        const distanceFactor = Math.max(0, 1 - normalizedDistance);
                        const smoothFactor = Math.pow(distanceFactor, 2);
                        
                        const rippleOffset = Math.sin(normalizedDistance * Math.PI * 2 - ripple.phase) 
                            * ripple.amplitude 
                            * smoothFactor;
                        
                        totalOffset += rippleOffset;
                    }
                }
                
                const maxOffset = height * 0.15;
                totalOffset = maxOffset * Math.tanh(totalOffset / maxOffset);
                
                ctx.lineTo(x + i, baseY + totalOffset);
            }
            
            ctx.lineTo(x + width, y + height);
            ctx.lineTo(x, y + height);
            
            ctx.fillStyle = gradient;
            ctx.fill();

            const waveGradient = ctx.createLinearGradient(x, y, x, y + 5);
            waveGradient.addColorStop(0, 'rgba(106,198,241,0.6)');
            waveGradient.addColorStop(1, 'rgba(255, 255, 255, 0)');
            ctx.fillStyle = waveGradient;
            ctx.fill();

            // glowy glowy glowy
            ctx.save();
            ctx.globalCompositeOperation = 'lighter';
            const glowGradient = ctx.createRadialGradient(
                x + width / 2, y + height / 2, 0,
                x + width / 2, y + height / 2, width / 2
            );
            glowGradient.addColorStop(0, 'rgba(255, 255, 255, 0.05)');
            glowGradient.addColorStop(1, 'rgba(255, 255, 255, 0)');
            ctx.fillStyle = glowGradient;
            ctx.fillRect(x, y, width, height);
            ctx.restore();
        };

        const drawCup = () => {
            ctx.clearRect(0, 0, cupWidth, cupHeight);
            
            if (isShaking) {
                ctx.save();
                const elapsed = Date.now() - shakeStartTimeRef.current;
                const shakeProgress = Math.min(1, elapsed / 1200);
                
                const easeOutElastic = (x) => {
                    const c4 = (2 * Math.PI) / 3;
                    return x === 0 ? 0 : x === 1 ? 1
                        : Math.pow(2, -10 * x) * Math.sin((x * 10 - 0.75) * c4) + 1;
                };
                
                const smoothDecay = 1 - easeOutElastic(shakeProgress);
                const initialImpact = Math.max(0, 1 - elapsed / 300) * 3.5;
                
                const shakeFreq = 0.025 - shakeProgress * 0.012;
                const shakeAmount = (
                    initialImpact * Math.sin(elapsed * 0.045) +
                    5.5 * smoothDecay * Math.sin(elapsed * shakeFreq)
                );
                
                ctx.translate(cupWidth / 2, cupHeight / 2);
                ctx.rotate(shakeAmount * Math.PI / 180);
                ctx.translate(-cupWidth / 2, -cupHeight / 2);
            }

            const waterHeight = 80 * (currentDrops / maxDrops);
            drawShimmeringWater(ctx, 22, 100 - waterHeight, 76, waterHeight);

            //ripples
            ctx.beginPath();
            ctx.moveTo(22, 100 - waterHeight);
            for (let x = 22; x <= 98; x++) {
                let y = 100 - waterHeight;
                for (const ripple of ripples) {
                    const dx = x - ripple.x;
                    const dy = y - ripple.y;
                    const distance = Math.sqrt(dx * dx + dy * dy);
                    y += Math.sin(distance / ripple.wavelength - ripple.phase) * ripple.amplitude;
                }
                ctx.lineTo(x, y);
            }
            ctx.lineTo(98, 100 - waterHeight);
            ctx.closePath();
            ctx.fill();

            //cup outline, TODO: fix this
            ctx.beginPath();
            ctx.moveTo(20, 17);
            ctx.lineTo(20, 80);
            ctx.quadraticCurveTo(20, 100, 40, 100);
            ctx.lineTo(80, 100);
            ctx.quadraticCurveTo(100, 100, 100, 80);
            ctx.lineTo(100, 17);

            ctx.lineWidth = 9;

            if (isGolden) {
                const goldGradient = ctx.createLinearGradient(10, 20, 100, 80);
                goldGradient.addColorStop(0, '#FFD700');
                goldGradient.addColorStop(0.5, '#FFF3A0');
                goldGradient.addColorStop(1, '#FFD700');
                ctx.strokeStyle = goldGradient;
                
                //GOLD GLOW!!!!!
                ctx.save();
                ctx.shadowColor = '#FFD700';
                ctx.shadowBlur = 5;
                ctx.stroke();
                ctx.restore();
            } else {
                ctx.strokeStyle = getComputedStyle(document.documentElement).getPropertyValue('--cup-outline-color').trim();
                ctx.stroke();
            }

            if (isShaking) {
                ctx.restore();
            }
        };

        const animate = () => {
            drawCup();
            if (isShaking) {
                const elapsed = Date.now() - shakeStartTimeRef.current;
                if (elapsed > 700) {
                    setIsShaking(false);
                }
            }
            animationFrameRef.current = requestAnimationFrame(animate);
        };

        animate();

        return () => {
            if (animationFrameRef.current) {
                cancelAnimationFrame(animationFrameRef.current);
            }
        };
    }, [currentDrops, maxDrops, isGolden, ripples, goldParticles, isShaking]);

    const shake = () => {
        if (!isShaking) {
            setIsShaking(true);
            shakeStartTimeRef.current = Date.now();

            const addShakeRipples = () => {
                if (Math.random() < 0.3) {
                    const x = 22 + Math.random() * 76;
                    const y = 100 - 80 * (currentDrops / maxDrops);
                    setRipples(prevRipples => [
                        ...prevRipples,
                        {
                            x,
                            y,
                            amplitude: 1.5 + Math.random(),
                            wavelength: 8 + Math.random() * 6,
                            speed: 0.12 + Math.random() * 0.08,
                            phase: 0
                        }
                    ]);
                }
            };

            const rippleInterval = setInterval(addShakeRipples, 50);

            setTimeout(() => {
                clearInterval(rippleInterval);
            }, 700);
        }
    };

    useEffect(() => {
        const updateRipples = () => {
            setRipples(prevRipples => 
                prevRipples
                    .map(ripple => ({
                        ...ripple,
                        phase: ripple.phase + ripple.speed,
                        amplitude: ripple.amplitude - (ripple.amplitude * 0.02)
                    }))
                    .filter(ripple => ripple.amplitude > 0.1)
            );
        };

        const rippleInterval = setInterval(updateRipples, 16);
        return () => clearInterval(rippleInterval);
    }, []);

    const lerpColor = (color1, color2, amount) => {
        const parseColor = (color) => {
            const match = color.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)(?:,\s*(\d+(?:\.\d+)?))?\)/);
            return {
                r: parseInt(match[1]),
                g: parseInt(match[2]),
                b: parseInt(match[3]),
                a: match[4] ? parseFloat(match[4]) : 1
            };
        };

        const c1 = parseColor(color1);
        const c2 = parseColor(color2);

        return `rgba(${
            Math.floor(c1.r + (c2.r - c1.r) * amount)
        }, ${
            Math.floor(c1.g + (c2.g - c1.g) * amount)
        }, ${
            Math.floor(c1.b + (c2.b - c1.b) * amount)
        }, ${
            c1.a + (c2.a - c1.a) * amount
        })`;
    };

    return (
        <div id="cupContainer" onClick={shake}>
            <canvas ref={canvasRef} id="cupCanvas"></canvas>
            <div id="cupInfo" className={isGolden ? 'golden' : ''}></div>
        </div>
    );
};

export default Cup; 