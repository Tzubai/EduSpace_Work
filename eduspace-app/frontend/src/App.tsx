import { useEffect, useMemo, useState } from "react";
import { api, clearToken, getToken, useLoad } from "./api";
import Auth from "./pages/Auth";
import Dashboard from "./pages/Dashboard";
import Students from "./pages/Students";
import People from "./pages/People";
import Setup from "./pages/Setup";
import Attendance from "./pages/Attendance";
import Scores from "./pages/Scores";
import Fees from "./pages/Fees";
import Announcements from "./pages/Announcements";
import Parent from "./pages/Parent";
import Landing from "./pages/Landing";
import RoleDashboard from "./pages/RoleDashboard";
import { Admissions, Timetable, StaffAttendance } from "./pages/PremiumPages";
import AdminDashboard from "./pages/AdminDashboard";
import Reports from "./pages/Reports";
import ParentMessaging from "./pages/ParentMessaging";
import { AcademicManagement } from "./pages/AcademicManagement";

export type Term = { id: string; name: string; session: string };
export type Me = {
  full_name: string; institute_name: string; role: string; institute_id?: string;
  logo_url?: string; motto?: string; address?: string; phone?: string;
  contact_email?: string; website?: string; primary_color?: string; secondary_color?: string;
};

const SCREENS: Record<string, string[]> = {
  owner: ["Dashboard", "Students", "People", "Setup", "Admissions", "Timetable", "Staff attendance", "Attendance", "Scores", "Academic management", "Fees", "Reports", "Parent messaging", "Announcements"],
  admin: ["Dashboard", "Students", "People", "Setup", "Admissions", "Timetable", "Staff attendance", "Attendance", "Scores", "Academic management", "Fees", "Reports", "Parent messaging", "Announcements"],
  bursar: ["Dashboard", "Fees", "Announcements"],
  teacher: ["Dashboard", "Timetable", "Attendance", "Scores", "Academic management", "Announcements"],
  parent: ["Dashboard", "My children", "Academic results", "Attendance history", "Fee account", "Child timetable", "Announcements", "School messages"],
  student: ["Announcements"],
};
const NAV_GROUPS = [
  { label: "OVERVIEW", pages: ["Dashboard"] },
  { label: "SCHOOL MANAGEMENT", pages: ["Students", "People", "Setup", "Admissions", "Staff attendance"] },
  { label: "ACADEMICS", pages: ["Timetable", "Attendance", "Scores", "Academic management"] },
  { label: "FINANCE", pages: ["Fees"] },
  { label: "COMMUNICATION", pages: ["Announcements", "Parent messaging", "My children"] },
  { label: "REPORTS & ANALYTICS", pages: ["Reports"] },
  { label: "CHILD PORTAL", pages: ["My children", "Academic results", "Attendance history", "Fee account", "Child timetable"] },
];
const NAV_ICON: Record<string, string> = {
  Dashboard: "⌂", Students: "♙", People: "♧", Setup: "▦", Attendance: "✓",
  Scores: "▤", Fees: "₦", Announcements: "✉", "My children": "♡",
  Admissions: "✦", Timetable: "▣", "Staff attendance": "◎", Reports: "◫", "Academic management": "◈",
  "Academic results": "▤", "Attendance history": "✓", "Fee account": "₦", "Child timetable": "▣", "School messages": "✉",
};

export default function App() {
  const [me, setMe] = useState<Me | null>(null);
  const [ready, setReady] = useState(false);
  const [page, setPage] = useState("");
  const [publicPage, setPublicPage] = useState<"home" | "login" | "register">("home");
  const [termId, setTermId] = useState("");
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [navSearch, setNavSearch] = useState("");
  const terms = useLoad<Term[]>(me && me.role !== "student" ? "/terms" : null);
  const allowedScreens = useMemo(() => (me?.role ? SCREENS[me.role] || ["Dashboard"] : ["Dashboard"]), [me?.role]);
  const visibleGroups = NAV_GROUPS.map((group) => ({
    ...group,
    pages: group.pages.filter((p) => allowedScreens.includes(p) && p.toLowerCase().includes(navSearch.toLowerCase())),
  })).filter((group) => group.pages.length > 0);

  useEffect(() => {
    if (!getToken()) { setReady(true); return; }
    api<Me>("GET", "/auth/me").then(setMe).catch(clearToken).finally(() => setReady(true));
  }, []);
  useEffect(() => { if (me) { setPage("Dashboard"); } }, [me, allowedScreens]);
  useEffect(() => { if (!termId && terms.data?.length) setTermId(terms.data[0].id); }, [terms.data, termId]);

  if (!ready) return <div className="app-loading"><div className="app-loading-mark">E</div><span>Preparing EduSpace…</span></div>;
  if (!me && publicPage === "home") return <Landing onLogin={() => setPublicPage("login")} onRegister={() => setPublicPage("register")} />;
  if (!me) return <Auth initialMode={publicPage === "register" ? "register" : "login"} onBack={() => setPublicPage("home")} onDone={() => api<Me>("GET", "/auth/me").then(setMe)} />;

  const props = { me, termId, terms: terms.data || [], reloadTerms: terms.reload };
  const navigateWorkspace = (next: string) => {
    if (next === "Dashboard") { setPage("Dashboard"); return; }
    setPage(next);
  };

  return (
    <div className={`edu-shell ${sidebarCollapsed ? "sidebar-is-collapsed" : ""}`}>
      <aside className="edu-sidebar">
        <div className="edu-brand-row">
          <div className="edu-brand-symbol">E</div>
          {!sidebarCollapsed && <div className="edu-brand-text"><strong>EduSpace</strong><span>School workspace</span></div>}
          <button className="edu-collapse" onClick={() => setSidebarCollapsed((v) => !v)} aria-label="Toggle sidebar">{sidebarCollapsed ? "›" : "‹"}</button>
        </div>
        <div className="edu-school-chip">
          {me.logo_url ? <img src={me.logo_url} alt="School logo" /> : <div className="edu-school-avatar">{me.institute_name.split(/\s+/).filter(Boolean).slice(0, 2).map((x) => x[0]).join("").toUpperCase()}</div>}
          {!sidebarCollapsed && <div className="edu-school-chip-copy"><strong>{me.institute_name}</strong><small>{me.role.replace(/_/g, " ")}</small></div>}
        </div>
        {!sidebarCollapsed && <label className="edu-nav-search"><span>⌕</span><input placeholder="Find a page…" value={navSearch} onChange={(e) => setNavSearch(e.target.value)} /></label>}
        <div className="edu-sidebar-scroll">
          {visibleGroups.map((group) => <div className="edu-nav-group" key={group.label}>
            {!sidebarCollapsed && <div className="edu-nav-label">{group.label}</div>}
            {group.pages.map((item) => <button key={item} title={sidebarCollapsed ? item : undefined} className={`edu-nav-item ${page === item ? "active" : ""}`} onClick={() => navigateWorkspace(item)}>
              <span className="edu-nav-icon">{NAV_ICON[item] || "•"}</span>{!sidebarCollapsed && <span>{item}</span>}{!sidebarCollapsed && page === item && <i className="edu-nav-active-dot" />}
            </button>)}
          </div>)}
        </div>
        {!sidebarCollapsed && <div className="edu-sidebar-footer"><div className="edu-help-icon">?</div><div><strong>Need some help?</strong><small>Explore EduSpace support</small></div><span>›</span></div>}
      </aside>
      <div className="edu-main-area">
        <header className="edu-topbar">
          <div className="edu-breadcrumb"><span>Workspace</span><b>/</b><strong>{page}</strong></div>
          <div className="edu-top-actions">
            {page !== "parent" && terms.data && terms.data.length > 0 && <label className="edu-term-select"><span>Session</span><select value={termId} onChange={(e) => setTermId(e.target.value)}>{terms.data.map((t) => <option key={t.id} value={t.id}>{t.session} · {t.name}</option>)}</select></label>}
            <button className="edu-icon-button" title="Notifications" aria-label="Notifications" onClick={() => navigateWorkspace("Announcements")}>♧<span className="notification-dot" /></button>
            <div className="edu-user-menu"><div className="edu-user-avatar">{me.full_name.split(/\s+/).slice(0, 2).map((x) => x[0]).join("").toUpperCase()}</div><div className="edu-user-copy"><strong>{me.full_name}</strong><small>{me.role}</small></div><button className="edu-user-chevron" onClick={() => { clearToken(); location.reload(); }} title="Log out">⌄</button></div>
          </div>
        </header>
        <main className="edu-page-content">
          {page === "Dashboard" && (me.role === "owner" || me.role === "admin"
            ? <AdminDashboard {...props} onNavigate={navigateWorkspace} />
            : me.role === "teacher" || me.role === "parent"
              ? <RoleDashboard {...props} onNavigate={navigateWorkspace} />
              : <Dashboard {...props} onNavigate={navigateWorkspace} />)}
          {page === "Students" && <Students {...props} />}
          {page === "People" && <People {...props} />
          }
          {page === "Setup" && <Setup {...props} />}
          {page === "Attendance" && <Attendance {...props} />}
          {page === "Scores" && <Scores {...props} />}
          {page === "Fees" && <Fees {...props} />}
          {page === "Announcements" && <Announcements {...props} />}
          {page === "Admissions" && <Admissions {...props} />}
          {page === "Timetable" && <Timetable {...props} />}
          {page === "Staff attendance" && <StaffAttendance {...props} />}
          {page === "Reports" && <Reports {...props} />}
          {page === "Parent messaging" && <ParentMessaging />}
          {page === "Academic management" && <AcademicManagement {...props} />}
          {page === "My children" && <Parent {...props} initialTab="Overview" />}
          {page === "Academic results" && <Parent {...props} initialTab="Results" />}
          {page === "Attendance history" && <Parent {...props} initialTab="Attendance" />}
          {page === "Fee account" && <Parent {...props} initialTab="Fees" />}
          {page === "Child timetable" && <Parent {...props} initialTab="Timetable" />}
          {page === "School messages" && <Parent {...props} initialTab="School messages" />}
          {!allowedScreens.includes(page) && <div className="edu-not-found"><h2>Page not available</h2><p>This page isn't available for your account.</p></div>}
        </main>
      </div>
    </div>
  );
}
