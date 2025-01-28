import { useEffect, useRef, useState } from 'react';
import './Cup.css';

const Cup = () => {
    const canvasRef = useRef(null);
    const [isGolden, setIsGolden] = useState(false);
    const [currentDrops, setCurrentDrops] = useState(30);
    const [maxDrops, setMaxDrops] = useState(50);
    const [ripples, setRipples] = useState([]);
    const [goldParticles, setGoldParticles] = useState([]);
    const animationFrameRef = useRef();

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
            ctx.moveTo(x, y);
            for (let i = 0; i <= width; i++) {
                ctx.lineTo(x + i, y + Math.sin((i + time * 30) * 0.1) * 2);
            }
            ctx.lineTo(x + width, y + height);
            ctx.lineTo(x, y + height);
            ctx.closePath();

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
            ctx.lineWidth = 9;
        };

        const animate = () => {
            drawCup();
            animationFrameRef.current = requestAnimationFrame(animate);
        };

        animate();

        return () => {
            if (animationFrameRef.current) {
                cancelAnimationFrame(animationFrameRef.current);
            }
        };
    }, [currentDrops, maxDrops, isGolden, ripples, goldParticles]);

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
        <div id="cupContainer">
            <canvas ref={canvasRef} id="cupCanvas"></canvas>
            <div id="cupInfo" className={isGolden ? 'golden' : ''}></div>
        </div>
    );
};

export default Cup; 