import { useEffect, useRef, useState } from "react";

type LandingProps = { onLogin: () => void; onRegister: () => void };

const features = [
  {
    eyebrow: "STUDENT ADMINISTRATION",
    title: "Every student record, organised.",
    description: "Manage admissions, student profiles, class placement and parent or guardian records from one secure school workspace.",
    points: ["Admissions and enrolment", "Student profiles and history", "Classes, arms and promotion", "Parent and guardian records"],
    icon: "01",
  },
  {
    eyebrow: "ACADEMIC MANAGEMENT",
    title: "Stay in control of the academic year.",
    description: "Coordinate sessions, terms, attendance, assessments and report cards with clear responsibilities for authorised staff.",
    points: ["Sessions and terms", "Daily attendance", "Assessments and results", "Grading and report cards"],
    icon: "02",
  },
  {
    eyebrow: "SCHOOL FINANCE",
    title: "Know what is billed, paid and outstanding.",
    description: "Keep school fee invoices, payment records, receipts and outstanding balances together for better financial visibility.",
    points: ["Fee structures and invoices", "Cash, transfer and POS records", "Part-payment tracking", "Financial summaries"],
    icon: "03",
  },
  {
    eyebrow: "TEACHERS AND STAFF",
    title: "Give every team member the right tools.",
    description: "Organise staff records and teaching responsibilities while keeping access limited to each person's role.",
    points: ["Staff profiles", "Class and subject assignments", "Role-based permissions", "Teacher workspaces"],
    icon: "04",
  },
  {
    eyebrow: "PARENT ENGAGEMENT",
    title: "Keep families informed and connected.",
    description: "Give parents a dedicated place to follow linked children's school information, updates and important notices.",
    points: ["Linked child profiles", "Attendance and result updates", "Fee information", "School announcements"],
    icon: "05",
  },
  {
    eyebrow: "REPORTS AND OVERSIGHT",
    title: "Make decisions with a clearer picture.",
    description: "Bring essential school information together so authorised leaders can review activity and follow up confidently.",
    points: ["Academic summaries", "Attendance insights", "Fee collection overview", "Accountability and activity records"],
    icon: "06",
  },
];

export default function Landing({ onLogin, onRegister }: LandingProps) {
  const [active, setActive] = useState(0);
  const [paused, setPaused] = useState(false);
  const touchStart = useRef<number | null>(null);
  useEffect(() => {
    if (paused) return;
    const timer = window.setInterval(() => setActive((current) => (current + 1) % features.length), 6500);
    return () => window.clearInterval(timer);
  }, [paused]);

  const feature = features[active];
  const move = (direction: number) => setActive((current) => (current + direction + features.length) % features.length);

  return (
    <div className="marketing">
      <header className="marketing-nav">
        <a className="brand" href="#top" aria-label="EduSpace home"><img className="brand-logo" src="/eduspace-logo.png" alt="EduSpace" /></a>
        <nav className="marketing-links" aria-label="Main navigation">
          <a href="#features">Features</a><a href="#how-it-works">How It Works</a><a href="#for-schools">For Schools</a>
        </nav>
        <div className="marketing-actions"><button className="text-button" onClick={onLogin}>Sign In</button><button className="marketing-button compact" onClick={onRegister}>Get Started</button></div>
      </header>

      <section className="hero" id="top">
        <div className="hero-content">
          <p className="eyebrow">SCHOOL MANAGEMENT, MADE CLEARER</p>
          <h1>Your School.<br /><span>Your Space. Everything in One Place.</span></h1>
          <p className="hero-copy">A simpler way to manage your school's people, academics, finances and everyday operations — all from one dedicated digital workspace.</p>
          <div className="hero-actions"><button className="marketing-button" onClick={onRegister}>Bring Your School Online <span>→</span></button><a className="outline-button" href="#features">Explore Features</a></div>
          <p className="hero-note">Built for the realities of schools in Nigeria.</p>
        </div>
        <div className="hero-preview" aria-label="Illustrative school dashboard preview">
          <div className="preview-top"><i /><i /><i /><span>School Workspace</span></div>
          <div className="preview-body"><aside><b>ES</b><span /><span /><span /><span /></aside><div className="preview-main"><small>GOOD MORNING</small><h3>Your School Overview</h3><div className="preview-stats"><div><small>Students</small><strong>1,248</strong></div><div><small>Teachers</small><strong>64</strong></div><div><small>Classes</small><strong>32</strong></div></div><div className="preview-chart"><div><b>School activity</b><small>Illustrative preview</small></div><div className="chart-bars"><i /><i /><i /><i /><i /><i /><i /></div></div></div></div>
          <p className="preview-disclaimer">Illustrative dashboard preview</p>
        </div>
      </section>

      <section className="trust-strip"><span>ONE PLATFORM</span><b>Student records</b><b>Academic operations</b><b>School finances</b><b>Parent communication</b></section>

      <section className="section feature-section" id="features">
        <div className="section-heading"><p className="eyebrow">ONE CONNECTED WORKSPACE</p><h2>Everything your school needs to run smoothly.</h2><p>Explore the tools that help your team stay organised, informed and in control.</p></div>
        <div className="feature-carousel" onMouseEnter={() => setPaused(true)} onMouseLeave={() => setPaused(false)} onFocus={() => setPaused(true)} onBlur={(event) => { if (!event.currentTarget.contains(event.relatedTarget as Node)) setPaused(false); }} onTouchStart={(event) => { touchStart.current = event.changedTouches[0]?.clientX ?? null; }} onTouchEnd={(event) => { if (touchStart.current === null) return; const delta = event.changedTouches[0].clientX - touchStart.current; if (Math.abs(delta) > 45) move(delta < 0 ? 1 : -1); touchStart.current = null; }}>
          <div className="feature-copy"><div className="feature-count">{feature.icon} <span> / 06</span></div><p className="eyebrow">{feature.eyebrow}</p><h3>{feature.title}</h3><p>{feature.description}</p><ul>{feature.points.map((point) => <li key={point}><span>✓</span>{point}</li>)}</ul><div className="carousel-controls"><button aria-label="Previous feature" onClick={() => move(-1)}>←</button><button aria-label="Next feature" onClick={() => move(1)}>→</button><div className="carousel-dots">{features.map((item, index) => <button key={item.icon} aria-label={`Show feature ${index + 1}`} aria-current={active === index} className={active === index ? "active" : ""} onClick={() => setActive(index)} />)}</div></div></div>
          <div className="feature-visual"><div className="visual-window"><div className="visual-header"><span /><span /><span /><small>EduSpace · {feature.eyebrow.toLowerCase()}</small></div><div className="visual-content"><div className="visual-sidebar"><b>ES</b><i /><i /><i /><i /></div><div className="visual-dashboard"><small>YOUR SCHOOL WORKSPACE</small><h4>{feature.title}</h4><div className="visual-metrics"><div><small>Overview</small><b>Organised</b></div><div><small>Workspace</small><b>Connected</b></div></div><div className="visual-list"><b>Key tools</b>{feature.points.map((point) => <span key={point}><i />{point}</span>)}</div></div></div></div><p className="visual-caption">A preview of how your school's workspace can be organised.</p></div>
        </div>
      </section>

      <section className="section split-section" id="for-schools"><div><p className="eyebrow">MADE FOR YOUR SCHOOL</p><h2>Your school identity.<br /><span>Your own workspace.</span></h2><p>Each school gets a dedicated workspace with its own identity, users and records. Your administrators manage your school, while teachers, bursars and parents access only the tools relevant to their roles.</p><ul className="check-list"><li>Dedicated school workspace</li><li>Role-based access for school teams</li><li>Responsive access on phones, tablets and computers</li><li>School records kept separate from other schools</li></ul><button className="marketing-button" onClick={onRegister}>Get Started <span>→</span></button></div><div className="workspace-card"><div className="workspace-card-head"><span className="school-avatar">G</span><div><b>Greenfield School</b><small>School workspace preview</small></div><span className="workspace-status">● Active</span></div><div className="workspace-card-body"><div><small>STUDENTS</small><b>1,248</b></div><div><small>TEACHERS</small><b>64</b></div><div><small>CLASSES</small><b>32</b></div></div><div className="workspace-lines"><b>School management</b><i /><i /><i /></div><small className="visual-caption">Example school identity — illustrative only</small></div></section>

      <section className="section how-section" id="how-it-works"><div className="section-heading"><p className="eyebrow">SIMPLE TO GET STARTED</p><h2>How EduSpace works.</h2><p>From registration to a dedicated workspace for your school.</p></div><div className="steps-grid">{[{n:"01",t:"Choose your plan",d:"Select the subscription option that fits your school's size and needs."},{n:"02",t:"Register your school",d:"Provide your school details and create your administrator account."},{n:"03",t:"Set up your workspace",d:"Add your school identity, staff, classes and essential records."},{n:"04",t:"Manage with confidence",d:"Use your dedicated workspace to coordinate everyday school operations."}].map((step) => <article className="step-card" key={step.n}><span>{step.n}</span><h3>{step.t}</h3><p>{step.d}</p></article>)}</div></section>

      <section className="closing-cta"><p className="eyebrow">A BETTER WAY TO RUN YOUR SCHOOL</p><h2>Give your school a space<br /><span>to work smarter.</span></h2><p>Bring your school operations together with EduSpace.</p><button className="light-button" onClick={onRegister}>Start Your School Registration <span>→</span></button></section>
      <footer className="marketing-footer"><a className="brand footer-brand" href="#top"><img className="brand-logo" src="/eduspace-logo.png" alt="EduSpace" /></a><p>Simplifying school management.</p><div><a href="#features">Features</a><a href="#how-it-works">How It Works</a><button onClick={onLogin}>Sign In</button></div><small>© {new Date().getFullYear()} EduSpace. All rights reserved.</small></footer>
    </div>
  );
}
