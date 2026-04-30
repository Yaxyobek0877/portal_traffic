// app.js — landing page interactions and animations
// All effects respect prefers-reduced-motion.

(() => {
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  // ---------------------------------------------------------------
  // 1. Reveal-on-scroll with stagger
  // ---------------------------------------------------------------
  const io = new IntersectionObserver((entries) => {
    for (const e of entries) {
      if (e.isIntersecting) {
        e.target.classList.add('in');
        io.unobserve(e.target);
      }
    }
  }, { threshold: 0.12, rootMargin: '0px 0px -40px 0px' });

  document.querySelectorAll('.reveal').forEach((el, i) => {
    el.style.transitionDelay = (i % 6) * 60 + 'ms';
    io.observe(el);
  });

  // ---------------------------------------------------------------
  // 2. Copy-to-clipboard buttons
  // ---------------------------------------------------------------
  document.querySelectorAll('.copy[data-copy]').forEach((btn) => {
    btn.addEventListener('click', async () => {
      try {
        await navigator.clipboard.writeText(btn.dataset.copy);
        const old = btn.textContent;
        btn.textContent = 'copied ✓';
        btn.classList.add('flash');
        setTimeout(() => { btn.textContent = old; btn.classList.remove('flash'); }, 1400);
      } catch (e) { /* ignore */ }
    });
  });

  if (reduced) return; // motion-sensitive users get static UI from here

  // ---------------------------------------------------------------
  // 3. Particle network canvas
  //    Subtle drifting points; lines connect close pairs. Mouse
  //    repels gently. Renders behind everything via CSS z-index.
  // ---------------------------------------------------------------
  const canvas = document.getElementById('particles');
  if (canvas) {
    const ctx = canvas.getContext('2d');
    let w, h, dpr, particles, mouse = { x: -9999, y: -9999 };

    function resize() {
      dpr = Math.min(window.devicePixelRatio || 1, 2);
      w = canvas.clientWidth = window.innerWidth;
      h = canvas.clientHeight = window.innerHeight;
      canvas.width  = w * dpr;
      canvas.height = h * dpr;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      // Density scales with viewport area; clamp so phones don't melt.
      const target = Math.min(120, Math.max(40, Math.floor((w * h) / 18000)));
      particles = Array.from({ length: target }, () => spawn());
    }
    function spawn() {
      return {
        x: Math.random() * w,
        y: Math.random() * h,
        vx: (Math.random() - 0.5) * 0.18,
        vy: (Math.random() - 0.5) * 0.18,
        r: Math.random() * 1.4 + 0.4,
      };
    }

    function tick() {
      ctx.clearRect(0, 0, w, h);
      const linkDist = 130;
      const linkDist2 = linkDist * linkDist;
      const repelDist = 110;
      const repelDist2 = repelDist * repelDist;

      // Update + draw nodes
      for (const p of particles) {
        // Mouse repulsion
        const dx = p.x - mouse.x, dy = p.y - mouse.y;
        const d2 = dx * dx + dy * dy;
        if (d2 < repelDist2) {
          const f = (1 - d2 / repelDist2) * 0.4;
          p.vx += (dx / Math.sqrt(d2 + 0.001)) * f * 0.05;
          p.vy += (dy / Math.sqrt(d2 + 0.001)) * f * 0.05;
        }
        p.x += p.vx; p.y += p.vy;
        p.vx *= 0.99; p.vy *= 0.99;
        if (p.x < 0) { p.x = 0; p.vx *= -1; }
        if (p.y < 0) { p.y = 0; p.vy *= -1; }
        if (p.x > w) { p.x = w; p.vx *= -1; }
        if (p.y > h) { p.y = h; p.vy *= -1; }

        ctx.beginPath();
        ctx.arc(p.x, p.y, p.r, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(180, 195, 230, 0.55)';
        ctx.fill();
      }

      // Connect lines under threshold
      for (let i = 0; i < particles.length; i++) {
        const a = particles[i];
        for (let j = i + 1; j < particles.length; j++) {
          const b = particles[j];
          const dx = a.x - b.x, dy = a.y - b.y;
          const d2 = dx * dx + dy * dy;
          if (d2 < linkDist2) {
            const t = 1 - d2 / linkDist2;
            ctx.strokeStyle = `rgba(139,92,246,${t * 0.18})`;
            ctx.lineWidth = 0.7;
            ctx.beginPath();
            ctx.moveTo(a.x, a.y); ctx.lineTo(b.x, b.y);
            ctx.stroke();
          }
        }
      }
      raf = requestAnimationFrame(tick);
    }

    let raf;
    window.addEventListener('mousemove', (e) => { mouse.x = e.clientX; mouse.y = e.clientY; });
    window.addEventListener('mouseleave', () => { mouse.x = mouse.y = -9999; });
    window.addEventListener('resize', resize);
    document.addEventListener('visibilitychange', () => {
      if (document.hidden) { cancelAnimationFrame(raf); }
      else { raf = requestAnimationFrame(tick); }
    });
    resize();
    raf = requestAnimationFrame(tick);
  }

  // ---------------------------------------------------------------
  // 4. Wormhole hero — mouse parallax tilt
  // ---------------------------------------------------------------
  const wh = document.querySelector('.wormhole-wrap');
  const hero = document.querySelector('section.hero');
  if (wh && hero) {
    let tx = 0, ty = 0, cx = 0, cy = 0;
    hero.addEventListener('mousemove', (e) => {
      const r = hero.getBoundingClientRect();
      const px = (e.clientX - r.left) / r.width  - 0.5;
      const py = (e.clientY - r.top)  / r.height - 0.5;
      tx = px * 16;
      ty = py * 16;
    });
    hero.addEventListener('mouseleave', () => { tx = 0; ty = 0; });
    function frame() {
      cx += (tx - cx) * 0.08;
      cy += (ty - cy) * 0.08;
      wh.style.transform = `translate3d(${cx * 0.6}px, ${cy * 0.6}px, 0) rotateY(${cx}deg) rotateX(${-cy}deg)`;
      requestAnimationFrame(frame);
    }
    frame();
  }

  // ---------------------------------------------------------------
  // 5. 3D tilt on feature cards
  // ---------------------------------------------------------------
  document.querySelectorAll('.card.tilt').forEach((card) => {
    let raf, tx = 0, ty = 0, cx = 0, cy = 0, gx = 50, gy = 50;
    card.addEventListener('mousemove', (e) => {
      const r = card.getBoundingClientRect();
      const px = (e.clientX - r.left) / r.width  - 0.5;
      const py = (e.clientY - r.top)  / r.height - 0.5;
      tx = -py * 8;
      ty = px * 8;
      gx = ((e.clientX - r.left) / r.width)  * 100;
      gy = ((e.clientY - r.top)  / r.height) * 100;
      if (!raf) raf = requestAnimationFrame(apply);
    });
    card.addEventListener('mouseleave', () => {
      tx = 0; ty = 0; gx = 50; gy = 50;
      if (!raf) raf = requestAnimationFrame(apply);
    });
    function apply() {
      cx += (tx - cx) * 0.16;
      cy += (ty - cy) * 0.16;
      card.style.transform = `perspective(900px) rotateX(${cx}deg) rotateY(${cy}deg) translateY(${Math.abs(cx) + Math.abs(cy) > 0.05 ? -3 : 0}px)`;
      card.style.setProperty('--mx', gx + '%');
      card.style.setProperty('--my', gy + '%');
      raf = (Math.abs(tx - cx) + Math.abs(ty - cy) > 0.01) ? requestAnimationFrame(apply) : null;
    }
  });

  // ---------------------------------------------------------------
  // 6. Number counters (data-count attribute)
  // ---------------------------------------------------------------
  const counterIO = new IntersectionObserver((entries) => {
    for (const e of entries) {
      if (!e.isIntersecting) continue;
      counterIO.unobserve(e.target);
      const el = e.target;
      const target = parseFloat(el.dataset.count);
      const decimals = (el.dataset.count.split('.')[1] || '').length;
      const start = performance.now();
      const dur = 1400;
      function step(t) {
        const k = Math.min(1, (t - start) / dur);
        const eased = 1 - Math.pow(1 - k, 3);
        el.textContent = (target * eased).toFixed(decimals);
        if (k < 1) requestAnimationFrame(step);
        else el.textContent = el.dataset.suffix
          ? (target.toFixed(decimals) + el.dataset.suffix)
          : target.toFixed(decimals);
      }
      requestAnimationFrame(step);
    }
  }, { threshold: 0.5 });
  document.querySelectorAll('[data-count]').forEach((el) => counterIO.observe(el));

  // ---------------------------------------------------------------
  // 7. Live mesh visualization (animated SVG)
  //    A small physics-light rendering: 5 peers around a circle,
  //    full mesh edges, pulses traveling along edges in random
  //    directions to evoke "data flowing".
  // ---------------------------------------------------------------
  const meshSvg = document.getElementById('live-mesh');
  if (meshSvg) {
    const N = 5;
    const cx = 250, cy = 200, R = 130;
    const peers = Array.from({ length: N }, (_, i) => {
      const a = (i / N) * Math.PI * 2 - Math.PI / 2;
      return { id: i, x: cx + Math.cos(a) * R, y: cy + Math.sin(a) * R };
    });

    const NS = 'http://www.w3.org/2000/svg';
    const layer = (cls) => {
      const g = document.createElementNS(NS, 'g');
      g.setAttribute('class', cls);
      meshSvg.appendChild(g);
      return g;
    };
    const lineLayer = layer('mesh-lines');
    const pulseLayer = layer('mesh-pulses');
    const nodeLayer = layer('mesh-nodes');

    // Edges
    const edges = [];
    for (let i = 0; i < N; i++) {
      for (let j = i + 1; j < N; j++) {
        const a = peers[i], b = peers[j];
        const ln = document.createElementNS(NS, 'line');
        ln.setAttribute('x1', a.x); ln.setAttribute('y1', a.y);
        ln.setAttribute('x2', b.x); ln.setAttribute('y2', b.y);
        ln.setAttribute('stroke', 'url(#edgeGrad)');
        ln.setAttribute('stroke-width', '1');
        ln.setAttribute('opacity', '0.35');
        lineLayer.appendChild(ln);
        edges.push({ a, b, line: ln });
      }
    }

    // Nodes
    peers.forEach((p, i) => {
      const ring = document.createElementNS(NS, 'circle');
      ring.setAttribute('cx', p.x); ring.setAttribute('cy', p.y);
      ring.setAttribute('r', 18);
      ring.setAttribute('fill', 'none');
      ring.setAttribute('stroke', 'url(#nodeRing)');
      ring.setAttribute('stroke-width', '1');
      ring.setAttribute('opacity', '0.5');
      nodeLayer.appendChild(ring);

      const dot = document.createElementNS(NS, 'circle');
      dot.setAttribute('cx', p.x); dot.setAttribute('cy', p.y);
      dot.setAttribute('r', 8);
      dot.setAttribute('fill', 'url(#nodeFill)');
      dot.setAttribute('class', 'mesh-node');
      dot.style.transformOrigin = `${p.x}px ${p.y}px`;
      dot.style.animation = `nodePulse 3s ease-in-out infinite ${i * 0.4}s`;
      nodeLayer.appendChild(dot);

      const lbl = document.createElementNS(NS, 'text');
      const off = 28;
      const ang = Math.atan2(p.y - cy, p.x - cx);
      lbl.setAttribute('x', p.x + Math.cos(ang) * off);
      lbl.setAttribute('y', p.y + Math.sin(ang) * off + 4);
      lbl.setAttribute('text-anchor', 'middle');
      lbl.setAttribute('fill', '#98a2b3');
      lbl.setAttribute('font-family', 'JetBrains Mono, monospace');
      lbl.setAttribute('font-size', '11');
      lbl.textContent = `peer-${i + 1}`;
      nodeLayer.appendChild(lbl);
    });

    // Periodic pulses traveling along random edges
    function emitPulse() {
      if (document.hidden) { setTimeout(emitPulse, 800); return; }
      const e = edges[Math.floor(Math.random() * edges.length)];
      const reverse = Math.random() > 0.5;
      const a = reverse ? e.b : e.a;
      const b = reverse ? e.a : e.b;

      const p = document.createElementNS(NS, 'circle');
      p.setAttribute('r', '3');
      p.setAttribute('fill', '#22d3ee');
      p.setAttribute('opacity', '0.95');
      p.style.filter = 'drop-shadow(0 0 6px #22d3ee)';
      pulseLayer.appendChild(p);

      const dur = 600 + Math.random() * 500;
      const start = performance.now();
      function step(t) {
        const k = Math.min(1, (t - start) / dur);
        const x = a.x + (b.x - a.x) * k;
        const y = a.y + (b.y - a.y) * k;
        p.setAttribute('cx', x);
        p.setAttribute('cy', y);
        p.setAttribute('opacity', String(0.95 * (1 - k * 0.5)));
        if (k < 1) requestAnimationFrame(step);
        else p.remove();
      }
      requestAnimationFrame(step);

      // Highlight the edge briefly.
      e.line.setAttribute('opacity', '0.9');
      e.line.setAttribute('stroke-width', '1.7');
      setTimeout(() => {
        e.line.setAttribute('opacity', '0.35');
        e.line.setAttribute('stroke-width', '1');
      }, dur);

      setTimeout(emitPulse, 220 + Math.random() * 380);
    }
    setTimeout(emitPulse, 600);
  }

  // ---------------------------------------------------------------
  // 8. Tagline typewriter effect (h1 sub-line)
  // ---------------------------------------------------------------
  const phrases = [
    'WebRTC mesh.',
    "Server o'rtada yo'q.",
    "DTLS + secretbox.",
    "16 ta peer, to'liq mesh.",
    "Chat, fayl, port tunnel.",
  ];
  const tw = document.getElementById('typewriter');
  if (tw) {
    let pi = 0, ci = 0, deleting = false;
    function step() {
      const word = phrases[pi];
      if (!deleting) {
        ci++;
        tw.textContent = word.slice(0, ci);
        if (ci === word.length) { deleting = true; setTimeout(step, 1700); return; }
        setTimeout(step, 55 + Math.random() * 40);
      } else {
        ci--;
        tw.textContent = word.slice(0, ci);
        if (ci === 0) { deleting = false; pi = (pi + 1) % phrases.length; setTimeout(step, 250); return; }
        setTimeout(step, 25);
      }
    }
    setTimeout(step, 1100);
  }

  // ---------------------------------------------------------------
  // 9. Smooth-scroll for in-page anchors
  // ---------------------------------------------------------------
  document.querySelectorAll('a[href^="#"]').forEach((a) => {
    a.addEventListener('click', (e) => {
      const id = a.getAttribute('href').slice(1);
      if (!id) return;
      const target = document.getElementById(id);
      if (!target) return;
      e.preventDefault();
      target.scrollIntoView({ behavior: 'smooth', block: 'start' });
    });
  });

  // ---------------------------------------------------------------
  // 10. Header shadow on scroll
  // ---------------------------------------------------------------
  const nav = document.querySelector('nav.top');
  if (nav) {
    const onScroll = () => {
      if (window.scrollY > 4) nav.classList.add('scrolled');
      else nav.classList.remove('scrolled');
    };
    onScroll();
    window.addEventListener('scroll', onScroll, { passive: true });
  }
})();
